package actions

import (
	"net/http"

	"github.com/ottogroup/penelope/pkg/builder"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opentelemetry.io/otel"
)

type ConfigStorageClassesHandler struct {
	processorBuilder *builder.ProcessorBuilder
}

func NewConfigStorageClassesHandler(processorBuilder *builder.ProcessorBuilder) *ConfigStorageClassesHandler {
	return &ConfigStorageClassesHandler{processorBuilder: processorBuilder}
}

// ServeHTTP will handle BucketListing operation
func (bl *ConfigStorageClassesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("").Start(r.Context(), "ConfigRegionsHandler.ServeHTTP")
	defer span.End()

	var request requestobjects.EmptyRequest

	handleRequestByProcessor(ctx, w, r, request, http.StatusOK, bl.processorBuilder.ProcessorForConfigStorageClasses)
}
