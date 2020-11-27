package processor

import (
    "context"
    "github.com/ottogroup/penelope/pkg/repository"
    "fmt"
    "go.opencensus.io/trace"
    "time"
)

// CloudStorageJobCreator will create Transfer Jobs in GCP to backup GCS
type CloudStorageJobCreator struct {
    backupRepository repository.BackupRepository
    jobRepository    repository.JobRepository
}

// NewCloudStorageJobCreator return instance of CloudStorageJobCreator
func NewCloudStorageJobCreator(ctxIn context.Context, backupRepository repository.BackupRepository, jobRepository repository.JobRepository) *CloudStorageJobCreator {
    _, span := trace.StartSpan(ctxIn, "NewCloudStorageJobCreator")
    defer span.End()

    return &CloudStorageJobCreator{
        backupRepository: backupRepository,
        jobRepository:    jobRepository,
    }
}

// PrepareJobs for GCS backup
func (b *CloudStorageJobCreator) PrepareJobs(ctxIn context.Context, backup *repository.Backup) error {
    ctx, span := trace.StartSpan(ctxIn, "(*CloudStorageJobCreator).PrepareJobs")
    defer span.End()

    if repository.Mirror == backup.Strategy || repository.Snapshot == backup.Strategy {
        return b.prepareSnapshotJobs(ctx, backup)
    }
    return fmt.Errorf("unsupported strategy %s", backup.Strategy)
}

func (b *CloudStorageJobCreator) prepareSnapshotJobs(ctxIn context.Context, backup *repository.Backup) error {
    ctx, span := trace.StartSpan(ctxIn, "(*CloudStorageJobCreator).prepareSnapshotJobs")
    defer span.End()

    job := &repository.Job{
        ID:       generateNewID(),
        BackupID: backup.ID,
        Status:   repository.NotScheduled,
        Source:   backup.CloudStorageOptions.Bucket,
        Type:     repository.CloudStorage,
    }

    err := b.jobRepository.AddJob(ctx, job)
    if err == nil {
        err = b.backupRepository.UpdateLastScheduledTime(ctx, backup.ID, time.Now(), repository.Prepared)
    }

    return err
}
