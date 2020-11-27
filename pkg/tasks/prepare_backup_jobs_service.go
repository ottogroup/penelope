package tasks

import (
    "context"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "github.com/ottogroup/penelope/pkg/processor"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/service/bigquery"
    "fmt"
    "github.com/golang/glog"
    "go.opencensus.io/trace"
    "time"
)

type prepareBackupJobsService struct {
    scheduleProcessor   processor.ScheduleProcessor
    tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func newPrepareBackupJobsService(ctxIn context.Context, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) (*prepareBackupJobsService, error) {
    ctx, span := trace.StartSpan(ctxIn, "newPrepareBackupJobsService")
    defer span.End()

    scheduleProcessor, err := processor.NewScheduleProcessor(ctx, credentialsProvider)
    if err != nil {
        return &prepareBackupJobsService{}, fmt.Errorf("could not instantiate new ScheduleProcessor: %s", err)
    }

    return &prepareBackupJobsService{scheduleProcessor: scheduleProcessor, tokenSourceProvider: tokenSourceProvider}, nil
}

func (j *prepareBackupJobsService) Run(ctxIn context.Context) {
    ctx, span := trace.StartSpan(ctxIn, "(*prepareBackupJobsService).Run")
    defer span.End()

    for _, t := range repository.BackupTypes {
        backups, err := j.scheduleProcessor.GetScheduledBackups(ctx, t)
        if err != nil {
            glog.Errorf("could not get list of scheduled backups for backup type %s: %s", t.String(), err)
            return
        }
        if len(backups) == 0 {
            glog.Infof("No backups to prepare for type %s", t.String())
            continue
        }
        glog.Infof("Preparing jobs for %d backups for type %s", len(backups), t.String())
        for _, backup := range backups {
            j.scheduleJob(ctx, t, backup)
        }
    }
}

func (j *prepareBackupJobsService) scheduleJob(ctxIn context.Context, backupType repository.BackupType, backup *repository.Backup) {
    ctx, span := trace.StartSpan(ctxIn, "(*prepareBackupJobsService).scheduleJob")
    defer span.End()

    switch backupType {
    case repository.BigQuery:
        j.createBigQueryBackupJobs(ctx, backup)
    case repository.CloudStorage:
        j.createCloudStorageBackupJobs(ctx, backup)
    }
}

func (j *prepareBackupJobsService) createBigQueryBackupJobs(ctxIn context.Context, backup *repository.Backup) {
    ctx, span := trace.StartSpan(ctxIn, "(*prepareBackupJobsService).createBigQueryBackupJobs")
    defer span.End()

    if !isNextScheduleTime(backup) {
        glog.Infof("Backup with id %s don't need to be scheduled", backup.ID)
        return
    }

    glog.Infof("[START] Preparing backup jobs for backup %s", backup)
    bq, err := bigquery.NewBigQueryClient(ctx, j.tokenSourceProvider, backup.SourceProject, backup.SinkOptions.TargetProject)
    if err != nil {
        glog.Warningf("[FAIL] Error creating bigquery client for backup %s: %s", backup, err)
    } else {
        err = j.scheduleProcessor.CreateBigQueryJobCreator(ctx, bq).PrepareJobs(ctx, backup)
        if err != nil {
            glog.Warningf("[FAIL] Error preparing backup jobs for backup %s: %s", backup, err)
        } else {
            glog.Infof("[SUCCESS] Persisting backup job finished successfully for backup %s", backup)
        }
    }
}

func (j *prepareBackupJobsService) createCloudStorageBackupJobs(ctxIn context.Context, backup *repository.Backup) {
    ctx, span := trace.StartSpan(ctxIn, "(*prepareBackupJobsService).createCloudStorageBackupJobs")
    defer span.End()

    if !isNextScheduleTime(backup) {
        glog.Infof("Backup with id %s don't need to be scheduled", backup.ID)
        return
    }

    glog.Infof("[START] Preparing backup jobs for backup %s", backup)
    err := j.scheduleProcessor.CreateCloudStorageJobCreator(ctx).PrepareJobs(ctx, backup)
    if err != nil {
        glog.Warningf("[FAIL] Error preparing backup jobs for backup %s: %s", backup, err)
    } else {
        glog.Infof("[SUCCESS] Persisting backup job finished successfully for backup %s", backup)
    }
}

var getCurrentTime = func() time.Time {
    return time.Now().UTC()
}

func isNextScheduleTime(backup *repository.Backup) bool {
    return isNextSnapshotTime(backup) || isNextMirrorTime(backup)
}

func isNextMirrorTime(backup *repository.Backup) bool {
    // partitions are made on hourly basis
    nextScheduledTime := backup.LastScheduledTime.Add(time.Hour)
    return backup.Strategy == repository.Mirror &&
        (backup.LastScheduledTime.IsZero() || nextScheduledTime.Before(getCurrentTime()))
}

func isNextSnapshotTime(backup *repository.Backup) bool {
    nextScheduledTime := backup.LastScheduledTime.Add(time.Hour * time.Duration(backup.FrequencyInHours))
    currentTime := getCurrentTime()
    return backup.Strategy == repository.Snapshot && ((backup.FrequencyInHours == 0) || //is one-shot
        (backup.LastScheduledTime.IsZero() || nextScheduledTime.Before(currentTime)) && currentTime.Minute() == 0) //is scheduled on full-hour
}
