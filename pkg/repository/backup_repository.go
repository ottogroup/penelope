package repository

import (
    "context"
    "fmt"
    "github.com/go-pg/pg/v10/orm"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/service"
    "go.opencensus.io/trace"
    "time"
)

//TODO include defer .close() for .conn() in each method
// BackupFilter possible filtering for backups in Query
type BackupFilter struct {
    Project string
}

// UpdateFields what is changing
type UpdateFields struct {
    BackupID       string
    Status         BackupStatus
    IncludePath    []string
    ExcludePath    []string
    Table          []string
    ExcludedTables []string
    MirrorTTL      uint
    SnapshotTTL    uint
    ArchiveTTM     uint
}

// BackupRepository defines operations for a Backup
type BackupRepository interface {
    AddBackup(context.Context, *Backup) (*Backup, error)
    GetBackup(ctxIn context.Context, backupID string) (*Backup, error)
    GetBackups(context.Context, BackupFilter) ([]*Backup, error)
    MarkStatus(ctxIn context.Context, backupId string, status BackupStatus) error
    MarkDeleted(context.Context, string) error
    UpdateBackup(ctxIn context.Context, updateFields UpdateFields) error
    UpdateLastScheduledTime(ctxIn context.Context, backupID string, lastScheduledTime time.Time, status BackupStatus) error
    UpdateLastCleanupTime(ctxIn context.Context, backupID string, lastCleanupTime time.Time) error
    GetByBackupStatus(ctxIn context.Context, status BackupStatus) ([]*Backup, error)
    GetByBackupStrategy(ctxIn context.Context, strategy Strategy) ([]*Backup, error)
    GetExpired(context.Context, BackupType) ([]*Backup, error)
    GetExpiredBigQueryMirrorRevisions(ctxIn context.Context, maxRevisionLifetimeInWeeks int) ([]*MirrorRevision, error)
    GetBigQueryOneShotSnapshots(ctxIn context.Context, status BackupStatus) ([]*Backup, error)
    GetScheduledBackups(context.Context, BackupType) ([]*Backup, error)
}

// defaultBackupRepository implements BackupRepository
type defaultBackupRepository struct {
    storageService *service.Service
    ctx            context.Context
}

// NewBackupRepository return instance of BackupRepository
func NewBackupRepository(ctxIn context.Context, credentialsProvider secret.SecretProvider) (BackupRepository, error) {
    ctx, span := trace.StartSpan(ctxIn, "NewBackupRepository")
    defer span.End()
    storageService, err := service.NewStorageService(ctx, credentialsProvider)
    if err != nil {
        return nil, err
    }

    return &defaultBackupRepository{storageService: storageService, ctx: ctx}, nil
}

// AddBackup create new backup
func (d *defaultBackupRepository) AddBackup(ctxIn context.Context, backup *Backup) (*Backup, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).AddBackup")
    defer span.End()

    _, err := d.storageService.DB().Model(backup).Insert()
    return backup, err
}

// GetBackups list backups with filtering
func (d *defaultBackupRepository) GetBackups(ctxIn context.Context, backupFilter BackupFilter) ([]*Backup, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).GetBackups")
    defer span.End()

    var backups []*Backup
    query := d.storageService.DB().Model(&backups)
    if 0 < len(backupFilter.Project) {
        query = query.Where("project = ?", backupFilter.Project)
    }
    err := query.Select()
    if err != nil {
        logQueryError("GetBackups", err)
        return nil, err
    }
    return backups, nil
}

// GetByBackupStrategy return backups by strategy
func (d *defaultBackupRepository) GetByBackupStrategy(ctxIn context.Context, strategy Strategy) ([]*Backup, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).GetByBackupStrategy")
    defer span.End()

    var backups []*Backup

    err := d.storageService.DB().Model(&backups).
        Where("strategy = ?", strategy).
        Where("audit_deleted_timestamp IS NULL").
        Select()

    if err != nil {
        logQueryError("GetByBackupStrategy", err)
        return nil, err
    }

    return backups, nil
}

// GetBackup get backup details
func (d *defaultBackupRepository) GetBackup(ctxIn context.Context, backupID string) (*Backup, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).GetBackup")
    defer span.End()

    backup := new(Backup)
    err := d.storageService.DB().Model(backup).Where("id = ?", backupID).Select()
    if err != nil {
        logQueryError("GetBackup", err)
        return nil, err
    }
    return backup, nil
}

