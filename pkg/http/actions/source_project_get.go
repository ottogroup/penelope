package actions

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ottogroup/penelope/pkg/builder"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
)

type SourceProjectGetHandler struct {
	processorBuilder *builder.ProcessorBuilder
}

func NewSourceProjectHandler(processorBuilder *builder.ProcessorBuilder) *SourceProjectGetHandler {
	return &SourceProjectGetHandler{processorBuilder: processorBuilder}
}

// ServeHTTP will handle SourceProject operation
func (bl *SourceProjectGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "SourceProjectGetHandler.ServeHTTP")
	defer span.End()

	projectID, exist := mux.Vars(r)["project_id"]
	if !exist {
		msg := "Bad request missing parameter: project_id"
		prepareResponse(w, msg, msg, http.StatusBadRequest)
		return
	}

	var request requestobjects.SourceProjectGetRequest
	request.Project = projectID

	handleRequestByProcessor(ctx, w, r, request, http.StatusOK, bl.processorBuilder.ProcessorForSourceProjectGet)
}
