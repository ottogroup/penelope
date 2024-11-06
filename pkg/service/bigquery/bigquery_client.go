package bigquery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	bq "cloud.google.com/go/bigquery"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"go.opencensus.io/trace"
	gimpersonate "google.golang.org/api/impersonate"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// Table store information for BigQuery table changes
type Table struct {
	Name             string
	Checksum         string
	SizeInBytes      float64
	LastModifiedTime time.Time
}

func newTableEntry(name string, tableMetadata *bq.TableMetadata) *Table {
	return &Table{
		Name:             name,
		Checksum:         tableMetadata.ETag,
		SizeInBytes:      float64(tableMetadata.NumBytes),
		LastModifiedTime: tableMetadata.LastModifiedTime,
	}
}

// Client define operations for BigQuery
type Client interface {
	IsInitialized(ctxIn context.Context) bool
	ExtractTableToGcsAsAvro(ctxIn context.Context, dataset, table, gcsURI string) *bq.Extractor
	GetExtractJobStatus(ctxIn context.Context, extractJobID string) (*bq.JobStatus, error)
	DoesDatasetExists(ctxIn context.Context, project string, dataset string) (bool, error)
	GetTable(ctxIn context.Context, project string, dataset string, table string) (*Table, error)
	GetTablesInDataset(ctxIn context.Context, project string, dataset string) ([]*Table, error)
	HasTablePartitions(ctxIn context.Context, project string, dataset string, table string) (bool, error)
	GetTablePartitions(ctxIn context.Context, project string, dataset string, table string) ([]*Table, error)
	GetDatasets(ctxIn context.Context, project string) ([]string, error)
	DeleteExtractJob(ctxIn context.Context, extractJobID string, location string) error
	GetDatasetDetails(ctxIn context.Context, datasetId string) (*bq.DatasetMetadata, error)
}

// defaultBigQueryClient represent BigqUEry Client implementation
type defaultBigQueryClient struct {
	client          *bq.Client
	sourceProjectID string
	targetProjectID string
}

func (d *defaultBigQueryClient) DeleteExtractJob(ctxIn context.Context, extractJobID string, location string) error {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultBigQueryClient).DeleteExtractJob")
	defer span.End()

	job, err := d.client.JobFromIDLocation(ctx, extractJobID, location)
	if err != nil {
		return err
	}

	return job.Delete(ctx)
}

// NewBigQueryClient crete new instance of defaultBigQueryClient
func NewBigQueryClient(ctxIn context.Context, targetPrincipalProvider impersonate.TargetPrincipalForProjectProvider, sourceProjectID string, targetProjectID string) (Client, error) {
	ctx, span := trace.StartSpan(ctxIn, "NewBigQueryClient")
	defer span.End()

	target, delegates, err := targetPrincipalProvider.GetTargetPrincipalForProject(ctx, targetProjectID)
	if err != nil {
		return nil, err
	}

	var options []option.ClientOption
	if config.UseDefaultHttpClient.GetBoolOrDefault(false) {
		options = []option.ClientOption{
			option.WithHTTPClient(http.DefaultClient),
		}
	} else {
		tokenSource, err := gimpersonate.CredentialsTokenSource(ctx, gimpersonate.CredentialsConfig{
			TargetPrincipal: target,
			Scopes:          []string{cloudPlatformAPIScope, defaultAPIScope},
			Delegates:       delegates,
		})
		if err != nil {
			return nil, err
		}

		options = []option.ClientOption{
			option.WithTokenSource(tokenSource),
		}
	}
	client, err := bq.NewClient(ctx, targetProjectID, options...)
	if err != nil {
		return &defaultBigQueryClient{}, fmt.Errorf("failed to create client: %s", err)
	}
	return &defaultBigQueryClient{client: client, sourceProjectID: sourceProjectID, targetProjectID: targetProjectID}, nil
}

// IsInitialized check if BigQuery client is initialized
func (d *defaultBigQueryClient) IsInitialized(ctxIn context.Context) bool {
	_, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).IsInitialized")
	defer span.End()

	return d.client != nil
}

// ExtractTableToGcsAsAvro will export data into GCS Bucket in AVRO format
// FIXME: method overlapping with ExtractJobHandler
func (d *defaultBigQueryClient) ExtractTableToGcsAsAvro(ctxIn context.Context, dataset, table, sinkURI string) *bq.Extractor {
	_, span := trace.StartSpan(ctxIn, "(*defaultBigQueryClient).ExtractTableToGcsAsAvro")
	defer span.End()

	gcsURI := bq.NewGCSReference(sinkURI)
	extractor := d.client.DatasetInProject(d.sourceProjectID, dataset).Table(table).ExtractorTo(gcsURI)
	extractor.Dst.DestinationFormat = bq.Avro
	return extractor
}

// GetExtractJobStatus return status for extract job
func (d *defaultBigQueryClient) GetExtractJobStatus(ctxIn context.Context, extractJobID string) (*bq.JobStatus, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultBigQueryClient).GetExtractJobStatus")
	defer span.End()

	job, err := d.client.JobFromID(ctx, extractJobID)
	if err != nil {
		return &bq.JobStatus{}, err
	}
	status, err := job.Status(ctx)
	if err != nil {
		return &bq.JobStatus{}, err
	}

	return status, nil
}

