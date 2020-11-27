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
    "reflect"
)

type AddBackupHandler struct {
    processorBuilder *builder.ProcessorBuilder
}

func NewAddBackupHandler(processorBuilder *builder.ProcessorBuilder) *AddBackupHandler {
    return &AddBackupHandler{processorBuilder: processorBuilder}
}

// HandleAddBackup will handle Creating operation
func (dl *AddBackupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, span := trace.StartSpan(r.Context(), "AddBackupHandler.ServeHTTP")
    defer span.End()

    bodyBytes, err := ioutil.ReadAll(r.Body)
    if !checkRequestBodyIsValid(w, err) {
        return
    }

    var request requestobjects.CreateRequest
    err = json.Unmarshal(bodyBytes, &request)
    body := string(bodyBytes)
    if !checkParsingBodyIsValid(w, err, body) {
        return
    }
    if !validateCreateRequest(w, request, body) {
        return
    }

    principal, isValid := getPrincipalOrElsePrepareFailedResponse(w, r)
    if !isValid {
        return
    }

    // business logic
    processor, err := dl.processorBuilder.ProcessorForRequestType(ctx, requestobjects.Creating)
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
    if backup == nil || reflect.ValueOf(backup).IsNil() {
        logMsg := "backup was not created"
        prepareResponse(w, logMsg, logMsg, http.StatusInternalServerError)
        return
    }

    backupResponse := mapBackupToResponse(backup, []*repository.Job{})
    responseBody, err := json.Marshal(&backupResponse)
    if err != nil {
        logMsg := fmt.Sprintf("Error creating response body. Err: %s", err)
        respMsg := "Could not handle request"
        prepareResponse(w, logMsg, respMsg, http.StatusInternalServerError)
        return
    }

    msg := fmt.Sprintf("Backup with id %s successfully created", backup.ID)
    glog.Info(msg)
    w.WriteHeader(http.StatusCreated)
    _, err = w.Write(responseBody)
    if err != nil {
        logMsg := fmt.Sprintf("Error creating response body. Err: %s", err)
        respMsg := "Could not handle request"
        prepareResponse(w, logMsg, respMsg, http.StatusInternalServerError)
        return
    }
}

func getUnsetMandatoryFields(request requestobjects.CreateRequest) []string {
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

func checkMandatoryFieldsAreSet(w http.ResponseWriter, unsetMandatoryFields []string, body string) bool {
    if len(unsetMandatoryFields) > 0 {
        logMsg := fmt.Sprintf("Error request does not contain all mandatory fields\n body: %s", body)
        respMsg := "Request does not contain all mandatory fields:\n"
        for _, field := range unsetMandatoryFields {
            respMsg += field + "\n"
        }
        prepareResponse(w, logMsg, respMsg, http.StatusBadRequest)

        return false
    }

    return true
}

func checkStrategyIsValid(w http.ResponseWriter, strategy string, body string) bool {
    validStrategy := false
    for _, s := range repository.Strategies {
        if s.EqualTo(strategy) {
            validStrategy = true
            break
        }
    }

    if !validStrategy {
        logMsg := fmt.Sprintf("Error unknown strategy %s\n body: %s", strategy, body)
        respMsg := "Provided unknown strategy: " + strategy
        prepareResponse(w, logMsg, respMsg, http.StatusBadRequest)
        return false
    }

    return true
}

func checkTypeIsValid(w http.ResponseWriter, backupType string, body string) bool {
    validType := false
    for _, t := range repository.BackupTypes {
        if t.EqualTo(backupType) {
            validType = true
            break
        }
    }

    if !validType {
        logMsg := fmt.Sprintf("Error unknown backup type %s\n body: %s", backupType, body)
        respMsg := "Provided unknown backup type: " + backupType
        prepareResponse(w, logMsg, respMsg, http.StatusBadRequest)
        return false
    }

    return true
}

func checkRegionIsValid(w http.ResponseWriter, region string, body string) bool {
    validRegion := false
    for _, r := range repository.Regions {
        if r.EqualTo(region) {
            validRegion = true
            break
        }
    }

    if !validRegion {
        logMsg := fmt.Sprintf("Error invalid region %s\n body: %s", region, body)
        respMsg := "Provided invalid region: " + region
        prepareResponse(w, logMsg, respMsg, http.StatusBadRequest)
        return false
    }

    return true
}

func checkStorageClassIsValid(w http.ResponseWriter, storageClass string, body string) bool {
    if storageClass == "" { //this will fall back to default
        return true
    }

    validRegion := false

    for _, r := range repository.StorageClasses {
        if r.EqualTo(storageClass) {
            validRegion = true
            break
        }
    }

    if !validRegion {
        logMsg := fmt.Sprintf("Error invalid storage class %s\n body: %s", storageClass, body)
        respMsg := "Provided invalid storage class: " + storageClass
        prepareResponse(w, logMsg, respMsg, http.StatusBadRequest)
        return false
    }

    return true
}

func checkSourceOptionsAreValid(w http.ResponseWriter, request requestobjects.CreateRequest) bool {
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

func validateCreateRequest(w http.ResponseWriter, request requestobjects.CreateRequest, body string) bool {
    if !checkMandatoryFieldsAreSet(w, getUnsetMandatoryFields(request), body) {
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

    if !checkSourceOptionsAreValid(w, request) {
        return false
    }
    return true
}
