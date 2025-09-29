package bigquery

import (
	"context"
	"os"
	"testing"

	"github.com/ottogroup/penelope/pkg/http/mock"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("PENELOPE_USE_DEFAULT_HTTP_CLIENT", "true")
}

func TestNewBigQueryClient(t *testing.T) {
	httpMockHandler := mock.NewHTTPMockHandler()
	httpMocks := []mock.MockedHTTPRequest{ // /bigquery/v2/projects/.*/datasets/unknown-dataset
		mock.ImpersonationHTTPMock, mock.RetrieveAccessTokenHTTPMock,
		mock.DatasetInfoHTTPMock, mock.TableInfoHTTPMock,
		mock.TablePartitionJobHTTPMock, mock.TableMetadataPartitionResultHTTPMock,
		mock.ExtractJobResultOkHTTPMock,
	}
	httpMockHandler.Register(httpMocks...)
	httpMockHandler.Start()
	defer httpMockHandler.Stop()

	ctx := context.Background()
	provider := &mockImpersonatedTokenConfigProvider{}
	client, err := NewBigQueryClient(ctx, provider, "test-project", "test-project-backup")
	assert.NoError(t, err)

	partitions, err := client.GetTablePartitions(ctx, "test-project", "example-dataset", "example-table")
	assert.NoError(t, err)
	assert.NotEmpty(t, partitions)
	assert.Len(t, partitions, 2)
}

type mockImpersonatedTokenConfigProvider struct {
}

func (mi *mockImpersonatedTokenConfigProvider) GetTargetPrincipalForProject(_ context.Context, _ string) (string, []string, error) {
	return "example@test-project-backup.iam.gserviceaccount.com", nil, nil
}
