package repository

import (
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/ottogroup/penelope/pkg/service"
    "testing"
    "time"
)

func TestDefaultBackupRepository_AddBackup_Simple(t *testing.T) {
    backupRepository := setUpAndGetBackupRepository(t)
    ctx := context.Background()

    simpleBackupId := "simple-backup-id"

    _, err := backupRepository.AddBackup(ctx, createBackup(simpleBackupId, "", ""))
    assert.NoError(t, err)
}

func TestDefaultBackupRepository_AddBackup_Table(t *testing.T) {
    backupRepository := setUpAndGetBackupRepository(t)
    ctx := context.Background()

    simpleBackupId := "simple-backup-id"

    backup := createBackup(simpleBackupId, "", "")
    backup.Table = []string{"notExistingTable"}

    _, err := backupRepository.AddBackup(ctx, backup)
    assert.NoError(t, err)
}

func TestDefaultBackupRepository_AddBackup_RedundantPK(t *testing.T) {
    backupRepository := setUpAndGetBackupRepository(t)
    ctx := context.Background()

    simpleBackupId := "simple-backup-id"

    _, err := backupRepository.AddBackup(ctx, createBackup(simpleBackupId, "", ""))
    assert.NoError(t, err)

    _, err = backupRepository.AddBackup(ctx, createBackup(simpleBackupId, "", ""))
    assert.Error(t, err, "Expected redundant primary key error")
}

func TestDefaultBackupRepository_AddGet_Simple(t *testing.T) {
    backupRepository := setUpAndGetBackupRepository(t)
    ctx := context.Background()

    simpleBackupId := "simple-backup-id"
    status := NotStarted
    typ := BigQuery

    _, err := backupRepository.AddBackup(ctx, createBackup(simpleBackupId, status, typ))
    assert.NoError(t, err)

    backup, err := backupRepository.GetBackup(ctx, simpleBackupId)
    assert.NoError(t, err)
    assert.NotNil(t, backup)
    assert.Equal(t, status, backup.Status)
    assert.Equal(t, typ, backup.Type)
}

func TestDefaultBackupRepository_GetBackup_NotFound(t *testing.T) {
    backupRepository := setUpAndGetBackupRepository(t)
    ctx := context.Background()

    simpleBackupId := "simple-backup-id"

    backup, err := backupRepository.GetBackup(ctx, simpleBackupId)
    assert.Error(t, err)
    assert.Nil(t, backup)
}

func TestDefaultBackupRepository_MarkDeleted(t *testing.T) {
    backupRepository := setUpAndGetBackupRepository(t)
    ctx := context.Background()

    simpleBackupId := "simple-backup-id"

    _, err := backupRepository.AddBackup(ctx, createBackup(simpleBackupId, "", ""))
    assert.NoError(t, err)

    err = backupRepository.MarkDeleted(ctx, simpleBackupId)
    assert.NoError(t, err)

    backup, err := backupRepository.GetBackup(ctx, simpleBackupId)
    assert.NoError(t, err)
    assert.Equal(t, backup.Status, BackupDeleted)
}

func TestDefaultBackupRepository_MarkDeleted_OnlyOneRowShouldBeAffected(t *testing.T) {
    ctx, storageService := prepareTest(t)

    backupRepository := &defaultBackupRepository{storageService: storageService}

    err := setBackups(backupRepository.storageService, []*Backup{
        {ID: "backup-id-1232", Status: Paused},
        {ID: "backup-id-2412", Status: Paused},
    })
    assert.NoError(t, err)


    err = backupRepository.MarkDeleted(ctx, "backup-id-1232")
    assert.NoError(t, err)

    count, _ := storageService.DB().Model(&Backup{}).Where("status = ?", BackupDeleted).Count()
    assert.Equal(t, 1, count, "only one row should be affected")

    count, _ = storageService.DB().Model(&Backup{}).Where("audit_deleted_timestamp IS NOT NULL").Count()
    assert.Equal(t, 1, count, "expected one audit_deleted_timestamp not null")
}

func TestDefaultBackupRepository_MarkStatus_FirstDeleteThenNotStarted(t *testing.T) {
    backupRepository := setUpAndGetBackupRepository(t)
    ctx := context.Background()

    simpleBackupId := "simple-backup-id"

    _, err := backupRepository.AddBackup(ctx, createBackup(simpleBackupId, "", ""))
    require.NoError(t, err)

    err = backupRepository.MarkDeleted(ctx, simpleBackupId)
    require.NoError(t, err)

    backup, err := backupRepository.GetBackup(ctx, simpleBackupId)
    require.NoError(t, err)
    require.Equal(t, backup.Status, BackupDeleted)
    assert.False(t, backup.DeletedTimestamp.IsZero())

    err = backupRepository.MarkStatus(ctx, simpleBackupId, NotStarted)
    assert.NoError(t, err)

    backup, err = backupRepository.GetBackup(ctx, simpleBackupId)
    assert.NoError(t, err)
    assert.Equal(t, backup.Status, NotStarted)
    assert.True(t, backup.DeletedTimestamp.IsZero(), "backup deleted timestamp with status NotStarted should be zero: %s", backup)
}

