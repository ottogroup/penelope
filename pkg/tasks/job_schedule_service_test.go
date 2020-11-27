package tasks

import (
    "bytes"
    "context"
    "fmt"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/ottogroup/penelope/pkg/processor"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/service"
    "github.com/ottogroup/penelope/pkg/service/bigquery"
    "io"
    "os"
    "strings"
    "testing"
    "time"
)

const (
    scheduleServiceJobID    = "schedule-uuid-1234"
    scheduleServiceBackupID = "schedule-uuid-5678"
)


type MockScheduleProcessor struct {
    shouldReturnValidJob    bool
    shouldReturnValidBackup bool
    updatedStatus           repository.JobStatus
    updatedExternalID       string
    scheduledBackup         *repository.Backup
    mirrorRevision          *repository.MirrorRevision
    ctx                     context.Context
}

func (m MockScheduleProcessor) FilterExistingTrashcanEntries(context.Context, []processor.TrashcanEntry) ([]processor.TrashcanEntry, error) {
    panic("implement me")
}

func (m MockScheduleProcessor) UpdateLastCleanupTime(ctxIn context.Context, backupID string, lastCleanupTime time.Time) error {
    panic("implement me")
}

func (m MockScheduleProcessor) AddTrashcanEntry(ctxIn context.Context, backupID string, source string, timestamp time.Time) error {
    panic("implement me")
}

func (m MockScheduleProcessor) DeleteTrashcanEntry(ctxIn context.Context, backupID string, source string) error {
    panic("implement me")
}

func (m MockScheduleProcessor) GetEntriesInTrashcanBefore(ctxIn context.Context, deltaWeeks int) ([]*repository.SourceTrashcan, error) {
    panic("implement me")
}

func (m MockScheduleProcessor) GetJobsForBackupID(ctxIn context.Context, backupID string, jobPage repository.JobPage) ([]*repository.Job, error) {
    panic("implement me")
}

func (m MockScheduleProcessor) MarkBackupDeleted(ctxIn context.Context, id string) error {
    panic("implement me")
}

func (m MockScheduleProcessor) MarkSourceMetadataDeleted(ctxIn context.Context, id int) error {
    panic("implement me")
}

func (m MockScheduleProcessor) MarkJobDeleted(ctxIn context.Context, id string) error {
    panic("implement me")
}

func (m MockScheduleProcessor) GetExpiredBigQueryMirrorRevisions(ctxIn context.Context, maxRevisionLifetimeInWeeks int) ([]*repository.MirrorRevision, error) {
    if m.shouldReturnValidBackup {
        return []*repository.MirrorRevision{m.mirrorRevision}, nil
    }
    return nil, fmt.Errorf("GetBackupForID failed")
}

func (m MockScheduleProcessor) CreateCloudStorageJobCreator(ctxIn context.Context) *processor.CloudStorageJobCreator {
    panic("implement me")
}

func (m MockScheduleProcessor) GetByStatusAndAfter(context.Context, []repository.JobStatus, int) ([]*repository.Job, error) {
    panic("implement me")
}

func (m MockScheduleProcessor) CreateBigQueryJobCreator(c context.Context, bigQueryClient bigquery.Client) *processor.BigQueryJobCreator {
    panic("implement me")
}

func (m MockScheduleProcessor) AddJobs(jobs []*repository.Job) error {
    panic("implement me")
}

func (m MockScheduleProcessor) GetScheduledBackups(ctxIn context.Context, backupType repository.BackupType) ([]*repository.Backup, error) {
    if m.shouldReturnValidBackup {
        return []*repository.Backup{m.scheduledBackup}, nil
    }
    return nil, fmt.Errorf("GetBackupForID failed")
}

func (m MockScheduleProcessor) GetExpired(ctxIn context.Context, backupType repository.BackupType) ([]*repository.Backup, error) {
    if m.shouldReturnValidBackup {
        return []*repository.Backup{{
            ID:            scheduleServiceBackupID,
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
        }}, nil
    }
    return nil, fmt.Errorf("GetBackupForID failed")
}

func (m MockScheduleProcessor) UpdateJob(ctxIn context.Context, backupType repository.BackupType, jobID string, status repository.JobStatus, externalID string) error {
    return nil
}

func (m MockScheduleProcessor) UpdateBackupStatus(ctxIn context.Context, id string, status repository.BackupStatus) error {
    panic("implement me")
}

