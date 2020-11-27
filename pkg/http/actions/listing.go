package actions

import (
    "encoding/json"
    "fmt"
    "github.com/ottogroup/penelope/pkg/builder"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "go.opencensus.io/trace"
    "net/http"
)

type ListingBackupHandler struct {
    processorBuilder *builder.ProcessorBuilder
}

func NewListingBackupHandler(processorBuilder *builder.ProcessorBuilder) *ListingBackupHandler {
    return &ListingBackupHandler{processorBuilder: processorBuilder}
}

// ServeHTTP will handle Listing operation
func (dl *ListingBackupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, span := trace.StartSpan(r.Context(), "ListingBackupHandler.ServeHTTP")
    defer span.End()

    request := requestobjects.ListRequest{Project: r.URL.Query().Get("project")}

    principal, isValid := getPrincipalOrElsePrepareFailedResponse(w, r)
    if !isValid {
        return
    }

    // business logic
    processor, err := dl.processorBuilder.ProcessorForRequestType(ctx, requestobjects.Listing)
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

    listResponse := requestobjects.ListingResponse{}
    listResponse.Backups = []requestobjects.BackupResponse{}
    for _, b := range result.GetBackups() {
        listResponse.Backups = append(listResponse.Backups, mapBackupToResponse(b, []*repository.Job{}))
    }

    responseBody, err := json.Marshal(&listResponse)
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
