package repository

import (
    "context"
    "github.com/aws/aws-sdk-go/service/backup"
    "github.com/stretchr/testify/assert"
    "github.com/ottogroup/penelope/pkg/service"
    "github.com/ottogroup/penelope/pkg/service/sql"
    "os"
    "testing"
)

func TestDefaultSourceMetadataJobRepository_Add_Simple(t *testing.T) {
    metadata := []SourceMetadata{
        {
            ID:             1,
            BackupID:       "",
            Source:         "",
            SourceChecksum: "",
            Operation:      "",
        },
    }

    jobs := []Job{
        {
            ID:           "123",
            BackupID:     "",
            Type:         "",
            Status:       backup.JobStateCreated,
            Source:       "",
            ForeignJobID: ForeignJobID{},
            EntityAudit:  EntityAudit{},
        },
    }

    sourceMetadataJob := SourceMetadataJob{
        SourceMetadataID: 1,
        JobId:            "123",
    }

    err := testAdd(t, metadata, jobs, &sourceMetadataJob)
    assert.NoError(t, err)
}

func TestDefaultSourceMetadataJobRepository_Add_UnknownJob(t *testing.T) {
    metadata := []SourceMetadata{
        {
            ID:             1,
            BackupID:       "",
            Source:         "",
            SourceChecksum: "",
            Operation:      "",
        },
    }

    var jobs []Job

    sourceMetadataJob := SourceMetadataJob{
        SourceMetadataID: 1,
        JobId:            "123",
    }

    err := testAdd(t, metadata, jobs, &sourceMetadataJob)
    assert.Error(t, err)
}

func TestDefaultSourceMetadataJobRepository_Add_UnknownSourceMetadata(t *testing.T) {
    var metadata []SourceMetadata

    jobs := []Job{
        {
            ID:           "123",
            BackupID:     "",
            Type:         "",
            Status:       backup.JobStateCreated,
            Source:       "",
            ForeignJobID: ForeignJobID{},
            EntityAudit:  EntityAudit{},
        },
    }

    sourceMetadataJob := SourceMetadataJob{
        SourceMetadataID: 1,
        JobId:            "123",
    }

    err := testAdd(t, metadata, jobs, &sourceMetadataJob)
    assert.Error(t, err)
}

func getTestConnectOptions() sql.ConnectOptions {
    host := "127.0.0.1"
    port := "5432"
    database := "backupdatabase"
    password := "backupuserpassword"
    user := "backupuser"
    debug := "false"

    if os.Getenv("POSTGRES_PORT") != "" {
        port = os.Getenv("POSTGRES_PORT")
    }
    if os.Getenv("POSTGRES_HOST") != "" {
        host = os.Getenv("POSTGRES_HOST")
    }
    if os.Getenv("POSTGRES_DB") != "" {
        database = os.Getenv("POSTGRES_DB")
    }
    if os.Getenv("POSTGRES_USER") != "" {
        user = os.Getenv("POSTGRES_USER")
    }
    if os.Getenv("POSTGRES_PASSWORD") != "" {
        password = os.Getenv("POSTGRES_PASSWORD")
    }
    if os.Getenv("POSTGRES_DEBUG_QUERIES") != "" {
        debug = os.Getenv("POSTGRES_DEBUG_QUERIES")
    }
    return sql.ConnectOptions{
        Host:         host,
        Port:         port,
        User:         user,
        Password:     password,
        Database:     database,
        DebugQueries: debug,
    }
}

func clearDatabase(client *service.Service) error {
    if _, err := client.DB().Model(new(SourceMetadataJob)).Where("true").Delete(); err != nil {
        return err
    }
    if _, err := client.DB().Model(new(SourceTrashcan)).Where("true").Delete(); err != nil {
        return err
    }
    if _, err := client.DB().Model(new(SourceMetadata)).Where("true").Delete(); err != nil {
        return err
    }
    if _, err := client.DB().Model(new(Job)).Where("true").Delete(); err != nil {
        return err
    }
    if _, err := client.DB().Model(new(Backup)).Where("true").Delete(); err != nil {
        return err
    }
    return nil
}

func setDatabase(client *service.Service, backups []Backup, jobs []Job, metadata []SourceMetadata, smjs []SourceMetadataJob) error {
    for _, b := range backups {
        if _, err := client.DB().Model(&b).Insert(); err != nil {
            return err
        }
    }

    for _, job := range jobs {
        if _, err := client.DB().Model(&job).Insert(); err != nil {
            return err
        }
    }

    for _, data := range metadata {
        if _, err := client.DB().Model(&data).Insert(); err != nil {
            return err
        }
    }

    for _, smj := range smjs {
        if _, err := client.DB().Model(&smj).Insert(); err != nil {
            return err
        }
    }

    return nil
}

func testAdd(t *testing.T, metadata []SourceMetadata, jobs []Job, sourceMetadataJob *SourceMetadataJob) error {
    options := getTestConnectOptions()
    ctx := context.Background()

    storageService, err := service.NewStorageServiceWithConnectionOptions(ctx, options)

    assert.NoError(t, err)

    err = clearDatabase(storageService)
    assert.NoError(t, err)

    err = setDatabase(storageService, []Backup{}, jobs, metadata, []SourceMetadataJob{})
    assert.NoError(t, err)

    repository := &DefaultSourceMetadataJobRepository{storageService: storageService}
    err = repository.Add(ctx, sourceMetadataJob.SourceMetadataID, sourceMetadataJob.JobId)
    return err
}
