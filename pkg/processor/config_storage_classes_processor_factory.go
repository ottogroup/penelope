package processor

import (
	"context"

	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
)

type ConfigStorageClassesProcessorFactory interface {
	CreateProcessor(ctxIn context.Context) (Operation[requestobjects.EmptyRequest, requestobjects.StorageClassListResponse], error)
}

// configStorageClassesProcessorFactory create Process for list storage classes
type configStorageClassesProcessorFactory struct {
}

func NewConfigStorageClassesProcessorFactory() ConfigStorageClassesProcessorFactory {
	return &configStorageClassesProcessorFactory{}
}

// CreateProcessor return instance of Operations for list storage classes
func (c *configStorageClassesProcessorFactory) CreateProcessor(ctxIn context.Context) (Operation[requestobjects.EmptyRequest, requestobjects.StorageClassListResponse], error) {
	return &configStorageClassesProcessor{}, nil
}

type configStorageClassesProcessor struct {
}

// Process request
func (l configStorageClassesProcessor) Process(ctxIn context.Context, args *Argument[requestobjects.EmptyRequest]) (requestobjects.StorageClassListResponse, error) {
	_, span := trace.StartSpan(ctxIn, "(bucketListingProcessor).Process")
	defer span.End()

	var classes []string
	for _, class := range StorageClasses {
		classes = append(classes, class.String())
	}

	return requestobjects.StorageClassListResponse{
		StorageClasses: classes,
	}, nil
}
