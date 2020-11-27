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

type BucketListingHandler struct {
    processorBuilder *builder.ProcessorBuilder
}

func NewBucketListingHandler(processorBuilder *builder.ProcessorBuilder) *BucketListingHandler {
    return &BucketListingHandler{processorBuilder: processorBuilder}
}

// ServeHTTP will handle BucketListing operation
func (bl *BucketListingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, span := trace.StartSpan(r.Context(), "BucketListingHandler.ServeHTTP")
    defer span.End()

    projectID, exist := mux.Vars(r)["project_id"]
    if !exist {
        msg := "Bad request missing parameter: project_id"
        prepareResponse(w, msg, msg, http.StatusBadRequest)
        return
    }

    var request requestobjects.BucketListRequest
    request.Project = projectID

    principal, isValid := getPrincipalOrElsePrepareFailedResponse(w, r)
    if !isValid {
        return
    }

    // business logic
    processor, err := bl.processorBuilder.ProcessorForRequestType(ctx, requestobjects.BucketListing)
    if err != nil {
        logMsg := fmt.Sprintf("Error creating new backup processor. Err: %s", err)
        respMsg := "Could not handle request"
        prepareResponse(w, logMsg, respMsg, http.StatusInternalServerError)
        return
    }
    processorArguments := bl.processorBuilder.ProcessorArgumentsForRequest(&request, principal)
    result, err := processor.Process(ctx, &processorArguments)
    if err != nil {
        logMsg := fmt.Sprintf("Error dataset listing processing backup entity. Err: %s", err)
        errMsg := fmt.Sprintf("could not handle request because of: %s", err)
        prepareResponse(w, logMsg, errMsg, http.StatusPreconditionFailed)
        return
    }

    responseBody, err := json.Marshal(result.BucketListResponse)
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