func TestDefaultBackupRepository_UpdateBackup(t *testing.T) {
    backupRepository := setUpAndGetBackupRepository(t)
    ctx := context.Background()

    simpleBackupId := "simple-backup-id"

    _, err := backupRepository.AddBackup(ctx, createBackup(simpleBackupId, "", CloudStorage))
    assert.NoError(t, err)
    backupStatus := BackupDeleted
    backupIncludePath := []string{"first", "second"}
    backupExcludePath := []string{"third", "four"}
    table := []string{"table-1", "table-1"}
    excludedTables := []string{"table-excluded-1", "table-excluded-1"}

    err = backupRepository.UpdateBackup(ctx, UpdateFields{
        BackupID:       simpleBackupId,
        Status:         backupStatus,
        IncludePath:    backupIncludePath,
        ExcludePath:    backupExcludePath,
        Table:          table,
        ExcludedTables: excludedTables,
        MirrorTTL:      1000,
        SnapshotTTL:    1001,
        ArchiveTTM:     1002,
    })
    assert.NoError(t, err)

    backup, err := backupRepository.GetBackup(ctx, simpleBackupId)
    assert.NoError(t, err)
    assert.NotNil(t, backup)
    assert.Equal(t, backup.Status, backupStatus)
    assertContainsSameElements(t, backupIncludePath, backup.IncludePath)
    assertContainsSameElements(t, backupExcludePath, backup.ExcludePath)
    assertContainsSameElements(t, table, backup.Table)
    assertContainsSameElements(t, excludedTables, backup.ExcludedTables)
    assert.Equal(t, uint(1000), backup.MirrorOptions.LifetimeInDays)
    assert.Equal(t, uint(1001), backup.SnapshotOptions.LifetimeInDays)
    assert.Equal(t, uint(1002), backup.ArchiveTTM)

    assert.Equal(t, CloudStorage, backup.Type, "type should not change")
}

func TestDefaultBackupRepository_UpdateBackup_ToDeletedAndNotStarted(t *testing.T) {
    backupRepository := setUpAndGetBackupRepository(t)
    ctx := context.Background()

    simpleBackupId := "simple-backup-id"

    _, err := backupRepository.AddBackup(ctx, createBackup(simpleBackupId, NotStarted, CloudStorage))
    require.NoError(t, err)

    err = backupRepository.UpdateBackup(ctx, UpdateFields{
        BackupID: "simple-backup-id",
        Status:   NotStarted,
    })
    require.NoError(t, err)

    backup, err := backupRepository.GetBackup(ctx, "simple-backup-id")
    require.NoError(t, err)
    assert.True(t, backup.DeletedTimestamp.IsZero())

    err = backupRepository.UpdateBackup(ctx, UpdateFields{
        BackupID: "simple-backup-id",
        Status:   BackupDeleted,
    })
    require.NoError(t, err)

    backup, err = backupRepository.GetBackup(ctx, "simple-backup-id")
    require.NoError(t, err)
    assert.False(t, backup.DeletedTimestamp.IsZero())
}

func TestDefaultBackupRepository_UpdateBackupOnlyOneRowShouldBeAffected(t *testing.T) {
    ctx, storageService := prepareTest(t)


    backupRepository := &defaultBackupRepository{storageService: storageService}

    simpleBackupId := "simple-backup-id"
    secondBackupId := "second-backup-id"

    _, err := backupRepository.AddBackup(ctx, createBackup(simpleBackupId, "", ""))
    assert.NoError(t, err)
    _, err = backupRepository.AddBackup(ctx, createBackup(secondBackupId, "", ""))
    assert.NoError(t, err)
    backupStatus := BackupDeleted
    backupIncludePath := []string{"first", "second"}
    backupExcludePath := []string{"third", "four"}
    tables := []string{"table-one", "table-two"}

    err = backupRepository.UpdateBackup(ctx, UpdateFields{
        BackupID:       simpleBackupId,
        Status:         backupStatus,
        IncludePath:    backupIncludePath,
        ExcludePath:    backupExcludePath,
        Table:          tables,
        ExcludedTables: nil,
        MirrorTTL:      0,
        SnapshotTTL:    0,
        ArchiveTTM:     0,
    })
    assert.NoError(t, err)

    backup, err := backupRepository.GetBackup(ctx, simpleBackupId)
    assert.NoError(t, err)
    assert.NotNil(t, backup)
    assert.Equal(t, backup.Status, backupStatus)
    assertContainsSameElements(t, backupIncludePath, backup.IncludePath)
    assertContainsSameElements(t, backupExcludePath, backup.ExcludePath)
    assertContainsSameElements(t, tables, backup.Table)

    count, _ := storageService.DB().
        Model(&Backup{}).Where("status = ?", BackupDeleted).Count()
    assert.Equal(t, 1, count)
}

