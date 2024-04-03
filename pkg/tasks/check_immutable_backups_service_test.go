package tasks

import (
	"context"
	"github.com/ottogroup/penelope/pkg/http/mock"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/repository/memory"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheckImmutableBackupsService_Run_Unsafe(t *testing.T) {
	httpMockHandler.Start()
	defer httpMockHandler.Stop()

	httpMockHandler.Register(mock.ListPoliciesUnsafeHTTPMock)

	ctx := context.Background()
	backupRepository := &memory.BackupRepository{}
	impersonatedTokenConfigProvider := provider.NewDefaultImpersonatedTokenConfigProvider()
	service := &checkImmutableBackupsService{
		backupRepository:    backupRepository,
		tokenSourceProvider: impersonatedTokenConfigProvider,
	}

	backup := &repository.Backup{
		ID:              testBackupID,
		Status:          repository.NotStarted,
		Type:            repository.BigQuery,
		SnapshotOptions: repository.SnapshotOptions{},
		SinkOptions: repository.SinkOptions{
			TargetProject: "test-example-unsafe",
		},
	}
	_, _ = backupRepository.AddBackup(ctx, backup)

	service.Run(ctx)
	assert.False(t, backup.SinkIsImmutable, "target sink should be unsafe: %s", backup.TargetProject)
}

func TestCheckImmutableBackupsService_Run_Safe(t *testing.T) {
	httpMockHandler.Start()
	defer httpMockHandler.Stop()

	httpMockHandler.Register(mock.ListPoliciesSafeHTTPMock)

	ctx := context.Background()
	backupRepository := &memory.BackupRepository{}
	impersonatedTokenConfigProvider := provider.NewDefaultImpersonatedTokenConfigProvider()
	service := &checkImmutableBackupsService{
		backupRepository:    backupRepository,
		tokenSourceProvider: impersonatedTokenConfigProvider,
	}

	backup := &repository.Backup{
		ID:              testBackupID,
		Status:          repository.NotStarted,
		Type:            repository.BigQuery,
		SnapshotOptions: repository.SnapshotOptions{},
		SinkOptions: repository.SinkOptions{
			TargetProject: "test-example-safe",
		},
	}

	_, _ = backupRepository.AddBackup(ctx, backup)

	service.Run(ctx)
	assert.True(t, backup.SinkIsImmutable, "target sink should be safe: %s", backup.TargetProject)
}
