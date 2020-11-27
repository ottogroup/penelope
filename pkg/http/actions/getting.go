package actions

import (
    "encoding/json"
    "fmt"
    "github.com/golang/glog"
    "github.com/gorilla/mux"
    "github.com/ottogroup/penelope/pkg/builder"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "go.opencensus.io/trace"
    "net/http"
    "strconv"
)

type GettingBackupHandler struct {
	processorBuilder *builder.ProcessorBuilder
}

func NewGettingBackupHandler(processorBuilder *builder.ProcessorBuilder) *GettingBackupHandler {
	return &GettingBackupHandler{processorBuilder: processorBuilder}
}

// ServeHTTP will handle Getting operation
func (dl *GettingBackupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "GettingBackupHandler.ServeHTTP")
	defer span.End()

	backupID, ok := mux.Vars(r)["backup_id"]
	if !ok {
        msg := "Bad request missing parameter: backup_id"
        prepareResponse(w, msg, msg, http.StatusBadRequest)
		return
	}

	request := requestobjects.GetRequest{}
	request.BackupID = backupID
	q := r.URL.Query()
	if q.Get("size") != "" {
		i, err := strconv.Atoi(q.Get("size"))
		if err != nil {
			BadRequestResponse(w, r)
			return
		}
		request.Page.Size = i
	}
	if q.Get("page") != "" {
		i, err := strconv.Atoi(q.Get("page"))
		if err != nil {
			BadRequestResponse(w, r)
            return
		}
		request.Page.Number = i
	}

    principal, isValid := getPrincipalOrElsePrepareFailedResponse(w, r)
    if !isValid {
        return
    }

	// business logic
	processor, err := dl.processorBuilder.ProcessorForRequestType(ctx, requestobjects.Getting)
	if err != nil {
		logMsg := fmt.Sprintf("Error creating new backup processor. Err: %s", err)
		respMsg := "Could not handle request"
		prepareResponse(w, logMsg, respMsg, http.StatusInternalServerError)
		return
	}

	processorArguments := dl.processorBuilder.ProcessorArgumentsForRequest(&request, principal)
	result, err := processor.Process(ctx, &processorArguments)
	if err != nil {
		logMsg := fmt.Sprintf("Error creating processing backup entity. Err: %s", err)
		errMsg := fmt.Sprintf("could not handle request because of: %s", err)
		prepareResponse(w, logMsg, errMsg, http.StatusPreconditionFailed)
		return
	}

	backup := result.GetBackup()
	if !checkBackupIsFound(w, backup, request.BackupID) {
		return
	}
	backupResponse := mapBackupToResponse(backup, result.GetJobs())
	backupResponse.JobsTotal = result.JobsTotal
	responseBody, err := json.Marshal(&backupResponse)
	if err != nil {
		logMsg := fmt.Sprintf("Error creating response body. Err: %s", err)
		respMsg := "Could not handle request"
		prepareResponse(w, logMsg, respMsg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(responseBody)
	if err != nil {
		logMsg := fmt.Sprintf("Error creating response body. Err: %s", err)
		respMsg := "Could not handle request"
		prepareResponse(w, logMsg, respMsg, http.StatusInternalServerError)
		return
	}
}

func BadRequestResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	if _, err := fmt.Fprintf(w, "Unkown api endpoint %s", r.URL.Path); err != nil {
		glog.Warningf("Error writing response for %s: %s", r.URL.Path, err)
	}
}