func TestDefaultBackupRepository_UpdateLastScheduledTime(t *testing.T) {
    ctx, storageService := prepareTest(t)


    backupRepository := &defaultBackupRepository{storageService: storageService}

    err := setBackups(backupRepository.storageService, []*Backup{
        {ID: "backup-id-1232", Status: Paused},
        {ID: "backup-id-2412", Status: Paused},
    })
    assert.NoError(t, err)

    lastScheduledTime, _ := time.Parse("2006-01-02", "2000-01-01")

    err = backupRepository.UpdateLastScheduledTime(ctx, "backup-id-1232", lastScheduledTime, Finished)
    assert.NoError(t, err)

    count, _ := storageService.DB().Model(&Backup{}).Where("status = ?", Finished).Count()
    assert.Equal(t, 1, count, "one row should be affected")

    count, _ = storageService.DB().Model(&Backup{}).Where("DATE(last_scheduled_timestamp) = '2000-01-01'").Count()
    assert.Equal(t, 1, count, "one row should be affected")
}

func TestDefaultBackupRepository_UpdateLastCleanupTime(t *testing.T) {
    ctx, storageService := prepareTest(t)


    backupRepository := &defaultBackupRepository{storageService: storageService}

    err := setBackups(backupRepository.storageService, []*Backup{
        {ID: "backup-id-1232", Status: Paused},
        {ID: "backup-id-2412", Status: Paused},
    })
    assert.NoError(t, err)

    lastCleanupTime, _ := time.Parse("2006-01-02", "2000-01-01")

    err = backupRepository.UpdateLastCleanupTime(ctx, "backup-id-1232", lastCleanupTime)
    assert.NoError(t, err)

    count, _ := storageService.DB().Model(&Backup{}).Where("DATE(last_cleanup_timestamp) = '2000-01-01'").Count()
    assert.Equal(t, 1, count, "one row should be affected")
}

func TestDefaultBackupRepository_GetByBackupStatus(t *testing.T) {
    ctx, storageService := prepareTest(t)


    backupRepository := &defaultBackupRepository{storageService: storageService}

    err := setBackups(backupRepository.storageService, []*Backup{
        {ID: "backup-id-1232", Status: Paused},
        {ID: "backup-id-2412", Status: Paused},

        {ID: "backup-id-3412", Status: Finished},

        {ID: "backup-id-4412", Status: BackupDeleted},
        {ID: "backup-id-5412", Status: BackupDeleted},
        {ID: "backup-id-6412", Status: BackupDeleted},
    })
    assert.NoError(t, err)

    backupsWithStatusDeleted, err := backupRepository.GetByBackupStatus(ctx, BackupDeleted)
    assert.NoError(t, err)
    assert.Len(t, backupsWithStatusDeleted, 3)

    backupsWithStatusFinished, err := backupRepository.GetByBackupStatus(ctx, Finished)
    assert.NoError(t, err)
    assert.Len(t, backupsWithStatusFinished, 1)
    assert.Equal(t, "backup-id-3412", backupsWithStatusFinished[0].ID)

    backupsWithStatusPaused, err := backupRepository.GetByBackupStatus(ctx, Paused)
    assert.NoError(t, err)
    assert.Len(t, backupsWithStatusPaused, 2)

    backupsWithStatusNotStarted, err := backupRepository.GetByBackupStatus(ctx, NotStarted)
    assert.NoError(t, err)
    assert.Len(t, backupsWithStatusNotStarted, 0)
}

func TestDefaultBackupRepository_GetByBackupStrategy(t *testing.T) {
    ctx, storageService := prepareTest(t)


    backupRepository := &defaultBackupRepository{storageService: storageService}

    err := setBackups(backupRepository.storageService, []*Backup{
        {ID: "backup-id-1232", Strategy: Mirror},
        {ID: "backup-id-2412", Strategy: Mirror},

        {ID: "backup-id-3412"},
    })
    assert.NoError(t, err)

    backupsWithStrategyMirror, err := backupRepository.GetByBackupStrategy(ctx, Mirror)
    assert.NoError(t, err)
    assert.Len(t, backupsWithStrategyMirror, 2)

    backupsWithStrategySnapshot, err := backupRepository.GetByBackupStrategy(ctx, Snapshot)
    assert.NoError(t, err)
    assert.Len(t, backupsWithStrategySnapshot, 0)
}

