package memory

import (
	"context"

	"go.opentelemetry.io/otel"
)

// SourceMetadataJob defines table version backup job
type SourceMetadataJob struct {
	SourceMetadataID int
	JobID            string
}

// DefaultSourceMetadataJobRepository gives possibility to add new SourceMetadataJob
type DefaultSourceMetadataJobRepository struct {
	SourceMetadataJobs []*SourceMetadataJob
}

// Add gives possibility add new SourceMetadataJob
func (r *DefaultSourceMetadataJobRepository) Add(ctxIn context.Context, sourceMetadataID int, jobID string) error {
	_, span := otel.Tracer("").Start(ctxIn, "(*DefaultSourceMetadataJobRepository).Add")
	defer span.End()

	r.SourceMetadataJobs = append(r.SourceMetadataJobs, &SourceMetadataJob{SourceMetadataID: sourceMetadataID, JobID: jobID})
	return nil
}
