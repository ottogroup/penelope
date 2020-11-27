package actions

import (
    "encoding/json"
    "fmt"
    "github.com/gorilla/mux"
    "github.com/ottogroup/penelope/pkg/builder"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "go.opencensus.io/trace"
    "net/http"
)

type RestoringBackupHandler struct {
    processorBuilder *builder.ProcessorBuilder
}

func NewRestoringBackupHandler(processorBuilder *builder.ProcessorBuilder) *RestoringBackupHandler {
    return &RestoringBackupHandler{processorBuilder: processorBuilder}
}

// ServeHTTP will handle Updating Restoring
func (rb *RestoringBackupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, span := trace.StartSpan(r.Context(), "RestoringBackupHandler.ServeHTTP")
    defer span.End()

    backupID, exist := mux.Vars(r)["backup_id"]
    if !exist {
        msg := "Bad request missing parameter: backup_id"
        prepareResponse(w, msg, msg, http.StatusBadRequest)
        return
    }

    var request requestobjects.RestoreRequest
    request.BackupID = backupID
    request.JobIDForTimestamp = r.URL.Query().Get("jobIDForTimestamp")

    principal, isValid := getPrincipalOrElsePrepareFailedResponse(w, r)
    if !isValid {
        return
    }

    // business logic
    processor, err := rb.processorBuilder.ProcessorForRequestType(ctx, requestobjects.Restoring)
    if err != nil {
        logMsg := fmt.Sprintf("Error creating new backup processor. Err: %s", err)
        respMsg := "Could not handle request"
        prepareResponse(w, logMsg, respMsg, http.StatusInternalServerError)
        return
    }

    processorArguments := rb.processorBuilder.ProcessorArgumentsForRequest(&request, principal)
    result, err := processor.Process(ctx, &processorArguments)
    if err != nil {
        logMsg := fmt.Sprintf("Error creating processing backup entity. Err: %s", err)
        errMsg := fmt.Sprintf("could not handle request because of: %s", err)
        prepareResponse(w, logMsg, errMsg, http.StatusPreconditionFailed)
        return
    }

    backup := result.GetBackup()
    if !checkBackupIsFound(w, backup, backupID) {
        return
    }
    restoreResponse := mapToRestoreResponse(backup, result.GetJobs())
    responseBody, err := json.Marshal(&restoreResponse)
    if err != nil {
        logMsg := fmt.Sprintf("Error restoring response body. Err: %s", err)
        respMsg := "Could not handle request"
        prepareResponse(w, logMsg, respMsg, http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _, err = w.Write(responseBody)
    if err != nil {
        logMsg := fmt.Sprintf("Error restoring response body. Err: %s", err)
        respMsg := "Could not handle request"
        prepareResponse(w, logMsg, respMsg, http.StatusInternalServerError)
        return
    }
}
