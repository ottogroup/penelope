package tasks

import (
    "context"
    "fmt"
    "github.com/go-pg/pg/v10/orm"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/ottogroup/penelope/pkg/http/mock"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/secret"
    service2 "github.com/ottogroup/penelope/pkg/service"
    "os"
    "strings"
    "testing"
    "time"
)

const (
    prepareBackupID   = "prepareBackup-uuid-5678"
    prepareBackupSink = "prepareBackup-uuid-5678-123456"
)

func prepareBackupServiceBigQueryBackup() *repository.Backup {
    return &repository.Backup{
        ID:            prepareBackupID,
        Status:        repository.Finished,
        SourceProject: "local-ability",
        Strategy:      repository.Snapshot,
        Type:          repository.BigQuery,
        SinkOptions: repository.SinkOptions{
            TargetProject: "local-ability-backup",
            Sink:          prepareBackupSink,
            Region:        "europe-west1",
            StorageClass:  repository.Nearline.String(),
        },
        BackupOptions: repository.BackupOptions{
            BigQueryOptions: repository.BigQueryOptions{"demo_delete_me_backup_target", []string{"gcp_billing_budget_amount_plan"}, []string{}},
        },
    }
}

func prepareBackupServiceBigQueryMirrorBackup() *repository.Backup {
    return &repository.Backup{
        ID:            prepareBackupID,
        Status:        repository.NotStarted,
        SourceProject: "local-ability",
        Strategy:      repository.Mirror,
        Type:          repository.BigQuery,
        SinkOptions: repository.SinkOptions{
            TargetProject: "local-ability-backup",
            Sink:          prepareBackupSink,
            Region:        "europe-west1",
            StorageClass:  repository.Nearline.String(),
        },
        BackupOptions: repository.BackupOptions{
            BigQueryOptions: repository.BigQueryOptions{"demo_delete_me_backup_target", []string{"gcp_billing_budget_amount_plan"}, []string{}},
        },
    }
}

func prepareBackupServiceCloudStorageBackup() *repository.Backup {
    return &repository.Backup{
        ID:            prepareBackupID,
        Status:        repository.Finished,
        SourceProject: "local-ability",
        Strategy:      repository.Snapshot,
        Type:          repository.CloudStorage,
        SinkOptions: repository.SinkOptions{
            TargetProject: "local-ability-backup",
            Sink:          prepareBackupSink,
            Region:        "europe-west1",
            StorageClass:  repository.Nearline.String(),
        },
        BackupOptions: repository.BackupOptions{
            CloudStorageOptions: repository.CloudStorageOptions{
                Bucket: "test-bucket",
            },
        },
    }
}

func TestPrepareBackupJobsService_WithoutValidJob(t *testing.T) {
    ctx := context.Background()
    service, _ := newPrepareBackupJobsService(ctx, nil, secret.NewEnvSecretProvider())
    service.scheduleProcessor = MockScheduleProcessor{
        shouldReturnValidJob:    false,
        shouldReturnValidBackup: false,
        ctx:                     ctx,
    }

    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })
    require.NoError(t, err)

    logMsg := "could not get list of scheduled backups for backup type BigQuery"
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
}

func TestPrepareBackupJobsService_WithFinishedBigQueryBackup(t *testing.T) {
    ctx := context.Background()
    backupRepository, err := repository.NewBackupRepository(context.Background(), secret.NewEnvSecretProvider())
    require.NoError(t, err, "BackupRepository should be instantiate")

    _, err = backupRepository.AddBackup(ctx, prepareBackupServiceBigQueryBackup())
    require.NoError(t, err, "should be able to add new backup")
    defer func() { deleteBackup(prepareBackupID) }()

    service, _ := newPrepareBackupJobsService(context.Background(), nil, secret.NewEnvSecretProvider())
    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := "No backups to prepare for type " + repository.BigQuery.String()
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
}

