package tasks

import (
	"context"
	"fmt"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/secret"
	"go.opencensus.io/trace"
	"regexp"
	"time"

	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/processor"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"github.com/ottogroup/penelope/pkg/service/logging"
	"github.com/pkg/errors"
	"google.golang.org/api/googleapi"
)

const maxMirrorRevisionLifetimeInWeeks = 4

type cleanupBackupService struct {
	scheduleProcessor   processor.ScheduleProcessor
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func newCleanupExpiredSinkService(ctxIn context.Context, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) (*cleanupBackupService, error) {
	ctx, span := trace.StartSpan(ctxIn, "newCleanupExpiredSinkService")
	defer span.End()

	scheduleProcessor, err := processor.NewScheduleProcessor(ctx, credentialsProvider)
	if err != nil {
		return &cleanupBackupService{}, fmt.Errorf("could not instantiate new ScheduleProcessor: %s", err)
	}

	return &cleanupBackupService{
		scheduleProcessor:   scheduleProcessor,
		tokenSourceProvider: tokenSourceProvider,
	}, nil
}

func (j *cleanupBackupService) Run(ctxIn context.Context) {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupBackupService).Run")
	defer span.End()

	for _, t := range repository.BackupTypes {
		j.handleExpiredBackups(ctx, t)

		if repository.BigQuery.EqualTo(t.String()) {
			j.handleBigQueryMirror(ctx, t)
		} else if repository.CloudStorage.EqualTo(t.String()) {
			j.handleCloudStorageMirror(ctx, t)
		}
	}
}

func (j *cleanupBackupService) handleExpiredBackups(ctxIn context.Context, t repository.BackupType) {
	ctx, span := trace.StartSpan(ctxIn, "(*handleExpiredBackups).handleExpiredBackups")
	defer span.End()

	backups, err := j.scheduleProcessor.GetExpired(ctx, t)
	if err != nil {
		glog.Errorf("could not get list of expired backup backups for backup type %s: %s", t.String(), err)
	}
	if len(backups) == 0 {
		glog.Infof("No backups to clean up for type %s", t.String())
	} else {
		glog.Infof("Cleaning up %d sinks for type %s", len(backups), t.String())
	}

	for _, backup := range backups {
		glog.Infof("[START] Deleting sink for backup %s", backup)
		err := j.cleanupBackup(ctx, backup)
		if err != nil {
			glog.Warningf("[FAIL] Error deleting sink for backup %s: %s", backup, err)
		} else {
			glog.Infof("[SUCCESS] Deleting sink finished for backup %s", backup)
		}
	}
}

func (j *cleanupBackupService) handleBigQueryMirror(ctxIn context.Context, t repository.BackupType) {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupBackupService).handleBigQueryMirror")
	defer span.End()

	revisions, err := j.scheduleProcessor.GetExpiredBigQueryMirrorRevisions(ctx, maxMirrorRevisionLifetimeInWeeks)
	if err != nil {
		glog.Errorf("could not get list of expired mirror revisions for backup type %s: %s", t.String(), err)
	} else if len(revisions) == 0 {
		glog.Infof("No revisions to clean up for type %s", t.String())
	} else {
		for _, revision := range revisions {
			glog.Infof("[START] Deleting old BigQuery revision %s", revision)
			err = j.deleteBigQueryRevision(ctx, revision)
			if err != nil {
				glog.Warningf("[FAIL] Error deleting old BigQuery revision %s: %s", revision, err)
			} else {
				glog.Infof("[SUCCESS] Deleting old BigQuery revision finished %s", revision)
			}
		}
	}
}

