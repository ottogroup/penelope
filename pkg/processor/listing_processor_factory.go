package processor

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/http/auth"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/requestobjects"
    "github.com/ottogroup/penelope/pkg/secret"
    "go.opencensus.io/trace"
)

// ListingProcessorFactory create Process for Listing
type ListingProcessorFactory struct {
    tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
    credentialsProvider secret.SecretProvider
}

func NewListingProcessorFactory(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) *ListingProcessorFactory {
    return &ListingProcessorFactory{tokenSourceProvider, credentialsProvider}
}

// DoMatchRequestType does request type match Listing
func (c ListingProcessorFactory) DoMatchRequestType(requestType requestobjects.RequestType) bool {
	return requestobjects.Listing.EqualTo(requestType.String())
}

// CreateProcessor return instance of Operations for Listing
func (c ListingProcessorFactory) CreateProcessor(ctxIn context.Context) (Operations, error) {
	processor, err := c.newListingProcessor(ctxIn)
	if err != nil {
		return nil, err
	}

	return processor, nil
}

type listingProcessor struct {
	BackupRepository repository.BackupRepository
	Context          context.Context
}

func (c ListingProcessorFactory) newListingProcessor(ctxIn context.Context) (*listingProcessor, error) {
	ctx, span := trace.StartSpan(ctxIn, "newListingProcessor")
	defer span.End()

	backupRepository, err := repository.NewBackupRepository(ctx, c.credentialsProvider)
	if err != nil {
		glog.Error(err)
		return &listingProcessor{}, err
	}

	return &listingProcessor{BackupRepository: backupRepository}, nil
}

// Process request
func (l listingProcessor) Process(ctxIn context.Context, args *Arguments) (*Result, error) {
	ctx, span := trace.StartSpan(ctxIn, "(listingProcessor).Process")
	defer span.End()

	var request *requestobjects.ListRequest
	if args.Request == nil {
		return nil, fmt.Errorf("nil request object for processing backup listing request")
	}
	request, ok := args.Request.(*requestobjects.ListRequest)
	if !ok {
		return nil, fmt.Errorf("wrong request object for processing backup listing request")
	}

	backupFilter := repository.BackupFilter{Project: request.Project}
	backups, err := l.BackupRepository.GetBackups(ctx, backupFilter)
	var filteredBackups []*repository.Backup
	if err != nil {
		return &Result{backups: filteredBackups}, err
	}
	for _, backup := range backups {
		if auth.CheckRequestIsAllowed(args.Principal, requestobjects.Listing, backup.SourceProject) {
			filteredBackups = append(filteredBackups, backup)
		} else {
			glog.V(2).Infof("%s is not allowed for user %q on project %q", requestobjects.Listing.String(), args.Principal.User.Email, backup.TargetProject)
		}
	}
	return &Result{backups: filteredBackups}, nil
}
