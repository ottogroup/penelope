package tasks

import (
    "context"
    "github.com/golang/glog"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/secret"
    "go.opencensus.io/trace"
)

type oneShotBackupStatusService struct {
    backupRepository repository.BackupRepository
    jobRepository    repository.JobRepository
}

func newOneShotBackupStatusService(ctxIn context.Context, credentialsProvider secret.SecretProvider) (*oneShotBackupStatusService, error) {
    ctx, span := trace.StartSpan(ctxIn, "newOneShotBackupStatusService")
    defer span.End()

    backupRepository, err := repository.NewBackupRepository(ctx, credentialsProvider)
    if err != nil {
        return nil, err
    }

    jobRepository, err := repository.NewJobRepository(ctx, credentialsProvider)
    if err != nil {
        return nil, err
    }

    return &oneShotBackupStatusService{backupRepository: backupRepository, jobRepository: jobRepository}, nil
}

func (b *oneShotBackupStatusService) Run(ctxIn context.Context) {
    ctx, span := trace.StartSpan(ctxIn, "(*oneShotBackupStatusService).Run")
    defer span.End()

    backups, err := b.backupRepository.GetBigQueryOneShotSnapshots(ctx, repository.Prepared)
    if err != nil {
        glog.Errorf("could not get prepared one shot snapshot backups: %s", err)
        return
    }
    for _, backup := range backups {
        jobStatistics, err := b.jobRepository.GetStatisticsForBackupID(ctx, backup.ID)
        if err != nil {
            glog.Errorf("could not get job statistics for backup %s: %s", backup.ID, err)
            continue
        }

        jobsFinished := jobStatistics[repository.FinishedOk] + jobStatistics[repository.FinishedError] + jobStatistics[repository.FinishedQuotaError] + jobStatistics[repository.Error]
        if jobsFinished == 0{
            // no job was finished
            // backup is freshly created or there are some jobs in progress
            continue
        }

        jobsInProgress := jobStatistics[repository.NotScheduled] + jobStatistics[repository.Scheduled] + jobStatistics[repository.Pending]
        if 0 < jobsInProgress {
            // there are unfinished jobs
            continue
        }

        b.backupRepository.MarkStatus(ctx, backup.ID, repository.Finished)
    }
}
