package processor

import (
    "context"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/service/bigquery"
    "fmt"
    "github.com/golang/glog"
    "go.opencensus.io/trace"
    "google.golang.org/api/googleapi"
    "strings"
    "time"
)

// BigQueryJobCreator prepares jobs for BigQuery
type BigQueryJobCreator struct {
    BackupRepository            repository.BackupRepository
    JobRepository               repository.JobRepository
    SourceMetadataRepository    repository.SourceMetadataRepository
    SourceMetadataJobRepository repository.SourceMetadataJobRepository
    BigQuery                    bigquery.Client
}

// NewBigQueryJobCreator return instance of BigQueryJobCreator
func NewBigQueryJobCreator(ctxIn context.Context, backupRepository repository.BackupRepository, jobRepository repository.JobRepository, bigQueryClient bigquery.Client,
    sourceMetadataRepository repository.SourceMetadataRepository, sourceMetadataJobRepository repository.SourceMetadataJobRepository) *BigQueryJobCreator {
    _, span := trace.StartSpan(ctxIn, "NewBigQueryJobCreator")
    defer span.End()

    return &BigQueryJobCreator{
        BackupRepository:            backupRepository,
        JobRepository:               jobRepository,
        SourceMetadataRepository:    sourceMetadataRepository,
        SourceMetadataJobRepository: sourceMetadataJobRepository,
        BigQuery:                    bigQueryClient,
    }
}

// PrepareJobs new BigQuery extract job
func (b *BigQueryJobCreator) PrepareJobs(ctxIn context.Context, backup *repository.Backup) error {
    ctx, span := trace.StartSpan(ctxIn, "(*BigQueryJobCreator).PrepareJobs")
    defer span.End()

    if repository.Mirror == backup.Strategy {
        return b.prepareMirrorJobs(ctx, backup)
    } else if repository.Snapshot == backup.Strategy {
        return b.prepareSnapshotJobs(ctx, backup)
    } else {
        return fmt.Errorf("unkown strategy %s", backup.Strategy)
    }
}

func (b *BigQueryJobCreator) prepareSnapshotJobs(ctxIn context.Context, backup *repository.Backup) error {
    ctx, span := trace.StartSpan(ctxIn, "(*BigQueryJobCreator).prepareSnapshotJobs")
    defer span.End()

    tables, err := b.flattenTables(ctx, backup)
    if err != nil {
        return err
    }

    var jobs []*repository.Job
    for _, table := range tables {
        jobs = append(jobs, newJob(backup.ID, table.Name))
    }

    err = b.JobRepository.AddJobs(ctx, jobs)
    if err == nil {
        err = b.BackupRepository.UpdateLastScheduledTime(ctx, backup.ID, time.Now(), repository.Prepared)
    }

    return err
}

func (b *BigQueryJobCreator) prepareMirrorJobs(ctxIn context.Context, backup *repository.Backup) error {
    ctx, span := trace.StartSpan(ctxIn, "(*BigQueryJobCreator).prepareMirrorJobs")
    defer span.End()

    tables, err := b.flattenTables(ctx, backup)
    if err != nil {
        return err
    }

    jobDescriptors, err := b.collateState(ctx, backup.ID, tables)
    if err != nil {
        return err
    }

    var jobs []*repository.Job
    for _, descriptor := range jobDescriptors {
        jobs = append(jobs, newJob(backup.ID, descriptor.table))
    }

    err = b.JobRepository.AddJobs(ctx, jobs)
    if err != nil {
        return err
    }

    for _, descriptor := range jobDescriptors {
        for _, job := range jobs {
            if descriptor.matchJob(job) {
                err = b.SourceMetadataJobRepository.Add(ctx, descriptor.sourceMetadaID, job.ID)
                if err != nil {
                    return err
                }
                break
            }
        }
    }

    err = b.BackupRepository.UpdateLastScheduledTime(ctx, backup.ID, time.Now(), repository.Prepared)
    return err
}

func (b *BigQueryJobCreator) flattenTables(ctxIn context.Context, backup *repository.Backup) (flattenedTables []*bigquery.Table, err error) {
    ctx, span := trace.StartSpan(ctxIn, "(*BigQueryJobCreator).prepareMirrorJobs")
    defer span.End()

    var tablesToInspect []string

    // inspect only given tables
    if 0 < len(backup.BigQueryOptions.Table) {
        tablesToInspect = append(tablesToInspect, backup.BigQueryOptions.Table...)
    } else {
        // inspect whole dataset
        tablesInDataset, err := b.BigQuery.GetTablesInDataset(ctx, backup.SourceProject, backup.Dataset)

        if err != nil {
            return []*bigquery.Table{}, err
        }

        for _, t := range tablesInDataset {
            if len(backup.ExcludedTables) > 0 && containsTableWithName(t.Name, backup.ExcludedTables) {
                continue
            }
            tablesToInspect = append(tablesToInspect, t.Name)
        }
    }

    for _, t := range tablesToInspect {
        resultingTables, err := b.listBigQueryTable(ctx, backup, t)
        if err != nil {
            if e, ok := err.(*googleapi.Error); ok && e.Code == 404 {
                glog.Infof("404 Error: table with id %s not found", t)
                continue
            } else {
                return []*bigquery.Table{}, err
            }
        }
        flattenedTables = append(flattenedTables, resultingTables...)
    }

    return flattenedTables, err
}

