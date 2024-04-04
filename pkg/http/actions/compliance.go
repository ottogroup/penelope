package actions

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/ottogroup/penelope/pkg/builder"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
)

type ComplianceBackupHandler struct {
	processorBuilder *builder.ProcessorBuilder
}

func NewComplianceBackupHandler(processorBuilder *builder.ProcessorBuilder) *ComplianceBackupHandler {
	return &ComplianceBackupHandler{processorBuilder: processorBuilder}
}

// ServeHTTP will handle Calculating operation
func (dl *ComplianceBackupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "ComplianceBackupHandler.ServeHTTP")
	defer span.End()

	bodyBytes, err := io.ReadAll(r.Body)
	if !checkRequestBodyIsValid(w, err) {
		return
	}

	var request requestobjects.ComplianceRequest
	err = json.Unmarshal(bodyBytes, &request)
	body := string(bodyBytes)
	if !checkParsingBodyIsValid(w, err, body) {
		return
	}

	handleRequestByProcessor(ctx, w, r, request, http.StatusOK, dl.processorBuilder.ProcessorForCompliance)
}
