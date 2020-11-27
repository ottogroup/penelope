package provider

import (
    "context"
    "fmt"
    "github.com/ottogroup/penelope/pkg/config"
    "github.com/ottogroup/penelope/pkg/service/gcs"
    "go.opencensus.io/trace"
    "gopkg.in/yaml.v2"
)

type SinkGCPProjectProvider interface {
    GetSinkGCPProjectID(ctxIn context.Context, sourceGCPProjectID string) (string, error)
}

type defaultGCPProjectProvider struct {
    client gcs.CloudStorageClient
}

func NewDefaultGCPBackupProvider(ctxIn context.Context, gcsClient gcs.CloudStorageClient) (SinkGCPProjectProvider, error) {
    ctx, span := trace.StartSpan(ctxIn, "NewDefaultGCPBackupProvider")
    defer span.End()

    if gcsClient == nil || !gcsClient.IsInitialized(ctx) {
        return &defaultGCPProjectProvider{}, fmt.Errorf("can not create instance of defaultGCSBackupProvider with unititialized GcsClient")
    }

    return &defaultGCPProjectProvider{gcsClient}, nil
}

func (p *defaultGCPProjectProvider) GetSinkGCPProjectID(ctxIn context.Context, sourceID string) (string, error) {
    ctx, span := trace.StartSpan(ctxIn, "(*defaultGCPProjectProvider).GetSinkGCPProjectID")
    defer span.End()

    bucketName := config.DefaultProviderBucketEnv.MustGet()
    objectName := config.DefaultProviderSinkForProjectPathEnv.MustGet()

    object, err := p.client.ReadObject(ctx, bucketName, objectName)
    if err != nil {
        return "", err
    }

    var projectBackups []struct{
        Project string
        Backup  string
    }

    if err = yaml.Unmarshal(object, &projectBackups); err != nil {
        return "", fmt.Errorf("can not parse yaml file %s", err)
    }

    for _, projectBackup := range projectBackups {
        if projectBackup.Project == sourceID {
            return projectBackup.Backup, nil
        }
    }

    return "", fmt.Errorf("could not find backup for %s in backupProjectsPath %s", sourceID, config.DefaultProviderSinkForProjectPathEnv.MustGet())
}
