package processor

import (
    "context"
    "fmt"
    "github.com/pkg/errors"
    "github.com/ottogroup/penelope/pkg/http/auth"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "github.com/ottogroup/penelope/pkg/provider"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "github.com/ottogroup/penelope/pkg/service/bigquery"
    "go.opencensus.io/trace"
)

// DatasetListingProcessorFactory create Process for DatasetListing
type DatasetListingProcessorFactory struct {
    backupProvider      provider.SinkGCPProjectProvider
    tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func NewDatasetListingProcessorFactory(backupProvider provider.SinkGCPProjectProvider, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) *DatasetListingProcessorFactory {
    return &DatasetListingProcessorFactory{backupProvider: backupProvider, tokenSourceProvider: tokenSourceProvider}
}

// DoMatchRequestType does request type match Listing
func (c *DatasetListingProcessorFactory) DoMatchRequestType(requestType requestobjects.RequestType) bool {
    return requestobjects.DatasetListing.EqualTo(requestType.String())
}

// CreateProcessor return instance of Operations for DatasetListing
func (c *DatasetListingProcessorFactory) CreateProcessor(ctxIn context.Context) (Operations, error) {
    _, span := trace.StartSpan(ctxIn, "(*DatasetListingProcessorFactory).CreateProcessor")
    defer span.End()

    processor, err := c.newDatasetListingProcessor()
    if err != nil {
        return nil, err
    }

    return processor, nil
}

type datasetListingProcessor struct {
    backupProvider      provider.SinkGCPProjectProvider
    tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func (c *DatasetListingProcessorFactory) newDatasetListingProcessor() (*datasetListingProcessor, error) {
    return &datasetListingProcessor{
        backupProvider: c.backupProvider,
        tokenSourceProvider: c.tokenSourceProvider,
    }, nil
}

// Process request
func (l datasetListingProcessor) Process(ctxIn context.Context, args *Arguments) (*Result, error) {
    ctx, span := trace.StartSpan(ctxIn, "(*datasetListingProcessor).Process")
    defer span.End()

    var request *requestobjects.DatasetListRequest
    if args.Request == nil {
        return nil, fmt.Errorf("nil request object for processing dataset list request")
    }
    request, ok := args.Request.(*requestobjects.DatasetListRequest)
    if !ok {
        return nil, fmt.Errorf("wrong request object for processing dataset list request")
    }

    sourceProject := request.Project
    targetProject, err := l.backupProvider.GetSinkGCPProjectID(ctx, sourceProject)
    if err != nil {
        return nil, err
    }

    if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.DatasetListing, sourceProject) {
        return nil, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.DatasetListing.String(), args.Principal.User.Email, sourceProject)
    }

    bigQueryClient, err := bigquery.NewBigQueryClient(ctx, l.tokenSourceProvider, sourceProject, targetProject)
    if err != nil {
        return nil, errors.Wrapf(err, "NewBigQueryClient failed source/target %s/%s", sourceProject, targetProject)
    }
    dataSets, err := bigQueryClient.GetDatasets(ctx, sourceProject)
    if err != nil {
        return nil, errors.Wrap(err, "GetDatasets failed")
    }
    var datasetListResponse = requestobjects.DatasetListResponse{}
    datasetListResponse.Datasets = dataSets

    return &Result{DatasetListResponse: &datasetListResponse}, err
}
