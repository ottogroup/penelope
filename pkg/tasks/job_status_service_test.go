package tasks

import (
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/secret"
    "strings"
    "testing"
    "time"
)

const (
    statusServiceJobID    = "status-uuid-1234"
    statusServiceBackupID = "status-uuid-5678"
)

func TestJobStatusService_WithoutValidJob(t *testing.T) {
    ctx := context.Background()
    service, err := newJobStatusService(ctx, nil, secret.NewEnvSecretProvider())
    require.NoError(t, err)

    service.scheduleProcessor = MockScheduleProcessor{
        shouldReturnValidJob:    false,
        shouldReturnValidBackup: false,
        ctx:                     ctx,
    }

    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := "could not get scheduled backup jobs for backup type BigQuery"
    if !strings.Contains(strings.TrimSpace(stdErr), logMsg) {
        t.Errorf("Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
    }
}

func TestJobStatusService_WithValidJobValidBackup(t *testing.T) {
    httpMockHandler.Start()
    defer httpMockHandler.Stop()

    ctx := context.Background()
    backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "BackupRepository should be instantiate")

    jobRepository, err := repository.NewJobRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "JobRepository should be instantiate")

    configProvider := &MockImpersonatedTokenConfigProvider{
        TargetPrincipal: "backup-tooling@local-test-prod.iam.gserviceaccount.com",
        Error:           nil,
    }

    service, err := newJobStatusService(ctx, configProvider, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "JobStatusService should be instantiate")

    _, err = backupRepository.AddBackup(ctx, &repository.Backup{
        ID:            statusServiceBackupID,
        Status:        repository.Prepared,
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
        EntityAudit: repository.EntityAudit{
            CreatedTimestamp: time.Now(),
        },
    })
    require.NoError(t, err)
    defer func() { deleteBackup(statusServiceBackupID) }()

    job := repository.Job{
        ID:       statusServiceJobID,
        Source:   "amount_budget_plan",
        Status:   repository.NotScheduled,
        BackupID: statusServiceBackupID,
        Type:     repository.BigQuery,
    }
    err = jobRepository.AddJob(ctx, &job)
    require.NoError(t, err, "should add new job")
    err = jobRepository.PatchJobStatus(ctx, repository.JobPatch{ID: statusServiceJobID, Status: repository.Scheduled, ForeignJobID: repository.ForeignJobID{BigQueryID: "extractJobId"}})
    require.NoError(t, err)
    defer func() { jobRepository.DeleteJob(ctx, statusServiceJobID) }()

    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := "Checking status of 1 jobs"
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)

    updatedJob, err := jobRepository.GetJob(ctx, statusServiceJobID)
    require.NoError(t, err)
    assert.Equal(t, repository.FinishedOk, updatedJob.Status)

    updatedBackup, err := backupRepository.GetBackup(ctx, statusServiceBackupID)
    require.NoError(t, err)
    assert.Equal(t, repository.Prepared, updatedBackup.Status)
}
