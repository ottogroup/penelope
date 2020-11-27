package repository

import (
    "context"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/service"
    "go.opencensus.io/trace"
)

// SourceMetadataJobRepository defines operation for sourceMetadata
type SourceMetadataJobRepository interface {
    Add(ctxIn context.Context, sourceMetadataID int, jobID string) error
}

// DefaultSourceMetadataJobRepository implements instance of SourceMetadataJobRepository
type DefaultSourceMetadataJobRepository struct {
    storageService *service.Service
}

// NewSourceMetadataJobRepository return instance of SourceMetadataJobRepository
func NewSourceMetadataJobRepository(ctxIn context.Context, credentialsProvider secret.SecretProvider) (SourceMetadataJobRepository, error) {
    ctx, span := trace.StartSpan(ctxIn, "NewSourceMetadataJobRepository")
    defer span.End()

    storageService, err := service.NewStorageService(ctx, credentialsProvider)
    if err != nil {
        return nil, err
    }

    return &DefaultSourceMetadataJobRepository{storageService: storageService}, nil
}

// Add mew sourceMetadata entry
func (d *DefaultSourceMetadataJobRepository) Add(ctxIn context.Context, sourceMetadataID int, jobID string) error {
    _, span := trace.StartSpan(ctxIn, "(*DefaultSourceMetadataJobRepository).Add")
    defer span.End()

    _, err := d.storageService.DB().Model(&SourceMetadataJob{
        SourceMetadataID: sourceMetadataID,
        JobId:            jobID,
    }).Insert()

    return err
}