func TestDefaultBackupRepository_GetExpiredBigQueryMirrorRevisions_WithOneExpiredMirrorRevision(t *testing.T) {
    backups := []Backup{
        {ID: "backup-id-1", MirrorOptions: MirrorOptions{LifetimeInDays: 1}, Status: NotStarted, Strategy: Mirror},
    }
    jobs := []Job{
        {ID: "job-id-1", Status: FinishedOk},
    }
    metadata := []SourceMetadata{
        {ID: 1, BackupID: "backup-id-1", CreatedTimestamp: time.Now().AddDate(0,0, -2)},
    }
    metadataJobs := []SourceMetadataJob{
        {SourceMetadataID: 1, JobId: "job-id-1"},
    }

    ctx, storageService := prepareTest(t)

    repository := &defaultBackupRepository{storageService: storageService}

    err := setDatabase(storageService, backups, jobs, metadata, metadataJobs)
    assert.NoError(t, err)

    revisions, err := repository.GetExpiredBigQueryMirrorRevisions(ctx, 0)
    assert.NoError(t, err)
    assert.Len(t, revisions, 1)
}

func TestDefaultBackupRepository_GetExpiredBigQueryMirrorRevisions_WithNoExpiredMirrorRevision(t *testing.T) {
    backups := []Backup{
        {ID: "backup-id-1", MirrorOptions: MirrorOptions{LifetimeInDays: 3}, Status: NotStarted, Strategy: Mirror},
    }
    jobs := []Job{
        {ID: "job-id-1", Status: FinishedOk},
    }
    metadata := []SourceMetadata{
        {ID: 1, BackupID: "backup-id-1", CreatedTimestamp: time.Now().AddDate(0,0, -2)},
    }
    metadataJobs := []SourceMetadataJob{
        {SourceMetadataID: 1, JobId: "job-id-1"},
    }

    ctx, storageService := prepareTest(t)

    repository := &defaultBackupRepository{storageService: storageService}

    err := setDatabase(storageService, backups, jobs, metadata, metadataJobs)
    assert.NoError(t, err)

    revisions, err := repository.GetExpiredBigQueryMirrorRevisions(ctx, 0)
    assert.NoError(t, err)
    assert.Len(t, revisions, 0)
}

func TestDefaultBackupRepository_GetScheduledBackups(t *testing.T) {
    backups := []*Backup{
        {ID: "backup-id-1232", Type: BigQuery, Status: NotStarted},
        {ID: "backup-id-2232", Type: BigQuery, Status: Prepared},
        {ID: "backup-id-3232", Type: BigQuery, Status: Finished},
        {ID: "backup-id-4232", Type: CloudStorage, Status: Paused},
    }


    ctx, storageService := prepareTest(t)


    backupRepository := &defaultBackupRepository{storageService: storageService}

    err := setBackups(backupRepository.storageService, backups)
    assert.NoError(t, err)

    scheduledBackups, err := backupRepository.GetScheduledBackups(ctx, BigQuery)
    assert.NoError(t, err)
    assert.Len(t, scheduledBackups, 2)

    scheduledBackups, err = backupRepository.GetScheduledBackups(ctx, CloudStorage)
    assert.NoError(t, err)
    assert.Len(t, scheduledBackups, 0)
}

func assertContainsSameElements(t *testing.T, first []string, second []string) {
    for _, path := range first {
        assert.Contains(t, second, path)
    }

    for _, path := range second {
        assert.Contains(t, first, path)
    }
}

func setUpAndGetBackupRepository(t *testing.T) *defaultBackupRepository {
    options := getTestConnectOptions()
    ctx := context.Background()

    storageService, err := service.NewStorageServiceWithConnectionOptions(ctx, options)
    assert.NoError(t, err)

    err = clearDatabase(storageService)
    assert.NoError(t, err)

    return &defaultBackupRepository{storageService: storageService}
}

func createBackup(id string, status BackupStatus, typ BackupType) *Backup {
    return &Backup{
        ID:                id,
        Status:            status,
        Type:              typ,
        Strategy:          "",
        SourceProject:     "",
        LastScheduledTime: time.Time{},
        LastCleanupTime:   time.Time{},
        SinkOptions:       SinkOptions{},
        SnapshotOptions:   SnapshotOptions{},
        BackupOptions:     BackupOptions{},
        EntityAudit:       EntityAudit{},
        MirrorOptions:     MirrorOptions{},
    }
}
