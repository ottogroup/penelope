package rest

import (
    "encoding/json"
    "fmt"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/ottogroup/penelope/pkg/http/auth"
    "github.com/ottogroup/penelope/pkg/http/auth/model"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "github.com/ottogroup/penelope/pkg/http/mock"
    "github.com/ottogroup/penelope/pkg/provider"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "github.com/ottogroup/penelope/pkg/secret"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"
)

func restAPIFactoryWithRealFactory(t *testing.T, principalRoleBindings []model.ProjectRoleBinding, backupProvider provider.SinkGCPProjectProvider, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) *httptest.Server {
    emptyTokenValidator := auth.NewEmptyTokenValidator()
    authenticationMiddleware, err := auth.NewAuthenticationMiddleware(emptyTokenValidator, givenDefaultPrincipalRetrieverWithRoles(principalRoleBindings))
    if err != nil {
        t.Error("expected", "instance of AuthenticationMiddleware can be created", "got", fmt.Sprintf("error: %s", err))
        os.Exit(1)
    }
    app := NewRestAPI(createBuilder(backupProvider, tokenSourceProvider, secret.NewEnvSecretProvider()), authenticationMiddleware, nil, secret.NewEnvSecretProvider())
    return httptest.NewServer(authenticationMiddleware.AddAuthentication(app.ServeHTTP))
}

func TestCreateSnapshotRequestOneShot(t *testing.T) {
    httpMockHandler.Start()
    defer httpMockHandler.Stop()

    mockBackupProvider := &mockBackupProvider{
        Backup: "gcp-project-backup",
        Error:  nil,
    }

    tokenConfigProvider := &MockImpersonatedTokenConfigProvider{
        TargetPrincipal: "backup-project@local-test-prod.iam.gserviceaccount.com",
        Error:           nil,
    }

    s := restAPIFactoryWithRealFactory(t, []model.ProjectRoleBinding{{
        Role:    model.Owner,
        Project: defaultProjectID,
    }}, mockBackupProvider, tokenConfigProvider)
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    body := requestobjects.CreateRequest{
        Strategy: "snapshot",
        Type:     "bigquery",
        Project:  defaultProjectID,
        TargetOptions: requestobjects.TargetOptions{
            Region: "europe-west1",
        },
        BigQueryOptions: requestobjects.BigQueryOptions{
            Dataset: "demo_delete_me_backup_target",
            Table:   []string{"bq_tables_storage_statistics"},
        },
    }

    resp, responseBody := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusCreated, resp.StatusCode)

    var entity requestobjects.BackupResponse
    err := json.Unmarshal(responseBody, &entity)
    require.NoError(t, err, "response body be json formated")

    require.NotEmpty(t, entity.ID, "response entity has an ID")
    defer func() { deleteBackup(entity.ID) }()

    require.NotEmpty(t, entity.TargetOptions.StorageClass, "response entity has an StorageClass")
    require.NotEmpty(t, entity.Sink, "response entity has an Sink")
    require.NotEmpty(t, entity.CreatedTimestamp, "response entity has an CreatedTimestamp")
    require.Empty(t, entity.UpdatedTimestamp, "response entity has no UpdatedTimestamp")
    require.Empty(t, entity.DeletedTimestamp, "response entity has no DeletedTimestamp")
}

func TestCreateSnapshotRequest_WithTTL(t *testing.T) {
    httpMockHandler.Start()
    defer httpMockHandler.Stop()

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

    body := requestobjects.CreateRequest{
        Strategy: "snapshot",
        Type:     "bigquery",
        Project:  defaultProjectID,
        TargetOptions: requestobjects.TargetOptions{
            Region: "europe-west1",
        },
        SnapshotOptions: requestobjects.SnapshotOptions{
            LifetimeInDays: 1,
        },
        BigQueryOptions: requestobjects.BigQueryOptions{
            Dataset: "demo_delete_me_backup_target",
            Table:   []string{"bq_tables_storage_statistics"},
        },
    }

    resp, responseBody := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusCreated, resp.StatusCode)

    var entity requestobjects.BackupResponse
    err := json.Unmarshal(responseBody, &entity)
    require.NoError(t, err, "response body be json formated")

    defer func() { deleteBackup(entity.ID) }()
    assert.NotEqual(t, entity.CreateRequest.SnapshotOptions.LifetimeInDays, -1, "response entity should have LifetimeInDays of 1")
}

