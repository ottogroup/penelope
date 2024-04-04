package actions

import (
	"net/http"

	"github.com/ottogroup/penelope/pkg/builder"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
)

type ConfigRegionsHandler struct {
	processorBuilder *builder.ProcessorBuilder
}

func NewConfigRegionsHandler(processorBuilder *builder.ProcessorBuilder) *ConfigRegionsHandler {
	return &ConfigRegionsHandler{processorBuilder: processorBuilder}
}

// ServeHTTP will handle BucketListing operation
func (bl *ConfigRegionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "ConfigRegionsHandler.ServeHTTP")
	defer span.End()

	var request requestobjects.EmptyRequest

	handleRequestByProcessor(ctx, w, r, request, http.StatusOK, bl.processorBuilder.ProcessorForConfigRegions)
}
