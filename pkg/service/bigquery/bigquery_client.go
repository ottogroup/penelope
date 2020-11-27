package bigquery

import (
    bq "cloud.google.com/go/bigquery"
    "context"
    "fmt"
    "github.com/pkg/errors"
    "github.com/ottogroup/penelope/pkg/config"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "go.opencensus.io/trace"
    "google.golang.org/api/iterator"
    "google.golang.org/api/option"
    "net/http"
    "strconv"
    "time"
)

// Table store information for BigQuery table changes
type Table struct {
    Name        string
    Checksum    string
    SizeInBytes float64
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
}

// defaultBigQueryClient represent BigqUEry Client implementation
type defaultBigQueryClient struct {
    client          *bq.Client
    sourceProjectID string
    targetProjectID string
}

// NewBigQueryClient crete new instance of defaultBigQueryClient
func NewBigQueryClient(ctxIn context.Context, targetPrincipalProvider impersonate.TargetPrincipalForProjectProvider, sourceProjectID string, targetProjectID string) (Client, error) {
    ctx, span := trace.StartSpan(ctxIn, "NewBigQueryClient")
    defer span.End()

    target, err := targetPrincipalProvider.GetTargetPrincipalForProject(ctx, targetProjectID)
    if err != nil {
        return nil, err
    }

    options := []option.ClientOption{
        option.WithScopes(cloudPlatformAPIScope, defaultAPIScope),
        option.ImpersonateCredentials(target),
    }

    if config.UseDefaultHttpClient.GetBoolOrDefault(false) {
        options = append(options, option.WithHTTPClient(http.DefaultClient))
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
    m, err := oDataset.Table(table).Metadata(ctx)
    if err != nil {
        return &Table{}, err
    }
    return &Table{Name: table, Checksum: m.ETag, SizeInBytes: float64(m.NumBytes)}, nil
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
            tables = append(tables, &Table{Name: oTable.TableID, Checksum: tableMetadata.ETag, SizeInBytes: float64(tableMetadata.NumBytes)})
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
    Total int64     `bigquery:"total"`
    P     time.Time `bigquery:"p"`
}

// GetTablePartitions list all partitions in table
func (d *defaultBigQueryClient) GetTablePartitions(ctxIn context.Context, project string, dataset string, table string) ([]*Table, error) {
    ctx, span := trace.StartSpan(ctxIn, "(*defaultBigQueryClient).GetTablePartitions")
    defer span.End()

    metadata, err := d.client.DatasetInProject(project, dataset).Table(table).Metadata(ctx)
    if err != nil {
        return nil, err
    }

    timePartitioningField := "_PARTITIONTIME"
    if metadata.TimePartitioning.Field != "" {
        timePartitioningField = metadata.TimePartitioning.Field
    }

    var partitions []*Table
    q := fmt.Sprintf("SELECT count(*) as total, %s as p FROM `%s.%s.%s` WHERE %s IS NOT NULL GROUP BY p",
        timePartitioningField,
        project, dataset, table,
        timePartitioningField,
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
        if s.Total == 0{
            continue
        }
        month := zerofill(int(s.P.Month()))
        day := zerofill(s.P.Day())
        partitionTable := fmt.Sprintf("%s$%d%s%s", table, s.P.Year(), month, day)
        tableMetadata, err := d.client.DatasetInProject(project, dataset).Table(partitionTable).Metadata(ctx)
        if err != nil {
            return []*Table{}, err
        }
        partitions = append(partitions, &Table{Name: partitionTable, Checksum: tableMetadata.ETag, SizeInBytes: float64(tableMetadata.NumBytes)})
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
