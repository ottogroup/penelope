package actions

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ottogroup/penelope/pkg/builder"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
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

	handleRequestByProcessor(ctx, w, r, request, http.StatusOK, bl.processorBuilder.ProcessorForBucketListing)
}