// MarkDeleted mark backup as deleted
func (d *defaultBackupRepository) MarkDeleted(ctxIn context.Context, backupID string) error {
    ctx, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).MarkDeleted")
    defer span.End()

    err := d.MarkStatus(ctx, backupID, BackupDeleted)

    if err != nil {
        logQueryError("MarkDeleted", err)
        return fmt.Errorf("error during executing updating backup statemant: %s", err)
    }
    return nil
}

// MarkStatus marks backup as specified status
func (d *defaultBackupRepository) MarkStatus(ctxIn context.Context, id string, status BackupStatus) error {
    _, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).MarkStatus")
    defer span.End()

    backup := &Backup{
        ID:id,
        Status:status,
        EntityAudit: EntityAudit{
            UpdatedTimestamp: time.Now(),
        },
    }

    if status == BackupDeleted {
        backup.DeletedTimestamp = time.Now()
    }

    query := d.storageService.DB().Model(backup).
        Column("status", "audit_updated_timestamp", "audit_deleted_timestamp")

    _, err := query.
        WherePK().
        Update()

    if err != nil {
        logQueryError("MarkStatus", err)
        return fmt.Errorf("error during executing updating backup statemant: %s", err)
    }
    return nil
}

// UpdateBackupStatus change backup status
func (d *defaultBackupRepository) UpdateBackup(ctxIn context.Context, fields UpdateFields) error {
    _, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).UpdateBackupStatus")
    defer span.End()

    backup := &Backup{
        ID:                fields.BackupID,
        Status:            fields.Status,
        SnapshotOptions:   SnapshotOptions{
            LifetimeInDays: fields.SnapshotTTL,
        },
        BackupOptions:     BackupOptions{
            BigQueryOptions:     BigQueryOptions{
                Table:          fields.Table,
                ExcludedTables: fields.ExcludedTables,
            },
            CloudStorageOptions: CloudStorageOptions{
                IncludePath: fields.IncludePath,
                ExcludePath: fields.ExcludePath,
            },
        },
        EntityAudit:       EntityAudit{
            UpdatedTimestamp: time.Now(),
        },
        MirrorOptions:     MirrorOptions{
            LifetimeInDays: fields.MirrorTTL,
        },
        SinkOptions: SinkOptions{
            ArchiveTTM:    fields.ArchiveTTM,
        },
    }

    if fields.Status.EqualTo(BackupDeleted.String()) {
        backup.DeletedTimestamp = time.Now()
    }

    result, err := d.storageService.DB().
        Model(backup).
        Column(
            "status",
            "snapshot_lifetime_in_days",
            "bigquery_table",
            "bigquery_excluded_tables",
            "cloudstorage_include_path",
            "cloudstorage_exclude_path",
            "audit_updated_timestamp",
            "audit_deleted_timestamp",
            "mirror_lifetime_in_days",
            "archive_ttm",
        ).
        WherePK().
        Update()

    if err != nil {
        return fmt.Errorf("error during executing updating backup statemant: %s", err)
    }

    if 1 < result.RowsAffected() {
        return fmt.Errorf("error during validation of updating backup statemant:  expected one row to be updated but was %v", result.RowsAffected())
    }

    return nil
}

// UpdateLastScheduledTime set last time when backup was scheduled
func (d *defaultBackupRepository) UpdateLastScheduledTime(ctxIn context.Context, backupID string, lastScheduledTime time.Time, status BackupStatus) error {
    _, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).UpdateLastScheduledTime")
    defer span.End()

    backup := &Backup{
        ID:backupID,
        Status:status,
        LastScheduledTime:lastScheduledTime,
        EntityAudit: EntityAudit{
            UpdatedTimestamp: time.Now(),
        },
    }

    res, err := d.storageService.DB().Model(backup).
        Column("last_scheduled_timestamp", "status", "audit_updated_timestamp").
        WherePK().
        Where("audit_deleted_timestamp IS NULL").
        Update()

    if err != nil {
        return fmt.Errorf("error during executing updating backup statemant: %s", err)
    }

    rowsAffected := res.RowsAffected()
    if 1 < rowsAffected {
        return fmt.Errorf("error during validation of updating backup statemant:  expected one row to be updated but was %v", rowsAffected)
    }

    return nil
}

// UpdateLastCleanupTime update cleanup time operation
func (d *defaultBackupRepository) UpdateLastCleanupTime(ctxIn context.Context, backupID string, lastCleanupTime time.Time) error {
    _, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).UpdateLastCleanupTime")
    defer span.End()

    backup := &Backup{
        ID:backupID,
        LastCleanupTime:lastCleanupTime,
        EntityAudit: EntityAudit{
            UpdatedTimestamp: time.Now(),
        },
    }

    res, err := d.storageService.DB().Model(backup).
        Column("last_cleanup_timestamp", "audit_updated_timestamp").
        WherePK().
        Where("audit_deleted_timestamp IS NULL").
        Update()

    if err != nil {
        return fmt.Errorf("error during executing updating backup statemant: %s", err)
    }

    rowsAffected := res.RowsAffected()
    if 1 < rowsAffected {
        return fmt.Errorf("error during validation of updating backup statemant:  expected one row to be updated but was %v", rowsAffected)
    }

    return nil
}

