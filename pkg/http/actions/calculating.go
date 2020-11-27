package actions

import (
    "encoding/json"
    "fmt"
    "github.com/ottogroup/penelope/pkg/builder"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "go.opencensus.io/trace"
    "io/ioutil"
    "net/http"
)

type CalculateBackupHandler struct {
    processorBuilder *builder.ProcessorBuilder
}

func NewCalculateBackupHandler(processorBuilder *builder.ProcessorBuilder) *CalculateBackupHandler {
    return &CalculateBackupHandler{processorBuilder: processorBuilder}
}

// ServeHTTP will handle Calculating operation
func (dl *CalculateBackupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, span := trace.StartSpan(r.Context(), "CalculateBackupHandler.ServeHTTP")
    defer span.End()

    bodyBytes, err := ioutil.ReadAll(r.Body)
    if !checkRequestBodyIsValid(w, err) {
        return
    }

    var request requestobjects.CalculateRequest
    err = json.Unmarshal(bodyBytes, &request)
    body := string(bodyBytes)
    if !checkParsingBodyIsValid(w, err, body) {
        return
    }
    if !validateCancelRequest(w, request, body) {
        return
    }

    principal, isValid := getPrincipalOrElsePrepareFailedResponse(w, r)
    if !isValid {
        return
    }

    // business logic
    processor, err := dl.processorBuilder.ProcessorForRequestType(ctx, requestobjects.Calculating)
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

    responseBody, err := json.Marshal(result.CalculateResponse)
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

func validateCancelRequest(w http.ResponseWriter, request requestobjects.CalculateRequest, body string) bool {
    if !checkMandatoryFieldsAreSet(w, getUnsetMandatoryFieldsForCalculateRequest(request), body) {
        return false
    }

    if !checkStrategyIsValid(w, request.Strategy, body) {
        return false
    }

    if !checkTypeIsValid(w, request.Type, body) {
        return false
    }

    if !checkRegionIsValid(w, request.TargetOptions.Region, body) {
        return false
    }

    if !checkStorageClassIsValid(w, request.TargetOptions.StorageClass, body) {
        return false
    }

    if !checkSourceOptionsAreValidForCalculateRequest(w, request) {
        return false
    }
    return true
}

func getUnsetMandatoryFieldsForCalculateRequest(request requestobjects.CalculateRequest) []string {
    var unsetMandatoryFields []string

    if request.TargetOptions.Region == "" {
        unsetMandatoryFields = append(unsetMandatoryFields, "region")
    } else if request.Type == "" {
        unsetMandatoryFields = append(unsetMandatoryFields, "type")
    } else if request.Strategy == "" {
        unsetMandatoryFields = append(unsetMandatoryFields, "strategy")
    } else if request.Project == "" {
        unsetMandatoryFields = append(unsetMandatoryFields, "project")
    }

    return unsetMandatoryFields
}

func checkSourceOptionsAreValidForCalculateRequest(w http.ResponseWriter, request requestobjects.CalculateRequest) bool {
    if repository.BigQuery.EqualTo(request.Type) && request.BigQueryOptions.Dataset == "" {
        logMsg := "Error bigquery backup type missing mandatory dataset field"
        respMsg := "Missing mandatory bigquery dataset name"
        prepareResponse(w, logMsg, respMsg, http.StatusBadRequest)
        return false
    } else if repository.CloudStorage.EqualTo(request.Type) && request.GCSOptions.Bucket == "" {
        logMsg := "Error cloudstorage backup type missing mandatory bucket field"
        respMsg := "Missing mandatory cloudstorage bucket name"
        prepareResponse(w, logMsg, respMsg, http.StatusBadRequest)
        return false
    }

    return true
}