func (b *BigQueryJobCreator) listBigQueryTable(ctxIn context.Context, backup *repository.Backup, table string) (tables []*bigquery.Table, err error) {
    ctx, span := trace.StartSpan(ctxIn, "(*BigQueryJobCreator).listBigQueryTable")
    defer span.End()

    dataset := backup.BigQueryOptions.Dataset

    if isSinglePartitionTable(table) {
        bqTable, err := b.BigQuery.GetTable(ctx, backup.SourceProject, dataset, table)
        if err != nil {
            return tables, err
        }
        if bqTable != nil {
            tables = append(tables, bqTable)
        }
        return tables, err
    }

    partitions, err := b.BigQuery.HasTablePartitions(ctx, backup.SourceProject, dataset, table)
    if err != nil {
        return nil, err
    }

    if !partitions {
        bqTable, err := b.BigQuery.GetTable(ctx, backup.SourceProject, dataset, table)
        if err != nil {
            return tables, err
        }
        if bqTable != nil {
            tables = append(tables, bqTable)
        }
        return tables, err
    }

    tablePartitions, err := b.BigQuery.GetTablePartitions(ctx, backup.SourceProject, dataset, table)
    if err != nil {
        return tables, err
    }
    tables = append(tables, tablePartitions...)

    return tables, nil
}

func (b *BigQueryJobCreator) collateState(ctxIn context.Context, backupID string, tables []*bigquery.Table) (descriptors []*jobDescriptor, err error) {
    ctx, span := trace.StartSpan(ctxIn, "(*BigQueryJobCreator).collateState")
    defer span.End()

    var toAdd, toUpdate, toDelete []*repository.SourceMetadata
    sourceMetadata, err := b.SourceMetadataRepository.GetLastByBackupID(ctx, backupID)
    if err != nil {
        return descriptors, err
    }

    for _, table := range tables {
        meta := b.getMetadataForTable(table.Name, sourceMetadata)
        if meta == nil {
            // table present and no meta: ADD
            toAdd = append(toAdd, &repository.SourceMetadata{BackupID: backupID, Source: table.Name, SourceChecksum: table.Checksum, Operation: repository.Add.String()})
            descriptors = append(descriptors, &jobDescriptor{backupID: backupID, table: table.Name})
            continue
        } else if meta.SourceChecksum == table.Checksum {
            // table and meta have same checksum: do nothing
            continue
        } else if meta.SourceChecksum != table.Checksum {
            // table and meta have different checksum: UPDATE
            toUpdate = append(toUpdate, &repository.SourceMetadata{BackupID: backupID, Source: table.Name, SourceChecksum: table.Checksum, Operation: repository.Update.String()})
            descriptors = append(descriptors, &jobDescriptor{backupID: backupID, table: table.Name})
            continue
        }
    }

    for _, meta := range sourceMetadata {
        table := b.getTableForMetadata(meta.Source, tables)
        if !repository.Delete.EqualTo(meta.Operation) && table == nil {
            // BigQuery table was deleted: DELETE
            toDelete = append(toDelete, &repository.SourceMetadata{BackupID: backupID, Source: meta.Source, SourceChecksum: meta.SourceChecksum, Operation: repository.Delete.String()})
        }
    }

    glog.Infof("preparing mirror jobs for backup with id %s. Source metadata FilteredTables %d ToAdd %d, ToUpdate %d, ToDelete %d",
        backupID, len(descriptors), len(toAdd), len(toUpdate), len(toDelete))

    var totalList []*repository.SourceMetadata
    totalList = append(totalList, toAdd...)
    totalList = append(totalList, toUpdate...)
    totalList = append(totalList, toDelete...)
    addedSourceMetadata, err := b.SourceMetadataRepository.Add(ctx, totalList)
    if err != nil {
        return descriptors, err
    }

    for _, descriptor := range descriptors {
        meta := b.getMetadataForTable(descriptor.table, addedSourceMetadata)
        if meta == nil {
            return descriptors, fmt.Errorf("got not expected added source metadata for backupID=%s and table=%s", backupID, descriptor.table)
        }

        descriptor.sourceMetadaID = meta.ID
    }

    return descriptors, err
}

func isSinglePartitionTable(table string) bool {
    return strings.Contains(table, "$")
}

type jobDescriptor struct {
    backupID       string
    table          string
    sourceMetadaID int
}

func (j *jobDescriptor) matchJob(job *repository.Job) bool {
    return job.BackupID == j.backupID && job.Source == j.table
}

func newJob(backupID string, table string) *repository.Job {
    return &repository.Job{
        ID:       generateNewID(),
        BackupID: backupID,
        Status:   repository.NotScheduled,
        Source:   table,
        Type:     repository.BigQuery,
    }
}

func (b *BigQueryJobCreator) getMetadataForTable(tableName string, sourceMetadata []*repository.SourceMetadata) *repository.SourceMetadata {
    for _, meta := range sourceMetadata {
        if meta.Source == tableName {
            return meta
        }
    }
    return nil
}

func (b *BigQueryJobCreator) getTableForMetadata(tableName string, tables []*bigquery.Table) *bigquery.Table {
    for _, table := range tables {
        if table.Name == tableName {
            return table
        }
    }
    return nil
}

func containsTableWithName(needle string, haystack []string) bool {
    for _, element := range haystack {
        if element == needle {
            return true
        }
    }
    return false
}
