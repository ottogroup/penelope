package tasks

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/processor"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/secret"
	bq "github.com/ottogroup/penelope/pkg/service/bigquery"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"go.opencensus.io/trace"
)

const (
	lastWeekInDays = 7
)

type cleanupExpiredJobsService struct {
	scheduleProcessor   processor.ScheduleProcessor
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
	backupRepository    repository.BackupRepository
}

func newCleanupExpiredJobsService(ctxIn context.Context, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) (*cleanupExpiredJobsService, error) {
	ctx, span := trace.StartSpan(ctxIn, "newCleanupExpiredSinkService")
	defer span.End()

	scheduleProcessor, err := processor.NewScheduleProcessor(ctx, credentialsProvider)
	if err != nil {
		return &cleanupExpiredJobsService{}, fmt.Errorf("could not instantiate new ScheduleProcessor: %s", err)
	}

	backupRepository, err := repository.NewBackupRepository(ctx, credentialsProvider)
	if err != nil {
		return nil, err
	}

	return &cleanupExpiredJobsService{
		scheduleProcessor:   scheduleProcessor,
		tokenSourceProvider: tokenSourceProvider,
		backupRepository:    backupRepository,
	}, nil
}

func (s cleanupExpiredJobsService) Run(ctxIn context.Context) {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupExpiredJobsService).Run")
	defer span.End()

	jobs, err := s.scheduleProcessor.ListExpiredJobs(ctx, lastWeekInDays)
	if err != nil {
		glog.Errorf("could not list expired jobs: %s", err)
		return
	}

	glog.Infof("Found %d expired jobs", len(jobs))

	var errorCount = 0

	for _, job := range jobs {
		backup, err := s.backupRepository.GetBackup(ctx, job.BackupID)
		if err != nil {
			glog.Errorf("could not get backup %s: %s", job.BackupID, err)
			continue
		}

		switch backup.Type {
		case repository.BigQuery:
			extractJobHandler, err := bq.NewExtractJobHandler(ctx, s.tokenSourceProvider, backup.SourceProject, backup.TargetProject)
			if err != nil {
				glog.Errorf("could not create ExtractJobHandler: %s", err)
				return
			}

			err = extractJobHandler.DeleteJob(ctx, job.BigQueryID.String())
			if err != nil {
				glog.Errorf("could not delete job %s: %s", job.BigQueryID, err)
				errorCount++
				continue
			}
			extractJobHandler.Close()
		case repository.CloudStorage:
			transferJobHandler, err := gcs.NewTransferJobHandler(ctx, s.tokenSourceProvider, backup.TargetProject)
			if err != nil {
				glog.Errorf("could not create TransferJobHandler: %s", err)
				return
			}

			err = transferJobHandler.DeleteTransferJob(ctx, backup.TargetProject, job.CloudStorageID.String())
			if err != nil {
				glog.Errorf("could not delete job %s: %s", job.CloudStorageID, err)
				errorCount++
				continue
			}
			transferJobHandler.Close(ctx)
		}

		err = s.scheduleProcessor.MarkJobDeleted(ctx, job.ID)
		if err != nil {
			glog.Errorf("could not mark job %s as deleted: %s", job.ID, err)
			return
		}

		if errorCount > 9 {
			glog.Errorf("too many errors stop processing expired jobs")
			return
		}
	}
}
