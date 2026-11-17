package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/ottogroup/penelope/pkg/service/bigquery"
	"go.opentelemetry.io/otel"
)

// MaintenanceTaskHandler must honor cancellation and make external operations idempotent.
type MaintenanceTaskHandler func(context.Context, *repository.BackupMaintenanceTask) error

type MaintenanceWorker struct {
	repository        repository.BackupMaintenanceTaskRepository
	workerID          string
	leaseDuration     time.Duration
	heartbeatInterval time.Duration
	handlers          map[repository.TaskName]MaintenanceTaskHandler
}

type MaintenanceService struct {
	worker              *MaintenanceWorker
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
	credentialsProvider secret.SecretProvider
}

func (s *MaintenanceService) Run(ctx context.Context) error {
	return s.worker.Run(ctx)
}

func newMaintenanceService(ctx context.Context, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) (MaintenanceService, error) {
	repo, err := repository.NewBackupMaintenanceTaskRepository(ctx, credentialsProvider)
	if err != nil {
		return MaintenanceService{}, fmt.Errorf("could not instantiate new BackupMaintenanceTaskRepository: %s", err)
	}
	workerID := uuid.NewString()
	handlers := map[repository.TaskName]MaintenanceTaskHandler{
		repository.UpdateTableSnapshotExpirationDate: (&updateTableSnapshotExpirationDate{
			tokenSourceProvider: tokenSourceProvider,
			secretProvider:      credentialsProvider,
		}).do,
	}
	worker, err := newMaintenanceWorker(repo, workerID, time.Hour, time.Minute, handlers)
	return MaintenanceService{worker: worker, tokenSourceProvider: tokenSourceProvider, credentialsProvider: credentialsProvider}, err
}

func newMaintenanceWorker(repo repository.BackupMaintenanceTaskRepository, workerID string, leaseDuration, heartbeatInterval time.Duration, handlers map[repository.TaskName]MaintenanceTaskHandler) (*MaintenanceWorker, error) {
	if repo == nil || strings.TrimSpace(workerID) == "" {
		return nil, fmt.Errorf("maintenance repository and worker ID are required")
	}
	if heartbeatInterval < time.Microsecond || leaseDuration < time.Microsecond ||
		heartbeatInterval >= leaseDuration/2 {
		return nil, fmt.Errorf("heartbeat interval must be at least one microsecond and less than half the lease duration")
	}
	if len(handlers) == 0 {
		return nil, fmt.Errorf("at least one maintenance task handler is required")
	}
	handlerCopy := make(map[repository.TaskName]MaintenanceTaskHandler, len(handlers))
	for name, handler := range handlers {
		if strings.TrimSpace(string(name)) == "" || handler == nil {
			return nil, fmt.Errorf("maintenance task names and handlers must not be empty")
		}
		handlerCopy[name] = handler
	}
	return &MaintenanceWorker{
		repository: repo, workerID: workerID, leaseDuration: leaseDuration,
		heartbeatInterval: heartbeatInterval, handlers: handlerCopy,
	}, nil
}

// Run drains available work sequentially and stops on the first failure or empty queue.
func (w *MaintenanceWorker) Run(ctx context.Context) error {
	for {
		claimed, err := w.RunOnce(ctx)
		if err != nil || !claimed {
			return err
		}
	}
}

// RunOnce returns false when no work is available. Failed tasks retain their lease until expiry.
func (w *MaintenanceWorker) RunOnce(ctxIn context.Context) (bool, error) {
	ctx, span := otel.Tracer("").Start(ctxIn, "(*MaintenanceWorker).RunOnce")
	defer span.End()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	claimCtx, cancelClaim := context.WithTimeout(ctx, w.heartbeatInterval)
	task, err := w.repository.Claim(claimCtx, w.workerID, w.leaseDuration)
	cancelClaim()
	if err != nil {
		return false, fmt.Errorf("could not claim maintenance task: %w", err)
	}
	if task == nil {
		return false, nil
	}
	if task.LockToken == nil || *task.LockToken == uuid.Nil {
		return true, fmt.Errorf("claimed maintenance task %d has no lock token", task.ID)
	}
	handler, exists := w.handlers[task.TaskName]
	if !exists {
		return true, fmt.Errorf("no handler registered for maintenance task %q (id=%d)", task.TaskName, task.ID)
	}

	// Copy the fencing identity before passing the task to caller-provided code.
	id, token := task.ID, *task.LockToken
	workCtx, cancelWork := context.WithCancelCause(ctx)
	defer cancelWork(nil)
	stop := make(chan struct{})
	heartbeatDone := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(w.heartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				heartbeatDone <- nil
				return
			case <-workCtx.Done():
				heartbeatDone <- context.Cause(workCtx)
				return
			case <-ticker.C:
				heartbeatCtx, cancel := context.WithTimeout(workCtx, w.heartbeatInterval)
				_, err := w.repository.Heartbeat(heartbeatCtx, id, token, w.leaseDuration)
				cancel()
				if err != nil {
					err = fmt.Errorf("could not renew maintenance task %d: %w", id, err)
					cancelWork(err)
					heartbeatDone <- err
					return
				}
			}
		}
	}()

	var heartbeatErr error
	handlerErr := func() error {
		defer func() {
			close(stop)
			heartbeatErr = <-heartbeatDone
		}()
		return handler(workCtx, task)
	}()
	if err := errors.Join(handlerErr, heartbeatErr, ctx.Err()); err != nil {
		return true, fmt.Errorf("maintenance task %d failed: %w", id, err)
	}
	finishCtx, cancelFinish := context.WithTimeout(ctx, w.heartbeatInterval)
	defer cancelFinish()
	if err := w.repository.Finish(finishCtx, id, token); err != nil {
		return true, fmt.Errorf("could not finish maintenance task %d: %w", id, err)
	}
	return true, nil
}

type updateTableSnapshotExpirationDate struct {
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
	secretProvider      secret.SecretProvider
}

func (w *updateTableSnapshotExpirationDate) do(ctx context.Context, task *repository.BackupMaintenanceTask) error {
	r, err := repository.NewBackupRepository(ctx, w.secretProvider)
	if err != nil {
		return fmt.Errorf("could not create backup repository: %w", err)
	}
	b, err := r.GetBackup(ctx, task.BackupID)
	if err != nil {
		return fmt.Errorf("could not get backup %s: %w", task.BackupID, err)
	}
	if b.Type != repository.BigQuery || !b.UseNativeTableSnapshots {
		return fmt.Errorf("backup %s does not use native BigQuery table snapshots", task.BackupID)
	}
	destDataset := repository.BuildSnapshotDatasetName(b.BigQueryOptions.Dataset, b.ID)
	client, err := bigquery.NewBigQueryClient(ctx, w.tokenSourceProvider, b.SourceProject, b.TargetProject)
	if err != nil {
		return fmt.Errorf("could not create BigQuery client for backup %s: %w", b.ID, err)
	}
	if err := client.UpdateDatasetTableExpirations(ctx, b.TargetProject, destDataset, b.SnapshotOptions.LifetimeInDays); err != nil {
		return fmt.Errorf("could not update snapshot expiration for backup %s: %w", b.ID, err)
	}
	return nil
}
