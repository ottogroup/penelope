package repository

import (
    "context"
    "fmt"
    "github.com/go-pg/pg/v10"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/service"
    "go.opencensus.io/trace"
    "time"
)

// SourceMetadataRepository defines operation for SourceMetadata
type SourceMetadataRepository interface {
    Add(context.Context, []*SourceMetadata) ([]*SourceMetadata, error)
    GetLastByBackupID(ctxIn context.Context, backupID string) ([]*SourceMetadata, error)
    MarkDeleted(context.Context, int) error
}

// NewSourceMetadataRepository return instance of SourceMetadataRepository
func NewSourceMetadataRepository(ctxIn context.Context, credentialsProvider secret.SecretProvider) (SourceMetadataRepository, error) {
    ctx, span := trace.StartSpan(ctxIn, "NewSourceMetadataRepository")
    defer span.End()

    storageService, err := service.NewStorageService(ctx, credentialsProvider)
    if err != nil {
        return nil, err
    }
    return &defaultSourceMetadataRepository{storageService: storageService}, nil
}

type defaultSourceMetadataRepository struct {
    storageService *service.Service
}

// Add new SourceMetadata entries
func (b *defaultSourceMetadataRepository) Add(ctxIn context.Context, sourceMetadata []*SourceMetadata) ([]*SourceMetadata, error) {
    ctx, span := trace.StartSpan(ctxIn, "(*defaultSourceMetadataRepository).Add")
    defer span.End()

    err := b.storageService.DB().RunInTransaction(ctx, func(tx *pg.Tx) error {
        for _, sm := range sourceMetadata {
            _, err := b.storageService.DB().Model(sm).Insert()
            if err != nil {
                return err
            }
        }
        return nil
    })

    return sourceMetadata, err
}

// GetLastByBackupID get the latest created source metadata for backup
func (b *defaultSourceMetadataRepository) GetLastByBackupID(ctxIn context.Context, backupID string) ([]*SourceMetadata, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultSourceMetadataRepository).GetLastByBackupID")
    defer span.End()
    
    var sourceMetadata []*SourceMetadata
    db := b.storageService.DB()
    subselect := db.Model().
        Table("source_metadata").
        Column("*").
        ColumnExpr("rank() over (partition by source order by audit_created_timestamp desc) as inner_rank").
        Where("backup_id = ?", backupID)

    err := db.Model().TableExpr("(?) AS s", subselect).
        Column(
            "id", 
            "backup_id", 
            "source", 
            "source_checksum", 
            "operation", 
            "audit_created_timestamp", 
            "audit_deleted_timestamp",
        ).
        Where("inner_rank = 1").
        Select(&sourceMetadata)

    return sourceMetadata, err
}

// MarkDeleted
func (b *defaultSourceMetadataRepository) MarkDeleted(ctxIn context.Context, id int) error {
    _, span := trace.StartSpan(ctxIn, "(*defaultSourceMetadataRepository).MarkDeleted")
    defer span.End()

    sourceMetadata := &SourceMetadata{
        ID: id,
        DeletedTimestamp: time.Now(),
    }

    _, err := b.storageService.DB().Model(sourceMetadata).
        Column("audit_deleted_timestamp").
        WherePK().
        Where("audit_deleted_timestamp IS NULL").
        Update()

    if err != nil {
        logQueryError("MarkDeleted", err)
        return fmt.Errorf("error during executing updating backup statemant: %s", err)
    }

    return nil
}
