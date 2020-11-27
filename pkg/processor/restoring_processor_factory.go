package processor

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/ottogroup/penelope/pkg/http/auth"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/requestobjects"
    "github.com/ottogroup/penelope/pkg/secret"
    "go.opencensus.io/trace"
)

// RestoringProcessorFactory create Operations for Restoring
type RestoringProcessorFactory struct {
    tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
    credentialsProvider secret.SecretProvider
}

func NewRestoringProcessorFactory(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) *RestoringProcessorFactory {
    return &RestoringProcessorFactory{tokenSourceProvider, credentialsProvider}
}

// DoMatchRequestType does request type match Restoring
func (c RestoringProcessorFactory) DoMatchRequestType(requestType requestobjects.RequestType) bool {
	return requestobjects.Restoring.EqualTo(requestType.String())
}

// CreateProcessor return Operations for Restoring
func (c RestoringProcessorFactory) CreateProcessor(ctxIn context.Context) (Operations, error) {
	processor, err := c.newRestoringProcessor(ctxIn)
	if err != nil {
		return nil, err
	}

	return processor, nil
}

type restoringProcessor struct {
	BackupRepository repository.BackupRepository
	JobRepository    repository.JobRepository
	Context          context.Context
}

func (c RestoringProcessorFactory) newRestoringProcessor(ctxIn context.Context) (*restoringProcessor, error) {
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

func (l restoringProcessor) Process(ctxIn context.Context, args *Arguments) (*Result, error) {
	ctx, span := trace.StartSpan(ctxIn, "(restoringProcessor).Process")
	defer span.End()

	var request *requestobjects.RestoreRequest
	if args.Request == nil {
		return nil, fmt.Errorf("nil request object for processing bucket restore reques")
	}
	request, ok := args.Request.(*requestobjects.RestoreRequest)
	if !ok {
		return nil, fmt.Errorf("wrong request object for processing bucket restore request")
	}

	backup, err := l.BackupRepository.GetBackup(ctx, request.BackupID)
	if err != nil {
		return nil, errors.Wrapf(err, "get backup failed %s", request.BackupID)
	}
	jobs, err := l.JobRepository.GetBackupRestoreJobs(ctx, backup.ID, request.JobIDForTimestamp)
	if err != nil {
		return nil, errors.Wrapf(err, "job repository GetBackupJobs failed  %s", request.JobIDForTimestamp)
	}

	if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.Restoring, backup.SourceProject) {
		return nil, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.Restoring.String(), args.Principal.User.Email, backup.TargetProject)
	}

	return &Result{backups: []*repository.Backup{backup}, jobs: jobs}, err
}
