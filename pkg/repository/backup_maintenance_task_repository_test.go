package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/stretchr/testify/require"
)

func TestMaintenanceTaskValidation(t *testing.T) {
	ctx, storageService := prepareTest(t)
	repo := &defaultBackupMaintenanceTaskRepository{storageService: storageService}
	for _, test := range []struct {
		backup, name string
		parameters   TaskParameters
	}{
		{"", "cleanup", nil},
		{"backup", " ", nil},
		{"backup", "cleanup", TaskParameters{}},
	} {
		_, err := repo.Add(ctx, test.backup, TaskName(test.name), test.parameters)
		require.Error(t, err)
	}
	_, err := repo.Claim(ctx, "", time.Minute)
	require.Error(t, err)
	for _, duration := range []time.Duration{-time.Second, 0, time.Nanosecond} {
		_, err := repo.Claim(ctx, "worker", duration)
		require.Error(t, err)
		_, err = repo.Heartbeat(ctx, 1, uuid.New(), duration)
		require.Error(t, err)
	}
	for _, id := range []int64{-1, 0} {
		_, err := repo.Heartbeat(ctx, id, uuid.New(), time.Minute)
		require.Error(t, err)
		require.Error(t, repo.Finish(ctx, id, uuid.New()))
	}
	_, err = repo.Heartbeat(ctx, 1, uuid.Nil, time.Minute)
	require.Error(t, err)
	require.Error(t, repo.Finish(ctx, 1, uuid.Nil))
}

func TestMaintenanceTaskLifecycle(t *testing.T) {
	ctx, storageService := prepareTest(t)
	backup := prepareBackup(t, ctx)
	repo := &defaultBackupMaintenanceTaskRepository{storageService: storageService}

	task, err := repo.Add(ctx, backup.ID, "cleanup", TaskParameters{ExpirationInDays: 14})
	require.NoError(t, err)
	require.Positive(t, task.ID)
	require.Equal(t, TaskParameters{ExpirationInDays: float64(14)}, task.TaskParameters)
	require.False(t, task.CreatedTimestamp.IsZero())
	require.Nil(t, task.LockToken)
	_, err = repo.Add(ctx, backup.ID, "cleanup", TaskParameters{ExpirationInDays: 30})
	require.ErrorIs(t, err, ErrMaintenanceTaskAlreadyPending)

	claimed, err := repo.Claim(ctx, "worker", time.Minute)
	require.NoError(t, err)
	require.Equal(t, task.ID, claimed.ID)
	require.Equal(t, "worker", *claimed.LockedBy)
	require.NotNil(t, claimed.LockToken)
	require.True(t, claimed.LeaseExpiresAt.After(*claimed.LastHeartbeatAt))
	unavailable, err := repo.Claim(ctx, "another-worker", time.Minute)
	require.NoError(t, err)
	require.Nil(t, unavailable)

	_, err = repo.Heartbeat(ctx, task.ID, uuid.New(), time.Minute)
	require.ErrorIs(t, err, ErrMaintenanceTaskLeaseLost)
	require.ErrorIs(t, repo.Finish(ctx, task.ID, uuid.New()), ErrMaintenanceTaskLeaseLost)
	renewed, err := repo.Heartbeat(ctx, task.ID, *claimed.LockToken, 2*time.Minute)
	require.NoError(t, err)
	require.True(t, renewed.LeaseExpiresAt.After(*claimed.LeaseExpiresAt))
	require.True(t, renewed.UpdatedTimestamp.After(task.UpdatedTimestamp))

	_, err = repo.storageService.DB().ExecContext(ctx, `UPDATE backup_maintenance_tasks
SET last_heartbeat_at = now() - interval '2 minutes',
    lease_expires_at = now() - interval '1 minute' WHERE id = ?`, task.ID)
	require.NoError(t, err)
	_, err = repo.Heartbeat(ctx, task.ID, *claimed.LockToken, time.Minute)
	require.ErrorIs(t, err, ErrMaintenanceTaskLeaseLost)
	require.ErrorIs(t, repo.Finish(ctx, task.ID, *claimed.LockToken), ErrMaintenanceTaskLeaseLost)
	reclaimed, err := repo.Claim(ctx, "worker", time.Minute)
	require.NoError(t, err)
	require.Equal(t, task.ID, reclaimed.ID)
	require.NotEqual(t, *claimed.LockToken, *reclaimed.LockToken)
	require.ErrorIs(t, repo.Finish(ctx, task.ID, *claimed.LockToken), ErrMaintenanceTaskLeaseLost)
	require.NoError(t, repo.Finish(ctx, task.ID, *reclaimed.LockToken))
	finished := new(BackupMaintenanceTask)
	_, err = repo.storageService.DB().QueryOneContext(ctx, finished, "SELECT * FROM backup_maintenance_tasks WHERE id = ?", task.ID)
	require.NoError(t, err)
	require.NotNil(t, finished.FinishedTimestamp)
	require.Nil(t, finished.LockToken)
	require.Nil(t, finished.LockedBy)
	require.Nil(t, finished.LeaseExpiresAt)
	require.Nil(t, finished.LastHeartbeatAt)
	_, err = repo.Heartbeat(ctx, task.ID, *reclaimed.LockToken, time.Minute)
	require.ErrorIs(t, err, ErrMaintenanceTaskLeaseLost)
	next, err := repo.Add(ctx, backup.ID, "cleanup", nil)
	require.NoError(t, err)
	require.Greater(t, next.ID, task.ID)
	require.Len(t, next.TaskParameters, 0)
}

func prepareBackupServiceBigQuerySnapshotBackup(backupID, sink string) *Backup {
	{
		return &Backup{
			ID:            backupID,
			Status:        NotStarted,
			SourceProject: "local-ability",
			Strategy:      Snapshot,
			Type:          BigQuery,
			SinkOptions: SinkOptions{
				TargetProject: "local-ability-backup",
				Sink:          sink,
				Region:        "europe-west1",
				StorageClass:  "NEARLINE",
			},
			BackupOptions: BackupOptions{
				BigQueryOptions: BigQueryOptions{Dataset: "demo_delete_me_backup_target", Table: []string{"gcp_billing_budget_amount_plan"}, ExcludedTables: []string{}},
			},
			SnapshotOptions: SnapshotOptions{LifetimeInDays: 7, FrequencyInHours: 24},
		}
	}
}

func prepareBackup(t *testing.T, ctx context.Context) *Backup {
	backupRepository, err := NewBackupRepository(ctx, secret.NewEnvSecretProvider())
	require.NoError(t, err, "BackupRepository should be instantiate")

	backup := prepareBackupServiceBigQuerySnapshotBackup("backupID", "sink")
	_, err = backupRepository.AddBackup(ctx, backup)
	require.NoError(t, err, "should be able to add new backup")
	return backup
}

func TestMaintenanceTaskConcurrentClaims(t *testing.T) {
	ctx, storageService := prepareTest(t)
	backup := prepareBackup(t, ctx)
	repo := &defaultBackupMaintenanceTaskRepository{storageService: storageService}

	task, err := repo.Add(ctx, backup.ID, "cleanup", nil)
	require.NoError(t, err)
	var wg sync.WaitGroup
	claims := make(chan *BackupMaintenanceTask, 8)
	failures := make(chan error, 8)
	for range 8 {
		wg.Go(func() {
			claimed, err := repo.Claim(ctx, "worker", time.Minute)
			if err != nil {
				failures <- err
			}
			if claimed != nil {
				claims <- claimed
			}
		})
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
	require.Len(t, claims, 1)
	require.Equal(t, task.ID, (<-claims).ID)
}
