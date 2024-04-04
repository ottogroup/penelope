package processor

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/http/auth"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type RestoringProcessorFactory interface {
	CreateProcessor(ctxIn context.Context) (Operation[requestobjects.RestoreRequest, requestobjects.RestoreResponse], error)
}

// RestoringProcessorFactory create Operations for Restoring
type restoringProcessorFactory struct {
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
	credentialsProvider secret.SecretProvider
}

func NewRestoringProcessorFactory(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) RestoringProcessorFactory {
	return &restoringProcessorFactory{tokenSourceProvider, credentialsProvider}
}

// CreateProcessor return Operations for Restoring
func (c restoringProcessorFactory) CreateProcessor(ctxIn context.Context) (Operation[requestobjects.RestoreRequest, requestobjects.RestoreResponse], error) {
	ctx, span := trace.StartSpan(ctxIn, "newRestoringProcessor")
	defer span.End()

	backupRepository, err := repository.NewBackupRepository(ctx, c.credentialsProvider)
	if err != nil {
		glog.Error(err)
		return &restoringProcessor{}, err
	}
	jobRepository, err := repository.NewJobRepository(ctx, c.credentialsProvider)
	if err != nil {
		glog.Error(err)
		return &restoringProcessor{}, err
	}

	return &restoringProcessor{BackupRepository: backupRepository, JobRepository: jobRepository}, nil
}

type restoringProcessor struct {
	BackupRepository repository.BackupRepository
	JobRepository    repository.JobRepository
	Context          context.Context
}

func (l restoringProcessor) Process(ctxIn context.Context, args *Argument[requestobjects.RestoreRequest]) (requestobjects.RestoreResponse, error) {
	ctx, span := trace.StartSpan(ctxIn, "(restoringProcessor).Process")
	defer span.End()

	var request requestobjects.RestoreRequest = args.Request

	backup, err := l.BackupRepository.GetBackup(ctx, request.BackupID)
	if err != nil {
		if err == pg.ErrNoRows {
			return requestobjects.RestoreResponse{}, requestobjects.ApiError{
				Code:    404,
				Message: fmt.Sprintf("no backup with id %q found", request.BackupID),
			}
		}
		return requestobjects.RestoreResponse{}, errors.Wrapf(err, "get backup failed %s", request.BackupID)
	}
	jobs, err := l.JobRepository.GetBackupRestoreJobs(ctx, backup.ID, request.JobIDForTimestamp)
	if err != nil {
		return requestobjects.RestoreResponse{}, errors.Wrapf(err, "job repository GetBackupJobs failed  %s", request.JobIDForTimestamp)
	}

	if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.Restoring, backup.SourceProject) {
		return requestobjects.RestoreResponse{}, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.Restoring.String(), args.Principal.User.Email, backup.TargetProject)
	}

	return mapToRestoreResponse(backup, jobs), err
}
