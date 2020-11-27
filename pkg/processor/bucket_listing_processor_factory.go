package processor

import (
    "context"
    "fmt"
    "github.com/pkg/errors"
    "github.com/ottogroup/penelope/pkg/http/auth"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "github.com/ottogroup/penelope/pkg/provider"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "github.com/ottogroup/penelope/pkg/service/gcs"
    "go.opencensus.io/trace"
)

// BucketListingProcessorFactory create Process for BucketListing
type BucketListingProcessorFactory struct {
    backupProvider      provider.SinkGCPProjectProvider
    tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func NewBucketListingProcessorFactory(backupProvider provider.SinkGCPProjectProvider, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) *BucketListingProcessorFactory {
    return &BucketListingProcessorFactory{
        backupProvider:      backupProvider,
        tokenSourceProvider: tokenSourceProvider,
    }
}

// DoMatchRequestType does request type match BucketListing
func (c *BucketListingProcessorFactory) DoMatchRequestType(requestType requestobjects.RequestType) bool {
    return requestobjects.BucketListing.EqualTo(requestType.String())
}

// CreateProcessor return instance of Operations for BucketListing
func (c *BucketListingProcessorFactory) CreateProcessor(ctxIn context.Context) (Operations, error) {
    processor, err := c.newBucketListingProcessor()
    if err != nil {
        return nil, err
    }

    return processor, nil
}

type bucketListingProcessor struct {
    backupProvider      provider.SinkGCPProjectProvider
    tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func (c *BucketListingProcessorFactory) newBucketListingProcessor() (*bucketListingProcessor, error) {
    return &bucketListingProcessor{
        backupProvider: c.backupProvider,
        tokenSourceProvider: c.tokenSourceProvider,
    }, nil
}

// Process request
func (l bucketListingProcessor) Process(ctxIn context.Context, args *Arguments) (*Result, error) {
    ctx, span := trace.StartSpan(ctxIn, "(bucketListingProcessor).Process")
    defer span.End()

    var request *requestobjects.BucketListRequest
    if args.Request == nil {
        return nil, fmt.Errorf("nil request object for processing bucket list request")
    }
    request, ok := args.Request.(*requestobjects.BucketListRequest)
    if !ok {
        return nil, fmt.Errorf("wrong request object for processing bucket list request")
    }

    sourceProject := request.Project
    targetProject, err := l.backupProvider.GetSinkGCPProjectID(ctx, request.Project)
    if err != nil {
        return nil, err
    }

    if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.BucketListing, sourceProject) {
        return nil, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.BucketListing.String(), args.Principal.User.Email, sourceProject)
    }

    cloudStorageClient, err := gcs.NewCloudStorageClient(ctx, l.tokenSourceProvider, targetProject)
    if err != nil {
        return nil, errors.Wrapf(err, "NewCloudStorageClient failed for project %s", targetProject)
    }
    buckets, err := cloudStorageClient.GetBuckets(ctx, sourceProject)
    if err != nil {
        return nil, errors.Wrapf(err, "GetBuckets failed for source project %s", sourceProject)
    }
    defer cloudStorageClient.Close(ctx)
    var bucketListResponse = requestobjects.BucketListResponse{}
    bucketListResponse.Buckets = buckets

    return &Result{BucketListResponse: &bucketListResponse}, err
}
