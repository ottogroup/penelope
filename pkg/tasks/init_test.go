package tasks

import (
    "context"
    "flag"
    "github.com/ottogroup/penelope/pkg/http/mock"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/service"
    "os"
    "testing"
)

var httpMockHandler *mock.HTTPMockHandler

func init() {
    testing.Init()
    os.Setenv("GCP_PROJECT_ID", "local-project")
    os.Setenv("POSTGRES_HOST", "127.0.0.1")
    os.Setenv("POSTGRES_USER", "backupuser")
    os.Setenv("POSTGRES_DB", "backupdatabase")
    os.Setenv("POSTGRES_PASSWORD", "backupuserpassword")

    os.Setenv("DEFAULT_BUCKET_STORAGE_CLASS", "REGIONAL")
    os.Setenv("CLOUD_SQL_SECRETS_PATH", "path/to/secret")
    os.Setenv("CLOUD_SQL_SECRETS_READING_STRATEGY", "ENV")

    os.Setenv("PENELOPE_USE_DEFAULT_HTTP_CLIENT", "true")

    flag.Lookup("logtostderr").Value.Set("true")
    flag.Parse()

    httpMockHandler = mock.NewHTTPMockHandler()
    httpMockHandler.Register(mock.OauthHTTPMock, mock.ImpersonationHTTPMock, mock.RetrieveAccessTokenHTTPMock, mock.TablePartitionQueryHTTPMock, mock.TableInfoHTTPMock, mock.DatasetInfoHTTPMock)
    httpMockHandler.Register(mock.ObjectsExistsHTTPMock, mock.SinkNotExistsHTTPMock, mock.SinkCreatedHTTPpMock, mock.SinkDeletedHTTPMock)
    httpMockHandler.Register(mock.TablePartitionJobHTTPMock, mock.TablePartitionResultHTTPMock, mock.ExtractJobResultOkHTTPMock)
    httpMockHandler.Register(mock.NewMockedHTTPRequest("GET", "/local-kebab-database/"+os.Getenv("CLOUD_SQL_SECRETS_PATH"), mock.SQLPasswordStorageResponse))

    defer httpMockHandler.Stop()
    httpMockHandler.Start()
    storageService, err := service.NewStorageService(context.Background(), secret.NewEnvSecretProvider())
    if err != nil {
        panic(err)
    }

    storageService.DB().Model(&repository.SourceTrashcan{}).Where("true").Delete()
    storageService.DB().Model(&repository.SourceMetadata{}).Where("true").Delete()
    storageService.DB().Model(&repository.SourceMetadataJob{}).Where("true").Delete()
    storageService.DB().Model(&repository.Job{}).Where("true").Delete()
    storageService.DB().Model(&repository.Backup{}).Where("true").Delete()
}

type MockImpersonatedTokenConfigProvider struct {
    TargetPrincipal string
    Error           error
}

func (mi *MockImpersonatedTokenConfigProvider) GetTargetPrincipalForProject(context.Context, string) (string, error) {
    return mi.TargetPrincipal, mi.Error
}
