package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/http/auth"
	"github.com/ottogroup/penelope/pkg/http/auth/model"
	"github.com/ottogroup/penelope/pkg/processor"
	"github.com/ottogroup/penelope/pkg/repository"
)

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func checkRequestBodyIsValid(w http.ResponseWriter, err error) bool {
	if err != nil {
		logMsg := fmt.Sprintf("Error reading request body. Err: %s", err)
		respMsg := "Could not read body of request"
		prepareResponse(w, logMsg, respMsg, http.StatusUnprocessableEntity)
		return false
	}

	return true
}

func getPrincipalOrElsePrepareFailedResponse(w http.ResponseWriter, r *http.Request) (*model.Principal, bool) {
	principal, ok := r.Context().Value(auth.CtxPrincipalKey).(*model.Principal)
	if !ok || principal == nil {
		prepareResponse(w, "no principal found in context", "could not retrieve user-info", http.StatusInternalServerError)
		return nil, false
	}
	return principal, true
}

func checkBackupIsFound(w http.ResponseWriter, backup *repository.Backup, backupID string) bool {
	if backup == nil || reflect.ValueOf(backup).IsNil() {
		logMsg := fmt.Sprintf("no backup with id %q found", backupID)
		prepareResponse(w, logMsg, logMsg, http.StatusNotFound)
		return false
	}
	return true
}

func checkParsingBodyIsValid(w http.ResponseWriter, err error, body string) bool {
	if err != nil {
		logMsg := fmt.Sprintf("Error parsing json request body. Err: %s\n body: %s", err, body)
		respMsg := "Could not parsing json request body of request"
		prepareResponse(w, logMsg, respMsg, http.StatusUnprocessableEntity)
		return false
	}

	return true
}

func prepareResponse(w http.ResponseWriter, logMsg string, responseMsg string, responseCode int) {
	glog.Info(logMsg)
	w.WriteHeader(responseCode)
	if _, err := fmt.Fprint(w, responseMsg); err != nil {
		glog.Warningf("Error writing response: %s", err)
	}
}

func handleRequestByProcessor[T, R any](ctx context.Context, w http.ResponseWriter, r *http.Request, request T, processorBuilder func(context.Context) (processor.Operation[T, R], error)) {
	principal, isValid := getPrincipalOrElsePrepareFailedResponse(w, r)
	if !isValid {
		return
	}

	// business logic
	p, err := processorBuilder(ctx)
	if err != nil {
		logMsg := fmt.Sprintf("Error creating new backup processor. Err: %s", err)
		respMsg := "Could not handle request"
		prepareResponse(w, logMsg, respMsg, http.StatusInternalServerError)
		return
	}
	args := processor.Argument[T]{
		Request:   request,
		Principal: principal,
	}
	result, err := p.Process(ctx, &args)
	if err != nil {
		logMsg := fmt.Sprintf("Error dataset listing processing backup entity. Err: %s", err)
		errMsg := fmt.Sprintf("could not handle request because of: %s", err)
		prepareResponse(w, logMsg, errMsg, http.StatusPreconditionFailed)
		return
	}

	responseBody, err := json.Marshal(result)
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
