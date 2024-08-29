package tasks

import (
	"context"
	"fmt"
	"github.com/ottogroup/penelope/pkg/http/mock"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestCleanupTrashcansService_Schedule(t *testing.T) {
	httpMockHandler.Start()
	defer httpMockHandler.Stop()

	ctx := context.Background()
	service, err := newCleanupTrashcansService(ctx, provider.NewDefaultImpersonatedTokenConfigProvider(), secret.NewEnvSecretProvider())
	require.NoError(t, err)

	backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
	require.NoErrorf(t, err, "backupRepository should be instantiate")

	backup, err := backupRepository.AddBackup(ctx, scheduleTrashcanBackup())
	require.NoError(t, err, "should add new backup")
	defer func() { _ = deleteBackup(backup.ID) }()

	httpMockHandler.Register(mock.ListObjectsHTTPMockFunc(backup.Sink))

	_, stdErr, err := captureStderr(func() {
		service.Run(ctx)
	})

	fmt.Println(stdErr)

	cleanupTrashcanBucket, err := backupRepository.GetBackup(ctx, backup.ID)
	assert.Equal(t, repository.NoopCleanupTrashcanCleanupStatus, cleanupTrashcanBucket.TrashcanCleanup.Status)

	require.NoError(t, err)
	logMsg := "trashcan cleanup for backup completed:"
	if !strings.Contains(strings.TrimSpace(stdErr), logMsg) {
		t.Errorf("Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
	}
}

func TestCleanupTrashcansService_Noop(t *testing.T) {
	httpMockHandler.Start()
	defer httpMockHandler.Stop()

	ctx := context.Background()
	service, err := newCleanupTrashcansService(ctx, provider.NewDefaultImpersonatedTokenConfigProvider(), secret.NewEnvSecretProvider())
	require.NoError(t, err)

	backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
	require.NoErrorf(t, err, "backupRepository should be instantiate")

	backup, err := backupRepository.AddBackup(ctx, noopTrashcanBackup())
	require.NoError(t, err, "should add new backup")
	defer func() { _ = deleteBackup(backup.ID) }()

	_, stdErr, err := captureStderr(func() {
		service.Run(ctx)
	})

	cleanupTrashcanBucket, err := backupRepository.GetBackup(ctx, backup.ID)
	assert.Equal(t, repository.NoopCleanupTrashcanCleanupStatus, cleanupTrashcanBucket.TrashcanCleanup.Status)

	require.NoError(t, err)
	logMsg := ""
	if !strings.Contains(strings.TrimSpace(stdErr), logMsg) {
		t.Errorf("Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
	}
}

func scheduleTrashcanBackup() *repository.Backup {
	return &repository.Backup{
		ID:              "somerandom-id",
		Status:          repository.Prepared,
		TrashcanCleanup: repository.TrashcanCleanup{Status: repository.ScheduledTrashcanCleanupStatus},
		SourceProject:   "local-ability",
		Strategy:        repository.Snapshot,
		Type:            repository.BigQuery,
		SinkOptions: repository.SinkOptions{
			TargetProject: "local-ability-backup",
			Sink:          "uuid-5678-123456",
			Region:        "europe-west1",
			StorageClass:  "NEARLINE",
		},
		BackupOptions: repository.BackupOptions{
			BigQueryOptions: repository.BigQueryOptions{"demo_delete_me_backup_target", []string{"gcp_billing_budget_amount_plan"}, []string{}},
		},
	}
}

func noopTrashcanBackup() *repository.Backup {
	return &repository.Backup{
		ID:              "somerandom-id",
		Status:          repository.Prepared,
		TrashcanCleanup: repository.TrashcanCleanup{Status: repository.NoopCleanupTrashcanCleanupStatus},
		SourceProject:   "local-ability",
		Strategy:        repository.Snapshot,
		Type:            repository.BigQuery,
		SinkOptions: repository.SinkOptions{
			TargetProject: "local-ability-backup",
			Sink:          "uuid-5678-123456",
			Region:        "europe-west1",
			StorageClass:  "NEARLINE",
		},
		BackupOptions: repository.BackupOptions{
			BigQueryOptions: repository.BigQueryOptions{"demo_delete_me_backup_target", []string{"gcp_billing_budget_amount_plan"}, []string{}},
		},
	}
}
