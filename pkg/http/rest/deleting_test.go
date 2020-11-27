package rest

import (
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/ottogroup/penelope/pkg/http/auth/model"
    "github.com/ottogroup/penelope/pkg/http/mock"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "github.com/ottogroup/penelope/pkg/secret"
    "net/http"
    "testing"
    "time"
)

const deletingBackupID = "test-backup-id"

func TestDeleting_WithNonExistingBackup(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()

    s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    body := requestobjects.UpdateRequest{
        BackupID: "NonExistingBackupID",
        Status:   repository.ToDelete.String(),
    }
    resp, _ := patch(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleting_WithScheduledBackup(t *testing.T) {
    httpMockHandler.Start()
    defer httpMockHandler.Stop()

    ctx := context.Background()

    httpMockHandler.Register(mock.BucketAttrsHTTPMock, mock.PatchBucketAttrsHTTPMock)

    backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
    require.NoError(t, err, "BackupRepository should be instantiate")

    backup := deletingBackup()
    backup.LastScheduledTime = time.Now()
    backup.Status = repository.Prepared
    _, err = backupRepository.AddBackup(ctx, backup)
    require.NoError(t, err, "should add new backup")

    mockBackupProvider := &mockBackupProvider{
        Backup: "gcp-project-backup",
        Error:  nil,
    }

    mockTokenConfigProvider := &MockImpersonatedTokenConfigProvider{
        TargetPrincipal: "backup-project@local-test-prod.iam.gserviceaccount.com",
        Error:           nil,
    }

    defer func() { deleteBackup(deletingBackupID) }()
    s := restAPIFactoryWithRealFactory(t, []model.ProjectRoleBinding{{
        Role:    model.Owner,
        Project: defaultProjectID,
    }}, mockBackupProvider, mockTokenConfigProvider)
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    body := requestobjects.UpdateRequest{
        BackupID: deletingBackupID,
        Status:   repository.ToDelete.String(),
    }
    resp, _ := patch(t, s, buildBackupRequestPath()+"/"+deletingBackupID, body)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    backup, err = backupRepository.GetBackup(ctx, deletingBackupID)
    require.NoError(t, err, "GetBackup with id %s should be found", deletingBackupID)
    assert.Equalf(t, repository.ToDelete, backup.Status, "GetBackup with id %s should be in state %s", deletingBackupID, repository.ToDelete)
}

func TestDeleting_WithNotScheduledBackup(t *testing.T) {
    defer httpMockHandler.Stop()
    httpMockHandler.Start()
    ctx := context.Background()

    mockBackupProvider := &mockBackupProvider{
        Backup: "gcp-project-backup",
        Error:  nil,
    }

    mockTokenConfigProvider := &MockImpersonatedTokenConfigProvider{
        TargetPrincipal: "backup-project@local-test-prod.iam.gserviceaccount.com",
        Error:           nil,
    }

    s := restAPIFactoryWithRealFactory(t, []model.ProjectRoleBinding{{
        Role:    model.Owner,
        Project: defaultProjectID,
    }}, mockBackupProvider, mockTokenConfigProvider)
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    backupRepository, err := repository.NewBackupRepository(context.Background(), secret.NewEnvSecretProvider())
    require.NoError(t, err, "BackupRepository should be instantiate")

    backup := deletingBackup()
    _, err = backupRepository.AddBackup(ctx, backup)
    require.NoError(t, err, "should add new backup")

    defer func() { deleteBackup(deletingBackupID) }()
    body := requestobjects.UpdateRequest{
        BackupID: deletingBackupID,
        Status:   repository.ToDelete.String(),
    }
    resp, _ := patch(t, s, buildBackupRequestPath()+"/"+deletingBackupID, body)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    backup, err = backupRepository.GetBackup(ctx, deletingBackupID)
    require.NoError(t, err, "GetBackup with id %s should be found", deletingBackupID)
    assert.Equalf(t, repository.BackupDeleted, backup.Status, "GetBackup with id %s should be in state %s", deletingBackupID, repository.ToDelete)
}

func deletingBackup() *repository.Backup {
    return &repository.Backup{
        ID:            deletingBackupID,
        Status:        repository.NotStarted,
        SourceProject: defaultProjectID,
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
    }
}
