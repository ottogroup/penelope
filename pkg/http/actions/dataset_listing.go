package actions

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ottogroup/penelope/pkg/builder"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
)

type DatasetListingHandler struct {
	processorBuilder *builder.ProcessorBuilder
}

func NewDatasetListingHandler(processorBuilder *builder.ProcessorBuilder) *DatasetListingHandler {
	return &DatasetListingHandler{processorBuilder: processorBuilder}
}

// ServeHTTP will handle DatasetListing operation
func (dl *DatasetListingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "DatasetListingHandler.ServeHTTP")
	defer span.End()

	projectID, exist := mux.Vars(r)["project_id"]
	if !exist {
		msg := "Bad request missing parameter: project_id"
		prepareResponse(w, msg, msg, http.StatusBadRequest)
		return
	}

	var request requestobjects.DatasetListRequest
	request.Project = projectID

	handleRequestByProcessor(ctx, w, r, request, dl.processorBuilder.ProcessorForDatasetListing)
}
