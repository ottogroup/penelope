package actions

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ottogroup/penelope/pkg/builder"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
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

	bodyBytes, err := io.ReadAll(r.Body)
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
	handleRequestByProcessor(ctx, w, r, request, http.StatusOK, dl.processorBuilder.ProcessorForUpdating)
}