func TestPrepareBackupJobsService_WithFinishedCloudStorageBackup(t *testing.T) {
    ctx := context.Background()
    backupRepository, err := repository.NewBackupRepository(context.Background(), secret.NewEnvSecretProvider())
    require.NoError(t, err, "BackupRepository should be instantiate")

    _, err = backupRepository.AddBackup(ctx, prepareBackupServiceCloudStorageBackup())
    require.NoError(t, err, "should be able to add new backup")
    defer func() { deleteBackup(prepareBackupID) }()

    service, _ := newPrepareBackupJobsService(context.Background(), nil, secret.NewEnvSecretProvider())
    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := "No backups to prepare for type " + repository.CloudStorage.String()
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
}

func TestPrepareBackupJobsService_SchedulingTimeNotReached(t *testing.T) {
    ctx := context.Background()
    backup := prepareBackupServiceBigQueryBackup()
    backup.Status = repository.Prepared
    backup.LastScheduledTime = time.Now()
    backup.SnapshotOptions = repository.SnapshotOptions{
        FrequencyInHours: 12,
    }

    bkpGetCurrentTime := getCurrentTime
    getCurrentTime = func() time.Time {
        return time.Now().AddDate(0, -1, 0)
    }
    defer func() { getCurrentTime = bkpGetCurrentTime }()

    backupRepository, err := repository.NewBackupRepository(context.Background(), secret.NewEnvSecretProvider())
    require.NoError(t, err, "BackupRepository should be instantiate")

    _, err = backupRepository.AddBackup(ctx, backup)
    require.NoError(t, err, "should be able to add new backup")
    defer func() { deleteBackup(prepareBackupID) }()

    service, _ := newPrepareBackupJobsService(context.Background(), nil, secret.NewEnvSecretProvider())
    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := fmt.Sprintf("Backup with id %s don't need to be scheduled", prepareBackupID)
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
}

func TestPrepareBackupJobsService_WontScheduleAtNotFullHour(t *testing.T) {
    ctx := context.Background()
    backup := prepareBackupServiceBigQueryBackup()
    backup.Status = repository.Prepared
    backup.SnapshotOptions = repository.SnapshotOptions{
        FrequencyInHours: 12,
    }

    bkpGetCurrentTime := getCurrentTime
    getCurrentTime = func() time.Time {
        now := time.Now()
        return time.Date(now.Year(), now.Month(), now.Day(), 23, 23, 0, 0, now.Location())
    }
    defer func() { getCurrentTime = bkpGetCurrentTime }()

    backupRepository, err := repository.NewBackupRepository(context.Background(), secret.NewEnvSecretProvider())
    require.NoError(t, err, "BackupRepository should be instantiate")

    _, err = backupRepository.AddBackup(ctx, backup)
    require.NoError(t, err, "should be able to add new backup")
    defer func() { deleteBackup(prepareBackupID) }()

    service, _ := newPrepareBackupJobsService(context.Background(), nil, secret.NewEnvSecretProvider())
    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := fmt.Sprintf("Backup with id %s don't need to be scheduled", prepareBackupID)
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
}