// DoesDatasetExists check if dataset exist
func (d *defaultBigQueryClient) DoesDatasetExists(ctxIn context.Context, project string, dataset string) (bool, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultBigQueryClient).DoesDatasetExists")
	defer span.End()

	oDataset := d.client.DatasetInProject(project, dataset)
	metadata, err := oDataset.Metadata(ctx)
	if err != nil {
		return false, err
	}
	if metadata != nil {
		return true, nil
	}
	return false, nil
}

// GetTable return metadata of the table
func (d *defaultBigQueryClient) GetTable(ctxIn context.Context, project string, dataset string, table string) (*Table, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultBigQueryClient).GetTable")
	defer span.End()

	oDataset := d.client.DatasetInProject(project, dataset)
	tableMetadata, err := oDataset.Table(table).Metadata(ctx)
	if err != nil {
		return &Table{}, err
	}
	return newTableEntry(table, tableMetadata), nil
}

// GetTablesInDataset list all tables in a dataset
func (d *defaultBigQueryClient) GetTablesInDataset(ctxIn context.Context, project string, dataset string) ([]*Table, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultBigQueryClient).GetTablesInDataset")
	defer span.End()

	var tables []*Table

	tableIt := d.client.DatasetInProject(project, dataset).Tables(ctx)
	for {
		oTable, err := tableIt.Next()
		if err == iterator.Done {
			break
		}
		if err != iterator.Done && err != nil {
			return nil, err
		}
		if oTable == nil {
			return []*Table{}, fmt.Errorf("oTable was nil")
		}

		tableMetadata, err := oTable.Metadata(ctx)
		if err != nil {
			return []*Table{}, err
		}
		if tableMetadata.Type == bq.RegularTable {
			tables = append(tables, newTableEntry(oTable.TableID, tableMetadata))
		}
	}
	return tables, nil
}

// HasTablePartitions check if table has partitions
func (d *defaultBigQueryClient) HasTablePartitions(ctxIn context.Context, project string, dataset string, table string) (bool, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultBigQueryClient).HasTablePartitions")
	defer span.End()

	metadata, err := d.client.DatasetInProject(project, dataset).Table(table).Metadata(ctx)
	if err != nil {
		return false, err
	}
	return metadata.TimePartitioning != nil, nil
}

type tablePartition struct {
	Total       int64  `bigquery:"total"`
	PartitionID string `bigquery:"partition_id"`
}

// GetTablePartitions list all partitions in table
func (d *defaultBigQueryClient) GetTablePartitions(ctxIn context.Context, project string, dataset string, table string) ([]*Table, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultBigQueryClient).GetTablePartitions")
	defer span.End()

	metadata, err := d.client.DatasetInProject(project, dataset).Table(table).Metadata(ctx)
	if err != nil {
		return nil, err
	}

	if metadata.TimePartitioning.Type != bq.DayPartitioningType {
		return nil, fmt.Errorf("GetTablePartitions failed for `%s.%s.%s`, because partition other then DAY is not supported", project, dataset, table)
	}

	var partitions []*Table
	q := fmt.Sprintf("SELECT total_rows AS total, partition_id FROM `%s.%s.INFORMATION_SCHEMA.PARTITIONS` WHERE TABLE_NAME = '%s'",
		project, dataset, table,
	)

	run, err := d.client.Query(q).Run(ctx)
	if err != nil {
		return nil, err
	}
	rowIt, err := run.Read(ctx)
	if err != nil {
		return nil, err
	}
	partitionMetadataCollected := make(map[string]bool)
	for {
		var s tablePartition
		err := rowIt.Next(&s)
		if errors.Is(err, iterator.Done) {
			break
		} else if err != nil {
			return nil, err
		}
		if s.Total == 0 {
			continue
		}

		partition := s.PartitionID
		if partition == "" {
			return nil, fmt.Errorf("GetTablePartitions failed for `%s.%s.%s`, because partition_id is empty", project, dataset, table)
		}

		if _, exists := partitionMetadataCollected[partition]; exists {
			// tables that where updated multiple times in the same day are skipped
			continue
		}
		partitionTable := fmt.Sprintf("%s$%s", table, partition)
		tableMetadata, err := d.client.DatasetInProject(project, dataset).Table(partitionTable).Metadata(ctx)
		if err != nil {
			return []*Table{}, err
		}
		partitions = append(partitions, newTableEntry(partitionTable, tableMetadata))
		partitionMetadataCollected[partition] = true
	}
	return partitions, nil
}

// GetDatasets list all datasets in a project
func (d *defaultBigQueryClient) GetDatasets(ctxIn context.Context, project string) (datasets []string, err error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultBigQueryClient).GetDatasets")
	defer span.End()

	it := d.client.Datasets(ctx)
	it.ProjectID = project
	for {
		dataset, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		} else if err != nil {
			return []string{}, errors.Join(err, fmt.Errorf("Datasets.Next() failed for project %s", project))
		}
		if dataset == nil {
			return datasets, fmt.Errorf("datasets are nil for project %s", project)
		}
		datasets = append(datasets, dataset.DatasetID)
	}
	return datasets, err
}

// GetDatasetDetails get the details of a bigquery dataset
func (d *defaultBigQueryClient) GetDatasetDetails(ctxIn context.Context, datasetId string) (*bq.DatasetMetadata, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultBigQueryClient).GetDatasetDetails")
	defer span.End()

	return d.client.Dataset(datasetId).Metadata(ctx)
}
