package tasks

import (
    "context"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "github.com/ottogroup/penelope/pkg/processor"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/service/bigquery"
    "github.com/ottogroup/penelope/pkg/service/gcs"
    "fmt"
    "github.com/golang/glog"
    "github.com/pkg/errors"
    "go.opencensus.io/trace"
    "reflect"
)

type jobScheduleService struct {
    scheduleProcessor processor.ScheduleProcessor
    tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func newJobScheduleService(ctxIn context.Context, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) (*jobScheduleService, error) {
    ctx, span := trace.StartSpan(ctxIn, "newJobScheduleService")
    defer span.End()

    scheduleProcessor, err := processor.NewScheduleProcessor(ctx, credentialsProvider)
    if err != nil {
        return &jobScheduleService{}, fmt.Errorf("could not instantiate new ScheduleProcessor: %s", err)
    }

    return &jobScheduleService{
        scheduleProcessor: scheduleProcessor,
        tokenSourceProvider: tokenSourceProvider,
    }, nil
}

func (j *jobScheduleService) Run(ctxIn context.Context) {
    ctx, span := trace.StartSpan(ctxIn, "(*jobScheduleService).Run")
    defer span.End()

    for _, t := range repository.BackupTypes {
        jobs, err := j.scheduleProcessor.GetNextBackupJobs(ctx, t)
        if err != nil {
            glog.Errorf("could not get next backup jobs for backup type %s: %s", t.String(), err)
            return
        }
        if len(jobs) == 0 {
            glog.Infof("No jobs to schedule for type %s", t.String())
            continue
        }
        glog.Infof("Scheduling %d new jobs for type %s", len(jobs), t.String())
        for _, job := range jobs {
            err = j.scheduleJob(ctx, job)
            j.handleJobSchedulingError(ctx, err, job)
        }
    }
}

func (j *jobScheduleService) scheduleJob(ctxIn context.Context, job *repository.Job) error {
    ctx, span := trace.StartSpan(ctxIn, "(*jobScheduleService).scheduleJob")
    defer span.End()

    glog.Infof("[START] Scheduling job %s", job)
    switch job.Type {
    case repository.BigQuery:
        return j.scheduleBigQueryBackupJob(ctx, job)
    case repository.CloudStorage:
        return j.scheduleCloudStorageBackupJob(ctx, job)
    default:
        return &repository.InvalidBackupType{Type: job.Type}
    }
}

func (j *jobScheduleService) handleJobSchedulingError(ctxIn context.Context, err error, job *repository.Job) {
    ctx, span := trace.StartSpan(ctxIn, "(*jobScheduleService).handleJobSchedulingError")
    defer span.End()

    if err != nil {
        glog.Warningf("[FAIL] Error scheduling backup job %s: %s", job, err)
        err = j.scheduleProcessor.UpdateJob(ctx, job.Type, job.ID, repository.Error, "")
        if err != nil {
            glog.Warningf("[FAIL] Error marking backup job as failed %s: %s", job, err)
        }
    } else {
        glog.Infof("[SUCCESS] Scheduling finished for job %s", job)
    }
}

func (j *jobScheduleService) scheduleBigQueryBackupJob(ctxIn context.Context, job *repository.Job) error {
    ctx, span := trace.StartSpan(ctxIn, "(*jobScheduleService).scheduleBigQueryBackupJob")
    defer span.End()

    backup, err := j.getBackup(ctx, job.BackupID)
    if err != nil {
        return errors.Wrap(err, "getting backup failed")
    }

    jobHandler, err := bigquery.NewExtractJobHandler(ctx, j.tokenSourceProvider, backup.SourceProject, backup.TargetProject)
    if err != nil {
        return fmt.Errorf("could not create ExtractJobHandler: %s", err)
    }

    bigQueryOptions := backup.BackupOptions.BigQueryOptions
    sinkURI := repository.BuildFullObjectStoragePath(backup.Sink, bigQueryOptions.Dataset, job.Source, job.ID)
    glog.Infof("Creating bigquery extractJob with sink %s for job %s", sinkURI, job.ID)
    extractJobID, err := jobHandler.CreateAvroJob(ctx, bigQueryOptions.Dataset, job.Source, sinkURI)
    if err != nil {
        return fmt.Errorf("could not create avro job: %s", err)
    }
    glog.Infof("Successfully created bigquery extractJob with id %s for job %s", extractJobID, job.ID)

    state := repository.Scheduled
    err = j.scheduleProcessor.UpdateJob(ctx, job.Type, job.ID, state, extractJobID)
    if err != nil {
        return fmt.Errorf("could not update status of job with id %s to %s: %s", job.ID, state, err)
    }
    glog.Infof("Updating state job %s to %s of", state.String(), job.ID)

    return nil
}

func (j *jobScheduleService) scheduleCloudStorageBackupJob(ctxIn context.Context, job *repository.Job) error {
    ctx, span := trace.StartSpan(ctxIn, "(*jobScheduleService).scheduleCloudStorageBackupJob")
    defer span.End()

    backup, err := j.getBackup(ctx, job.BackupID)
    if err != nil {
        return errors.Wrap(err, "getting backup failed")
    }

    jobHandler, err := gcs.NewTransferJobHandler(ctx, j.tokenSourceProvider, backup.TargetProject)
    if err != nil {
        return fmt.Errorf("could not create TransferJobHandler: %s", err)
    }
    defer jobHandler.Close(ctx)

    glog.Infof("Creating cloudstorage transferJob with source %s for job %s", backup.Bucket, job.ID)
    transferJobID, err := jobHandler.CreateTransferJob(ctx, backup.SourceProject, backup.TargetProject, backup.Bucket, backup.Sink, backup.IncludePath, backup.ExcludePath)
    if err != nil {
        return fmt.Errorf("could not create transferJob: %s", err)
    }
    glog.Infof("Successfully created cloudstorage transferJob with id %s for job %s", transferJobID, job.ID)

    state := repository.Scheduled
    err = j.scheduleProcessor.UpdateJob(ctx, job.Type, job.ID, state, transferJobID)
    if err != nil {
        return fmt.Errorf("could not update status of job with id %s to %s: %s", job.ID, state, err)
    }
    glog.Infof("Updating state job %s to %s of", state.String(), job.ID)

    return nil
}

func (j *jobScheduleService) getBackup(ctxIn context.Context, backupID string) (*repository.Backup, error) {
    ctx, span := trace.StartSpan(ctxIn, "(*jobScheduleService).getBackup")
    defer span.End()

    backup, err := j.scheduleProcessor.GetBackupForID(ctx, backupID)
    if err != nil {
        return nil, fmt.Errorf("could not get backup with id %s: %s", backupID, err)
    }

    if backup == nil || reflect.ValueOf(backup).IsNil() {
        return nil, fmt.Errorf("got empty backup for id %s", backupID)
    }

    return backup, nil
}
