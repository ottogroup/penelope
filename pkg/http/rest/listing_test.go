package rest

import (
    "context"
    "fmt"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/ottogroup/penelope/pkg/http/auth/model"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/secret"
    "net/http"
    "testing"
    "time"
)

func TestListing_WithEmptyResponse(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()

    s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    resp, respString := get(t, s, buildBackupRequestPath()+"?project=test-project")
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    assert.Equal(t, `{"backups":[]}`, respString)
}

func TestListing_WithSingleResponse(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()
    ctx := context.Background()

    backupRepository, err := repository.NewBackupRepository(context.Background(), secret.NewEnvSecretProvider())
    require.NoError(t, err, "BackupRepository should be instantiate")

    backupID := "test-backup-id"
    _, err = backupRepository.AddBackup(ctx, &repository.Backup{
        ID:            backupID,
        Status:        repository.Prepared,
        SourceProject: defaultProjectID, // gcp-project-id
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
        EntityAudit: repository.EntityAudit{
            CreatedTimestamp: time.Now(),
        },
    })
    require.NoError(t, err, "should add new backup")

    mockBackupProvider := &mockBackupProvider{
        Backup: "",
        Error:  nil,
    }

    mockTokenConfigProvider := &MockImpersonatedTokenConfigProvider{
        TargetPrincipal: "",
        Error:           nil,
    }

    defer func() { deleteBackup(backupID) }()
    s := restAPIFactoryWithRealFactory(t, []model.ProjectRoleBinding{{
        Role:    model.Viewer,
        Project: defaultProjectID,
    }}, mockBackupProvider, mockTokenConfigProvider)
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    resp, respString := get(t, s, buildBackupRequestPath()+fmt.Sprintf("?project=%s", defaultProjectID))
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    assert.NotEmpty(t, respString)
    assert.Contains(t, respString, defaultProjectID)
}
