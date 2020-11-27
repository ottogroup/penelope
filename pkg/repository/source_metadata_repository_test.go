package repository

import (
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/ottogroup/penelope/pkg/service"
    "testing"
    "time"
)

func TestDefaultSourceMetadataRepository_Add_One(t *testing.T) {
    const backupId = "backup-id-102920"
    ctx, repository, _ := prepareTestForDefaultSourceMetadataRepository(t, backupId)
    
    sms, err := repository.Add(ctx, []*SourceMetadata{
        {
            BackupID: backupId,
        },
    })
    assert.NoError(t, err, "add source metadata failed")
    assert.Len(t, sms, 1)
}

func TestDefaultSourceMetadataRepository_Add_Two(t *testing.T) {
    const backupId = "backup-id-102920"
    ctx, repository, _ := prepareTestForDefaultSourceMetadataRepository(t, backupId)
    
    sms, err := repository.Add(ctx, []*SourceMetadata{
        {
            BackupID: backupId,
        },
        {
            BackupID: backupId,
        },
    })
    assert.NoError(t, err, "add source metadata failed")
    assert.Len(t, sms, 2)
}

func TestDefaultSourceMetadataRepository_GetLastByBackupID(t *testing.T) {
    const backupId = "backup-id-102920"
    ctx, repository, _ := prepareTestForDefaultSourceMetadataRepository(t, backupId)
    
    yesterday := time.Now().Add(-24*time.Hour)
    now := time.Now()
    sms, err := repository.Add(ctx, []*SourceMetadata{
        {
            BackupID: backupId,
            CreatedTimestamp: yesterday,
        },
        {
            BackupID: backupId,
            CreatedTimestamp: now,
        },
    })
    assert.NoError(t, err, "add source metadata failed")
    assert.Len(t, sms, 2)

    latestSms, err := repository.GetLastByBackupID(ctx, backupId)
    assert.NoError(t, err)
    assert.NotNil(t, latestSms)
    assert.Len(t, latestSms, 1)
    assert.Equal(t, latestSms[0].CreatedTimestamp.Day(), now.Day())
    assert.Equal(t, latestSms[0].CreatedTimestamp.Month(), now.Month())
    assert.Equal(t, latestSms[0].CreatedTimestamp.Year(), now.Year())
}

func TestDefaultSourceMetadataRepository_GetLastByBackupID_SameTime(t *testing.T) {
    const backupId = "backup-id-102920"
    ctx, repository, _ := prepareTestForDefaultSourceMetadataRepository(t, backupId)
    
    now := time.Now()
    sms, err := repository.Add(ctx, []*SourceMetadata{
        {
            BackupID: backupId,
            CreatedTimestamp: now,
        },
        {
            BackupID: backupId,
            CreatedTimestamp: now,
        },
    })
    assert.NoError(t, err, "add source metadata failed")
    assert.Len(t, sms, 2)

    latestSms, err := repository.GetLastByBackupID(ctx, backupId)
    assert.NoError(t, err)
    assert.NotNil(t, latestSms)
    assert.Len(t, latestSms, 2)
}

func TestDefaultSourceMetadataRepository_MarkDeleted(t *testing.T) {
    const backupId = "backup-id-102920"
    ctx, repository, _ := prepareTestForDefaultSourceMetadataRepository(t, backupId)
    
    sms, err := repository.Add(ctx, []*SourceMetadata{
        {
            BackupID: backupId,
        },
    })
    assert.NoError(t, err, "add source metadata failed")
    assert.Len(t, sms, 1)

    err = repository.MarkDeleted(ctx, sms[0].ID)
    assert.NoError(t, err)

    latestSms, err := repository.GetLastByBackupID(ctx, backupId)
    assert.NoError(t, err)
    assert.NotNil(t, latestSms)
    assert.Len(t, latestSms, 1)
    assert.Equal(t, latestSms[0].DeletedTimestamp.Year(), time.Now().Year())
    assert.Equal(t, latestSms[0].DeletedTimestamp.Month(), time.Now().Month())
    assert.Equal(t, latestSms[0].DeletedTimestamp.Day(), time.Now().Day())
}

func TestDefaultSourceMetadataRepository_MarkDeleted_MultipleBackups(t *testing.T) {
    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3"}
    ctx, repository, err := prepareTestForDefaultSourceMetadataRepository(t, backupIDs...)
    assert.NoError(t, err)

    sourceMetadata := []*SourceMetadata{
        {BackupID: "backup-id-1"},
        {BackupID: "backup-id-2"},
        {BackupID: "backup-id-3"},
    }

    sms, err := repository.Add(ctx, sourceMetadata)
    assert.NoError(t, err)
    assert.Len(t, sms, 3)

    err = repository.MarkDeleted(ctx, sms[0].ID)
    assert.NoError(t, err)

    count, err := repository.storageService.DB().
        Model(&SourceMetadata{}).
        Where("audit_deleted_timestamp IS NULL").
        Count()
    assert.NoError(t, err)
    assert.Equal(t, count, 2)

    metadata := SourceMetadata{ID: sms[0].ID}
    err = repository.storageService.DB().
        Model(&metadata).
        WherePK().
        Select()

    assert.NoError(t, err)
    assert.Equal(t, metadata.DeletedTimestamp.Day(), time.Now().Day())
    assert.Equal(t, metadata.DeletedTimestamp.Month(), time.Now().Month())
    assert.Equal(t, metadata.DeletedTimestamp.Year(), time.Now().Year())
}

func setBackupWithIDs(t *testing.T, storageService *service.Service, backups ...string) {
    for _, id := range backups {
        err := setBackups(storageService, []*Backup{
            {
                ID: id,
            },
        })
        require.NoError(t, err)
    }
}

func prepareTest(t *testing.T) (context.Context, *service.Service) {
    options := getTestConnectOptions()
    ctx := context.Background()

    storageService, err := service.NewStorageServiceWithConnectionOptions(ctx, options)
    assert.NoError(t, err)

    err = clearDatabase(storageService)
    assert.NoError(t, err)

    return ctx, storageService
}

func prepareTestForDefaultSourceMetadataRepository(t *testing.T, backupID ...string) (context.Context, defaultSourceMetadataRepository, error) {
    ctx, storageService := prepareTest(t)
    setBackupWithIDs(t, storageService, backupID...)
    repository := defaultSourceMetadataRepository{storageService: storageService}
    return ctx, repository, nil
}

func setBackups(client *service.Service, backups []*Backup) error {
    for _, backup := range backups {
        if _, err := client.DB().Model(backup).Insert(); err != nil {
            return err
        }
    }
    return nil
}
