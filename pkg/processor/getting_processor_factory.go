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

// GettingProcessorFactory create Process for Getting
type GettingProcessorFactory struct {
    tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
    credentialProvider  secret.SecretProvider
}

func NewGettingProcessorFactory(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialProvider  secret.SecretProvider) *GettingProcessorFactory {
    return &GettingProcessorFactory{tokenSourceProvider, credentialProvider}
}

// DoMatchRequestType does request type match Getting
func (c GettingProcessorFactory) DoMatchRequestType(requestType requestobjects.RequestType) bool {
	return requestobjects.Getting.EqualTo(requestType.String())
}

// CreateProcessor return instance of Operations for Getting
func (c GettingProcessorFactory) CreateProcessor(ctxIn context.Context) (Operations, error) {
	processor, err := c.newGettingProcessor(ctxIn)
	if err != nil {
		return nil, err
	}

	return processor, nil
}

type gettingProcessor struct {
	BackupRepository repository.BackupRepository
	JobRepository    repository.JobRepository
}

func (c GettingProcessorFactory) newGettingProcessor(ctxIn context.Context) (*gettingProcessor, error) {
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

// Process request
func (l gettingProcessor) Process(ctxIn context.Context, args *Arguments) (*Result, error) {
	ctx, span := trace.StartSpan(ctxIn, "(gettingProcessor).Process")
	defer span.End()

	var request *requestobjects.GetRequest
	if args.Request == nil {
		return nil, fmt.Errorf("nil request object for processing backup get request")
	}
	request, ok := args.Request.(*requestobjects.GetRequest)
	if !ok {
		return nil, fmt.Errorf("wrong request object for processing backup get request")
	}

	backup, err := l.BackupRepository.GetBackup(ctx, request.BackupID)
	if err != nil {
		return nil, errors.Wrapf(err, "get backup failed %s", request.BackupID)
	}

	if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.Getting, backup.SourceProject) {
		return nil, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.Getting.String(), args.Principal.User.Email, backup.TargetProject)
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
		return nil, errors.Wrapf(err, "job repository GetBackupJobs failed  %s", request.BackupID)
	}
	jobsStats, err := l.JobRepository.GetStatisticsForBackupID(ctx, backup.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "job repository GetStatisticsForBackupID failed  %s", request.BackupID)
	}
	var countedJobs uint64
	for _, status := range repository.JobStatutses {
		countedJobs += jobsStats[status]
	}

	return &Result{backups: []*repository.Backup{backup}, jobs: jobs, JobsTotal: countedJobs}, err
}
