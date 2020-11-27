package processor

import (
    "cloud.google.com/go/bigquery"
    "context"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/repository/memory"
    bq "github.com/ottogroup/penelope/pkg/service/bigquery"
    "github.com/ottogroup/penelope/pkg/service/gcs"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "regexp"
    "testing"
)

func TestBigQueryJobCreator_PrepareJobs_strategyNotExist(t *testing.T) {
    // Given
    ctx := context.Background()
    backup := newBigQueryMirrorBackup("strategyNotExist", "dataset", []string{})
    backup.Strategy = "not exist"
    testContext := givenATestContext()
    testContext.BigQuery.fDoesTableHasPartitions = false
    testContext.BackupRepository.AddBackup(ctx, backup)
    bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
    // When
    err := bigQueryJobCreator.PrepareJobs(ctx, backup)
    // Then
    if err == nil {
        t.Errorf("expected error after giving not existing strategy")
    }
}

func TestBigQueryJobCreator_PrepareJobs_Mirror_noTable(t *testing.T) {
    // Given
    ctx := context.Background()
    backup := newBigQueryMirrorBackup("noTable", "dataset", []string{})
    testContext := givenATestContext()
    testContext.BigQuery.fDoesTableHasPartitions = false
    testContext.BackupRepository.AddBackup(ctx, backup)
    bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
    // When
    err := bigQueryJobCreator.PrepareJobs(ctx, backup)
    // Then
    require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
}

func TestBigQueryJobCreator_PrepareJobs_Mirror_nonPartitionedTable_expectNewJob(t *testing.T) {
    // Given
    ctx := context.Background()
    backup := newBigQueryMirrorBackup("nonPartitionedTable_expectNewJob", "dataset", []string{})
    testContext := givenATestContext()
    testContext.BigQuery.fDoesTableHasPartitions = false
    testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "1", Checksum: "123"})
    testContext.BigQuery.fGetTable = &bq.Table{Name: "1", Checksum: "123"}

    testContext.BackupRepository.AddBackup(ctx, backup)
    bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
    // When
    err := bigQueryJobCreator.PrepareJobs(ctx, backup)
    require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
    // Then
    jobsForBackup, err := testContext.MemoryJobRepository.GetLastJobsForBackup(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
    assert.Equal(t, 1, len(jobsForBackup))
}

func TestBigQueryJobCreator_PrepareJobs_Mirror_nonPartitionedTable_expectNoNewJob(t *testing.T) {
    // Given
    ctx := context.Background()
    backup := newBigQueryMirrorBackup("nonPartitionedTable_expectNoNewJob", "dataset", []string{})
    testContext := givenATestContext()
    testContext.BigQuery.fDoesTableHasPartitions = false
    testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "table1", Checksum: "123"})
    testContext.SourceMetadataRepository.Add(ctx, []*repository.SourceMetadata{{BackupID: backup.ID, Source: "table1", SourceChecksum: "123"}})
    testContext.BackupRepository.AddBackup(ctx, backup)
    bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
    // When
    err := bigQueryJobCreator.PrepareJobs(ctx, backup)
    // Then
    require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
    jobsForBackup, err := testContext.MemoryJobRepository.GetLastJobsForBackup(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
    assert.Equal(t, 0, len(jobsForBackup))
}

