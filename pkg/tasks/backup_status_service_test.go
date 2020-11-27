package tasks

import (
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/repository/memory"
    "testing"
)

const testBackupID = "1234-TestBackupStatusService_Run"

func TestBackupStatusService_Run_NoJobs(t *testing.T) {
    ctx := context.Background()
    testContext := GivenABackupServiceTestContext()
    testContext.backupRepository.AddBackup(ctx, &repository.Backup{
        ID:              testBackupID,
        Status:          repository.NotStarted,
        Type:            repository.BigQuery,
        SnapshotOptions: repository.SnapshotOptions{},
    })
    testContext.service.Run(ctx)
    ExpectBackupWithStatus(testContext, t, 0, repository.Finished)
}

func TestBackupStatusService_Run_JobsInProgress(t *testing.T) {
    ctx := context.Background()
    testContext := GivenABackupServiceTestContext()
    testContext.backupRepository.AddBackup(ctx, &repository.Backup{
        ID:              testBackupID,
        Status:          repository.NotStarted,
        Type:            repository.BigQuery,
        Strategy:        repository.Snapshot,
        SnapshotOptions: repository.SnapshotOptions{},
    })
    testContext.jobRepository.AddJob(ctx, &repository.Job{
        BackupID: testBackupID,
        Status:   repository.Scheduled,
    })
    testContext.service.Run(ctx)
    ExpectBackupWithStatus(testContext, t, 0, repository.Finished)
}

func TestBackupStatusService_Run_IgnoreMirror(t *testing.T) {
    ctx := context.Background()
    testContext := GivenABackupServiceTestContext()
    testContext.backupRepository.AddBackup(ctx, &repository.Backup{
        ID:              testBackupID,
        Status:          repository.NotStarted,
        Type:            repository.BigQuery,
        Strategy:        repository.Mirror,
        SnapshotOptions: repository.SnapshotOptions{},
    })
    testContext.jobRepository.AddJob(ctx, &repository.Job{
        BackupID: testBackupID,
        Status:   repository.FinishedOk,
    })
    testContext.service.Run(ctx)
    ExpectBackupWithStatus(testContext, t, 0, repository.Finished)
}

func TestBackupStatusService_Run_IgnoreRegularSnapshot(t *testing.T) {
    ctx := context.Background()
    testContext := GivenABackupServiceTestContext()
    testContext.backupRepository.AddBackup(ctx, &repository.Backup{
        ID:              testBackupID,
        Status:          repository.NotStarted,
        Type:            repository.BigQuery,
        Strategy:        repository.Snapshot,
        SnapshotOptions: repository.SnapshotOptions{FrequencyInHours: 1},
    })
    testContext.jobRepository.AddJob(ctx, &repository.Job{
        BackupID: testBackupID,
        Status:   repository.FinishedOk,
    })
    testContext.service.Run(ctx)
    ExpectBackupWithStatus(testContext, t, 0, repository.Finished)
}

func TestBackupStatusService_Run_JobsFinished(t *testing.T) {
    ctx := context.Background()
    testContext := GivenABackupServiceTestContext()
    testContext.backupRepository.AddBackup(ctx, &repository.Backup{
        ID:              testBackupID,
        Status:          repository.NotStarted,
        Type:            repository.BigQuery,
        Strategy:        repository.Snapshot,
        SnapshotOptions: repository.SnapshotOptions{},
    })
    testContext.jobRepository.AddJob(ctx, &repository.Job{
        BackupID: testBackupID,
        Status:   repository.FinishedQuotaError,
    })
    testContext.jobRepository.AddJob(ctx, &repository.Job{
        BackupID: testBackupID,
        Status:   repository.FinishedOk,
    })
    testContext.service.Run(ctx)
    ExpectBackupWithStatus(testContext, t, 1, repository.Finished)
}

func ExpectBackupWithStatus(testContext BackupServiceTestContext, t *testing.T, expected int, status repository.BackupStatus) {
    ctx := context.Background()
    backups, err := testContext.backupRepository.GetBackups(ctx, repository.BackupFilter{})
    require.NoError(t, err)
    counted := 0
    for _, backup := range backups {
        if status == backup.Status {
            counted++
        }
    }
    assert.Equal(t, expected, counted)
}

type BackupServiceTestContext struct {
    backupRepository repository.BackupRepository
    jobRepository    repository.JobRepository
    service          oneShotBackupStatusService
    ctx              context.Context
}

func GivenABackupServiceTestContext() BackupServiceTestContext {
    backupRepository := memory.BackupRepository{}
    jobRepository := memory.JobRepository{}
    service := oneShotBackupStatusService{
        backupRepository: &backupRepository,
        jobRepository:    &jobRepository,
    }
    return BackupServiceTestContext{
        backupRepository: &backupRepository,
        jobRepository:    &jobRepository,
        service:          service,
        ctx:              context.Background(),
    }
}

