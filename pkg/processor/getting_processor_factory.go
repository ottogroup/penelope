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
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type GettingProcessorFactory interface {
	CreateProcessor(ctxIn context.Context) (Operation[requestobjects.GetRequest, requestobjects.BackupResponse], error)
}

// GettingProcessorFactory create Process for Getting
type gettingProcessorFactory struct {
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
	credentialProvider  secret.SecretProvider
}

func NewGettingProcessorFactory(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialProvider secret.SecretProvider) GettingProcessorFactory {
	return &gettingProcessorFactory{tokenSourceProvider, credentialProvider}
}

// CreateProcessor return instance of Operations for Getting
func (c gettingProcessorFactory) CreateProcessor(ctxIn context.Context) (Operation[requestobjects.GetRequest, requestobjects.BackupResponse], error) {
	ctx, span := trace.StartSpan(ctxIn, "newGettingProcessor")
	defer span.End()

	backupRepository, err := repository.NewBackupRepository(ctx, c.credentialProvider)
	if err != nil {
		glog.Error(err)
		return &gettingProcessor{}, err
	}
	jobRepository, err := repository.NewJobRepository(ctx, c.credentialProvider)
	if err != nil {
		glog.Error(err)
		return &gettingProcessor{}, err
	}

	return &gettingProcessor{BackupRepository: backupRepository, JobRepository: jobRepository}, nil
}

type gettingProcessor struct {
	BackupRepository repository.BackupRepository
	JobRepository    repository.JobRepository
}

// Process request
func (l gettingProcessor) Process(ctxIn context.Context, args *Argument[requestobjects.GetRequest]) (requestobjects.BackupResponse, error) {
	ctx, span := trace.StartSpan(ctxIn, "(gettingProcessor).Process")
	defer span.End()

	var request requestobjects.GetRequest = args.Request

	backup, err := l.BackupRepository.GetBackup(ctx, request.BackupID)
	if err != nil {
		return requestobjects.BackupResponse{}, errors.Wrapf(err, "get backup failed %s", request.BackupID)
	}

	if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.Getting, backup.SourceProject) {
		return requestobjects.BackupResponse{}, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.Getting.String(), args.Principal.User.Email, backup.TargetProject)
	}

	jobPage := repository.JobPage{Size: request.Page.Size, Number: request.Page.Number}
	if jobPage.Size == 0 || jobPage.Size < 0 {
		jobPage.Size = 100
	}
	if jobPage.Number < 0 {
		jobPage.Size = 0
	}
	jobs, err := l.JobRepository.GetJobsForBackupID(ctx, backup.ID, jobPage)
	if err != nil {
		return requestobjects.BackupResponse{}, errors.Wrapf(err, "job repository GetBackupJobs failed  %s", request.BackupID)
	}
	jobsStats, err := l.JobRepository.GetStatisticsForBackupID(ctx, backup.ID)
	if err != nil {
		return requestobjects.BackupResponse{}, errors.Wrapf(err, "job repository GetStatisticsForBackupID failed  %s", request.BackupID)
	}
	var countedJobs uint64
	for _, status := range repository.JobStatutses {
		countedJobs += jobsStats[status]
	}

	res := mapBackupToResponse(backup, jobs)
	res.JobsTotal = countedJobs
	return res, err
}
