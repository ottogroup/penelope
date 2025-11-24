package memory

import (
	"context"
	"fmt"

	"go.opencensus.io/trace"
)

// SourceMetadataJob defines table version backup job
type SourceMetadataJob struct {
	SourceMetadataID int
	JobID            string
}

// DefaultSourceMetadataJobRepository gives possibility to add new SourceMetadataJob
type DefaultSourceMetadataJobRepository struct {
	sourceMetadataJobs []*SourceMetadataJob
}

// Add gives possibility add new SourceMetadataJob
func (r *DefaultSourceMetadataJobRepository) Add(ctxIn context.Context, sourceMetadataID int, jobID string) error {
	_, span := trace.StartSpan(ctxIn, "(*DefaultSourceMetadataJobRepository).Add")
	defer span.End()

	r.sourceMetadataJobs = append(r.sourceMetadataJobs, &SourceMetadataJob{SourceMetadataID: sourceMetadataID, JobID: jobID})
	return nil
}

func (r *DefaultSourceMetadataJobRepository) SetSourceMetadataID(ctx context.Context, jobID string, sourceMetadataID int) error {
	for _, job := range r.sourceMetadataJobs {
		if job.JobID == jobID {
			job.SourceMetadataID = sourceMetadataID
			return nil
		}
	}
	return fmt.Errorf("source metadata job id %s not found", jobID)
}
