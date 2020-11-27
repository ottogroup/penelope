package actions

import (
    "encoding/json"
    "fmt"
    "github.com/golang/glog"
    "github.com/ottogroup/penelope/pkg/builder"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "go.opencensus.io/trace"
    "io/ioutil"
    "net/http"
)

type UpdateBackupHandler struct {
    processorBuilder *builder.ProcessorBuilder
}

func NewUpdateBackupHandler(processorBuilder *builder.ProcessorBuilder) *UpdateBackupHandler {
    return &UpdateBackupHandler{processorBuilder: processorBuilder}
}

// ServeHTTP will handle Updating operation
func (dl *UpdateBackupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, span := trace.StartSpan(r.Context(), "UpdateBackupHandler.ServeHTTP")
    defer span.End()

    bodyBytes, err := ioutil.ReadAll(r.Body)
    if !checkRequestBodyIsValid(w, err) {
        return
    }

    var request requestobjects.UpdateRequest
    err = json.Unmarshal(bodyBytes, &request)
    if err != nil {
        logMsg := fmt.Sprintf("Error creating new backup processor. Err: %s", err)
        respMsg := "Could not unmarshal request body"
        prepareResponse(w, logMsg, respMsg, http.StatusBadRequest)
        return
    }

    principal, isValid := getPrincipalOrElsePrepareFailedResponse(w, r)
    if !isValid {
        return
    }

    // business logic
    processor, err := dl.processorBuilder.ProcessorForRequestType(ctx, requestobjects.Updating)
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
        prepareResponse(w, logMsg, "could not handle request", http.StatusPreconditionFailed)
        return
    }
    backup := result.GetBackup()
    if !checkBackupIsFound(w, backup, request.BackupID) {
        return
    }

    updateResponse := prepareUpdateResponse(backup)
    responseBody, err := json.Marshal(&updateResponse)
    if err != nil {
        logMsg := fmt.Sprintf("Error creating response body. Err: %s", err)
        respMsg := "Could not handle request"
        prepareResponse(w, logMsg, respMsg, http.StatusInternalServerError)
        return
    }

    msg := fmt.Sprintf("Backup with id %s successfully updated", backup.ID)
    glog.Info(msg)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _, err = w.Write(responseBody)
    if err != nil {
        logMsg := fmt.Sprintf("Error creating response body. Err: %s", err)
        respMsg := "Could not handle request"
        prepareResponse(w, logMsg, respMsg, http.StatusInternalServerError)
    }
}

func prepareUpdateResponse(backup *repository.Backup) requestobjects.UpdateResponse {
    updateResponse := requestobjects.UpdateResponse{}
    updateResponse.Status = backup.Status.String()
    updateResponse.BackupID = backup.ID
    updateResponse.CreatedTimestamp = formatTime(backup.CreatedTimestamp)
    updateResponse.UpdatedTimestamp = formatTime(backup.UpdatedTimestamp)
    updateResponse.DeletedTimestamp = formatTime(backup.DeletedTimestamp)
    return updateResponse
}
