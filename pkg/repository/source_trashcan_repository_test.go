package repository

import (
    "context"
    "github.com/stretchr/testify/assert"
    "testing"
    "time"
)

func TestDefaultSourceTrashcan_Add(t *testing.T) {
    const backupID = "backup-id-192023"
    const sourceID = "source-id-193923"
    timestamp := time.Now()

    ctx, repository, err := prepareTestForDefaultSourceTrashcanRepository(t, backupID)

    assert.NoError(t, err)

    err = repository.Add(ctx, backupID, sourceID, timestamp)
    assert.NoError(t, err)
}

func TestDefaultSourceTrashcan_Add_UniqueSource(t *testing.T) {
    const backupID = "backup-id-192023"
    const sourceID = "source-id-193923"
    timestamp := time.Now()

    ctx, repository, err := prepareTestForDefaultSourceTrashcanRepository(t, backupID)

    assert.NoError(t, err)

    err = repository.Add(ctx, backupID, sourceID, timestamp)
    assert.NoError(t, err)

    err = repository.Add(ctx, backupID, sourceID, timestamp)
    assert.Error(t, err)
}

func TestDefaultSourceTrashcan_Delete(t *testing.T) {
    const backupID = "backup-id-192023"
    const sourceID = "source-id-193923"
    timestamp := time.Now()

    ctx, repository, err := prepareTestForDefaultSourceTrashcanRepository(t, backupID)

    assert.NoError(t, err)

    err = repository.Add(ctx, backupID, sourceID, timestamp)
    assert.NoError(t, err)

    err = repository.Delete(ctx, backupID, sourceID)
    assert.NoError(t, err)

    err = repository.Add(ctx, backupID, sourceID, timestamp)
    assert.NoError(t, err)
}

func TestDefaultSourceTrashcan_FilterExistingEntries_Empty(t *testing.T) {
    const backupID = "backup-id-192023"
    const sourceID = "source-id-193923"

    ctx, repository, err := prepareTestForDefaultSourceTrashcanRepository(t, backupID)

    assert.NoError(t, err)

    result, err := repository.FilterExistingEntries(ctx, []SourceTrashcan{
        {BackupID: backupID, Source: sourceID},
    })
    assert.NoError(t, err)
    assert.Len(t, result, 0)
}

func TestDefaultSourceTrashcan_FilterExistingEntries_Single(t *testing.T) {
    const backupID = "backup-id-192023"
    const sourceID = "source-id-193923"
    timestamp := time.Now()

    ctx, repository, err := prepareTestForDefaultSourceTrashcanRepository(t, backupID)

    assert.NoError(t, err)

    err = repository.Add(ctx, backupID, sourceID, timestamp)
    assert.NoError(t, err)

    result, err := repository.FilterExistingEntries(ctx, []SourceTrashcan{
        {BackupID: backupID, Source: sourceID},
    })
    assert.NoError(t, err)
    assert.Len(t, result, 1)
}

func TestDefaultSourceTrashcan_FilterExistingEntries_Multiple(t *testing.T) {
    timestamp := time.Now()

    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3","backup-id-4"}

    sourceTrashcans := []SourceTrashcan{
        {BackupID: "backup-id-1", Source: "source-id-1", CreatedTimestamp: timestamp},
        {BackupID: "backup-id-2", Source: "source-id-2", CreatedTimestamp: timestamp},
        {BackupID: "backup-id-3", Source: "source-id-3", CreatedTimestamp: timestamp},
        {BackupID: "backup-id-3", Source: "source-id-4", CreatedTimestamp: timestamp},
        {BackupID: "backup-id-4", Source: "source-id-5", CreatedTimestamp: timestamp},
    }

    filterSourceTrashcans := []SourceTrashcan{
        {BackupID: "backup-id-2", Source: "source-id-2", CreatedTimestamp: timestamp},
        {BackupID: "backup-id-3", Source: "source-id-3", CreatedTimestamp: timestamp},
    }

    ctx, repository, err := prepareTestForDefaultSourceTrashcanRepository(t, backupIDs...)
    assert.NoError(t, err)

    for _, sourceTrashcan := range sourceTrashcans {
        err := repository.Add(ctx, sourceTrashcan.BackupID, sourceTrashcan.Source, sourceTrashcan.CreatedTimestamp)
        assert.NoError(t, err)
    }

    result, err := repository.FilterExistingEntries(ctx, filterSourceTrashcans)
    assert.NoError(t, err)
    assert.Len(t, result, 2)
}

func TestDefaultSourceTrashcan_GetBefore(t *testing.T) {
    beforeOneWeek := time.Now().AddDate(0, 0, -7)
    now := time.Now()


    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3","backup-id-4"}

    sourceTrashcans := []SourceTrashcan{
        {BackupID: "backup-id-1", Source: "source-id-1", CreatedTimestamp: now},
        {BackupID: "backup-id-2", Source: "source-id-2", CreatedTimestamp: beforeOneWeek},
        {BackupID: "backup-id-3", Source: "source-id-3", CreatedTimestamp: now},
        {BackupID: "backup-id-3", Source: "source-id-4", CreatedTimestamp: beforeOneWeek},
        {BackupID: "backup-id-4", Source: "source-id-5", CreatedTimestamp: now},
    }

    ctx, repository, err := prepareTestForDefaultSourceTrashcanRepository(t, backupIDs...)
    assert.NoError(t, err)

    for _, sourceTrashcan := range sourceTrashcans {
        err := repository.Add(ctx, sourceTrashcan.BackupID, sourceTrashcan.Source, sourceTrashcan.CreatedTimestamp)
        assert.NoError(t, err)
    }

    trashcans, err := repository.GetBefore(ctx, 1)
    assert.NoError(t, err)
    assert.Len(t, trashcans, 2)
}

func prepareTestForDefaultSourceTrashcanRepository(t *testing.T, backupIDs ...string) (context.Context, defaultSourceTrashcan, error) {
    ctx, storageService := prepareTest(t)
    setBackupWithIDs(t, storageService, backupIDs...)
    repository := defaultSourceTrashcan{storageService: storageService}
    return ctx, repository, nil
}
