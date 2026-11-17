package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/google/uuid"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/ottogroup/penelope/pkg/service"
	"go.opentelemetry.io/otel"
)

var (
	ErrMaintenanceTaskAlreadyPending = errors.New("unfinished maintenance task already exists")
	ErrMaintenanceTaskLeaseLost      = errors.New("maintenance task lease expired or ownership lost")
)

type TaskParameter string

const (
	ExpirationInDays TaskParameter = "expiration_in_days"
)

type TaskName string

const (
	UpdateTableSnapshotExpirationDate TaskName = "update_table_snapshot_expiration_date"
)

type TaskParameters map[TaskParameter]any

type BackupMaintenanceTask struct {
	ID                int64
	BackupID          string
	TaskName          TaskName
	TaskParameters    TaskParameters `pg:",type:jsonb"`
	CreatedTimestamp  time.Time      `pg:"audit_created_timestamp"`
	UpdatedTimestamp  time.Time      `pg:"audit_updated_timestamp"`
	FinishedTimestamp *time.Time     `pg:"audit_finished_timestamp"`
	LockedBy          *string
	LockToken         *uuid.UUID
	LeaseExpiresAt    *time.Time
	LastHeartbeatAt   *time.Time
}

type BackupMaintenanceTaskRepository interface {
	Add(ctx context.Context, backupID string, taskName TaskName, parameters TaskParameters) (*BackupMaintenanceTask, error)
	// Claim returns nil, nil when no unfinished task has an available lease.
	Claim(ctx context.Context, worker string, leaseDuration time.Duration) (*BackupMaintenanceTask, error)
	Heartbeat(ctx context.Context, id int64, token uuid.UUID, leaseDuration time.Duration) (*BackupMaintenanceTask, error)
	Finish(ctx context.Context, id int64, token uuid.UUID) error
}

type defaultBackupMaintenanceTaskRepository struct {
	storageService *service.Service
}

func NewBackupMaintenanceTaskRepository(ctxIn context.Context, credentialsProvider secret.SecretProvider) (BackupMaintenanceTaskRepository, error) {
	ctx, span := otel.Tracer("").Start(ctxIn, "NewBackupMaintenanceTaskRepository")
	defer span.End()

	storageService, err := service.NewStorageService(ctx, credentialsProvider)
	if err != nil {
		return nil, fmt.Errorf("could not create maintenance task repository: %w", err)
	}
	return &defaultBackupMaintenanceTaskRepository{storageService: storageService}, nil
}

func (d *defaultBackupMaintenanceTaskRepository) Add(ctxIn context.Context, backupID string, taskName TaskName, parameters TaskParameters) (*BackupMaintenanceTask, error) {
	ctx, span := otel.Tracer("").Start(ctxIn, "(*defaultBackupMaintenanceTaskRepository).Add")
	defer span.End()

	if strings.TrimSpace(backupID) == "" || strings.TrimSpace(string(taskName)) == "" {
		return nil, fmt.Errorf("backup ID and maintenance task name must not be empty")
	}
	if len(parameters) == 0 {
		parameters = TaskParameters{}
	}
	rawParameters, err := json.Marshal(parameters)
	if err != nil {
		return nil, fmt.Errorf("could not marshal maintenance task parameters: %w", err)
	}
	if !json.Valid(rawParameters) {
		return nil, fmt.Errorf("maintenance task parameters must be valid JSON")
	}
	task := new(BackupMaintenanceTask)
	_, err = d.storageService.DB().QueryOneContext(ctx, task, `
INSERT INTO backup_maintenance_tasks (backup_id, task_name, task_parameters)
VALUES (?, ?, ?::jsonb)
ON CONFLICT (backup_id, task_name) WHERE audit_finished_timestamp IS NULL DO NOTHING
RETURNING *`, backupID, taskName, string(rawParameters))
	if errors.Is(err, pg.ErrNoRows) {
		return nil, fmt.Errorf("%w: backup=%s task=%s", ErrMaintenanceTaskAlreadyPending, backupID, taskName)
	}
	if err != nil {
		return nil, fmt.Errorf("could not add maintenance task for backup %s: %w", backupID, err)
	}
	return task, nil
}

