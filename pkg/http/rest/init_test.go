package rest

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

const defaultProjectID = "gcp-project-id"
const tokenHeaderKey = "X-Goog-IAP-JWT-Assertion"

func init() {
	testing.Init()
	os.Setenv("GCP_PROJECT_ID", "local-project")

	if os.Getenv("POSTGRES_HOST") == "" {
		os.Setenv("POSTGRES_HOST", "127.0.0.1")
	}
	if os.Getenv("POSTGRES_USER") == "" {
		os.Setenv("POSTGRES_USER", "backupuser")
	}
	if os.Getenv("POSTGRES_DB") == "" {
		os.Setenv("POSTGRES_DB", "backupdatabase")
	}
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		os.Setenv("POSTGRES_PASSWORD", "backupuserpassword")
	}

	os.Setenv("DEFAULT_BUCKET_STORAGE_CLASS", "REGIONAL")
	os.Setenv("CLOUD_SQL_SECRETS_PATH", "path/to/secret1")
	os.Setenv("CLOUD_SQL_SECRETS_READING_STRATEGY", "ENV")

	os.Setenv("PENELOPE_USE_DEFAULT_HTTP_CLIENT", "true")
	os.Setenv("PENELOPE_USE_GRPC_WITHOUT_AUTHENTICATION", "true")
	os.Setenv("TOKEN_HEADER_KEY", tokenHeaderKey)

	os.Setenv("TASKS_VALIDATION_HTTP_HEADER_NAME", "")
	os.Setenv("TASKS_VALIDATION_HTTP_HEADER_VALUE", "")
	os.Setenv("TASKS_VALIDATION_ALLOWED_IP_ADDRESSES", "")

	flag.Lookup("logtostderr").Value.Set("true")
	flag.Parse()

	sqlSecretPath := "/local-kebab-database/" + os.Getenv("CLOUD_SQL_SECRETS_PATH")
	httpMocks := []mock.MockedHTTPRequest{
		mock.ImpersonationHTTPMock, mock.RetrieveAccessTokenHTTPMock,
		mock.DatasetInfoHTTPMock, mock.TableInfoHTTPMock,
		mock.SinkNotExistsHTTPMock, mock.SinkCreatedHTTPpMock,
		mock.SinkDeletedHTTPMock, mock.TablePartitionQueryHTTPMock,
		mock.TablePartitionJobHTTPMock, mock.TablePartitionResultHTTPMock,
		mock.ExtractJobResultOkHTTPMock, mock.NewMockedHTTPRequest("GET", sqlSecretPath, mock.SQLPasswordStorageResponse),
	}

	httpMockHandler = mock.NewHTTPMockHandler()
	httpMockHandler.Register(httpMocks...)

	storageService, err := service.NewStorageService(context.Background(), secret.NewEnvSecretProvider())
	if err != nil {
		panic(err)
	}
	storageService.DB().Model(&repository.Job{}).Where("true").Delete()
	storageService.DB().Model(&repository.SourceMetadata{}).Where("true").Delete()
	storageService.DB().Model(&repository.SourceMetadataJob{}).Where("true").Delete()
	storageService.DB().Model(&repository.Backup{}).Where("true").Delete()
	storageService.DB().Model(&repository.SourceTrashcan{}).Where("true").Delete()
}