func (m MockScheduleProcessor) GetScheduledBackupJobs(ctxIn context.Context, backupType repository.BackupType) ([]*repository.Job, error) {
    if backupType == repository.BigQuery {
        if m.shouldReturnValidJob {
            return []*repository.Job{{
                Source:   "amount_budget_plan",
                Type:     repository.BigQuery,
                ID:       scheduleServiceJobID,
                Status:   repository.Scheduled,
                BackupID: scheduleServiceBackupID,
            }}, nil
        }
        return nil, fmt.Errorf("GetNextBackupJobs failed")
    } else if backupType == repository.CloudStorage {
        if m.shouldReturnValidJob {
            return []*repository.Job{{
                Source:   "test-bucket",
                ID:       scheduleServiceJobID,
                Type:     repository.CloudStorage,
                Status:   repository.Scheduled,
                BackupID: scheduleServiceBackupID,
            }}, nil
        }
        return nil, fmt.Errorf("GetNextBackupJobs failed")
    }
    return nil, fmt.Errorf("implement me")
}

func (m MockScheduleProcessor) GetNextBackupJobs(ctxIn context.Context, backupType repository.BackupType) ([]*repository.Job, error) {
    if backupType == repository.BigQuery {
        if m.shouldReturnValidJob {
            return []*repository.Job{{
                Source:   "amount_budget_plan",
                ID:       scheduleServiceJobID,
                Status:   repository.NotScheduled,
                Type:     repository.BigQuery,
                BackupID: scheduleServiceBackupID,
            }}, nil
        }
        return nil, fmt.Errorf("GetNextBackupJobs failed")
    } else if backupType == repository.CloudStorage {
        if m.shouldReturnValidJob {
            return []*repository.Job{{
                Source:   "test-bucket",
                ID:       scheduleServiceJobID,
                Status:   repository.NotScheduled,
                Type:     repository.CloudStorage,
                BackupID: scheduleServiceBackupID,
            }}, nil
        }
        return nil, fmt.Errorf("GetNextBackupJobs failed")
    }
    return nil, fmt.Errorf("implement me")
}

func (m MockScheduleProcessor) GetBackupForID(ctxIn context.Context, id string) (*repository.Backup, error) {
    if m.shouldReturnValidBackup {
        return &repository.Backup{
            ID:            scheduleServiceBackupID,
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
        }, nil
    }

    return &repository.Backup{}, fmt.Errorf("GetBackupForID failed")
}

func (m MockScheduleProcessor) UpdateBackupJob(backupType repository.BackupType, jobID string, status repository.JobStatus, externalID string) error {
    m.updatedStatus = status
    m.updatedExternalID = externalID
    return nil
}