// GetByBackupStatus return backups by status
func (d *defaultBackupRepository) GetByBackupStatus(ctxIn context.Context, status BackupStatus) ([]*Backup, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).GetByBackupStatus")
    defer span.End()

    var backups []*Backup

    err := d.storageService.DB().Model(&backups).
        Where("status = ?", status).
        Where("audit_deleted_timestamp IS NULL").
        Select()

    if err != nil {
        logQueryError("GetByBackupStatus", err)
        return backups, fmt.Errorf("error during executing get backup by status statement: %s", err)
    }

    return backups, nil
}

// GetExpired returns backups that are expired
func (d *defaultBackupRepository) GetExpired(ctxIn context.Context, backupType BackupType) ([]*Backup, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).GetExpired")
    defer span.End()

    var backups []*Backup
    err := d.storageService.DB().Model(&backups).
        Where("type = ?", backupType).
        Where("status = ?", ToDelete).
        WhereOrGroup(func(sub *orm.Query) (*orm.Query, error) {
            return sub.Where("status = ?", Finished).
                Where("audit_deleted_timestamp IS NULL").
                Where("snapshot_frequency_in_hours = 0").
                Where("snapshot_lifetime_in_days > 0").
                Where("audit_created_timestamp + INTERVAL ' 1 DAY ' * snapshot_lifetime_in_days < NOW()"), nil
    }).Select()

    if err != nil {
        logQueryError("GetExpired", err)
        return backups, fmt.Errorf("error during executing get backup by status statement: %s", err)
    }

    return backups, nil
}

func removeDuplicateRevisions(revisions []*MirrorRevision) []*MirrorRevision {
    keys := make(map[int]*MirrorRevision)
    var list []*MirrorRevision

    for _, entry := range revisions {
        keys[entry.SourceMetadataID] = entry
    }
    for _, entry := range keys {
        list = append(list, entry)
    }
    return list
}

// GetExpiredBigQueryMirrorRevisions get expired BigQuery mirror revisions
func (d *defaultBackupRepository) GetExpiredBigQueryMirrorRevisions(ctxIn context.Context, maxRevisionLifetimeInWeeks int) ([]*MirrorRevision, error) {
    ctx, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).GetExpiredBigQueryMirrorRevisions")
    defer span.End()

    revisions, err := d.getBigQueryMirrorPastRevisionsThatExpiredAfterXWeeks(ctx, maxRevisionLifetimeInWeeks)
    if err != nil {
        return nil, err
    }
    revisionsByTTL, err := d.getBigQueryMirrorRevisionsThatExpiredByTTL(ctx)
    if err != nil {
        return nil, err
    }

    revisions = append(revisions, revisionsByTTL...)
    revisions = removeDuplicateRevisions(revisions)

    return revisions, nil
}

func (d *defaultBackupRepository) getBigQueryMirrorPastRevisionsThatExpiredAfterXWeeks(ctxIn context.Context, maxRevisionLifetimeInWeeks int) ([]*MirrorRevision, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).getBigQueryMirrorPastRevisionsThatExpiredAfterXWeeks")
    defer span.End()

    var revisions []*MirrorRevision
    db := d.storageService.DB()
    subselect := db.Model().
        Table("source_metadata").
        Column("*").
        ColumnExpr("lead(operation) over (partition by source order by audit_created_timestamp asc) as descendant").
        ColumnExpr("lead(audit_created_timestamp) over (partition by source order by audit_created_timestamp asc) as descendant_audit_created_timestamp")

    err := db.Model().TableExpr("(?) AS s", subselect).
        Join("LEFT JOIN source_metadata_jobs as smj on s.id=smj.source_metadata_id").
        Join("LEFT JOIN jobs j on smj.job_id = j.id").
        Join("LEFT JOIN backups b on s.backup_id = b.id").
        WhereGroup(func(sub *orm.Query) (*orm.Query, error) {
            return sub.WhereOr("descendant != ?", Delete).
                WhereOr("descendant_audit_created_timestamp < NOW() - interval '1 week' * ?", maxRevisionLifetimeInWeeks).
                WhereOr("operation = ?", Delete), nil
        }).Where("s.audit_created_timestamp < NOW() - interval '1 week' * ?", maxRevisionLifetimeInWeeks).
        Where("s.audit_deleted_timestamp IS NULL").
        WhereGroup(func(sub *orm.Query) (*orm.Query, error) {
            return sub.WhereOr("j.status is NULL").WhereOr("j.status=?", FinishedOk), nil
        }).
        Where("b.strategy = ?", Mirror).
        WhereGroup(func(query *orm.Query) (*orm.Query, error) {
            return query.WhereOr("b.status is NULL").WhereOr("b.status != ?", BackupDeleted), nil
        }).
        ColumnExpr("s.id as source_metadata_id").
        ColumnExpr("CASE WHEN job_id is NULL THEN '' ELSE job_id END").
        ColumnExpr("b.id as backup_id").
        ColumnExpr("b.bigquery_dataset").
        ColumnExpr("s.source").
        ColumnExpr("b.target_project").
        ColumnExpr("b.target_sink").
        Select(&revisions)

    if err != nil {
        logQueryError("getBigQueryMirrorPastRevisionsThatExpiredAfterXWeeks", err)
        return revisions, fmt.Errorf("error during executing get backup by status statement: %s", err)
    }

    return revisions, nil
}

