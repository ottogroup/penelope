package processor

import (
	"context"

	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
)

type ConfigRegionsProcessorFactory interface {
	CreateProcessor(ctxIn context.Context) (Operation[requestobjects.EmptyRequest, requestobjects.RegionsListResponse], error)
}

// configRegionsProcessorFactory create Process for list regions
type configRegionsProcessorFactory struct {
}

func NewConfigRegionsProcessorFactory() ConfigRegionsProcessorFactory {
	return &configRegionsProcessorFactory{}
}

// CreateProcessor return instance of Operations for list regions
func (c *configRegionsProcessorFactory) CreateProcessor(ctxIn context.Context) (Operation[requestobjects.EmptyRequest, requestobjects.RegionsListResponse], error) {
	return &configRegionsProcessor{}, nil
}

type configRegionsProcessor struct {
}

// Process request
func (l configRegionsProcessor) Process(ctxIn context.Context, args *Argument[requestobjects.EmptyRequest]) (requestobjects.RegionsListResponse, error) {
	_, span := trace.StartSpan(ctxIn, "(bucketListingProcessor).Process")
	defer span.End()

	var regions []string
	for _, region := range Regions {
		regions = append(regions, region.String())
	}

	return requestobjects.RegionsListResponse{
		Regions: regions,
	}, nil
}