func (d *defaultBackupMaintenanceTaskRepository) Claim(ctxIn context.Context, worker string, leaseDuration time.Duration) (*BackupMaintenanceTask, error) {
	ctx, span := otel.Tracer("").Start(ctxIn, "(*defaultBackupMaintenanceTaskRepository).Claim")
	defer span.End()

	if strings.TrimSpace(worker) == "" {
		return nil, fmt.Errorf("maintenance task worker must not be empty")
	}
	if leaseDuration < time.Microsecond {
		return nil, fmt.Errorf("maintenance task lease duration must be at least one microsecond")
	}
	task := new(BackupMaintenanceTask)
	// The row lock and lease assignment occur in one statement, before external work starts.
	_, err := d.storageService.DB().QueryOneContext(ctx, task, `
WITH candidate AS (
    SELECT id FROM backup_maintenance_tasks
    WHERE audit_finished_timestamp IS NULL
      AND (lease_expires_at IS NULL OR lease_expires_at <= clock_timestamp())
    ORDER BY lease_expires_at NULLS FIRST, id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE backup_maintenance_tasks AS task
SET locked_by = ?, lock_token = ?,
    last_heartbeat_at = statement_timestamp(),
    lease_expires_at = statement_timestamp() + (? * interval '1 microsecond')
FROM candidate
WHERE task.id = candidate.id
RETURNING task.*`, worker, uuid.New(), leaseDuration.Microseconds())
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not claim maintenance task: %w", err)
	}
	return task, nil
}

func validateMaintenanceTaskClaim(id int64, token uuid.UUID) error {
	if id <= 0 || token == uuid.Nil {
		return fmt.Errorf("maintenance task ID must be positive and claim token must not be empty")
	}
	return nil
}

func (d *defaultBackupMaintenanceTaskRepository) Heartbeat(ctxIn context.Context, id int64, token uuid.UUID, leaseDuration time.Duration) (*BackupMaintenanceTask, error) {
	ctx, span := otel.Tracer("").Start(ctxIn, "(*defaultBackupMaintenanceTaskRepository).Heartbeat")
	defer span.End()

	if err := validateMaintenanceTaskClaim(id, token); err != nil {
		return nil, err
	}
	if leaseDuration < time.Microsecond {
		return nil, fmt.Errorf("maintenance task lease duration must be at least one microsecond")
	}
	task := new(BackupMaintenanceTask)
	_, err := d.storageService.DB().QueryOneContext(ctx, task, `
UPDATE backup_maintenance_tasks
SET last_heartbeat_at = statement_timestamp(),
    lease_expires_at = statement_timestamp() + (? * interval '1 microsecond')
WHERE id = ? AND lock_token = ?
  AND audit_finished_timestamp IS NULL AND lease_expires_at > clock_timestamp()
RETURNING *`, leaseDuration.Microseconds(), id, token)
	if errors.Is(err, pg.ErrNoRows) {
		return nil, fmt.Errorf("%w: task=%d", ErrMaintenanceTaskLeaseLost, id)
	}
	if err != nil {
		return nil, fmt.Errorf("could not renew maintenance task %d lease: %w", id, err)
	}
	return task, nil
}

func (d *defaultBackupMaintenanceTaskRepository) Finish(ctxIn context.Context, id int64, token uuid.UUID) error {
	ctx, span := otel.Tracer("").Start(ctxIn, "(*defaultBackupMaintenanceTaskRepository).Finish")
	defer span.End()

	if err := validateMaintenanceTaskClaim(id, token); err != nil {
		return err
	}
	result, err := d.storageService.DB().ExecContext(ctx, `
UPDATE backup_maintenance_tasks
SET audit_finished_timestamp = clock_timestamp(),
    locked_by = NULL, lock_token = NULL, lease_expires_at = NULL, last_heartbeat_at = NULL
WHERE id = ? AND lock_token = ?
  AND audit_finished_timestamp IS NULL AND lease_expires_at > clock_timestamp()`, id, token)
	if err != nil {
		return fmt.Errorf("could not finish maintenance task %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("%w: task=%d", ErrMaintenanceTaskLeaseLost, id)
	}
	return nil
}
