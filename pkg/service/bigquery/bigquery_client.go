package bigquery

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	bq "cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/pkg/errors"
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
	Total     int64          `bigquery:"total"`
	TIMESTAMP time.Time      `bigquery:"T_TIMESTAMP"`
	DATE      civil.Date     `bigquery:"T_DATE"`
	TIME      civil.Time     `bigquery:"T_TIME"`
	DATETIME  civil.DateTime `bigquery:"T_DATETIME"`
}

func (t *tablePartition) getPartitionFor(targetField string) (string, error) {
	var partitionName string
	switch targetField {
	case "T_TIMESTAMP":
		partitionName = fmt.Sprintf("%s%s%s",
			zerofill(t.TIMESTAMP.Year()),
			zerofill(int(t.TIMESTAMP.Month())),
			zerofill(t.TIMESTAMP.Day()))
	case "T_DATE":
		partitionName = fmt.Sprintf("%s%s%s",
			zerofill(t.DATE.Year),
			zerofill(int(t.DATE.Month)),
			zerofill(t.DATE.Day))
	case "T_DATETIME":
		partitionName = fmt.Sprintf("%s%s%s",
			zerofill(t.DATETIME.Date.Year),
			zerofill(int(t.DATETIME.Date.Month)),
			zerofill(t.DATETIME.Date.Day))
	default:
		return "", fmt.Errorf("partition for target field %q is not supported", targetField)
	}
	return partitionName, nil
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

	timePartitioningField := "_PARTITIONTIME"
	targetFieldInTablePartition := "T_TIMESTAMP"
	if metadata.TimePartitioning.Field != "" {
		timePartitioningField = metadata.TimePartitioning.Field
		for _, schema := range metadata.Schema.Relax() {
			if schema.Name == metadata.TimePartitioning.Field {
				targetFieldInTablePartition = fmt.Sprintf("T_%s", schema.Type)
				break
			}
		}
	}

	var partitions []*Table
	q := fmt.Sprintf("SELECT COUNT(*) AS total, %s AS %s FROM `%s.%s.%s` WHERE %s IS NOT NULL GROUP BY %s",
		timePartitioningField, targetFieldInTablePartition,
		project, dataset, table,
		timePartitioningField,
		targetFieldInTablePartition,
	)

	run, err := d.client.Query(q).Run(ctx)
	if err != nil {
		return nil, err
	}
	rowIt, err := run.Read(ctx)
	if err != nil {
		return nil, err
	}
	for {
		var s tablePartition
		err := rowIt.Next(&s)
		if err == iterator.Done {
			break
		}
		if err != iterator.Done && err != nil {
			return nil, err
		}
		if s.Total == 0 {
			continue
		}

		partition, err := s.getPartitionFor(targetFieldInTablePartition)
		if err != nil {
			return nil, fmt.Errorf("GetTablePartitions failed for `%s.%s.%s`, because partition with %s is not supported", project, dataset, table, targetFieldInTablePartition)
		}
		partitionTable := fmt.Sprintf("%s$%s", table, partition)
		tableMetadata, err := d.client.DatasetInProject(project, dataset).Table(partitionTable).Metadata(ctx)
		if err != nil {
			return []*Table{}, err
		}
		partitions = append(partitions, newTableEntry(partitionTable, tableMetadata))
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
		if err == iterator.Done {
			break
		}
		if err != nil && err != iterator.Done {
			return []string{}, errors.Wrap(err, fmt.Sprintf("Datasets.Next() failed for project %s", project))
		}
		if dataset == nil {
			return datasets, fmt.Errorf("datasets are nil for project %s", project)
		}
		datasets = append(datasets, dataset.DatasetID)
	}
	return datasets, err
}

func zerofill(intToFill int) string {
	if intToFill < 10 {
		return "0" + strconv.Itoa(intToFill)
	}
	return strconv.Itoa(intToFill)
}

// GetDatasetDetails get the details of a bigquery dataset
func (d *defaultBigQueryClient) GetDatasetDetails(ctxIn context.Context, datasetId string) (*bq.DatasetMetadata, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultBigQueryClient).GetDatasetDetails")
	defer span.End()

	return d.client.Dataset(datasetId).Metadata(ctx)
}
