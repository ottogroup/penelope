package provider

import (
	"context"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestDefaultSourceGCPProjectProvider_Found(t *testing.T) {
	_ = os.Setenv(config.DefaultProviderBucketEnv.String(), "local-xyz-dev.appspot.com")
	_ = os.Setenv(config.DefaultProviderGCPSourceProjectPathEnv.String(), "gcp-project-source.yaml")

	content := `
- project: local-account
  availability_class: A1
  data_owner: john.doe
`
	backupProvider, err := NewDefaultSourceGCPBackupProvider(context.Background(), &gcs.MockGcsClient{
		ClientInitialized: true,
		ShouldFail:        false,
		ObjectContent:     []byte(content),
	})
	assert.NoError(t, err)

	sourceProject, err := backupProvider.GetSourceGCPProject(context.Background(), "local-account")
	assert.NoError(t, err)
	assert.Equal(t, "john.doe", sourceProject.DataOwner)
	assert.Equal(t, A1Irrelevant, sourceProject.AvailabilityClass)
}

func TestDefaultSourceGCPProjectProvider_FoundFromCache(t *testing.T) {
	_ = os.Setenv(config.DefaultProviderBucketEnv.String(), "local-xyz-dev.appspot.com")
	_ = os.Setenv(config.DefaultProviderGCPSourceProjectPathEnv.String(), "gcp-project-source.yaml")

	content := `
- project: local-account
  availability_class: A1
  data_owner: john.doe
`
	mockGcsClient := &gcs.MockGcsClient{
		ClientInitialized: true,
		ShouldFail:        false,
		ObjectContent:     []byte(content),
	}
	backupProvider, err := NewDefaultSourceGCPBackupProvider(context.Background(), mockGcsClient)
	assert.NoError(t, err)

	sourceProject, err := backupProvider.GetSourceGCPProject(context.Background(), "local-account")
	assert.NoError(t, err)
	assert.Equal(t, "john.doe", sourceProject.DataOwner)
	assert.Equal(t, A1Irrelevant, sourceProject.AvailabilityClass)

	mockGcsClient.ObjectContent = []byte(`
- project: local-account
  availability_class: A2
  data_owner: john.doe2
`)
	sourceProject2, err := backupProvider.GetSourceGCPProject(context.Background(), "local-account")
	assert.NoError(t, err)
	assert.Equal(t, "john.doe", sourceProject2.DataOwner)
	assert.Equal(t, A1Irrelevant, sourceProject2.AvailabilityClass)

}

func TestDefaultSourceGCPProjectProvider_GetSinkGCPProjectID_NotFound(t *testing.T) {
	_ = os.Setenv(config.DefaultProviderBucketEnv.String(), "local-xyz-dev.appspot.com")
	_ = os.Setenv(config.DefaultProviderGCPSourceProjectPathEnv.String(), "gcp-project-source.yaml")

	content := ""
	backupProvider, err := NewDefaultSourceGCPBackupProvider(context.Background(), &gcs.MockGcsClient{
		ClientInitialized: true,
		ShouldFail:        false,
		ObjectContent:     []byte(content),
	})
	assert.NoError(t, err)

	sourceProject, err := backupProvider.GetSourceGCPProject(context.Background(), "local-account")
	assert.Error(t, err)
	assert.Equal(t, "", sourceProject.DataOwner)
	assert.Equal(t, A0Invalid, sourceProject.AvailabilityClass)
}
