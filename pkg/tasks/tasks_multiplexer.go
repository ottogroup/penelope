package tasks

import (
    "context"
    "fmt"
    "github.com/golang/glog"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "github.com/ottogroup/penelope/pkg/secret"
    "go.opencensus.io/trace"
)

const (
    // RunNewJobs is handled by task that start prepared jobs
    RunNewJobs = "run_new_jobs"
    // CheckJobsStatus is handled by task that update scheduled jobs status
    CheckJobsStatus = "check_jobs_status"
    // CheckOneShotBackupsStatus is handled by task that update jobs status for one shot backups
    CheckOneShotBackupsStatus = "check_backups_status"
    // CleanupExpiredSinks is handled by task that cleanup deleted backups or older files
    CleanupExpiredSinks = "cleanup_expired_sinks"
    // RescheduleJobsWithQuotaError is handled by task that start a new jobs that were interrupted due to quota exhaustion
    RescheduleJobsWithQuotaError = "reschedule_jobs_with_quota_error"
    // PrepareBackupJobs is handled by task that prepares new Jobs for scheduling
    PrepareBackupJobs = "prepare_backup_jobs"
    // CheckJobsStuck is handled by task that prepares check Jobs that are running to long
    CheckJobsStuck = "check_jobs_stuck"
)

// TaskRunner runs tasks
type TaskRunner interface {
    Run(context.Context)
}

// RunTask triggers specified task
func RunTask(task string, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) {
    background := context.TODO()
    ctx, span := trace.StartSpan(background, fmt.Sprintf("RunTask/%s", task))
    defer span.End()

    glog.Infof("Running task for action %s", task)

    switch task {
    case RunNewJobs:
        service, err := newJobScheduleService(ctx, tokenSourceProvider, credentialsProvider)
        if err != nil {
            glog.Errorf("could not instantiate new JobScheduleService: %s", err)
        }
        service.Run(ctx)
    case CheckJobsStatus:
        service, err := newJobStatusService(ctx, tokenSourceProvider, credentialsProvider)
        if err != nil {
            glog.Errorf("could not instantiate new JobStatusService: %s", err)
        }
        service.Run(ctx)
    case CheckOneShotBackupsStatus:
        service, err := newOneShotBackupStatusService(ctx, credentialsProvider)
        if err != nil {
            glog.Errorf("could not instantiate new BackupStatusService: %s", err)
        }
        service.Run(ctx)
    case CleanupExpiredSinks:
        service, err := newCleanupExpiredSinkService(ctx, tokenSourceProvider, credentialsProvider)
        if err != nil {
            glog.Errorf("could not instantiate new CleanupBackupService: %s", err)
        }
        service.Run(ctx)
    case PrepareBackupJobs:
        service, err := newPrepareBackupJobsService(ctx, tokenSourceProvider, credentialsProvider)
        if err != nil {
            glog.Errorf("could not instantiate new PrepareBackupJobsService: %s", err)
        }
        service.Run(ctx)
    case CheckJobsStuck:
        service, err := newJobsStuckService(ctx, credentialsProvider)
        if err != nil {
            glog.Errorf("could not instantiate new JobStuckService: %s", err)
        }
        service.Run(ctx)
    case RescheduleJobsWithQuotaError:
        service, err := newRescheduleJobsWithQuotaError(ctx, credentialsProvider)
        if err != nil {
            glog.Errorf("could not instantiate new RescheduleJobsWithQuotaErrorService: %s", err)
        }
        service.Run(ctx)
    default:
        glog.Warningf("no Service found for action: %s", task)
    }
}
