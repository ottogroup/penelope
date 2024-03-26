package processor

import (
	"context"
	"fmt"

	"github.com/ottogroup/penelope/pkg/http/auth"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type BucketListingProcessorFactory interface {
	CreateProcessor(ctxIn context.Context) (Operation[requestobjects.BucketListRequest, requestobjects.BucketListResponse], error)
}

// bucketListingProcessorFactory create Process for BucketListing
type bucketListingProcessorFactory struct {
	backupProvider      provider.SinkGCPProjectProvider
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func NewBucketListingProcessorFactory(backupProvider provider.SinkGCPProjectProvider, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) BucketListingProcessorFactory {
	return &bucketListingProcessorFactory{
		backupProvider:      backupProvider,
		tokenSourceProvider: tokenSourceProvider,
	}
}

// CreateProcessor return instance of Operations for BucketListing
func (c *bucketListingProcessorFactory) CreateProcessor(ctxIn context.Context) (Operation[requestobjects.BucketListRequest, requestobjects.BucketListResponse], error) {
	return &bucketListingProcessor{
		backupProvider:      c.backupProvider,
		tokenSourceProvider: c.tokenSourceProvider,
	}, nil
}

type bucketListingProcessor struct {
	backupProvider      provider.SinkGCPProjectProvider
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

// Process request
func (l bucketListingProcessor) Process(ctxIn context.Context, args *Argument[requestobjects.BucketListRequest]) (requestobjects.BucketListResponse, error) {
	ctx, span := trace.StartSpan(ctxIn, "(bucketListingProcessor).Process")
	defer span.End()

	var request *requestobjects.BucketListRequest = &args.Request

	sourceProject := request.Project
	targetProject, err := l.backupProvider.GetSinkGCPProjectID(ctx, request.Project)
	if err != nil {
		return requestobjects.BucketListResponse{}, err
	}

	if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.BucketListing, sourceProject) {
		return requestobjects.BucketListResponse{}, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.BucketListing.String(), args.Principal.User.Email, sourceProject)
	}

	cloudStorageClient, err := gcs.NewCloudStorageClient(ctx, l.tokenSourceProvider, targetProject)
	if err != nil {
		return requestobjects.BucketListResponse{}, errors.Wrapf(err, "NewCloudStorageClient failed for project %s", targetProject)
	}
	buckets, err := cloudStorageClient.GetBuckets(ctx, sourceProject)
	if err != nil {
		return requestobjects.BucketListResponse{}, errors.Wrapf(err, "GetBuckets failed for source project %s", sourceProject)
	}
	defer cloudStorageClient.Close(ctx)
	var bucketListResponse = requestobjects.BucketListResponse{}
	bucketListResponse.Buckets = buckets

	return bucketListResponse, err
}