func (j *cleanupBackupService) handleCloudStorageMirror(ctxIn context.Context, t repository.BackupType) {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupBackupService).handleCloudStorageMirror")
	defer span.End()

	backups, err := j.scheduleProcessor.GetScheduledBackups(ctx, t)

	if err != nil {
		glog.Errorf("could not get list of scheduled backups for backup type %s: %s", t.String(), err)
	} else if len(backups) == 0 {
		glog.Infof("No CloudStorage objects to clean up for type %s", t.String())
	} else {
		glog.Info("[START] Deleting old CloudStorage revision")
		err = j.cleanupCloudStorageObjects(ctx, backups)
		if err != nil {
			glog.Warningf("[FAIL] Error deleting old CloudStorage revision: %s", err)
		} else {
			glog.Info("[SUCCESS] Deleting old CloudStorage revision finished")
		}
	}
}

func (j *cleanupBackupService) cleanupBackup(ctxIn context.Context, backup *repository.Backup) error {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupBackupService).cleanupBackup")
	defer span.End()

	err := j.deleteSink(ctx, backup)
	if err != nil {
		return err
	}

	if repository.CloudStorage == backup.Type {
		return j.deleteTransferJobs(ctx, backup)
	}

	return nil
}

func (j *cleanupBackupService) deleteSink(ctxIn context.Context, backup *repository.Backup) error {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupBackupService).deleteSink")
	defer span.End()

	gcsClient, err := gcs.NewCloudStorageClient(ctx, j.tokenSourceProvider, backup.SinkOptions.TargetProject)
	if err != nil {
		return fmt.Errorf("could not instantiate CloudStorageClient for project %s: %s", backup.SinkOptions.TargetProject, err)
	}
	defer gcsClient.Close(ctx)

	prefix := ""
	if repository.BigQuery == backup.Type {
		prefix = repository.BuildStoragePath(backup.BackupOptions.BigQueryOptions.Dataset, "")
	}

	deletedObjects, err := gcsClient.DeleteObjectsWithObjectMatch(ctx, backup.Sink, prefix, nil)
	if err != nil {
		return err
	}
	glog.Infof("deleted %d objects for backup with id %s with prefix=%q", deletedObjects, backup.ID, prefix)

	// remove trashcan
	deletedObjects, err = gcsClient.DeleteObjectsWithObjectMatch(ctx, backup.Sink, backup.GetTrashcanPath(), nil)
	if err != nil {
		return err
	}
	glog.Infof("deleted %d objects for backup with id %s with prefix=%q", deletedObjects, backup.ID, prefix)

	err = gcsClient.DeleteBucket(ctx, backup.Sink)
	if err != nil {
		return fmt.Errorf("could not delete sink %s for project %s: %s", backup.Sink, backup.SinkOptions.TargetProject, err)
	}

	return j.scheduleProcessor.MarkBackupDeleted(ctx, backup.ID)
}

func (j *cleanupBackupService) deleteTransferJobs(ctxIn context.Context, backup *repository.Backup) error {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupBackupService).deleteTransferJobs")
	defer span.End()

	jobHandler, err := gcs.NewTransferJobHandler(ctx, j.tokenSourceProvider, backup.TargetProject)
	if err != nil {
		return fmt.Errorf("could not create TransferJobHandler: %s", err)
	}

	// errors are ignored from here after because they are not crucial
	jobPage := repository.JobPage{Size: repository.AllJobs}
	jobs, err := j.scheduleProcessor.GetJobsForBackupID(ctx, backup.ID, jobPage)
	if err == nil {
		for _, job := range jobs {
			jobHandler.DeleteTransferJob(ctx, backup.TargetProject, job.ForeignJobID.CloudStorageID.String())
		}
	}

	return nil
}

func (j *cleanupBackupService) deleteBigQueryRevision(ctxIn context.Context, revision *repository.MirrorRevision) error {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupBackupService).deleteBigQueryRevision")
	defer span.End()

	if revision.JobID != "" {
		err := j.deleteObjects(ctx, revision)
		if err != nil {
			return err
		}
		err = j.scheduleProcessor.MarkJobDeleted(ctx, revision.JobID)
		if err != nil {
			return err
		}
	}

	err := j.scheduleProcessor.MarkSourceMetadataDeleted(ctx, revision.SourceMetadataID)
	if err != nil {
		return err
	}

	return nil
}