func TestBigQueryJobCreator_PrepareJobs_Mirror_nonPartitionedTable_updateData(t *testing.T) {
    // Given
    ctx := context.Background()
    backup := newBigQueryMirrorBackup("nonPartitionedTable_updateData", "dataset", []string{})
    testContext := givenATestContext()
    testContext.BigQuery.fDoesTableHasPartitions = false
    testContext.BackupRepository.AddBackup(ctx, backup)
    testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "table_to_update", Checksum: "222"})
    testContext.BigQuery.fGetTable = &bq.Table{Name: "table_to_update", Checksum: "222"}
    testContext.SourceMetadataRepository.Add(ctx, []*repository.SourceMetadata{
        {BackupID: backup.ID, Source: "table_to_update", SourceChecksum: "old checksum"}},
    )
    bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)

    // When
    err := bigQueryJobCreator.PrepareJobs(ctx, backup)
    require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
    // Then
    jobsForBackup, err := testContext.MemoryJobRepository.GetLastJobsForBackup(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting last jobs for backup %s", backup.ID)
    assert.Equal(t, 1, len(jobsForBackup))

    sourceMetadataForBackup, err := testContext.SourceMetadataRepository.GetLastByBackupID(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
    assert.Equal(t, 2, len(sourceMetadataForBackup))

    totalElementsFound := 0
    for _, meta := range sourceMetadataForBackup {
        if meta.Source == "table_to_update" && meta.SourceChecksum == "222" {
            totalElementsFound++
        } else if meta.Source == "table_to_update" && meta.SourceChecksum == "old checksum" {
            totalElementsFound++
        }
    }
    assert.Equal(t, 2, totalElementsFound)
}

func TestBigQueryJobCreator_PrepareJobs_Mirror_nonPartitionedTable_removeData(t *testing.T) {
    // Given
    ctx := context.Background()
    backup := newBigQueryMirrorBackup("nonPartitionedTable", "dataset", []string{"t1", "t2"})
    testContext := givenATestContext()
    testContext.BigQuery.fDoesTableHasPartitions = false
    testContext.BackupRepository.AddBackup(ctx, backup)
    testContext.SourceMetadataRepository.Add(ctx, []*repository.SourceMetadata{
        {BackupID: backup.ID, Source: "table was removed", SourceChecksum: "n/a"}},
    )
    bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
    // When
    err := bigQueryJobCreator.PrepareJobs(ctx, backup)
    require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
    // Then
    jobsForBackup, err := testContext.MemoryJobRepository.GetLastJobsForBackup(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting last jobs for backup %s", backup.ID)
    assert.Equalf(t, 0, len(jobsForBackup), "expected no new job")

    sourceMetadataForBackup, err := testContext.SourceMetadataRepository.GetLastByBackupID(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
    assert.Equal(t, 2, len(sourceMetadataForBackup))
}

func TestBigQueryJobCreator_PrepareJobs_Mirror_partitionedTable_updateData(t *testing.T) {
    // Given
    ctx := context.Background()
    backup := newBigQueryMirrorBackup("partitionedTable_updateData", "dataset", []string{"partitioned_table$20190101"})
    testContext := givenATestContext()
    testContext.BigQuery.fDoesTableHasPartitions = false
    testContext.BigQuery.fGetTable = &bq.Table{Name: "partitioned_table$20190101", Checksum: "123"}
    testContext.BackupRepository.AddBackup(ctx, backup)
    bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
    // When
    err := bigQueryJobCreator.PrepareJobs(ctx, backup)
    require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
    // Then
    jobsForBackup, err := testContext.MemoryJobRepository.GetLastJobsForBackup(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting last jobs for backup %s", backup.ID)
    assert.Equal(t, 1, len(jobsForBackup))

    sourceMetadataForBackup, err := testContext.SourceMetadataRepository.GetLastByBackupID(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
    assert.Equal(t, 1, len(sourceMetadataForBackup))
}

func TestBigQueryJobCreator_PrepareJobs_Mirror_tablesWereDeletedOtherArePresentInDataset(t *testing.T) {
    // Given
    ctx := context.Background()
    backup := newBigQueryMirrorBackup("partitionedTable_updateData", "dataset", []string{"partitioned_table$20190101", "t2"})
    testContext := givenATestContext()
    testContext.BigQuery.fDoesTableHasPartitions = false
    testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "table not under backup #1", Checksum: "123"})
    testContext.BackupRepository.AddBackup(ctx, backup)
    bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
    // When
    err := bigQueryJobCreator.PrepareJobs(ctx, backup)
    require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)

    // Then
    jobsForBackup, err := testContext.MemoryJobRepository.GetLastJobsForBackup(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting last jobs for backup %s", backup.ID)
    assert.Equal(t, 0, len(jobsForBackup))

    sourceMetadataForBackup, err := testContext.SourceMetadataRepository.GetLastByBackupID(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
    assert.Equal(t, 0, len(sourceMetadataForBackup))
}

func TestBigQueryJobCreator_PrepareJobs_Mirror_partitionTables_expectChanges(t *testing.T) {
    // Given
    ctx := context.Background()
    testContext := givenATestContext()
    backup := newBigQueryMirrorBackup("partitionTables_expectChanges", "dataset", []string{})
    testContext.BackupRepository.AddBackup(ctx, backup)
    testContext.BigQuery.fDoesTableHasPartitions = true
    testContext.BigQuery.fGetTable = &bq.Table{Name: "partition", Checksum: "111"}
    testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "partition", Checksum: "000"})
    testContext.BigQuery.fGetTablePartitions = append(testContext.BigQuery.fGetTablePartitions, &bq.Table{Name: "partition$20190101", Checksum: "111"})
    testContext.BigQuery.fGetTablePartitions = append(testContext.BigQuery.fGetTablePartitions, &bq.Table{Name: "partition$20190102", Checksum: "222"})
    testContext.BigQuery.fGetTablePartitions = append(testContext.BigQuery.fGetTablePartitions, &bq.Table{Name: "partition$20190103", Checksum: "222"})
    testContext.SourceMetadataRepository.Add(ctx, []*repository.SourceMetadata{
        {BackupID: backup.ID, Source: "partition$20190101", SourceChecksum: "111"}, // do nothing
        {BackupID: backup.ID, Source: "partition$20190102", SourceChecksum: "111"}, // update
        {BackupID: backup.ID, Source: "partition$20190103", SourceChecksum: "111"}, // update
        {BackupID: backup.ID, Source: "partition$20190104", SourceChecksum: "111"}, // delete source
    })
    bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
    // When
    err := bigQueryJobCreator.PrepareJobs(ctx, backup)
    require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)

    // Then
    jobsForBackup, err := testContext.MemoryJobRepository.GetLastJobsForBackup(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting last jobs for backup %s", backup.ID)
    assert.Equal(t, 2, len(jobsForBackup))

    sourceMetadataForBackup, err := testContext.SourceMetadataRepository.GetLastByBackupID(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
    assert.Equal(t, 7, len(sourceMetadataForBackup))
}

func TestBigQueryJobCreator_PrepareJobs_Snapshot_nonPartitionTables_expectNewJob(t *testing.T) {
    // Given
    ctx := context.Background()
    testContext := givenATestContext()
    backup := newBigQuerySnapshotBackup("Snapshot_nonPartitionTables_expectChanges", "dataset", []string{})
    testContext.BackupRepository.AddBackup(ctx, backup)
    testContext.BigQuery.fDoesTableHasPartitions = false
    testContext.BigQuery.fGetTable = &bq.Table{Name: "non_partition", Checksum: "111"}
    testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "non_partition", Checksum: "000"})
    bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
    // When
    err := bigQueryJobCreator.PrepareJobs(ctx, backup)
    require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)

    // Then
    jobsForBackup, err := testContext.MemoryJobRepository.GetLastJobsForBackup(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
    assert.Equal(t, 1, len(jobsForBackup))
}

func TestBigQueryJobCreator_PrepareJobs_Snapshot_partitionTables_expectNewJobs(t *testing.T) {
    // Given
    ctx := context.Background()
    testContext := givenATestContext()
    backup := newBigQuerySnapshotBackup("Snapshot_partitionTables_expectNewJobs", "dataset", []string{})
    testContext.BackupRepository.AddBackup(ctx, backup)
    testContext.BigQuery.fDoesTableHasPartitions = true
    testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "partition", Checksum: "000"})
    testContext.BigQuery.fGetTable = &bq.Table{Name: "partition", Checksum: "111"}
    testContext.BigQuery.fGetTablePartitions = append(testContext.BigQuery.fGetTablePartitions, &bq.Table{Name: "partition$20190101", Checksum: "222"})
    testContext.BigQuery.fGetTablePartitions = append(testContext.BigQuery.fGetTablePartitions, &bq.Table{Name: "partition$20190102", Checksum: "222"})
    testContext.BigQuery.fGetTablePartitions = append(testContext.BigQuery.fGetTablePartitions, &bq.Table{Name: "partition$20190103", Checksum: "222"})
    testContext.BigQuery.fGetTablePartitions = append(testContext.BigQuery.fGetTablePartitions, &bq.Table{Name: "partition$20190104", Checksum: "222"})
    bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
    // When
    err := bigQueryJobCreator.PrepareJobs(ctx, backup)
    require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)

    // Then
    jobsForBackup, err := testContext.MemoryJobRepository.GetLastJobsForBackup(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
    assert.Equal(t, 4, len(jobsForBackup))
}