func (d *defaultBackupRepository) getBigQueryMirrorRevisionsThatExpiredByTTL(ctx context.Context) ([]*MirrorRevision, error) {
    _, span := trace.StartSpan(ctx, "(*defaultBackupRepository).getBigQueryMirrorRevisionsThatExpiredByTTL")
    defer span.End()

    var revisions []*MirrorRevision

    subselect := d.storageService.DB().
        Model().
        Table("source_metadata").
        Column("*")

    err := d.storageService.DB().
        Model().
        TableExpr("(?) AS s", subselect).
        Join("LEFT JOIN source_metadata_jobs as smj on s.id=smj.source_metadata_id").
        Join("LEFT JOIN jobs j on smj.job_id = j.id").
        Join("LEFT JOIN backups b on s.backup_id = b.id").
        Where("s.audit_created_timestamp < (NOW() - interval '1 day' * b.mirror_lifetime_in_days)").
        Where("s.audit_deleted_timestamp is NULL").
        WhereGroup(func(query *orm.Query) (*orm.Query, error) {
            return query.WhereOr("j.status is NULL").WhereOr("j.status=?", FinishedOk), nil
        }).
        Where("b.mirror_lifetime_in_days > 0").
        WhereGroup(func(query *orm.Query) (*orm.Query, error) {
            return query.WhereOr("b.status is NULL").WhereOr("b.status != ?", BackupDeleted), nil
        }).
        ColumnExpr("s.id as source_metadata_id").
        ColumnExpr("CASE WHEN job_id is NULL THEN '' ELSE job_id END").
        ColumnExpr("b.id as backup_id").
        ColumnExpr("b.bigquery_dataset").
        ColumnExpr("s.source").
        ColumnExpr("b.target_project").
        ColumnExpr("b.target_sink").
        Select(&revisions)

    if err != nil {
        logQueryError("getBigQueryMirrorRevisionsThatExpiredByTTL", err)
        return revisions, fmt.Errorf("error during executing get backup by status statement: %s", err)
    }

    return revisions, nil
}

// GetScheduledBackups list backups that can have a new job prepared
func (d *defaultBackupRepository) GetScheduledBackups(ctxIn context.Context, backupType BackupType) ([]*Backup, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).GetScheduledBackups")
    defer span.End()

    var backups []*Backup

    err := d.storageService.DB().Model(&backups).
        Where("status IN (?, ?)", NotStarted, Prepared).
        Where("type = ?", backupType).
        Where("audit_deleted_timestamp IS NULL").
        Select()

    if err != nil {
        return nil, fmt.Errorf("error during executing get scheduled backup by type %s", err)
    }

    return backups, nil
}

// GetBigQueryOneShotSnapshots return backups that are BigQuery with strategy Snapshot
func (d *defaultBackupRepository) GetBigQueryOneShotSnapshots(ctxIn context.Context, status BackupStatus) ([]*Backup, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultBackupRepository).GetBigQueryOneShotSnapshots")
    defer span.End()

    var backups []*Backup

    err := d.storageService.DB().Model(&backups).
        Where("status = ?", status).
        Where("strategy = ?", Snapshot).
        Where("audit_deleted_timestamp IS NULL").
        Where("snapshot_frequency_in_hours = 0").
        Select()

    if err != nil {
        return nil, fmt.Errorf("error during executing get one shot snapshots by status statement: %s", err)
    }

    return backups, nil
}