func TestCreateSnapshotRequestOneShot_DatasetNotExisting(t *testing.T) {
    httpMockHandler.Cleanup()
    httpMocks := []mock.MockedHTTPRequest{ // /bigquery/v2/projects/.*/datasets/unknown-dataset
        mock.ImpersonationHTTPMock, mock.RetrieveAccessTokenHTTPMock,
        mock.DatasetNotFoundInfoHTTPMock, mock.TableInfoHTTPMock,
        mock.SinkNotExistsHTTPMock, mock.SinkCreatedHTTPpMock,
        mock.SinkDeletedHTTPMock, mock.TablePartitionQueryHTTPMock,
        mock.TablePartitionJobHTTPMock, mock.TablePartitionResultHTTPMock,
        mock.ExtractJobResultOkHTTPMock, mock.NewMockedHTTPRequest("GET", "/local-kebab-database/"+os.Getenv("CLOUD_SQL_SECRETS_PATH"), mock.SQLPasswordStorageResponse),
    }
    httpMockHandler.Register(httpMocks...)

    defer httpMockHandler.Stop()
    httpMockHandler.Start()

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

    body := requestobjects.CreateRequest{
        Strategy: "snapshot",
        Type:     "bigquery",
        Project:  defaultProjectID,
        TargetOptions: requestobjects.TargetOptions{
            Region: "europe-west1",
        },
        BigQueryOptions: requestobjects.BigQueryOptions{
            Dataset: "unknown-dataset",
            Table:   []string{"bq_tables_storage_statistics"},
        },
    }

    resp, _ := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusPreconditionFailed, resp.StatusCode)
}

func TestCreateSnapshotRequestOneShot_InsufficientPermissions(t *testing.T) {
    httpMockHandler = mock.NewHTTPMockHandler()
    httpMockHandler.Register(mock.ImpersonationHTTPMock, mock.RetrieveAccessTokenHTTPMock, mock.RetrieveTokenForbiddenHTTPMock, mock.DatasetNotAllowedInfoHTTPMock)
    httpMockHandler.Register(mock.NewMockedHTTPRequest("GET", "/local-kebab-database/"+os.Getenv("CLOUD_SQL_SECRETS_PATH"), mock.SQLPasswordStorageResponse))

    defer httpMockHandler.Stop()
    httpMockHandler.Start()

    mockBackupProvider := &mockBackupProvider{
        Backup: "gcp-project-backup",
        Error:  nil,
    }

    mockTokenConfigProvider := &MockImpersonatedTokenConfigProvider{
        TargetPrincipal: "backup-project@local-test-prod.iam.gserviceaccount.com",
        Error:           nil,
    }

    s := restAPIFactoryWithRealFactory(t, []model.ProjectRoleBinding{{
        Role:    model.Viewer,
        Project: defaultProjectID,
    }}, mockBackupProvider, mockTokenConfigProvider)
    defer s.Close()
    httpMockHandler.RegisterLocalServer(s.URL)

    body := requestobjects.CreateRequest{
        Strategy: "snapshot",
        Type:     "bigquery",
        Project:  defaultProjectID,
        TargetOptions: requestobjects.TargetOptions{
            Region: "europe-west1",
        },
        BigQueryOptions: requestobjects.BigQueryOptions{
            Dataset: "not-allowed-dataset",
            Table:   []string{"bq_tables_storage_statistics"},
        },
    }

    resp, _ := post(t, s, buildBackupRequestPath(), body)
    assert.Equal(t, http.StatusPreconditionFailed, resp.StatusCode)
}