func TestBigQueryJobCreator_PrepareJobs_Snapshot_partitionTable_expectNewJobs(t *testing.T) {
    // Given
    ctx := context.Background()
    testContext := givenATestContext()
    backup := newBigQuerySnapshotBackup("Snapshot_partitionTables_expectNewJobs", "dataset", []string{"partition$20190101"})
    testContext.BackupRepository.AddBackup(ctx, backup)
    testContext.BigQuery.fDoesTableHasPartitions = false
    testContext.BigQuery.fGetTable = &bq.Table{Name: "partition$20190101", Checksum: "111"}
    bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
    // When
    err := bigQueryJobCreator.PrepareJobs(ctx, backup)
    require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)

    // Then
    jobsForBackup, err := testContext.MemoryJobRepository.GetLastJobsForBackup(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
    assert.Equal(t, 1, len(jobsForBackup))
}

func TestCloudStorageJobCreator_PrepareJobs_strategyNotExist(t *testing.T) {
    // Given
    ctx := context.Background()
    backup := newCloudStorageSnapshotBackup("strategyNotExist", "test-bucket")
    backup.Strategy = "not exist"
    testContext := givenACloudStorageJobTesttestContext()
    testContext.BackupRepository.AddBackup(ctx, backup)
    cloudStorageJobCreator := givenACloudStorageJobCreatorWithTestContext(testContext)
    // When
    err := cloudStorageJobCreator.PrepareJobs(ctx, backup)
    // Then
    require.Error(t, err, "expected error after giving not existing strategy")
}

