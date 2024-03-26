package builder

import (
	"context"
	"errors"

	"github.com/ottogroup/penelope/pkg/processor"
	"github.com/ottogroup/penelope/pkg/requestobjects"
)

// ProcessorBuilder is responsible for creating Operations for each request type
type ProcessorBuilder struct {
	creatingProcessorFactory       processor.CreatingProcessorFactory
	gettingProcessorFactory        processor.GettingProcessorFactory
	listingProcessorFactory        processor.ListingProcessorFactory
	updatingProcessorFactory       processor.UpdatingProcessorFactory
	restoringProcessorFactory      processor.RestoringProcessorFactory
	calculatingProcessorFactory    processor.CalculatingProcessorFactory
	complianceProcessorFactory     processor.ComplianceProcessorFactory
	bucketListingProcessorFactory  processor.BucketListingProcessorFactory
	datasetListingProcessorFactory processor.DatasetListingProcessorFactory
}

// NewProcessorBuilder created a new ProcessorBuilder
func NewProcessorBuilder(
	creatingProcessorFactory processor.CreatingProcessorFactory,
	gettingProcessorFactory processor.GettingProcessorFactory,
	listingProcessorFactory processor.ListingProcessorFactory,
	updatingProcessorFactory processor.UpdatingProcessorFactory,
	restoringProcessorFactory processor.RestoringProcessorFactory,
	calculatingProcessorFactory processor.CalculatingProcessorFactory,
	complianceProcessorFactory processor.ComplianceProcessorFactory,
	bucketListingProcessorFactory processor.BucketListingProcessorFactory,
	datasetListingProcessorFactory processor.DatasetListingProcessorFactory,
) *ProcessorBuilder {
	return &ProcessorBuilder{
		creatingProcessorFactory:       creatingProcessorFactory,
		gettingProcessorFactory:        gettingProcessorFactory,
		listingProcessorFactory:        listingProcessorFactory,
		updatingProcessorFactory:       updatingProcessorFactory,
		restoringProcessorFactory:      restoringProcessorFactory,
		calculatingProcessorFactory:    calculatingProcessorFactory,
		complianceProcessorFactory:     complianceProcessorFactory,
		bucketListingProcessorFactory:  bucketListingProcessorFactory,
		datasetListingProcessorFactory: datasetListingProcessorFactory,
	}
}

func (p *ProcessorBuilder) ProcessorForCreating(ctx context.Context) (processor.Operation[requestobjects.CreateRequest, requestobjects.BackupResponse], error) {
	if p.creatingProcessorFactory == nil {
		return nil, errors.New("factory not found")
	}
	return p.creatingProcessorFactory.CreateProcessor(ctx)
}

func (p *ProcessorBuilder) ProcessorForGetting(ctx context.Context) (processor.Operation[requestobjects.GetRequest, requestobjects.BackupResponse], error) {
	if p.gettingProcessorFactory == nil {
		return nil, errors.New("factory not found")
	}
	return p.gettingProcessorFactory.CreateProcessor(ctx)
}

func (p *ProcessorBuilder) ProcessorForListing(ctx context.Context) (processor.Operation[requestobjects.ListRequest, requestobjects.ListingResponse], error) {
	if p.listingProcessorFactory == nil {
		return nil, errors.New("factory not found")
	}
	return p.listingProcessorFactory.CreateProcessor(ctx)
}

func (p *ProcessorBuilder) ProcessorForUpdating(ctx context.Context) (processor.Operation[requestobjects.UpdateRequest, requestobjects.UpdateResponse], error) {
	if p.updatingProcessorFactory == nil {
		return nil, errors.New("factory not found")
	}
	return p.updatingProcessorFactory.CreateProcessor(ctx)
}

func (p *ProcessorBuilder) ProcessorForRestoring(ctx context.Context) (processor.Operation[requestobjects.RestoreRequest, requestobjects.RestoreResponse], error) {
	if p.restoringProcessorFactory == nil {
		return nil, errors.New("factory not found")
	}
	return p.restoringProcessorFactory.CreateProcessor(ctx)
}

func (p *ProcessorBuilder) ProcessorForCalclating(ctx context.Context) (processor.Operation[requestobjects.CalculateRequest, requestobjects.CalculatedResponse], error) {
	if p.calculatingProcessorFactory == nil {
		return nil, errors.New("factory not found")
	}
	return p.calculatingProcessorFactory.CreateProcessor(ctx)
}

func (p *ProcessorBuilder) ProcessorForCompliance(ctx context.Context) (processor.Operation[requestobjects.ComplianceRequest, requestobjects.ComplianceResponse], error) {
	if p.complianceProcessorFactory == nil {
		return nil, errors.New("factory not found")
	}
	return p.complianceProcessorFactory.CreateProcessor(ctx)
}

func (p *ProcessorBuilder) ProcessorForBucketListing(ctx context.Context) (processor.Operation[requestobjects.BucketListRequest, requestobjects.BucketListResponse], error) {
	if p.bucketListingProcessorFactory == nil {
		return nil, errors.New("factory not found")
	}
	return p.bucketListingProcessorFactory.CreateProcessor(ctx)
}

func (p *ProcessorBuilder) ProcessorForDatasetListing(ctx context.Context) (processor.Operation[requestobjects.DatasetListRequest, requestobjects.DatasetListResponse], error) {
	if p.datasetListingProcessorFactory == nil {
		return nil, errors.New("factory not found")
	}
	return p.datasetListingProcessorFactory.CreateProcessor(ctx)
}
