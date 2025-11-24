package memory

import (
	"context"
	"time"

	"github.com/ottogroup/penelope/pkg/repository"
	"go.opencensus.io/trace"
)

// SourceMetadataRepository storage for a source metadata
type SourceMetadataRepository struct {
	SourceMetadatas []*repository.SourceMetadata
}

// Add create a new source metadata
func (r *SourceMetadataRepository) Add(ctxIn context.Context, sourceMetadata []*repository.SourceMetadata) ([]*repository.SourceMetadata, error) {
	_, span := trace.StartSpan(ctxIn, "(*SourceMetadataRepository).Add")
	defer span.End()

	maxID := 0
	for _, s := range r.SourceMetadatas {
		if maxID < s.ID {
			maxID = s.ID
		}
	}
	for _, input := range sourceMetadata {
		maxID++
		input.ID = maxID
		r.SourceMetadatas = append(r.SourceMetadatas, input)
	}
	return sourceMetadata, nil
}

// GetLastByBackupID list all source metadata for a backup
func (r *SourceMetadataRepository) GetLastByBackupID(ctxIn context.Context, backupID string) (sourceMetadata []*repository.SourceMetadata, err error) {
	_, span := trace.StartSpan(ctxIn, "(*SourceMetadataRepository).GetLastByBackupID")
	defer span.End()

	// get the newest version for each source
	versions := make(map[string]bool)
	for i := len(r.SourceMetadatas) - 1; i >= 0; i-- {
		curr := r.SourceMetadatas[i]
		if _, ok := versions[curr.Source]; ok {
			continue
		}
		sourceMetadata = append(sourceMetadata, curr)
		versions[curr.Source] = true
	}
	return sourceMetadata, err
}

// MarkDeleted mark table as deleted
func (r *SourceMetadataRepository) MarkDeleted(ctxIn context.Context, id int) error {
	_, span := trace.StartSpan(ctxIn, "(*SourceMetadataRepository).MarkDeleted")
	defer span.End()

	for i, s := range r.SourceMetadatas {
		if s.ID == id {
			r.SourceMetadatas[i].DeletedTimestamp = time.Now()
			break
		}
	}
	return nil
}