func (j *cleanupBackupService) cleanupCloudStorageObjects(ctxIn context.Context, backups []*repository.Backup) error {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupBackupService).cleanupCloudStorageObjects")
	defer span.End()

	// Part 1
	glog.Info("Syncing cloud storage events")
	err := j.syncCloudStorageEvents(ctx, backups)
	if err != nil {
		return err
	}
	glog.Info("Finished syncing cloud storage events")

	// Part 2
	glog.Info("Deleting expired trashcan objects")
	for _, backup := range backups {
		err := j.deleteExpiredObjectsFromTrashcan(ctx, backup)
		if err != nil {
			glog.Warningf("[FAIL] error during deleting of expired objects from trashcan for backup %s: %s", backup, err)
		}
	}
	glog.Info("Finished deleting expired trashcan objects")

	return nil
}

func (j *cleanupBackupService) syncCloudStorageEvents(ctxIn context.Context, backups []*repository.Backup) error {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupBackupService).syncCloudStorageEvents")
	defer span.End()

	for _, backup := range backups {
		if repository.Mirror != backup.Strategy {
			// cleanup of deleted objects is done only for Mirror strategy
			continue
		}
		// 3 hours for all backups, we are running every 4 hours so margin of 1 hour should be enough
		iterationDeadline := time.Now().Add(time.Hour * time.Duration(3))
		err := j.syncCloudStorageEventsForBackup(ctx, backup, iterationDeadline)
		if err != nil {
			glog.Warningf("[FAIL] error during syncing cloud storage events for backup %s: %s", backup, err)
		}
	}

	return nil
}

