package processor

import (
	"context"
	"github.com/ottogroup/penelope/pkg/provider"

	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/http/auth"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"github.com/ottogroup/penelope/pkg/secret"
	"go.opencensus.io/trace"
)

type ListingProcessorFactory interface {
	CreateProcessor(ctxIn context.Context) (Operation[requestobjects.ListRequest, requestobjects.ListingResponse], error)
}

// ListingProcessorFactory create Process for Listing
type listingProcessorFactory struct {
	tokenSourceProvider      impersonate.TargetPrincipalForProjectProvider
	credentialsProvider      secret.SecretProvider
	sourceGCPProjectProvider provider.SourceGCPProjectProvider
}

func NewListingProcessorFactory(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider, sourceGCPProjectProvider provider.SourceGCPProjectProvider) ListingProcessorFactory {
	return &listingProcessorFactory{tokenSourceProvider, credentialsProvider, sourceGCPProjectProvider}
}

// CreateProcessor return instance of Operations for Listing
func (c listingProcessorFactory) CreateProcessor(ctxIn context.Context) (Operation[requestobjects.ListRequest, requestobjects.ListingResponse], error) {
	ctx, span := trace.StartSpan(ctxIn, "newListingProcessor")
	defer span.End()

	backupRepository, err := repository.NewBackupRepository(ctx, c.credentialsProvider)
	if err != nil {
		glog.Error(err)
		return &listingProcessor{}, err
	}

	return &listingProcessor{BackupRepository: backupRepository, sourceGCPProjectProvider: c.sourceGCPProjectProvider}, nil
}

type listingProcessor struct {
	BackupRepository         repository.BackupRepository
	Context                  context.Context
	sourceGCPProjectProvider provider.SourceGCPProjectProvider
}

// Process request
func (l listingProcessor) Process(ctxIn context.Context, args *Argument[requestobjects.ListRequest]) (requestobjects.ListingResponse, error) {
	ctx, span := trace.StartSpan(ctxIn, "(listingProcessor).Process")
	defer span.End()

	var request requestobjects.ListRequest = args.Request

	backupFilter := repository.BackupFilter{Project: request.Project}
	backups, err := l.BackupRepository.GetBackups(ctx, backupFilter)

	if err != nil {
		return requestobjects.ListingResponse{}, err
	}
	var filteredBackups []requestobjects.BackupResponse
	for _, backup := range backups {
		if auth.CheckRequestIsAllowed(args.Principal, requestobjects.Listing, backup.SourceProject) {
			sourceProject, err := l.sourceGCPProjectProvider.GetSourceGCPProject(ctx, backup.SourceProject)
			if err != nil {
				return requestobjects.ListingResponse{}, err
			}
			filteredBackups = append(filteredBackups, mapBackupToResponse(backup, nil, sourceProject))
		} else {
			glog.V(2).Infof("%s is not allowed for user %q on project %q", requestobjects.Listing.String(), args.Principal.User.Email, backup.TargetProject)
		}
	}

	if filteredBackups == nil {
		filteredBackups = []requestobjects.BackupResponse{}
	}

	return requestobjects.ListingResponse{
		Backups: filteredBackups,
	}, nil
}
