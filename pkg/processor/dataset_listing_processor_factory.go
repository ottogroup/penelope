package processor

import (
	"context"
	"fmt"

	"github.com/ottogroup/penelope/pkg/http/auth"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"github.com/ottogroup/penelope/pkg/service/bigquery"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type DatasetListingProcessorFactory interface {
	CreateProcessor(ctxIn context.Context) (Operation[requestobjects.DatasetListRequest, requestobjects.DatasetListResponse], error)
}

// DatasetListingProcessorFactory create Process for DatasetListing
type datasetListingProcessorFactory struct {
	backupProvider      provider.SinkGCPProjectProvider
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func NewDatasetListingProcessorFactory(backupProvider provider.SinkGCPProjectProvider, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) DatasetListingProcessorFactory {
	return &datasetListingProcessorFactory{backupProvider: backupProvider, tokenSourceProvider: tokenSourceProvider}
}

// CreateProcessor return instance of Operations for DatasetListing
func (c *datasetListingProcessorFactory) CreateProcessor(ctxIn context.Context) (Operation[requestobjects.DatasetListRequest, requestobjects.DatasetListResponse], error) {
	_, span := trace.StartSpan(ctxIn, "(*DatasetListingProcessorFactory).CreateProcessor")
	defer span.End()

	return &datasetListingProcessor{
		backupProvider:      c.backupProvider,
		tokenSourceProvider: c.tokenSourceProvider,
	}, nil
}

type datasetListingProcessor struct {
	backupProvider      provider.SinkGCPProjectProvider
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

// Process request
func (l datasetListingProcessor) Process(ctxIn context.Context, args *Argument[requestobjects.DatasetListRequest]) (requestobjects.DatasetListResponse, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*datasetListingProcessor).Process")
	defer span.End()

	var request requestobjects.DatasetListRequest = args.Request

	sourceProject := request.Project
	targetProject, err := l.backupProvider.GetSinkGCPProjectID(ctx, sourceProject)
	if err != nil {
		return requestobjects.DatasetListResponse{}, err
	}

	if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.DatasetListing, sourceProject) {
		return requestobjects.DatasetListResponse{}, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.DatasetListing.String(), args.Principal.User.Email, sourceProject)
	}

	bigQueryClient, err := bigquery.NewBigQueryClient(ctx, l.tokenSourceProvider, sourceProject, targetProject)
	if err != nil {
		return requestobjects.DatasetListResponse{}, errors.Wrapf(err, "NewBigQueryClient failed source/target %s/%s", sourceProject, targetProject)
	}
	dataSets, err := bigQueryClient.GetDatasets(ctx, sourceProject)
	if err != nil {
		return requestobjects.DatasetListResponse{}, errors.Wrap(err, "GetDatasets failed")
	}
	var datasetListResponse = requestobjects.DatasetListResponse{}
	if dataSets != nil {
		datasetListResponse.Datasets = dataSets
	} else {
		datasetListResponse.Datasets = []string{}
	}

	return datasetListResponse, err
}