func (j *cleanupBackupService) syncCloudStorageEventsForBackup(ctxIn context.Context, backup *repository.Backup, iterationDeadline time.Time) error {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupBackupService).syncCloudStorageEventsForBackup")
	defer span.End()

	glog.Infof("Syncing cloud events storage objects for backup: %s", backup.ID)

	client, err := logging.NewLoggingClient(ctx, j.tokenSourceProvider, backup.SourceProject, backup.TargetProject)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not instantiate LoggingClient for project %s", backup.TargetProject))
	}
	defer client.Close()

	gcsClient, err := gcs.NewCloudStorageClient(ctx, j.tokenSourceProvider, backup.TargetProject)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not instantiate CloudStorageClient for project %s", backup.TargetProject))
	}
	defer gcsClient.Close(ctx)

	backupID := backup.ID
	var timestampStart time.Time
	if !backup.LastCleanupTime.IsZero() {
		timestampStart = backup.LastCleanupTime
	} else {
		timestampStart = backup.CreatedTimestamp
	}

	srcBucketName := backup.CloudStorageOptions.Bucket
	targetBucketName := backup.Sink
	lastTimestamp, err := client.IterateOverBucketObjectEvents(ctx, backup, srcBucketName, timestampStart, iterationDeadline, func(objs []logging.BucketObjectEvent, eventType logging.ObjectEvent) error {

		switch eventType {
		case logging.Create:
			// error ignored from here on because they are not crucial
			var trashcanEntries []processor.TrashcanEntry
			for _, obj := range objs {
				trashcanEntries = append(trashcanEntries, processor.TrashcanEntry{BackupID: backupID, Source: obj.ObjectName})
			}
			exists, err := j.scheduleProcessor.FilterExistingTrashcanEntries(ctx, trashcanEntries)
			if err != nil {
				return errors.Wrapf(err, "FilterExistingTrashcanEntries failed for backup %s", backupID)
			}
			for _, exist := range exists {
				gcsClient.DeleteObject(ctx, targetBucketName, fmt.Sprintf("%s/%s", backup.GetTrashcanPath(), exist.Source))
				j.scheduleProcessor.DeleteTrashcanEntry(ctx, backupID, exist.Source)
				glog.Infof("Deleted object from trashcan %s", exist.Source)
			}
		case logging.Delete:
			for _, obj := range objs {
				err = gcsClient.MoveObject(ctx, targetBucketName, obj.ObjectName, fmt.Sprintf("%s/%s", backup.GetTrashcanPath(), obj.ObjectName))
				if err == nil {
					glog.Infof("Moved object from backup to trashcan %s", obj)
					if err := j.scheduleProcessor.AddTrashcanEntry(ctx, backupID, obj.ObjectName, obj.Timestamp); err != nil {
						glog.Errorf("could not add entry to trashcan table %s. err %v", obj, err)
					}
				} else {
					if errOK, ok := err.(*googleapi.Error); ok && errOK.Code == 404 {
						// object was deleted before file was put into backup
						continue
					} else {
						glog.Errorf("could not move object to trashcan %s. err %v", obj, err)
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not process objects from logs for %s: %s", backup, err))
	}
	glog.Infof("Finish syncing cloud storage objects events for backup: %s with last timestamp: %s", backup.ID, lastTimestamp)
	return j.scheduleProcessor.UpdateLastCleanupTime(ctx, backupID, lastTimestamp)
}

func (j *cleanupBackupService) deleteExpiredObjectsFromTrashcan(ctxIn context.Context, backup *repository.Backup) error {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupBackupService).deleteExpiredObjectsFromTrashcan")
	defer span.End()

	gcsClient, err := gcs.NewCloudStorageClient(ctx, j.tokenSourceProvider, backup.TargetProject)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not instantiate CloudStorageClient for project %s", backup.TargetProject))
	}
	defer gcsClient.Close(ctx)
	entries, err := j.scheduleProcessor.GetEntriesInTrashcanBefore(ctx, maxMirrorRevisionLifetimeInWeeks)
	if err != nil {
		return errors.Wrap(err, "could not get entries in trashcan after max revision lifetime")
	}

	glog.Infof("Deleting expired trashcan objects for backup: %s", backup)
	for _, entry := range entries {
		err := gcsClient.DeleteObject(ctx, backup.CloudStorageOptions.Bucket, fmt.Sprintf("%s/%s", backup.GetTrashcanPath(), entry.Source))
		if err, ok := err.(*googleapi.Error); ok && err.Code != 404 {
			return errors.Wrap(err, fmt.Sprintf("could not delete object %s:%s", backup.CloudStorageOptions.Bucket, fmt.Sprintf("%s/%s", backup.GetTrashcanPath(), entry.Source)))
		}

		err = j.scheduleProcessor.DeleteTrashcanEntry(ctx, backup.ID, entry.Source)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not delete db entry for backupID=%s : %s", backup.ID, entry.Source))
		}
	}

	return nil
}

func (j *cleanupBackupService) deleteObjects(ctxIn context.Context, revision *repository.MirrorRevision) error {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupBackupService).deleteObjects")
	defer span.End()

	gcsClient, err := gcs.NewCloudStorageClient(ctx, j.tokenSourceProvider, revision.TargetProject)
	if err != nil {
		return fmt.Errorf("could not instantiate CloudStorageClient for project %s: %s", revision.TargetProject, err)
	}
	defer gcsClient.Close(ctx)

	prefix := repository.BuildStoragePath(revision.BigqueryDataset, revision.Source)
	objectPattern := regexp.MustCompile(repository.BuildObjectStoragePathPattern(revision.BigqueryDataset, revision.Source, revision.JobID))
	deletedObjects, err := gcsClient.DeleteObjectsWithObjectMatch(ctx, revision.TargetSink, prefix, objectPattern)
	if err != nil {
		return err
	}

	glog.Infof("deleted %d objects for backupID=%s in %s", deletedObjects, revision.BackupID, prefix)
	return nil
}
