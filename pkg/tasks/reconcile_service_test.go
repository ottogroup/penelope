package tasks

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/service/bigquery"
	"github.com/stretchr/testify/require"
)

type reconcileBigQueryClient struct {
	bigquery.Client
	update func(project, dataset string, lifetimeInDays uint) error
}

func (c *reconcileBigQueryClient) UpdateDatasetTableExpirations(_ context.Context, project, dataset string, lifetimeInDays uint) error {
	return c.update(project, dataset, lifetimeInDays)
}

func TestReconcileService_SyncTableSnapshotExpiration(t *testing.T) {
	for _, days := range []uint{0, 10, 20} {
		t.Run(fmt.Sprint(days), func(t *testing.T) {
			backup := &repository.Backup{
				ID:          "backup-id",
				SinkOptions: repository.SinkOptions{TargetProject: "target"},
				BackupOptions: repository.BackupOptions{BigQueryOptions: repository.BigQueryOptions{
					Dataset: "source_dataset", UseNativeTableSnapshots: true,
				}},
				SnapshotOptions: repository.SnapshotOptions{LifetimeInDays: days},
			}
			calls := 0
			client := &reconcileBigQueryClient{update: func(project, dataset string, lifetimeInDays uint) error {
				calls++
				require.Equal(t, "target", project)
				require.Equal(t, repository.BuildSnapshotDatasetName("source_dataset", backup.ID), dataset)
				require.Equal(t, days, lifetimeInDays)
				return nil
			}}
			service := &reconcileService{bigQueryClients: map[string]bigquery.Client{"target": client}}
			require.NoError(t, service.syncTableSnapshotExpiration(context.Background(), backup))
			require.NoError(t, service.syncTableSnapshotExpiration(context.Background(), backup))
			require.Equal(t, 2, calls)
		})
	}
}

type reconcileBackupRepository struct {
	repository.BackupRepository
	backups []*repository.Backup
}

func (r *reconcileBackupRepository) GetBackups(_ context.Context, _ repository.BackupFilter) ([]*repository.Backup, error) {
	return r.backups, nil
}

func TestReconcileService_RunNativeSnapshots(t *testing.T) {
	for _, status := range []repository.BackupStatus{repository.Prepared, repository.NotStarted, repository.Paused, repository.Finished, repository.BackupDeleted} {
		t.Run(status.String(), func(t *testing.T) {
			calls := 0
			backup := &repository.Backup{
				ID: "backup", Type: repository.BigQuery, Status: status,
				SinkOptions: repository.SinkOptions{TargetProject: "target"},
				BackupOptions: repository.BackupOptions{BigQueryOptions: repository.BigQueryOptions{
					UseNativeTableSnapshots: true,
				}},
			}
			service := &reconcileService{
				db: &reconcileBackupRepository{backups: []*repository.Backup{backup}},
				bigQueryClients: map[string]bigquery.Client{"target": &reconcileBigQueryClient{
					update: func(_, _ string, lifetimeInDays uint) error {
						calls++
						require.Zero(t, lifetimeInDays)
						return nil
					},
				}},
			}
			service.Run(context.Background())
			if status == repository.BackupDeleted {
				require.Zero(t, calls)
			} else {
				require.Equal(t, 1, calls)
			}
		})
	}
}

func TestReconcileService_SnapshotExpirationFailure(t *testing.T) {
	queryErr := errors.New("query failed")
	service := &reconcileService{bigQueryClients: map[string]bigquery.Client{
		"target": &reconcileBigQueryClient{update: func(_, _ string, _ uint) error { return queryErr }},
	}}
	err := service.syncTableSnapshotExpiration(context.Background(), &repository.Backup{
		ID: "backup", SinkOptions: repository.SinkOptions{TargetProject: "target"},
	})
	require.ErrorIs(t, err, queryErr)
}
