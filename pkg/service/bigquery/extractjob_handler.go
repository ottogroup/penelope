package bigquery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/repository"
	"go.opentelemetry.io/otel"
	"google.golang.org/api/googleapi"
)

// ExtractJobHandler represent exporting data from BigQuery
type ExtractJobHandler struct {
	bq Client
}

// NewExtractJobHandler create new instance of ExtractJobHandler
func NewExtractJobHandler(ctxIn context.Context, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, srcProjectID, targetProjectID string) (*ExtractJobHandler, error) {
	ctx, span := otel.Tracer("").Start(ctxIn, "NewExtractJobHandler")
	defer span.End()

	bgClient, err := NewBigQueryClient(ctx, tokenSourceProvider, srcProjectID, targetProjectID)
	if err != nil {
		return &ExtractJobHandler{}, fmt.Errorf("can not create instance of ExtractJobHandler: %s", err)
	}
	if bgClient == nil || !bgClient.IsInitialized(ctx) {
		return &ExtractJobHandler{}, fmt.Errorf("can not create instance of ExtractJobHandler with unititialized Client")
	}

	return &ExtractJobHandler{bq: bgClient}, nil
}

// CreateAvroJob start a BigQuery job that export data in AVRO format
func (e *ExtractJobHandler) CreateAvroJob(ctxIn context.Context, dataset, table, sinkURI string) (repository.ExtractJobID, error) {
	ctx, span := otel.Tracer("").Start(ctxIn, "(*ExtractJobHandler).CreateAvroJob")
	defer span.End()

	extractor := e.bq.ExtractTableToGcsAsAvro(ctx, dataset, table, sinkURI)

	job, err := extractor.Run(ctx)
	if err != nil {
		return "", err
	}

	return repository.NewExtractJobIDWithLocation(job.ID(), job.Location()), nil
}

// CreateTableSnapshotJob start a BigQuery copy job that creates a native table snapshot
func (e *ExtractJobHandler) CreateTableSnapshotJob(ctxIn context.Context, dataset, table, dstTable string) (repository.ExtractJobID, error) {
	ctx, span := otel.Tracer("").Start(ctxIn, "(*ExtractJobHandler).CreateTableSnapshotJob")
	defer span.End()

	// snapshot destination dataset shares the source dataset's name, in the target project
	copier := e.bq.CopyTableAsSnapshot(ctx, dataset, table, dataset, dstTable)

	job, err := copier.Run(ctx)
	if err != nil {
		return "", err
	}

	return repository.NewExtractJobIDWithLocation(job.ID(), job.Location()), nil
}

// GetStatusOfJob get actual status for a BigQuery job
func (e *ExtractJobHandler) GetStatusOfJob(ctxIn context.Context, extractJobID repository.ExtractJobID) (ExtractJobState, error) {
	ctx, span := otel.Tracer("").Start(ctxIn, "(*ExtractJobHandler).GetStatusOfJob")
	defer span.End()

	jobStatus, err := e.bq.GetExtractJobStatus(ctx, extractJobID)
	if err != nil {
		return StateUnspecified, err
	}

	if jobStatus.Err() != nil {
		// handle non Quota Errors
		for _, jobError := range jobStatus.Errors {
			if jobError.Reason != "quotaExceeded" {
				return Failed, jobStatus.Err()
			}
		}
		// handle Quota Errors
		return FailedQuotaExceeded, jobStatus.Err()
	}

	return toJobState(jobStatus.State), nil
}

// DeleteExtractJob delete a BigQuery job
// If job does not exist, it returns nil
func (e *ExtractJobHandler) DeleteExtractJob(ctx context.Context, jobID repository.ExtractJobID) error {
	ctx, span := otel.Tracer("").Start(ctx, "(*ExtractJobHandler).DeleteExtractJob")
	defer span.End()

	err := e.bq.DeleteExtractJob(ctx, jobID)
	var googleAPIErr *googleapi.Error
	if err != nil && errors.As(err, &googleAPIErr) && googleAPIErr.Code == http.StatusNotFound {
		return nil
	}

	return err
}

// UpdateTableExpiration sets a table's expiration time, used as a best-effort TTL for native table snapshots
func (e *ExtractJobHandler) UpdateTableExpiration(ctxIn context.Context, project, dataset, table string, expiration time.Time) error {
	ctx, span := otel.Tracer("").Start(ctxIn, "(*ExtractJobHandler).UpdateTableExpiration")
	defer span.End()

	return e.bq.UpdateTableExpiration(ctx, project, dataset, table, expiration)
}

// DeleteTable deletes a table, e.g. a native table snapshot when its backup is removed
// If table does not exist, it returns nil
func (e *ExtractJobHandler) DeleteTable(ctx context.Context, project, dataset, table string) error {
	ctx, span := otel.Tracer("").Start(ctx, "(*ExtractJobHandler).DeleteTable")
	defer span.End()

	err := e.bq.DeleteTable(ctx, project, dataset, table)
	var googleAPIErr *googleapi.Error
	if err != nil && errors.As(err, &googleAPIErr) && googleAPIErr.Code == http.StatusNotFound {
		return nil
	}

	return err
}
