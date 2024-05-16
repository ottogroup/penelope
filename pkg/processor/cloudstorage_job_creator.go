package processor

import (
	"context"
	"errors"
	"fmt"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"go.opencensus.io/trace"
	"time"
)

// CloudStorageJobCreator will create Transfer Jobs in GCP to backup GCS
type CloudStorageJobCreator struct {
	backupRepository repository.BackupRepository
	jobRepository    repository.JobRepository
	gcsClient        gcs.CloudStorageClient
}

var BucketNotFound = errors.New("error: bucket not found")

// NewCloudStorageJobCreator return instance of CloudStorageJobCreator
func NewCloudStorageJobCreator(ctxIn context.Context, backupRepository repository.BackupRepository, jobRepository repository.JobRepository, gcsClient gcs.CloudStorageClient) *CloudStorageJobCreator {
	_, span := trace.StartSpan(ctxIn, "NewCloudStorageJobCreator")
	defer span.End()

	return &CloudStorageJobCreator{
		backupRepository: backupRepository,
		jobRepository:    jobRepository,
		gcsClient:        gcsClient,
	}
}

// PrepareJobs for GCS backup
func (b *CloudStorageJobCreator) PrepareJobs(ctxIn context.Context, backup *repository.Backup) error {
	ctx, span := trace.StartSpan(ctxIn, "(*CloudStorageJobCreator).PrepareJobs")
	defer span.End()

	var bucketExists, _ = b.gcsClient.DoesBucketExist(ctx, backup.SourceProject, backup.Bucket)
	if !bucketExists {
		return BucketNotFound
	}

	if repository.Mirror == backup.Strategy || repository.Snapshot == backup.Strategy {
		return b.prepareSnapshotJobs(ctx, backup)
	}
	return fmt.Errorf("unsupported strategy %s", backup.Strategy)
}

func (b *CloudStorageJobCreator) prepareSnapshotJobs(ctxIn context.Context, backup *repository.Backup) error {
	ctx, span := trace.StartSpan(ctxIn, "(*CloudStorageJobCreator).prepareSnapshotJobs")
	defer span.End()

	// In a CloudStorageTransferJob Snapshot case defensively try to re-use an old Job.
	fjId := repository.ForeignJobID{}
	if reusableJob, err := b.jobRepository.GetMostRecentJobForBackupID(ctx, backup.ID, repository.FinishedOk, repository.FinishedError); backup.Strategy == repository.Snapshot && err == nil && reusableJob != nil {
		fjId = repository.ForeignJobID{CloudStorageID: reusableJob.ForeignJobID.CloudStorageID}
	}

	job := &repository.Job{
		ID:           generateNewID(),
		BackupID:     backup.ID,
		Status:       repository.NotScheduled,
		Source:       backup.CloudStorageOptions.Bucket,
		Type:         repository.CloudStorage,
		ForeignJobID: fjId,
	}

	err := b.jobRepository.AddJob(ctx, job)
	if err == nil {
		err = b.backupRepository.UpdateLastScheduledTime(ctx, backup.ID, time.Now(), repository.Prepared)
	}

	return err
}