func TestPrepareBackupJobsService_ScheduleAtFullHour(t *testing.T) {
    httpMockHandler.Start()
    defer httpMockHandler.Stop()

    ctx := context.Background()

    bkpGetCurrentTime := getCurrentTime
    getCurrentTime = func() time.Time {
        now := time.Now()
        return time.Date(now.Year(), now.Month(), now.Day(), 22, 00, 53, 11, now.Location())
    }

    defer func() { getCurrentTime = bkpGetCurrentTime }()

    backup := prepareBackupServiceBigQueryBackup()
    backup.Status = repository.Prepared
    backup.SnapshotOptions = repository.SnapshotOptions{
        FrequencyInHours: 12,
    }

    backupRepository, err := repository.NewBackupRepository(context.Background(), secret.NewEnvSecretProvider())
    require.NoError(t, err, "BackupRepository should be instantiate")

    _, err = backupRepository.AddBackup(ctx, backup)
    require.NoError(t, err, "should be able to add new backup")
    defer func() { deleteBackup(prepareBackupID) }()
    defer func() { dropJobs(prepareBackupID) }()
    defer func() { dropSourceMetadata(prepareBackupID) }()

    configProvider := &MockImpersonatedTokenConfigProvider{
        TargetPrincipal: "backup-tooling@local-test-prod.iam.gserviceaccount.com",
        Error:           nil,
    }

    service, _ := newPrepareBackupJobsService(ctx, configProvider, secret.NewEnvSecretProvider())
    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := fmt.Sprintf("Persisting backup job finished successfully for backup backupID=%s", prepareBackupID)
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
}

func TestPrepareBackupJobsService_NewBigQueryMirror(t *testing.T) {
    httpMockHandler.Start()
    defer httpMockHandler.Stop()

    ctx := context.Background()

    bkpGetCurrentTime := getCurrentTime
    getCurrentTime = func() time.Time {
        now := time.Now()
        return time.Date(now.Year(), now.Month(), now.Day(), 22, 00, 53, 11, now.Location())
    }
    defer func() { getCurrentTime = bkpGetCurrentTime }()

    backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
    require.NoError(t, err, "BackupRepository should be instantiate")

    backup := prepareBackupServiceBigQueryMirrorBackup()
    _, err = backupRepository.AddBackup(ctx, backup)
    require.NoError(t, err, "should be able to add new backup")
    defer func() { deleteBackup(prepareBackupID) }()
    defer func() { dropJobs(prepareBackupID) }()
    defer func() { dropSourceMetadata(prepareBackupID) }()

    configProvider := &MockImpersonatedTokenConfigProvider{
        TargetPrincipal: "backup-tooling@local-test-prod.iam.gserviceaccount.com",
        Error:           nil,
    }

    service, _ := newPrepareBackupJobsService(context.Background(), configProvider, secret.NewEnvSecretProvider())
    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })

    require.NoError(t, err)
    logMsg := fmt.Sprintf("Persisting backup job finished successfully for backup backupID=%s", prepareBackupID)
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
}

func TestPrepareBackupJobsService_BigQueryMirror_MetadataRepositoryTracksDeletedTables(t *testing.T) {
    httpMockHandler.Start()
    defer httpMockHandler.Stop()

    ctx := context.Background()

    bkpGetCurrentTime := getCurrentTime
    getCurrentTime = func() time.Time {
        now := time.Now()
        return time.Date(now.Year(), now.Month(), now.Day(), 22, 00, 53, 11, now.Location())
    }
    defer func() { getCurrentTime = bkpGetCurrentTime }()

    sourceMetadataRepository, err := repository.NewSourceMetadataRepository(ctx, secret.NewEnvSecretProvider())
    require.NoError(t, err, "sourceMetadataRepository should be instantiate")

    backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
    require.NoError(t, err, "BackupRepository should be instantiate")

    backup := prepareBackupServiceBigQueryMirrorBackup()
    _, err = backupRepository.AddBackup(ctx, backup)
    require.NoError(t, err, "should be able to add new backup")

    source := "gcp_billing_budget_amount_plan$20181216"
    initalChecksum := "initial-checksum"
    _, err = sourceMetadataRepository.Add(ctx, []*repository.SourceMetadata{{BackupID: prepareBackupID, Source: source, SourceChecksum: initalChecksum, Operation: "Add"}})
    require.NoError(t, err, "sourceMetadata should be added")

    defer func() { deleteBackup(prepareBackupID) }()
    defer func() { dropJobs(prepareBackupID) }()
    defer func() { dropSourceMetadata(prepareBackupID) }()

    configProvider := &MockImpersonatedTokenConfigProvider{
        TargetPrincipal: "backup-tooling@local-test-prod.iam.gserviceaccount.com",
        Error:           nil,
    }

    service, _ := newPrepareBackupJobsService(ctx, configProvider, secret.NewEnvSecretProvider())
    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })

    normalizedStdErr := strings.ToLower(stdErr)
    if strings.Contains(normalizedStdErr, "fail") || strings.Contains(normalizedStdErr, "error") {
        t.Fatalf(fmt.Sprintf("service.Run failed: %s", stdErr))
    }

    require.NoError(t, err)
    logMsg := fmt.Sprintf("Persisting backup job finished successfully for backup backupID=%s", prepareBackupID)
    assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message '%s' but it logged\n\t%s", logMsg, stdErr)

    storageService, err := service2.NewStorageService(ctx, secret.NewEnvSecretProvider())

    require.NoError(t, err)

    count, err := storageService.DB().Model((*repository.SourceMetadata)(nil)).
        Where("backup_id = ?", prepareBackupID).
        WhereGroup(func(query *orm.Query) (*orm.Query, error) {
            query = query.
                WhereOrGroup(func(query *orm.Query) (*orm.Query, error) {
                    query = query.
                        Where("source_checksum= ?", initalChecksum).
                        Where("operation='Delete'")
                    return query, nil
                }).
                WhereOrGroup(func(query *orm.Query) (*orm.Query, error) {
                    query = query.
                        Where("source_checksum= ?", initalChecksum).
                        Where("operation='Add'")
                    return query, nil
                })
            return query, nil
        }).
        Count()
    require.NoError(t, err)

    assert.Equal(t, 2, count)
}

