package tasks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/stretchr/testify/require"
)

type maintenanceWorkerRepository struct {
	repository.BackupMaintenanceTaskRepository
	claim     func(context.Context) (*repository.BackupMaintenanceTask, error)
	heartbeat func(context.Context, int64, uuid.UUID) (*repository.BackupMaintenanceTask, error)
	finish    func(context.Context, int64, uuid.UUID) error
}

func (r *maintenanceWorkerRepository) Claim(ctx context.Context, _ string, _ time.Duration) (*repository.BackupMaintenanceTask, error) {
	return r.claim(ctx)
}

func (r *maintenanceWorkerRepository) Heartbeat(ctx context.Context, id int64, token uuid.UUID, _ time.Duration) (*repository.BackupMaintenanceTask, error) {
	return r.heartbeat(ctx, id, token)
}

func (r *maintenanceWorkerRepository) Finish(ctx context.Context, id int64, token uuid.UUID) error {
	return r.finish(ctx, id, token)
}

func TestMaintenanceWorkerLifecycle(t *testing.T) {
	token := uuid.New()
	task := &repository.BackupMaintenanceTask{ID: 1, TaskName: "cleanup", LockToken: &token}
	renewed := make(chan struct{}, 1)
	finished := false
	claims := 0
	repo := &maintenanceWorkerRepository{
		claim: func(context.Context) (*repository.BackupMaintenanceTask, error) {
			claims++
			if claims == 1 {
				return task, nil
			}
			return nil, nil
		},
		heartbeat: func(_ context.Context, id int64, claimToken uuid.UUID) (*repository.BackupMaintenanceTask, error) {
			if id != 1 || claimToken != token {
				return nil, errors.New("wrong heartbeat claim")
			}
			select {
			case renewed <- struct{}{}:
			default:
			}
			return task, nil
		},
		finish: func(_ context.Context, id int64, claimToken uuid.UUID) error {
			require.Equal(t, int64(1), id)
			require.Equal(t, token, claimToken)
			finished = true
			return nil
		},
	}
	worker, err := newMaintenanceWorker(repo, "worker", time.Second, 10*time.Millisecond,
		map[repository.TaskName]MaintenanceTaskHandler{"cleanup": func(ctx context.Context, task *repository.BackupMaintenanceTask) error {
			select {
			case <-renewed:
				// Handler mutation must not affect the worker's fencing identity.
				task.ID = 99
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, worker.Run(ctx))
	require.True(t, finished)
	require.Equal(t, 2, claims)
}

func TestMaintenanceWorkerFailures(t *testing.T) {
	for _, mode := range []string{"claim", "handler", "heartbeat", "heartbeat timeout", "finish", "unknown", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			token := uuid.New()
			task := &repository.BackupMaintenanceTask{ID: 1, TaskName: "cleanup", LockToken: &token}
			failure := errors.New(mode)
			finished := false
			repo := &maintenanceWorkerRepository{
				claim: func(context.Context) (*repository.BackupMaintenanceTask, error) {
					if mode == "claim" {
						return nil, failure
					}
					return task, nil
				},
				heartbeat: func(ctx context.Context, _ int64, _ uuid.UUID) (*repository.BackupMaintenanceTask, error) {
					if mode == "heartbeat timeout" {
						<-ctx.Done()
						return nil, ctx.Err()
					}
					return nil, failure
				},
				finish: func(context.Context, int64, uuid.UUID) error {
					finished = true
					return failure
				},
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			handler := func(workCtx context.Context, _ *repository.BackupMaintenanceTask) error {
				switch mode {
				case "heartbeat", "heartbeat timeout":
					<-workCtx.Done()
					return context.Cause(workCtx)
				case "handler":
					return failure
				case "canceled":
					cancel()
					return nil
				default:
					return nil
				}
			}
			if mode == "unknown" {
				task.TaskName = "unknown"
			}
			worker, err := newMaintenanceWorker(repo, "worker", time.Second, 10*time.Millisecond,
				map[repository.TaskName]MaintenanceTaskHandler{"cleanup": handler})
			require.NoError(t, err)
			_, err = worker.RunOnce(ctx)
			require.Error(t, err)
			if mode == "claim" || mode == "handler" || mode == "heartbeat" || mode == "finish" {
				require.ErrorIs(t, err, failure)
			}
			if mode == "heartbeat timeout" {
				require.ErrorIs(t, err, context.DeadlineExceeded)
			}
			require.Equal(t, mode == "finish", finished)
		})
	}
}

func TestMaintenanceWorkerValidation(t *testing.T) {
	repo := &maintenanceWorkerRepository{}
	handlers := map[repository.TaskName]MaintenanceTaskHandler{"cleanup": func(context.Context, *repository.BackupMaintenanceTask) error { return nil }}
	_, err := newMaintenanceWorker(nil, "worker", time.Second, time.Millisecond, handlers)
	require.Error(t, err)
	_, err = newMaintenanceWorker(repo, " ", time.Second, time.Millisecond, handlers)
	require.Error(t, err)
	for _, interval := range []time.Duration{0, time.Nanosecond, time.Second / 2, time.Second} {
		_, err := newMaintenanceWorker(repo, "worker", time.Second, interval, handlers)
		require.Error(t, err)
	}
	_, err = newMaintenanceWorker(repo, "worker", time.Second, time.Millisecond, nil)
	require.Error(t, err)
	_, err = newMaintenanceWorker(repo, "worker", time.Second, time.Millisecond, map[repository.TaskName]MaintenanceTaskHandler{"cleanup": nil})
	require.Error(t, err)
}
