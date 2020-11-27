package actions

import (
    "encoding/json"
    "fmt"
    "go.opencensus.io/trace"
    "net/http"
)

type GetUserMeHandler struct {
}

func NewGetUserMeHandler() *GetUserMeHandler {
    return &GetUserMeHandler{}
}

// ServeHTTP check user principal after authentication
func (g *GetUserMeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    _, span := trace.StartSpan(r.Context(), "GetUserMeHandler.ServeHTTP")
    defer span.End()

    principal, isValid := getPrincipalOrElsePrepareFailedResponse(w, r)
    if !isValid {
        return
    }

    responseBody, err := json.Marshal(&principal)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-type", "application/json")
    w.WriteHeader(http.StatusOK)
    _, err = w.Write(responseBody)
    if err != nil {
        logMsg := fmt.Sprintf("Error creating response body. Err: %s", err)
        respMsg := "Could not handle request"
        prepareResponse(w, logMsg, respMsg, http.StatusInternalServerError)
    }
}