func TestJobScheduleService_WithoutValidJob(t *testing.T) {
    ctx := context.Background()
    s, _ := newJobScheduleService(ctx, nil, secret.NewEnvSecretProvider())
    s.scheduleProcessor = MockScheduleProcessor{
        shouldReturnValidJob:    false,
        shouldReturnValidBackup: false,
        ctx:                     ctx,
    }

    _, stdErr, err := captureStderr(func() {
        s.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := "could not get next backup jobs for backup type BigQuery: GetNextBackupJobs failed"
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
}

func TestJobScheduleService_WithValidJobInvalidBackup(t *testing.T) {
    ctx := context.Background()
    s, _ := newJobScheduleService(ctx, nil, secret.NewEnvSecretProvider())
    s.scheduleProcessor = MockScheduleProcessor{
        shouldReturnValidJob:    true,
        shouldReturnValidBackup: false,
        ctx:                     ctx,
    }

    _, stdErr, err := captureStderr(func() {
        s.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := fmt.Sprintf("could not get backup with id %s", scheduleServiceBackupID)
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
}

func TestJobScheduleService_WithValidJobValidBigQueryBackup(t *testing.T) {
    httpMockHandler.Start()
    defer httpMockHandler.Stop()

    configProvider := &MockImpersonatedTokenConfigProvider{
        TargetPrincipal: "backup-tooling@local-test-prod.iam.gserviceaccount.com",
        Error:           nil,
    }

    ctx := context.Background()
    backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "BackupRepository should be instantiate")

    jobRepository, err := repository.NewJobRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "JobRepository should be instantiate")

    s, err := newJobScheduleService(ctx, configProvider, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "JobScheduleService should be instantiate")

    backup := repository.Backup{
        ID:            scheduleServiceBackupID,
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
    _, err = backupRepository.AddBackup(ctx, &backup)
    require.NoError(t, err, "should add new backup")
    defer func() { deleteBackup(scheduleServiceBackupID) }()

    job := repository.Job{
        ID:       scheduleServiceJobID,
        Source:   "amount_budget_plan",
        Status:   repository.NotScheduled,
        BackupID: scheduleServiceBackupID,
        Type:     repository.BigQuery,
    }
    err = jobRepository.AddJob(ctx, &job)
    require.NoError(t, err, "should add new job")
    defer func() { jobRepository.DeleteJob(ctx, scheduleServiceJobID) }()

    s.Run(ctx)

    require.NoError(t, err)
    updatedJob, err := jobRepository.GetJob(ctx, scheduleServiceJobID)
    require.NoError(t, err)
    assert.Equalf(t, repository.Scheduled, updatedJob.Status, "Job with id %s should be be scheduled but has status %s", scheduleServiceJobID, updatedJob.Status)
}

func TestJobScheduleService_WithValidJobValidCloudStorageBackup(t *testing.T) {
    httpMockHandler.Start()
    defer httpMockHandler.Stop()

    configProvider := &MockImpersonatedTokenConfigProvider{
        TargetPrincipal: "backup-tooling@local-test-prod.iam.gserviceaccount.com",
        Error:           nil,
    }

    ctx := context.Background()
    backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "BackupRepository should be instantiate")

    jobRepository, err := repository.NewJobRepository(ctx, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "JobRepository should be instantiate")

    scheduleService, err := newJobScheduleService(ctx, configProvider, secret.NewEnvSecretProvider())
    require.NoErrorf(t, err, "JobScheduleService should be instantiate")

    backup := repository.Backup{
        ID:            scheduleServiceBackupID,
        Status:        repository.NotStarted,
        SourceProject: "local-ability",
        Strategy:      repository.Snapshot,
        Type:          repository.CloudStorage,
        SinkOptions: repository.SinkOptions{
            TargetProject: "local-ability-backup",
            Sink:          "uuid-5678-123456",
            Region:        "europe-west1",
            StorageClass:  repository.Nearline.String(),
        },
        BackupOptions: repository.BackupOptions{
            CloudStorageOptions: repository.CloudStorageOptions{Bucket: "test-bucket"},
        },
    }
    _, err = backupRepository.AddBackup(ctx, &backup)
    require.NoError(t, err, "should add new backup")
    defer func() { deleteBackup(scheduleServiceBackupID) }()

    job := repository.Job{
        ID:       scheduleServiceJobID,
        Source:   "amount_budget_plan",
        Status:   repository.NotScheduled,
        BackupID: scheduleServiceBackupID,
        Type:     repository.BigQuery,
    }
    err = jobRepository.AddJob(ctx, &job)
    require.NoError(t, err, "should add new job")
    defer func() { jobRepository.DeleteJob(ctx, scheduleServiceJobID) }()

    scheduleService.Run(ctx)

    updatedJob, err := jobRepository.GetJob(ctx, scheduleServiceJobID)
    require.NoError(t, err)
    assert.Equalf(t, repository.Scheduled, updatedJob.Status, "Job with id %s should be be scheduled but has status %s", scheduleServiceJobID, updatedJob.Status)
}

func captureStderr(f func()) (string, string, error) {
    oldStderr := os.Stderr
    oldStdout := os.Stdout
    defer func() { os.Stderr = oldStderr }()
    defer func() { os.Stdout = oldStdout }()

    rStderr, wStderr, err := os.Pipe()
    if err != nil {
        return "", "", err
    }
    rStdout, wStdout, err := os.Pipe()
    if err != nil {
        return "", "", err
    }
    os.Stderr = wStderr
    os.Stdout = wStdout

    outStderrChan := make(chan string)
    outStdoutChan := make(chan string)
    // copy the output in a separate goroutine so printing can't block indefinitely
    go func() {
        var buf bytes.Buffer
        io.Copy(&buf, rStderr)
        outStderrChan <- buf.String()
    }()
    go func() {
        var buf bytes.Buffer
        io.Copy(&buf, rStdout)
        outStdoutChan <- buf.String()
    }()

    // calling function which stderr we are going to capture:
    f()

    // back to normal state
    wStderr.Close()
    wStdout.Close()
    return <-outStdoutChan, <-outStderrChan, nil
}

func deleteBackup(backupID string) error {
    storageService, err := service.NewStorageService(context.Background(), secret.NewEnvSecretProvider())
    if err != nil {
        panic(err)
    }
    _, err = storageService.DB().Model(&repository.Backup{ID: backupID}).WherePK().Delete()
    return err
}

func dropJobs(backupID string) error {
    storageService, err := service.NewStorageService(context.Background(), secret.NewEnvSecretProvider())
    if err != nil {
        panic(err)
    }

    _, err = storageService.DB().
        Model(&repository.Job{}).
        Where("backup_id = ?", backupID).
        Delete()
    return err
}

func dropSourceMetadata(backupID string) error {
    storageService, err := service.NewStorageService(context.Background(), secret.NewEnvSecretProvider())
    if err != nil {
        panic(err)
    }

    _, err = storageService.DB().
        Model(&repository.SourceMetadata{}).
        Where("backup_id = ?", backupID).
        Delete()
    return err
}
