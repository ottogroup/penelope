package memory

import (
    "context"
    "fmt"
    "github.com/ottogroup/penelope/pkg/repository"
    "go.opencensus.io/trace"
    "time"
)

// BackupRepository access to stored state of backups
type BackupRepository struct {
    backups []*repository.Backup
}

// UpdateBackupStatus is not implemented
func (r *BackupRepository) UpdateBackup(ctxIn context.Context, updateFields repository.UpdateFields) error {
    _, span := trace.StartSpan(ctxIn, "(*BackupRepository).UpdateBackupStatus")
    defer span.End()

    for _, backup := range r.backups {
        if backup.ID != updateFields.BackupID {
            continue
        }
        if updateFields.Status != "" && updateFields.Status != backup.Status {
            backup.Status = updateFields.Status
        }
        if repository.BigQuery == backup.Type {
            backup.Table = updateFields.Table
        }
        if repository.CloudStorage == backup.Type {
            backup.IncludePath = updateFields.IncludePath
            backup.ExcludePath = updateFields.ExcludePath
        }
        return nil
    }
    return fmt.Errorf("backup %s not found", updateFields.BackupID)
}

// GetBigQueryOneShotSnapshots return backups that are BigQuery with strategy Snapshot
func (r *BackupRepository) GetBigQueryOneShotSnapshots(ctxIn context.Context, status repository.BackupStatus) (backups []*repository.Backup, err error) {
    _, span := trace.StartSpan(ctxIn, "(*BackupRepository).GetBigQueryOneShotSnapshots")
    defer span.End()

    for _, backup := range r.backups {
        if backup.Strategy == repository.Snapshot && backup.SnapshotOptions.FrequencyInHours == 0 {
            backups = append(backups, backup)
        }
    }
    return backups, err
}

// UpdateLastCleanupTime is not implemented
func (r *BackupRepository) UpdateLastCleanupTime(ctxIn context.Context, backupID string, lastCleanupTime time.Time) error {
    _, span := trace.StartSpan(ctxIn, "(*BackupRepository).UpdateLastCleanupTime")
    defer span.End()

    panic("implement me")
}

// GetExpiredBigQueryMirrorRevisions is not implemented
func (r *BackupRepository) GetExpiredBigQueryMirrorRevisions(ctxIn context.Context, maxRevisionLifetimeInWeeks int) ([]*repository.MirrorRevision, error) {
    _, span := trace.StartSpan(ctxIn, "(*BackupRepository).GetExpiredBigQueryMirrorRevisions")
    defer span.End()

    panic("implement me")
}

// AddBackup create new backup
func (r *BackupRepository) AddBackup(ctxIn context.Context, backup *repository.Backup) (*repository.Backup, error) {
    _, span := trace.StartSpan(ctxIn, "(*BackupRepository).AddBackup")
    defer span.End()

    r.backups = append(r.backups, backup)
    backup.CreatedTimestamp = time.Now().UTC()
    return backup, nil
}

// GetBackup get backup details
func (r *BackupRepository) GetBackup(ctxIn context.Context, backupID string) (*repository.Backup, error) {
    _, span := trace.StartSpan(ctxIn, "(*BackupRepository).GetBackup")
    defer span.End()

    for _, b := range r.backups {
        if b.ID == backupID {
            return b, nil
        }
    }
    return nil, fmt.Errorf("backup %s not found", backupID)
}

// GetBackups list backups with filtering
func (r *BackupRepository) GetBackups(ctxIn context.Context, backupFilter repository.BackupFilter) (backups []*repository.Backup, err error) {
    _, span := trace.StartSpan(ctxIn, "(*BackupRepository).GetBackups")
    defer span.End()

    backups = append(backups, r.backups...)
    return backups, err
}

// MarkDeleted mark backup as deleted
func (r *BackupRepository) MarkDeleted(ctxIn context.Context, backupID string) error {
    ctx, span := trace.StartSpan(ctxIn, "(*BackupRepository).MarkDeleted")
    defer span.End()

    b, err := r.GetBackup(ctx, backupID)
    if err != nil {
        return err
    }
    b.DeletedTimestamp = time.Now().UTC()
    return nil
}

// MarkStatus mark backup as status
func (r *BackupRepository) MarkStatus(ctxIn context.Context, backupID string, status repository.BackupStatus) error {
    ctx, span := trace.StartSpan(ctxIn, "(*BackupRepository).MarkDeleted")
    defer span.End()

    b, err := r.GetBackup(ctx, backupID)
    if err != nil {
        return err
    }
    if b.Status == repository.BackupDeleted {
        b.DeletedTimestamp = time.Now().UTC()
    }
    b.Status = status
    return nil
}

// UpdateBackupStatus change backup status
func (r *BackupRepository) UpdateBackupStatus(ctxIn context.Context, backupID string, status repository.BackupStatus) error {
    ctx, span := trace.StartSpan(ctxIn, "(*BackupRepository).UpdateBackupStatus")
    defer span.End()

    b, err := r.GetBackup(ctx, backupID)
    if err != nil {
        return err
    }
    b.Status = status
    return nil
}

// UpdateLastScheduledTime set last time when backup was scheduled
func (r *BackupRepository) UpdateLastScheduledTime(ctxIn context.Context, backupID string, lastScheduledTime time.Time, status repository.BackupStatus) error {
    ctx, span := trace.StartSpan(ctxIn, "(*BackupRepository).UpdateLastScheduledTime")
    defer span.End()

    b, err := r.GetBackup(ctx, backupID)
    if err != nil {
        return err
    }
    b.LastScheduledTime = time.Now().UTC()
    b.Status = status
    return nil
}

// GetByBackupStatus return backups by status
func (r *BackupRepository) GetByBackupStatus(ctxIn context.Context, status repository.BackupStatus) (backups []*repository.Backup, err error) {
    _, span := trace.StartSpan(ctxIn, "(*BackupRepository).GetByBackupStatus")
    defer span.End()

    for _, b := range r.backups {
        if b.Status == status {
            backups = append(backups, b)
        }
    }
    return backups, err
}

// GetByBackupStrategy return backups by strategy
func (r *BackupRepository) GetByBackupStrategy(ctxIn context.Context, strategy repository.Strategy) (backups []*repository.Backup, err error) {
    _, span := trace.StartSpan(ctxIn, "(*BackupRepository).GetByBackupStrategy")
    defer span.End()

    for _, b := range r.backups {
        if b.Strategy == strategy {
            backups = append(backups, b)
        }
    }
    return backups, err
}

// GetExpired returns backups that are expired
func (r *BackupRepository) GetExpired(context.Context, repository.BackupType) (backups []*repository.Backup, err error) {
    now := time.Now().UTC()
    for _, b := range r.backups {
        if b.SnapshotOptions.LifetimeInDays < 1 {
            continue
        }
        expireAt := b.CreatedTimestamp.UTC().Add(time.Duration(b.SnapshotOptions.LifetimeInDays) * 24 * time.Hour)
        if now.Unix() > expireAt.Unix() {
            backups = append(backups, b)
        }
    }
    return backups, err
}

// GetScheduledBackups list backups that can have a new job prepared
func (r *BackupRepository) GetScheduledBackups(ctxIn context.Context, backupType repository.BackupType) (backups []*repository.Backup, err error) {
    _, span := trace.StartSpan(ctxIn, "(*BackupRepository).GetScheduledBackups")
    defer span.End()

    for _, b := range r.backups {
        if b.Type == backupType {
            backups = append(backups, b)
        }
    }
    return backups, err
}
