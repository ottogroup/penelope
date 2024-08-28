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

func NewTrashcanCleanUpProcessorFactory(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) TrashcanCleanUpProcessorFactory {
	return &trashcanCleanUpProcessorFactory{
		tokenSourceProvider: tokenSourceProvider,
		credentialsProvider: credentialsProvider,
	}
}

type TrashcanCleanUpProcessorFactory interface {
	CreateProcessor(ctxIn context.Context) (Operation[requestobjects.TrashcanCleanUpRequest, requestobjects.TrashcanCleanUpResponse], error)
}

// TrashcanCleanUpProcessorFactory create Process for TrashcanCleanUp
type trashcanCleanUpProcessorFactory struct {
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
	credentialsProvider secret.SecretProvider
}

func (p *trashcanCleanUpProcessorFactory) CreateProcessor(ctxIn context.Context) (Operation[requestobjects.TrashcanCleanUpRequest, requestobjects.TrashcanCleanUpResponse], error) {
	ctx, span := trace.StartSpan(ctxIn, "(*trashcanCleanUpProcessorFactory).CreateProcessor")
	defer span.End()

	backupRepository, err := repository.NewBackupRepository(ctx, p.credentialsProvider)
	if err != nil {
		glog.Error(err)
		return &trashcanCleanUpProcessor{}, err
	}

	return &trashcanCleanUpProcessor{
		backupRepository:    backupRepository,
		tokenSourceProvider: p.tokenSourceProvider,
	}, nil
}

type trashcanCleanUpProcessor struct {
	backupRepository    repository.BackupRepository
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func (p *trashcanCleanUpProcessor) Process(ctxIn context.Context, args *Argument[requestobjects.TrashcanCleanUpRequest]) (requestobjects.TrashcanCleanUpResponse, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*trashcanCleanUpProcessor).Process")
	defer span.End()

	var request = &args.Request

	backup, err := p.backupRepository.GetBackup(ctx, request.BackupID)
	if err != nil {
		if err == pg.ErrNoRows {
			return requestobjects.TrashcanCleanUpResponse{}, requestobjects.ApiError{
				Code:    404,
				Message: fmt.Sprintf("no backup with id %q found", request.BackupID),
			}
		}
		return requestobjects.TrashcanCleanUpResponse{}, errors.Wrapf(err, "get backup failed %s", request.BackupID)
	}

	if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.Cleanup, backup.SourceProject) {
		return requestobjects.TrashcanCleanUpResponse{}, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.Cleanup.String(), args.Principal.User.Email, backup.TargetProject)
	}

	err = p.backupRepository.MarkTrashcanCleanupStatus(ctx, backup.ID, repository.ScheduledTrashcanCleanupStatus)
	if err != nil {
		return requestobjects.TrashcanCleanUpResponse{}, errors.Wrapf(err, "mark trashcan cleanup status failed %s", backup.ID)
	}

	return requestobjects.TrashcanCleanUpResponse{}, nil
}
