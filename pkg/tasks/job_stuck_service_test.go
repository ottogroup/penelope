package tasks

import (
    "context"
    "fmt"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/secret"
    service2 "github.com/ottogroup/penelope/pkg/service"
    "strings"
    "testing"
)

const (
    jobStuckServiceJobID    = "jobStuck-uuid-1234"
    jobStuckServiceBackupID = "jobStuck-uuid-5678"
)

func TestJobsStuckService_FinishedJobsAreNotSelected(t *testing.T) {
    ctx := context.Background()
    backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "backupRepository should be instantiate")

    jobRepository, err := repository.NewJobRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "JobRepository should be instantiate")

    _, err = backupRepository.AddBackup(ctx, jobStuckServiceBackup())
    require.NoError(t, err, "should add new backup")
    defer func() { deleteBackup(jobStuckServiceBackupID) }()

    finishedOKJobID := jobStuckServiceJobID + "-finishedok"
    finishedErrorJobID := jobStuckServiceJobID + "-finishederror"
    deletedJobID := jobStuckServiceJobID + "-deleted"
    err = jobRepository.AddJob(ctx, jobStuckServiceJob(finishedOKJobID, repository.FinishedOk))
    require.NoError(t, err, "should add new job")
    err = jobRepository.AddJob(ctx, jobStuckServiceJob(finishedErrorJobID, repository.FinishedError))
    require.NoError(t, err, "should add new job")
    err = jobRepository.AddJob(ctx, jobStuckServiceJob(deletedJobID, repository.JobDeleted))
    require.NoError(t, err, "should add new job")

    defer func() {
        jobRepository.DeleteJob(ctx, finishedOKJobID)
        jobRepository.DeleteJob(ctx, finishedErrorJobID)
        jobRepository.DeleteJob(ctx, deletedJobID)
    }()

    service, _ := newJobsStuckService(ctx, secret.NewEnvSecretProvider())
    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := "No jobs stuck with status"
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
}

func TestJobsStuckService_NewJobsAreNotSelected(t *testing.T) {
    ctx := context.Background()
    backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "JobReposBackupRepositoryitory should be instantiate")

    jobRepository, err := repository.NewJobRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "JobRepository should be instantiate")

    _, err = backupRepository.AddBackup(ctx, jobStuckServiceBackup())
    require.NoError(t, err, "should add new backup")
    defer func() { deleteBackup(jobStuckServiceBackupID) }()

    pendingJobID := jobStuckServiceJobID + "-Pending"
    scheduledJobID := jobStuckServiceJobID + "-Scheduled"
    notScheduledJobID := jobStuckServiceJobID + "-NotScheduled"
    err = jobRepository.AddJob(ctx, jobStuckServiceJob(notScheduledJobID, repository.JobDeleted))
    require.NoError(t, err, "should add new job")
    err = jobRepository.AddJob(ctx, jobStuckServiceJob(pendingJobID, repository.JobDeleted))
    require.NoError(t, err, "should add new job")
    err = jobRepository.AddJob(ctx, jobStuckServiceJob(scheduledJobID, repository.JobDeleted))
    require.NoError(t, err, "should add new job")

    defer func() {
        jobRepository.DeleteJob(ctx, notScheduledJobID)
        jobRepository.DeleteJob(ctx, pendingJobID)
        jobRepository.DeleteJob(ctx, scheduledJobID)
    }()

    service, _ := newJobsStuckService(ctx, secret.NewEnvSecretProvider())
    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := "No jobs stuck with status"
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
}

func TestJobsStuckService_NotScheduledJobIsStuck(t *testing.T) {
    ctx := context.Background()
    backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "JobReposBackupRepositoryitory should be instantiate")

    jobRepository, err := repository.NewJobRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "JobRepository should be instantiate")

    _, err = backupRepository.AddBackup(ctx, jobStuckServiceBackup())
    require.NoError(t, err, "should add new backup")
    defer func() { deleteBackup(jobStuckServiceBackupID) }()

    notScheduledJobID := jobStuckServiceJobID + "-NotScheduled"
    foreignJobID := "ForeignBigQueryId"
    job := jobStuckServiceJob(notScheduledJobID, repository.NotScheduled)
    job.ForeignJobID = repository.ForeignJobID{BigQueryID: repository.ExtractJobID(foreignJobID)}
    err = jobRepository.AddJob(ctx, job)
    require.NoError(t, err, "should add new job")
    defer func() { jobRepository.DeleteJob(ctx, notScheduledJobID) }()

    storageService, err := service2.NewStorageService(ctx, secret.NewEnvSecretProvider())

    require.NoError(t, err)

    _, err = storageService.DB().Model(&repository.Job{}).
        Set("audit_created_timestamp=NOW()-interval ' 5 hour '").
        Set("bigquery_extract_job_id=?", foreignJobID).
        Where("id=?", notScheduledJobID).
        Update()
    require.NoError(t, err)

    service, _ := newJobsStuckService(ctx, secret.NewEnvSecretProvider())
    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := fmt.Sprintf("backupID=%s jobID=%s", jobStuckServiceBackupID, notScheduledJobID)
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
}

func TestJobsStuckService_ScheduledJobIsStuck(t *testing.T) {
    ctx := context.Background()
    backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "JobReposBackupRepositoryitory should be instantiate")

    jobRepository, err := repository.NewJobRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "JobRepository should be instantiate")

    _, err = backupRepository.AddBackup(ctx, jobStuckServiceBackup())
    require.NoError(t, err, "should add new backup")
    defer func() { deleteBackup(jobStuckServiceBackupID) }()

    scheduledJobID := jobStuckServiceJobID + "-Scheduled"
    err = jobRepository.AddJob(ctx, jobStuckServiceJob(scheduledJobID, repository.Scheduled))
    require.NoError(t, err, "should add new job")
    defer func() { jobRepository.DeleteJob(ctx, scheduledJobID) }()

    storageService, err := service2.NewStorageService(ctx, secret.NewEnvSecretProvider())

    require.NoError(t, err)

    _, err = storageService.DB().Model(&repository.Job{}).
        Set("audit_updated_timestamp=NOW()-interval ' 5 hour '").
        Where("id=?", scheduledJobID).
        Update()
    require.NoError(t, err)

    service, _ := newJobsStuckService(ctx, secret.NewEnvSecretProvider())
    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := fmt.Sprintf("backupID=%s jobID=%s", jobStuckServiceBackupID, scheduledJobID)
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
}

func jobStuckServiceBackup() *repository.Backup {
    return &repository.Backup{
        ID:            jobStuckServiceBackupID,
        Status:        repository.NotStarted,
        SourceProject: "local-ability",
        Strategy:      repository.Snapshot,
        Type:          repository.BigQuery,
        SinkOptions: repository.SinkOptions{
            TargetProject: "local-ability-backup",
            Sink:          "uuid-5678-123456",
            Region:        "europe-west1",
            StorageClass:  repository.Nearline.String(),
        },
        BackupOptions: repository.BackupOptions{
            BigQueryOptions: repository.BigQueryOptions{"demo_delete_me_backup_target", []string{"gcp_billing_budget_amount_plan"}, []string{}},
        },
    }
}

func jobStuckServiceJob(id string, status repository.JobStatus) *repository.Job {
    return &repository.Job{
        ID:       id,
        Source:   "amount_budget_plan",
        Status:   status,
        BackupID: jobStuckServiceBackupID,
        Type:     repository.BigQuery,
    }
}
