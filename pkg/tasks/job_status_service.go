package tasks

import (
    "context"
    "fmt"
    "github.com/golang/glog"
    "github.com/pkg/errors"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "github.com/ottogroup/penelope/pkg/processor"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/service/bigquery"
    "github.com/ottogroup/penelope/pkg/service/gcs"
    "go.opencensus.io/trace"
    "reflect"
)

type jobStatusService struct {
    scheduleProcessor processor.ScheduleProcessor
    tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func newJobStatusService(ctxIn context.Context, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) (*jobStatusService, error) {
    ctx, span := trace.StartSpan(ctxIn, "newJobStatusService")
    defer span.End()

    scheduleProcessor, err := processor.NewScheduleProcessor(ctx, credentialsProvider)
    if err != nil {
        return &jobStatusService{}, fmt.Errorf("could not instantiate new ScheduleProcessor: %s", err)
    }

    return &jobStatusService{
        scheduleProcessor: scheduleProcessor,
        tokenSourceProvider: tokenSourceProvider,
    }, nil
}

func (j *jobStatusService) Run(ctxIn context.Context) {
    ctx, span := trace.StartSpan(ctxIn, "(*jobStatusService).Run")
    defer span.End()

    for _, t := range repository.BackupTypes {
        jobs, err := j.scheduleProcessor.GetScheduledBackupJobs(ctx, t)
        if err != nil {
            glog.Errorf("could not get scheduled backup jobs for backup type %s: %s", t.String(), err)
            return
        }
        if len(jobs) == 0 {
            glog.Infof("No jobs to check status for type %s", t.String())
            continue
        }
        glog.Infof("Checking status of %d jobs for type %s", len(jobs), t.String())
        for _, job := range jobs {
            j.checkJobStatus(ctx, t, job)
        }
    }
}

func (j *jobStatusService) checkJobStatus(ctxIn context.Context, backupType repository.BackupType, job *repository.Job) {
    ctx, span := trace.StartSpan(ctxIn, "(*jobStatusService).checkJobStatus")
    defer span.End()

    switch backupType {
    case repository.BigQuery:
        glog.Infof("[START] Checking status of bigquery job %s", job)
        err := j.checkBigQueryBackupJob(ctx, job, backupType)
        if err != nil {
            glog.Warningf("[FAIL] Error checking status of bigquery backup job %s: %s", job, err)
        } else {
            glog.Infof("[SUCCESS] Checking status finished for bigquery job %s", job)
        }
    case repository.CloudStorage:
        glog.Infof("[START] Checking status of cloudstorage job %s", job)
        err := j.checkCloudStorageBackupJob(ctx, job, backupType)
        if err != nil {
            glog.Warningf("[FAIL] Error checking status of cloudstorage backup job %s: %s", job, err)
        } else {
            glog.Infof("[SUCCESS] Checking status finished for cloudstorage job %s", job)
        }
    }
}

func (j *jobStatusService) checkBigQueryBackupJob(ctxIn context.Context, job *repository.Job, backupType repository.BackupType) error {
    ctx, span := trace.StartSpan(ctxIn, "(*jobStatusService).checkBigQueryBackupJob")
    defer span.End()

    extractJobID := string(job.ForeignJobID.BigQueryID)
    if len(extractJobID) == 0 {
        return fmt.Errorf("could not check status of job with id %s without bigquery extractJobStatus for backup with id %s ", job.ID, job.BackupID)
    }

    backup, err := j.getBackup(ctx, job.BackupID)
    if err != nil {
        return errors.Wrap(err, "getting backup failed")
    }

    jobHandler, err := bigquery.NewExtractJobHandler(ctx, j.tokenSourceProvider, backup.SourceProject, backup.TargetProject)
    if err != nil {
        return fmt.Errorf("could not create ExtractJobHandler: %s", err)
    }

    glog.Infof("Checking status of bigquery extractJob with extractJobStatus %s for job %s", extractJobID, job.ID)
    extractJobStatus, err := jobHandler.GetStatusOfJob(ctx, extractJobID)
    if extractJobStatus == bigquery.StateUnspecified && err != nil {
        return fmt.Errorf("error getting status of extract job %s: %s", extractJobID, err)
    }
    glog.Infof("Successfully checked status of bigquery extractJob with id %s and status %s for job %s", extractJobStatus, extractJobID, job.ID)

    var state repository.JobStatus

    if extractJobStatus == bigquery.Done {
        state = repository.FinishedOk
    } else if extractJobStatus == bigquery.Pending || extractJobStatus == bigquery.Running {
        state = repository.Pending
    } else if extractJobStatus == bigquery.Failed {
        state = repository.FinishedError
    } else if extractJobStatus == bigquery.FailedQuotaExceeded {
        state = repository.FinishedQuotaError
    } else {
        return fmt.Errorf("extract job %s has unpredictable state for job with id %s to %s", extractJobID, state.String(), job.ID)
    }

    err = j.scheduleProcessor.UpdateJob(ctx, backupType, job.ID, state, extractJobID)
    if err != nil {
        return fmt.Errorf("could not update status of job with id %s to %s: %s", job.ID, state, err)
    }
    glog.Infof("Updating state to %s of job %s", state.String(), job.ID)
    if state == repository.FinishedError {
        glog.Infof("[FAIL] Job finished with error %s: %s", job, err)
    }
    if state == repository.FinishedQuotaError {
        glog.Warningf("[FAIL] Job finished with quota error %s: %s", job, err)
    }

    return nil
}

func (j *jobStatusService) checkCloudStorageBackupJob(ctxIn context.Context, job *repository.Job, backupType repository.BackupType) error {
    ctx, span := trace.StartSpan(ctxIn, "(*jobStatusService).checkCloudStorageBackupJob")
    defer span.End()

    transferJobID := string(job.ForeignJobID.CloudStorageID)
    if len(transferJobID) == 0 {
        return fmt.Errorf("could not check status of job with id %s without cloudstorage transferJobID for backup with id %s ", job.ID, job.BackupID)
    }

    backup, err := j.getBackup(ctx, job.BackupID)
    if err != nil {
        return errors.Wrap(err, "getting backup failed")
    }

    jobHandler, err := gcs.NewTransferJobHandler(ctx, j.tokenSourceProvider, backup.TargetProject)
    if err != nil {
        return fmt.Errorf("could not create TransferJobHandler: %s", err)
    }
    defer jobHandler.Close(ctx)

    glog.Infof("Checking status of cloudstorage transferJob with transferJobStatus %s for job %s", transferJobID, job.ID)
    transferJobStatus, err := jobHandler.GetStatusOfJob(ctx, backup.TargetProject, transferJobID)
    if transferJobStatus == gcs.StateUnspecified && err != nil {
        return fmt.Errorf("error getting status of extract job %s: %s", transferJobID, err)
    }
    glog.Infof("Successfully checked status of cloudstroage transferJob with id %s and status %s for job %s", transferJobStatus, transferJobID, job.ID)

    var state repository.JobStatus

    if transferJobStatus == gcs.Done {
        state = repository.FinishedOk
    } else if transferJobStatus == gcs.Pending || transferJobStatus == gcs.Running {
        state = repository.Pending
    } else if transferJobStatus == gcs.Failed {
        state = repository.FinishedError
    } else {
        return fmt.Errorf("extract job %s has unpredictable state for job with id %s to %s", transferJobID, state.String(), job.ID)
    }

    err = j.scheduleProcessor.UpdateJob(ctx, backupType, job.ID, state, transferJobID)
    if err != nil {
        return fmt.Errorf("could not update status of job with id %s to %s: %s", job.ID, state, err)
    }
    glog.Infof("Updating state to %s of job %s", state.String(), job.ID)
    if state == repository.FinishedError {
        glog.Infof("[FAIL] Job finished with error %s: %s", job, err)
    }

    //update status of backup if it is a oneshot snapshot
    if repository.Snapshot == backup.Strategy && backup.SnapshotOptions.FrequencyInHours == 0 {
        err = j.scheduleProcessor.UpdateBackupStatus(ctx, backup.ID, repository.Finished)
        if err != nil {
            return fmt.Errorf("could not update status of backup with id %s to %s: %s", backup.ID, repository.Finished.String(), err)
        }
    }

    return nil
}

func (j *jobStatusService) getBackup(ctxIn context.Context, backupID string) (*repository.Backup, error) {
    ctx, span := trace.StartSpan(ctxIn, "(*jobStatusService).getBackup")
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
