package bigquery

import (
	"context"
	"errors"
	"fmt"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"go.opencensus.io/trace"
	"google.golang.org/api/googleapi"
	"net/http"
)

// ExtractJobHandler represent exporting data from BigQuery
type ExtractJobHandler struct {
	bq Client
}

// NewExtractJobHandler create new instance of ExtractJobHandler
func NewExtractJobHandler(ctxIn context.Context, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, srcProjectID, targetProjectID string) (*ExtractJobHandler, error) {
	ctx, span := trace.StartSpan(ctxIn, "NewExtractJobHandler")
	defer span.End()

	bgClient, err := NewBigQueryClient(ctx, tokenSourceProvider, srcProjectID, targetProjectID)
	if err != nil || bgClient == nil || !bgClient.IsInitialized(ctx) {
		return &ExtractJobHandler{}, fmt.Errorf("can not create instance of ExtractJobHandler with unititialized Client")
	}

	return &ExtractJobHandler{bq: bgClient}, nil
}

// CreateAvroJob start a BigQuery job that export data in AVRO format
func (e *ExtractJobHandler) CreateAvroJob(ctxIn context.Context, dataset, table, sinkURI string) (string, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*ExtractJobHandler).CreateAvroJob")
	defer span.End()

	extractor := e.bq.ExtractTableToGcsAsAvro(ctx, dataset, table, sinkURI)

	job, err := extractor.Run(ctx)
	if err != nil {
		return "", err
	}

	return job.ID(), nil
}

// GetStatusOfJob get actuall status for a BigQuery job
func (e *ExtractJobHandler) GetStatusOfJob(ctxIn context.Context, extractJobID string) (ExtractJobState, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*ExtractJobHandler).GetStatusOfJob")
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
func (e *ExtractJobHandler) DeleteExtractJob(ctx context.Context, jobID string, location string) error {
	ctx, span := trace.StartSpan(ctx, "(*ExtractJobHandler).DeleteExtractJob")
	defer span.End()

	err := e.bq.DeleteExtractJob(ctx, jobID, location)
	var googleAPIErr *googleapi.Error
	if err != nil && errors.As(err, &googleAPIErr) && googleAPIErr.Code == http.StatusNotFound {
		return nil
	}

	return err
}