func TestCloudStorageJobCreator_PrepareJobs_SimpleBucket(t *testing.T) {
    // Given
    ctx := context.Background()
    backup := newCloudStorageSnapshotBackup("strategyNotExist", "test-bucket")
    testContext := givenACloudStorageJobTesttestContext()
    testContext.BackupRepository.AddBackup(ctx, backup)
    cloudStorageJobCreator := givenACloudStorageJobCreatorWithTestContext(testContext)
    // When
    err := cloudStorageJobCreator.PrepareJobs(ctx, backup)
    require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)

    // Then
    jobsForBackup, err := testContext.MemoryJobRepository.GetLastJobsForBackup(ctx, backup.ID)
    require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
    assert.Equal(t, 1, len(jobsForBackup))
}

func givenABigQueryJobCreatorWithTestContext(ctx *testContextBigQueryJobCreator) *BigQueryJobCreator {
    return NewBigQueryJobCreator(context.Background(), ctx.BackupRepository, ctx.JobRepository, &ctx.BigQuery, ctx.SourceMetadataRepository, ctx.SourceMetadataJobRepository)
}

func givenACloudStorageJobCreatorWithTestContext(ctx *testContextCloudStorageJobCreator) *CloudStorageJobCreator {
    return NewCloudStorageJobCreator(
        context.Background(),
        ctx.BackupRepository,
        ctx.JobRepository,
    )
}

func givenATestContext() *testContextBigQueryJobCreator {
    backupRepository := memory.BackupRepository{}
    jobRepository := memory.JobRepository{}
    bigQueryClient := testBigQueryClient{}
    clientFactory := testGcsClientFactory{}
    clientFactory.CloudStorageClient = &stubGcsClient{}
    sourceMetadataRepository := memory.SourceMetadataRepository{}
    sourceMetadataJobRepository := memory.DefaultSourceMetadataJobRepository{}
    return &testContextBigQueryJobCreator{
        BackupRepository:            &backupRepository,
        JobRepository:               &jobRepository,
        MemoryJobRepository:         &jobRepository,
        SourceMetadataRepository:    &sourceMetadataRepository,
        SourceMetadataJobRepository: &sourceMetadataJobRepository,
        BigQuery:                    bigQueryClient,
    }
}

func givenACloudStorageJobTesttestContext() *testContextCloudStorageJobCreator {
    backupRepository := memory.BackupRepository{}
    jobRepositry := memory.JobRepository{}
    bigQueryClient := testBigQueryClient{}
    sourceMetadataRepository := memory.SourceMetadataRepository{}
    return &testContextCloudStorageJobCreator{
        BackupRepository:         &backupRepository,
        JobRepository:            &jobRepositry,
        MemoryJobRepository:      &jobRepositry,
        SourceMetadataRepository: &sourceMetadataRepository,
        BigQuery:                 bigQueryClient,
        CloudStorageClient:       &stubGcsClient{},
    }
}

type testContextBigQueryJobCreator struct {
    BackupRepository            repository.BackupRepository
    JobRepository               repository.JobRepository
    MemoryJobRepository         *memory.JobRepository
    BigQuery                    testBigQueryClient
    SourceMetadataRepository    repository.SourceMetadataRepository
    SourceMetadataJobRepository repository.SourceMetadataJobRepository
}

type testContextCloudStorageJobCreator struct {
    BackupRepository         repository.BackupRepository
    JobRepository            repository.JobRepository
    MemoryJobRepository      *memory.JobRepository
    SourceMetadataRepository repository.SourceMetadataRepository
    BigQuery                 testBigQueryClient
    CloudStorageClient       gcs.CloudStorageClient
}

type testBigQueryClient struct {
    fDoesTableExists        bool
    fDoesTableHasPartitions bool
    Err                     error
    fGetTablePartitions     []*bq.Table
    fGetTablesInDataset     []*bq.Table
    fGetTable               *bq.Table
    fGetTableErr            error
}

func (t *testBigQueryClient) GetDatasets(context.Context, string) ([]string, error) {
    panic("implement me")
}

