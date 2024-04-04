package actions

import (
	"net/http"

	"github.com/ottogroup/penelope/pkg/builder"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
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

	handleRequestByProcessor(ctx, w, r, request, http.StatusOK, dl.processorBuilder.ProcessorForListing)
}
