package repository

import (
    "context"
    "github.com/go-pg/pg/v10/orm"
    "github.com/pkg/errors"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/service"
    "go.opencensus.io/trace"
    "time"
)

// SourceTrashcanRepository defines operation for SourceTrashcan
type SourceTrashcanRepository interface {
    Add(ctxIn context.Context, backupID string, source string, timestamp time.Time) error
    Delete(ctxIn context.Context, backupID string, source string) error
    FilterExistingEntries(ctxIn context.Context, sources []SourceTrashcan) ([]SourceTrashcan, error)
    GetBefore(ctxIn context.Context, deltaWeeks int) ([]*SourceTrashcan, error)
}

// defaultSourceTrashcan implements SourceTrashcanRepository
type defaultSourceTrashcan struct {
    storageService *service.Service
}

// NewSourceTrashcanRepository return instance of SourceTrashcanRepository
func NewSourceTrashcanRepository(ctxIn context.Context, credentialsProvider secret.SecretProvider) (SourceTrashcanRepository, error) {
    ctx, span := trace.StartSpan(ctxIn, "NewSourceTrashcanRepository")
    defer span.End()

    storageService, err := service.NewStorageService(ctx, credentialsProvider)
    if err != nil {
        return nil, err
    }

    return &defaultSourceTrashcan{storageService: storageService}, nil
}

// Add new Add source trashcan entry
func (d *defaultSourceTrashcan) Add(ctxIn context.Context, backupID string, source string, timestamp time.Time) error {
    _, span := trace.StartSpan(ctxIn, "(*defaultSourceTrashcan).Add")
    defer span.End()

    _, err := d.storageService.DB().Model(&SourceTrashcan{
        BackupID:         backupID,
        Source:           source,
        CreatedTimestamp: timestamp,
    }).Insert()

    if err != nil {
        return errors.Wrap(err, "error during executing add statement")
    }

    return nil
}


// Delete delete source trashcan entry
func (d *defaultSourceTrashcan) Delete(ctxIn context.Context, backupID string, source string) error {
    _, span := trace.StartSpan(ctxIn, "(*defaultSourceTrashcan).Delete")
    defer span.End()

    _, err := d.storageService.
        DB().
        Model(&SourceTrashcan{}).
            Where("backup_id = ?", backupID).
            Where("source = ?", source).
        Delete()

    if err != nil {
        return errors.Wrapf(err, "error during executing delete statement for backupID=%s and source=%s", backupID, source)
    }

    return nil
}


// FilterExistingEntries get source trashcan for a given sources
func (d *defaultSourceTrashcan) FilterExistingEntries(ctxIn context.Context, sources []SourceTrashcan) ([]SourceTrashcan, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultSourceTrashcan).FilterExistingEntries")
    defer span.End()

    var sourceTrashcans []SourceTrashcan
    query := d.storageService.
        DB().
        Model(&sourceTrashcans)

    for _, source := range sources {
        query = query.WhereOrGroup(func(query *orm.Query) (*orm.Query, error) {
            query = query.
                Where("backup_id = ?", source.BackupID).
                Where("source = ?", source.Source)
            return query, nil
        })
    }

    err := query.Select()
    if err != nil {
        return nil, errors.Wrapf(err, "error during executing filter existing entries")
    }

    return sourceTrashcans, nil
}


// GetBefore get entries after given time
func (d *defaultSourceTrashcan) GetBefore(ctxIn context.Context, deltaWeeks int) ([]*SourceTrashcan, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultSourceTrashcan).GetBefore")
    defer span.End()

    var sourceTrashcans []*SourceTrashcan

    err := d.storageService.DB().
        Model(&sourceTrashcans).
        Where("audit_created_timestamp < NOW() - (interval '1 week' * ?)", deltaWeeks).
        Select()

    if err != nil {
        return nil, errors.Wrapf(err, "error during executing get after delta weeks")
    }

    return sourceTrashcans, nil
}