func TestTableNotFound(t *testing.T) {
    httpMockHandler.Cleanup()
    httpMockHandler.Register(mock.OauthHTTPMock, mock.ImpersonationHTTPMock, mock.RetrieveAccessTokenHTTPMock, mock.TablePartitionQueryHTTPMock)
    httpMockHandler.Register(mock.ObjectsExistsHTTPMock, mock.SinkNotExistsHTTPMock, mock.SinkCreatedHTTPpMock, mock.SinkDeletedHTTPMock)
    httpMockHandler.Register(mock.TablePartitionJobHTTPMock, mock.TablePartitionResultHTTPMock, mock.ExtractJobResultOkHTTPMock)
    httpMockHandler.Register(mock.NewMockedHTTPRequest("GET", "/local-kebab-database/"+os.Getenv("CLOUD_SQL_SECRETS_PATH"), mock.SQLPasswordStorageResponse))
    httpMockHandler.Register(mock.TableNotFoundMock)

    httpMockHandler.Start()
    defer httpMockHandler.Stop()

    ctx := context.Background()

    backup := prepareBackupServiceBigQueryMirrorBackup()
    backup.Table = []string{"notExistingTable"}

    backupRepository, err := repository.NewBackupRepository(context.Background(), secret.NewEnvSecretProvider())
    require.NoError(t, err, "BackupRepository should be instantiate")

    _, err = backupRepository.AddBackup(ctx, backup)
    require.NoError(t, err, "should be able to add new backup")

    defer func() { deleteBackup(prepareBackupID) }()
    defer func() { dropJobs(prepareBackupID) }()
    defer func() { dropSourceMetadata(prepareBackupID) }()

    configProvider := &MockImpersonatedTokenConfigProvider{
        TargetPrincipal: "backup-tooling@local-test-prod.iam.gserviceaccount.com",
        Error:           nil,
    }

    service, _ := newPrepareBackupJobsService(context.Background(), configProvider, secret.NewEnvSecretProvider())
    _, stdErr, err := captureStderr(func() {
        service.Run(ctx)
    })
    require.NoError(t, err)

    assert.Contains(t, strings.TrimSpace(stdErr), fmt.Sprintf("404 Error: table with id %s not found", "notExistingTable"))
}
