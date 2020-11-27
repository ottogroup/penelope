package provider

import (
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/ottogroup/penelope/pkg/config"
    "github.com/ottogroup/penelope/pkg/service/gcs"
    "os"
    "testing"
)

func TestDefaultGCSBackupProvider_GetSinkGCPProjectID_Found(t *testing.T) {
    _ = os.Setenv(config.DefaultProviderBucketEnv.String(), "local-xyz-dev.appspot.com")
    _ = os.Setenv(config.DefaultProviderSinkForProjectPathEnv.String(), "project-backups.yaml")

    content := `
- project: local-account
  backup: local-account-backup
`
    backupProvider, err := NewDefaultGCPBackupProvider(context.Background(), &gcs.MockGcsClient{
        ClientInitialized: true,
        ShouldFail:        false,
        ObjectContent:     []byte(content),
    })
    assert.NoError(t, err)

    projectID, err := backupProvider.GetSinkGCPProjectID(context.Background(), "local-account")
    assert.NoError(t, err)
    assert.Equal(t, "local-account-backup", projectID)
}

func TestDefaultGCSBackupProvider_GetSinkGCPProjectID_NotFound(t *testing.T) {
    _ = os.Setenv(config.DefaultProviderBucketEnv.String(), "local-xyz-dev.appspot.com")
    _ = os.Setenv(config.DefaultProviderSinkForProjectPathEnv.String(), "project-backups.yaml")

    content := ""
    backupProvider, err := NewDefaultGCPBackupProvider(context.Background(), &gcs.MockGcsClient{
        ClientInitialized: true,
        ShouldFail:        false,
        ObjectContent:     []byte(content),
    })
    assert.NoError(t, err)

    projectID, err := backupProvider.GetSinkGCPProjectID(context.Background(), "local-account")
    assert.Error(t, err)
    assert.Equal(t, "", projectID)
}
