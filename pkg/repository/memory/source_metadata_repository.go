package memory

import (
    "context"
    "github.com/ottogroup/penelope/pkg/repository"
    "go.opencensus.io/trace"
)

// SourceMetadataRepository storage for a source metadata
type SourceMetadataRepository struct {
    sourceMetadatas []*repository.SourceMetadata
}

// Add create a new source metadata
func (r *SourceMetadataRepository) Add(ctxIn context.Context, sourceMetadata []*repository.SourceMetadata) ([]*repository.SourceMetadata, error) {
    _, span := trace.StartSpan(ctxIn, "(*SourceMetadataRepository).Add")
    defer span.End()

    maxID := 0
    for _, s := range r.sourceMetadatas {
        if maxID < s.ID {
            maxID = s.ID
        }
    }
    for _, input := range sourceMetadata {
        maxID++
        input.ID = maxID
        r.sourceMetadatas = append(r.sourceMetadatas, input)
    }
    return sourceMetadata, nil
}

// GetLastByBackupID list all source metadata for a backup
func (r *SourceMetadataRepository) GetLastByBackupID(ctxIn context.Context, backupID string) (sourceMetadata []*repository.SourceMetadata, err error) {
    _, span := trace.StartSpan(ctxIn, "(*SourceMetadataRepository).GetLastByBackupID")
    defer span.End()

    for _, s := range r.sourceMetadatas {
        if s.BackupID == backupID {
            sourceMetadata = append(sourceMetadata, s)
        }
    }
    return sourceMetadata, err
}

// MarkDeleted mark table as deleted
func (r *SourceMetadataRepository) MarkDeleted(ctxIn context.Context, id int) error {
    _, span := trace.StartSpan(ctxIn, "(*SourceMetadataRepository).MarkDeleted")
    defer span.End()

    for i, s := range r.sourceMetadatas {
        if s.ID == id {
            r.sourceMetadatas[i] = r.sourceMetadatas[len(r.sourceMetadatas)-1] // Replace it with the last one.
            r.sourceMetadatas = r.sourceMetadatas[:len(r.sourceMetadatas)-1]   // Chop off the last one.
            break
        }
    }
    return nil
}