func (t *testBigQueryClient) GetTablesInDataset(context.Context, string, string) ([]*bq.Table, error) {
    return t.fGetTablesInDataset, nil
}

func (t *testBigQueryClient) GetTable(context.Context, string, string, string) (*bq.Table, error) {
    return t.fGetTable, t.fGetTableErr
}

func (t *testBigQueryClient) GetTablePartitions(context.Context, string, string, string) ([]*bq.Table, error) {
    return t.fGetTablePartitions, t.Err
}

func (*testBigQueryClient) IsInitialized(context.Context) bool {
    panic("implement me")
}

func (*testBigQueryClient) ExtractTableToGcsAsAvro(c context.Context, dataset, table, gcsURI string) *bigquery.Extractor {
    panic("implement me")
}

func (*testBigQueryClient) GetExtractJobStatus(c context.Context, extractJobID string) (*bigquery.JobStatus, error) {
    panic("implement me")
}

func (*testBigQueryClient) DoesDatasetExists(c context.Context, project string, dataset string) (bool, error) {
    panic("implement me")
}

func (t *testBigQueryClient) DoesTableExists(c context.Context, project string, dataset string, table string) (bool, error) {
    return t.fDoesTableExists, nil
}

func (t *testBigQueryClient) HasTablePartitions(c context.Context, project string, dataset string, table string) (bool, error) {
    return t.fDoesTableHasPartitions, nil
}

type stubGcsClient struct {
    fDeleteObjectsErr error
}

func (g *stubGcsClient) Close(context.Context) {
    panic("implement me")
}

func (g *stubGcsClient) MoveObject(c context.Context, bucketName, oldObjectName, newObjectName string) error {
    panic("implement me")
}

func (g *stubGcsClient) CreateObject(c context.Context, bucketName, objectName, content string) error {
    panic("implement me")
}

func (g *stubGcsClient) DeleteObject(c context.Context, bucketName string, objectName string) error {
    panic("implement me")
}

func (g *stubGcsClient) GetBuckets(c context.Context, project string) ([]string, error) {
    panic("implement me")
}

func (g *stubGcsClient) BucketUsageInBytes(c context.Context, project string, bucket string) (float64, error) {
    panic("implement me")
}

func (g *stubGcsClient) DeleteObjectsWithObjectMatch(c context.Context, bucketName string, prefix string, objectPattern *regexp.Regexp) (deleted int, err error) {
    return 0, g.fDeleteObjectsErr
}

func (*stubGcsClient) IsInitialized(c context.Context) bool {
    panic("implement me")
}

func (*stubGcsClient) DoesBucketExist(c context.Context, project string, bucket string) (bool, error) {
    panic("implement me")
}

func (*stubGcsClient) CreateBucket(c context.Context, project, bucket, location, storageClass string, lifetimeInDays uint, archiveTTM uint) error {
    panic("implement me")
}

func (*stubGcsClient) UpdateBucket(ctxIn context.Context, bucket string, lifetimeInDays uint, archiveTTM uint) error {
    panic("implement me")
}

func (*stubGcsClient) DeleteBucket(c context.Context, bucket string) error {
    panic("implement me")
}

func (*stubGcsClient) ReadObject(c context.Context, bucketName, objectName string) ([]byte, error) {
    panic("implement me")
}

type testGcsClientFactory struct {
    CloudStorageClient gcs.CloudStorageClient
}

func (f *testGcsClientFactory) NewCloudStorageClient(c context.Context, targetProjectID string) (gcs.CloudStorageClient, error) {
    return f.CloudStorageClient, nil
}

func newBigQuerySnapshotBackup(backupID string, dataset string, tables []string) *repository.Backup {
    return &repository.Backup{ID: backupID, Strategy: repository.Snapshot, Type: repository.BigQuery,
        BackupOptions: repository.BackupOptions{BigQueryOptions: repository.BigQueryOptions{
            Dataset: dataset, Table: tables,
        }},
    }
}

func newBigQueryMirrorBackup(backupID string, dataset string, tables []string) *repository.Backup {
    return &repository.Backup{ID: backupID, Strategy: repository.Mirror, Type: repository.BigQuery,
        BackupOptions: repository.BackupOptions{BigQueryOptions: repository.BigQueryOptions{
            Dataset: dataset, Table: tables,
        }},
    }
}

func newCloudStorageSnapshotBackup(backupID string, bucket string) *repository.Backup {
    return &repository.Backup{ID: backupID, Strategy: repository.Snapshot, Type: repository.CloudStorage,
        BackupOptions: repository.BackupOptions{CloudStorageOptions: repository.CloudStorageOptions{
            Bucket: bucket,
        }},
    }
}
