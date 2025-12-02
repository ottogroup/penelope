package processor

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"cloud.google.com/go/iam"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/repository/memory"
	bq "github.com/ottogroup/penelope/pkg/service/bigquery"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBigQueryJobCreator_PrepareJobs_strategyNotExist(t *testing.T) {
	// Given
	ctx := context.Background()
	backup := newBigQueryMirrorBackup("strategyNotExist", "dataset", []string{})
	backup.Strategy = "not exist"
	testContext := givenATestContext()
	testContext.BigQuery.fDoesDatasetExists = true
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
	testContext.BigQuery.fDoesDatasetExists = true
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
	testContext.BigQuery.fDoesDatasetExists = true
	testContext.BigQuery.fDoesTableHasPartitions = false
	testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "1", Checksum: "123"})
	testContext.BigQuery.fGetTable = &bq.Table{Name: "1", Checksum: "123"}

	testContext.BackupRepository.AddBackup(ctx, backup)
	bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
	// When
	err := bigQueryJobCreator.PrepareJobs(ctx, backup)
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
	// Then
	jobsForBackup, err := testContext.MemoryJobRepository.ListNotScheduledJobsForBackup(ctx, backup.ID)
	require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
	assert.Equal(t, 1, len(jobsForBackup))
}

func TestBigQueryJobCreator_PrepareJobs_Mirror_nonPartitionedTable_expectNoNewJob(t *testing.T) {
	// Given
	ctx := context.Background()
	backup := newBigQueryMirrorBackup("nonPartitionedTable_expectNoNewJob", "dataset", []string{})
	testContext := givenATestContext()
	testContext.BigQuery.fDoesDatasetExists = true
	testContext.BigQuery.fDoesTableHasPartitions = false
	testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "table1", Checksum: "123"})
	testContext.SourceMetadataRepository.Add(ctx, []*repository.SourceMetadata{{BackupID: backup.ID, Source: "table1", SourceChecksum: "123"}})
	testContext.BackupRepository.AddBackup(ctx, backup)
	bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
	// When
	err := bigQueryJobCreator.PrepareJobs(ctx, backup)
	// Then
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
	jobsForBackup, err := testContext.MemoryJobRepository.ListNotScheduledJobsForBackup(ctx, backup.ID)
	require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
	assert.Equal(t, 0, len(jobsForBackup))
}

func TestBigQueryJobCreator_PrepareJobs_Mirror_nonPartitionedTable_updateData(t *testing.T) {
	// Given
	ctx := context.Background()
	backup := newBigQueryMirrorBackup("nonPartitionedTable_updateData", "dataset", []string{})
	testContext := givenATestContext()
	testContext.BigQuery.fDoesDatasetExists = true
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
	jobsForBackup, err := testContext.MemoryJobRepository.ListNotScheduledJobsForBackup(ctx, backup.ID)
	require.NoErrorf(t, err, "should getting last jobs for backup %s", backup.ID)
	assert.Equal(t, 1, len(jobsForBackup))

	sourceMetadataForBackup, err := testContext.SourceMetadataRepository.GetLastByBackupID(ctx, backup.ID)
	require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
	assert.Equal(t, 1, len(sourceMetadataForBackup))
	assert.Equal(t, "222", sourceMetadataForBackup[0].SourceChecksum)
}

func TestBigQueryJobCreator_PrepareJobs_Mirror_nonPartitionedTable_removeData(t *testing.T) {
	// Given
	ctx := context.Background()
	backup := newBigQueryMirrorBackup("nonPartitionedTable", "dataset", []string{"t1", "t2"})
	testContext := givenATestContext()
	testContext.BigQuery.fDoesDatasetExists = true
	testContext.BigQuery.fDoesTableHasPartitions = false
	b, err := testContext.BackupRepository.AddBackup(ctx, backup)
	assert.NoError(t, err)
	assert.NotNil(t, b)
	sm, err := testContext.SourceMetadataRepository.Add(ctx, []*repository.SourceMetadata{
		{BackupID: backup.ID, Source: "table was removed", SourceChecksum: "n/a"}},
	)
	assert.NoError(t, err)
	assert.NotNil(t, sm)
	bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
	// When
	err = bigQueryJobCreator.PrepareJobs(ctx, backup)
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
	// Then
	// ctxIn context.Context, backupID string, jobPage repository.Page, status ...repository.JobStatus
	jobsForBackup, err := testContext.MemoryJobRepository.ListNotScheduledJobsForBackup(ctx, backup.ID)
	require.NoErrorf(t, err, "should getting last jobs for backup %s", backup.ID)
	assert.Equalf(t, 0, len(jobsForBackup), "expected no new job")

	sourceMetadataForBackup, err := testContext.SourceMetadataRepository.GetLastByBackupID(ctx, backup.ID)
	require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
	require.Equal(t, 1, len(sourceMetadataForBackup))
	assert.True(t, repository.Delete.EqualTo(sourceMetadataForBackup[0].Operation))
}

func TestBigQueryJobCreator_PrepareJobs_Mirror_partitionedTable_updateData(t *testing.T) {
	// Given
	ctx := context.Background()
	backup := newBigQueryMirrorBackup("partitionedTable_updateData", "dataset", []string{"partitioned_table$20190101"})
	testContext := givenATestContext()
	testContext.BigQuery.fDoesDatasetExists = true
	testContext.BigQuery.fDoesTableHasPartitions = false
	testContext.BigQuery.fGetTable = &bq.Table{Name: "partitioned_table$20190101", Checksum: "123"}
	testContext.BackupRepository.AddBackup(ctx, backup)
	bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
	// When
	err := bigQueryJobCreator.PrepareJobs(ctx, backup)
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
	// Then
	jobsForBackup, err := testContext.MemoryJobRepository.ListNotScheduledJobsForBackup(ctx, backup.ID)
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
	testContext.BigQuery.fDoesDatasetExists = true
	testContext.BigQuery.fDoesTableHasPartitions = false
	testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "table not under backup #1", Checksum: "123"})
	testContext.BackupRepository.AddBackup(ctx, backup)
	bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
	// When
	err := bigQueryJobCreator.PrepareJobs(ctx, backup)
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)

	// Then
	jobsForBackup, err := testContext.MemoryJobRepository.ListNotScheduledJobsForBackup(ctx, backup.ID)
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
	_, err := testContext.BackupRepository.AddBackup(ctx, backup)
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
	testContext.BigQuery.fDoesDatasetExists = true
	testContext.BigQuery.fDoesTableHasPartitions = true
	testContext.BigQuery.fGetTable = &bq.Table{Name: "partition", Checksum: "111"}
	testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "partition", Checksum: "000"})
	testContext.BigQuery.fGetTablePartitions = []*bq.Table{
		{Name: "partition$20190101", Checksum: "111"},
		{Name: "partition$20190102", Checksum: "222"},
		{Name: "partition$20190103", Checksum: "222"},
	}
	_, err = testContext.SourceMetadataRepository.Add(ctx, []*repository.SourceMetadata{
		{BackupID: backup.ID, Source: "partition$20190101", SourceChecksum: "111", Operation: repository.Add.String()}, // do nothing
		{BackupID: backup.ID, Source: "partition$20190102", SourceChecksum: "111", Operation: repository.Add.String()}, // update
		{BackupID: backup.ID, Source: "partition$20190103", SourceChecksum: "111", Operation: repository.Add.String()}, // update
		{BackupID: backup.ID, Source: "partition$20190104", SourceChecksum: "111", Operation: repository.Add.String()}, // delete source
	})
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
	bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
	// When
	err = bigQueryJobCreator.PrepareJobs(ctx, backup)
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)

	// Then
	jobsForBackup, err := testContext.MemoryJobRepository.ListNotScheduledJobsForBackup(ctx, backup.ID)
	require.NoErrorf(t, err, "should getting last jobs for backup %s", backup.ID)
	assert.Equal(t, 2, len(jobsForBackup))

	sourceMetadataForBackup, err := testContext.SourceMetadataRepository.GetLastByBackupID(ctx, backup.ID)
	require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
	require.Equal(t, 4, len(sourceMetadataForBackup))
	assert.True(t, repository.Add.EqualTo(findMetadataBySource(sourceMetadataForBackup, "partition$20190101").Operation))
	assert.True(t, repository.Update.EqualTo(findMetadataBySource(sourceMetadataForBackup, "partition$20190102").Operation))
	assert.True(t, repository.Update.EqualTo(findMetadataBySource(sourceMetadataForBackup, "partition$20190103").Operation))
	assert.True(t, repository.Delete.EqualTo(findMetadataBySource(sourceMetadataForBackup, "partition$20190104").Operation))
}

func TestBigQueryJobCreator_PrepareJobs_Snapshot_partitionTables_withJobsThatAreAlreadyScheduled_expectChanges(t *testing.T) {
	// Given
	ctx := context.Background()
	testContext := givenATestContext()
	backup := newBigQuerySnapshotBackup("partitionTables_expectChanges_snapshot", "dataset", []string{})
	_, err := testContext.BackupRepository.AddBackup(ctx, backup)
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
	testContext.BigQuery.fDoesDatasetExists = true
	testContext.BigQuery.fDoesTableHasPartitions = true
	testContext.BigQuery.fGetTable = &bq.Table{Name: "partition", Checksum: "111"}
	testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "partition", Checksum: "000"})
	testContext.BigQuery.fGetTablePartitions = []*bq.Table{
		{Name: "partition$20190101", Checksum: "111"}, // add
		{Name: "partition$20190102", Checksum: "111"}, // do nothing - job exist with status not scheduled
		{Name: "partition$20190103", Checksum: "111"}, // do nothing - job exist with status quota error
		{Name: "partition$20190104", Checksum: "111"}, // add
	}
	err = testContext.MemoryJobRepository.AddJobs(ctx, []*repository.Job{
		{BackupID: backup.ID, Source: "partition$20190102", ID: "existing-job-1", Type: repository.BigQuery, Status: repository.NotScheduled},
		{BackupID: backup.ID, Source: "partition$20190103", ID: "existing-job-2", Type: repository.BigQuery, Status: repository.FinishedQuotaError},
	})
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
	bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
	// When
	err = bigQueryJobCreator.PrepareJobs(ctx, backup)
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)

	// Then
	jobsForBackup, err := testContext.MemoryJobRepository.ListNotScheduledJobsForBackup(ctx, backup.ID)
	require.NoErrorf(t, err, "should getting last jobs for backup %s", backup.ID)
	assert.Equal(t, 3, len(jobsForBackup))
}

func TestBigQueryJobCreator_PrepareJobs_partitionTables_Mirror_withJobsThatAreAlreadyScheduled_expectChanges(t *testing.T) {
	// Given
	ctx := context.Background()
	testContext := givenATestContext()
	backup := newBigQueryMirrorBackup("partitionTables_expectChanges_mirror", "dataset", []string{})
	_, err := testContext.BackupRepository.AddBackup(ctx, backup)
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
	testContext.BigQuery.fDoesDatasetExists = true
	testContext.BigQuery.fDoesTableHasPartitions = true
	testContext.BigQuery.fGetTable = &bq.Table{Name: "partition", Checksum: "111"}
	testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "partition", Checksum: "000"})
	currentSourceMetadata, err := testContext.SourceMetadataRepository.Add(ctx, []*repository.SourceMetadata{
		{BackupID: backup.ID, Source: "partition$20190101", SourceChecksum: "111", Operation: repository.Add.String()},
		{BackupID: backup.ID, Source: "partition$20190102", SourceChecksum: "111", Operation: repository.Add.String()},
		{BackupID: backup.ID, Source: "partition$20190103", SourceChecksum: "111", Operation: repository.Add.String()},
		{BackupID: backup.ID, Source: "partition$20190104", SourceChecksum: "111", Operation: repository.Add.String()},
	})
	assert.Equal(t, 4, len(currentSourceMetadata))
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
	testContext.BigQuery.fGetTablePartitions = []*bq.Table{
		{Name: "partition$20190101", Checksum: "111"}, // same checksum - do nothing
		{Name: "partition$20190102", Checksum: "111"}, // same checksum - do nothing - job exist with status not scheduled
		{Name: "partition$20190103", Checksum: "111"}, // same checksum - do nothing - job exist with status quota error
		{Name: "partition$20190104", Checksum: "222"}, // new checksum - but job exist with status not scheduled - so new job
		{Name: "partition$20190105", Checksum: "111"}, // new partition - add
	}
	err = testContext.MemoryJobRepository.AddJobs(ctx, []*repository.Job{
		{BackupID: backup.ID, Source: "partition$20190101", ID: "existing-job-0", Type: repository.BigQuery, Status: repository.FinishedOk},
		{BackupID: backup.ID, Source: "partition$20190102", ID: "existing-job-1", Type: repository.BigQuery, Status: repository.NotScheduled},
		{BackupID: backup.ID, Source: "partition$20190103", ID: "existing-job-2", Type: repository.BigQuery, Status: repository.FinishedQuotaError},
		{BackupID: backup.ID, Source: "partition$20190104", ID: "existing-job-3", Type: repository.BigQuery, Status: repository.NotScheduled},
	})
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)
	for _, entry := range []struct {
		jobID      int
		metadataID string
	}{
		{jobID: currentSourceMetadata[0].ID, metadataID: "existing-job-0"},
		{jobID: currentSourceMetadata[1].ID, metadataID: "existing-job-1"},
		{jobID: currentSourceMetadata[2].ID, metadataID: "existing-job-2"},
		{jobID: currentSourceMetadata[3].ID, metadataID: "existing-job-3"},
	} {
		err = testContext.SourceMetadataJobRepository.Add(ctx, entry.jobID, entry.metadataID)
		require.NoErrorf(t, err, "should add connection between jobs and source_metadata backup %s", backup.ID)
	}

	bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
	// When
	err = bigQueryJobCreator.PrepareJobs(ctx, backup)
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)

	// Then
	jobsForBackup, err := testContext.MemoryJobRepository.ListNotScheduledJobsForBackup(ctx, backup.ID)
	require.NoErrorf(t, err, "should getting last jobs for backup %s", backup.ID)
	assert.Equal(t, 4, len(jobsForBackup))

	sourceMetadataForBackup, err := testContext.SourceMetadataRepository.GetLastByBackupID(ctx, backup.ID)
	require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
	assert.Equal(t, 5, len(sourceMetadataForBackup))

	rs, ok := testContext.SourceMetadataRepository.(*memory.SourceMetadataRepository)
	require.True(t, ok, "expected SourceMetadataRepository to be of type memory.SourceMetadataRepository")
	// check that partition$20190104 is marked as deleted
	var found bool
	for _, sourceMetadata := range rs.SourceMetadatas {
		if sourceMetadata.Source == "partition$20190104" && sourceMetadata.SourceChecksum == "111" {
			assert.False(t, sourceMetadata.DeletedTimestamp.IsZero())
			found = true
			break
		}
	}
	assert.True(t, found, "expected to find source metadata entry for partition$20190104 that is marked as deleted")

	// we let previously scheduled jobs partition$20190104 to run even if the checksum changed
	// otherwise we would need to delete rows from repository what is not a good idea for the backup solution
	assert.Equal(t, 2, countUniqueJobsForSource(jobsForBackup, "partition$20190104"))
	assert.Equal(t, 1, countUniqueJobsForSource(jobsForBackup, "partition$20190105"))

	smjr, ok := testContext.SourceMetadataJobRepository.(*memory.DefaultSourceMetadataJobRepository)
	require.True(t, ok, "expected SourceMetadataJobRepository to be of type memory.DefaultSourceMetadataJobRepository")

	var newJobs []*repository.Job
	for _, job := range jobsForBackup {
		if !strings.HasPrefix(job.ID, "existing-job-") {
			newJobs = append(newJobs, job)
		}
	}
	for _, job := range newJobs {
		var foundJob bool
		var foundSourceMetadataForJob bool
	outer:
		for _, sourceMetadataJob := range smjr.SourceMetadataJobs {
			if sourceMetadataJob.JobID == job.ID {
				foundJob = true
				for _, sourceMetadata := range rs.SourceMetadatas {
					if sourceMetadata.ID == sourceMetadataJob.SourceMetadataID {
						foundSourceMetadataForJob = true
						break outer
					}
				}
			}
		}
		assert.True(t, foundJob, "expected to find connection between job %s and source metadata", job.ID)
		assert.True(t, foundSourceMetadataForJob, "expected to find source metadata for job %s", job.ID)
	}
}

func countUniqueJobsForSource(jobs []*repository.Job, source string) int {
	jobsFound := make(map[string]int)
	for _, job := range jobs {
		if job.Source == source {
			jobsFound[job.ID]++
		}
	}
	var rs int
	for _, count := range jobsFound {
		if count == 1 {
			rs++
		}
	}
	return rs
}

func findMetadataBySource(metadata []*repository.SourceMetadata, source string) *repository.SourceMetadata {
	for _, sm := range metadata {
		if sm.Source == source {
			return sm
		}
	}
	return nil
}

func TestBigQueryJobCreator_PrepareJobs_Snapshot_nonPartitionTables_expectNewJob(t *testing.T) {
	// Given
	ctx := context.Background()
	testContext := givenATestContext()
	backup := newBigQuerySnapshotBackup("Snapshot_nonPartitionTables_expectChanges", "dataset", []string{})
	testContext.BackupRepository.AddBackup(ctx, backup)
	testContext.BigQuery.fDoesDatasetExists = true
	testContext.BigQuery.fDoesTableHasPartitions = false
	testContext.BigQuery.fGetTable = &bq.Table{Name: "non_partition", Checksum: "111"}
	testContext.BigQuery.fGetTablesInDataset = append(testContext.BigQuery.fGetTablesInDataset, &bq.Table{Name: "non_partition", Checksum: "000"})
	bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
	// When
	err := bigQueryJobCreator.PrepareJobs(ctx, backup)
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)

	// Then
	jobsForBackup, err := testContext.MemoryJobRepository.ListNotScheduledJobsForBackup(ctx, backup.ID)
	require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
	assert.Equal(t, 1, len(jobsForBackup))
}

func TestBigQueryJobCreator_PrepareJobs_Snapshot_partitionTables_expectNewJobs(t *testing.T) {
	// Given
	ctx := context.Background()
	testContext := givenATestContext()
	backup := newBigQuerySnapshotBackup("Snapshot_partitionTables_expectNewJobs", "dataset", []string{})
	testContext.BackupRepository.AddBackup(ctx, backup)
	testContext.BigQuery.fDoesDatasetExists = true
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
	jobsForBackup, err := testContext.MemoryJobRepository.ListNotScheduledJobsForBackup(ctx, backup.ID)
	require.NoErrorf(t, err, "should getting sourceMetadata for backup %s", backup.ID)
	assert.Equal(t, 4, len(jobsForBackup))
}

func TestBigQueryJobCreator_PrepareJobs_Snapshot_partitionTable_expectNewJobs(t *testing.T) {
	// Given
	ctx := context.Background()
	testContext := givenATestContext()
	backup := newBigQuerySnapshotBackup("Snapshot_partitionTables_expectNewJobs", "dataset", []string{"partition$20190101"})
	testContext.BackupRepository.AddBackup(ctx, backup)
	testContext.BigQuery.fDoesDatasetExists = true
	testContext.BigQuery.fDoesTableHasPartitions = false
	testContext.BigQuery.fGetTable = &bq.Table{Name: "partition$20190101", Checksum: "111"}
	bigQueryJobCreator := givenABigQueryJobCreatorWithTestContext(testContext)
	// When
	err := bigQueryJobCreator.PrepareJobs(ctx, backup)
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)

	// Then
	jobsForBackup, err := testContext.MemoryJobRepository.ListNotScheduledJobsForBackup(ctx, backup.ID)
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
	testContext.BigQuery.fDoesDatasetExists = true
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
	testContext.BigQuery.fDoesDatasetExists = true
	cloudStorageJobCreator := givenACloudStorageJobCreatorWithTestContext(testContext)
	// When
	err := cloudStorageJobCreator.PrepareJobs(ctx, backup)
	require.NoErrorf(t, err, "should prepare jobs for backup %s", backup.ID)

	// Then
	jobsForBackup, err := testContext.MemoryJobRepository.ListNotScheduledJobsForBackup(ctx, backup.ID)
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
		ctx.CloudStorageClient,
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
	fDoesDatasetExists      bool
	fDoesTableExists        bool
	fDoesTableHasPartitions bool
	Err                     error
	fGetTablePartitions     []*bq.Table
	fGetTablesInDataset     []*bq.Table
	fGetTable               *bq.Table
	fGetTableErr            error
}

func (t *testBigQueryClient) DeleteExtractJob(ctxIn context.Context, extractJobID repository.ExtractJobID) error {
	panic("implement me")
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

func (*testBigQueryClient) GetExtractJobStatus(c context.Context, extractJobID repository.ExtractJobID) (*bigquery.JobStatus, error) {
	panic("implement me")
}

func (t *testBigQueryClient) DoesDatasetExists(c context.Context, project string, dataset string) (bool, error) {
	return t.fDoesDatasetExists, nil
}

func (t *testBigQueryClient) DoesTableExists(c context.Context, project string, dataset string, table string) (bool, error) {
	return t.fDoesTableExists, nil
}

func (t *testBigQueryClient) HasTablePartitions(c context.Context, project string, dataset string, table string) (bool, error) {
	return t.fDoesTableHasPartitions, nil
}

func (t *testBigQueryClient) GetDatasetDetails(ctxIn context.Context, project string, dataset string) (*bigquery.DatasetMetadata, error) {
	panic("implement me")
}

type stubGcsClient struct {
	fDeleteObjectsErr error
}

func (g *stubGcsClient) DeleteObjectWithPrefix(ctxIn context.Context, bucket string, objectPrefixName string) error {
	panic("implement me")
}

func (g *stubGcsClient) GetProject(ctxIn context.Context, projectID string) (*resourcemanagerpb.Project, error) {
	panic("implement me")
}

func (g *stubGcsClient) SetBucketIAMPolicy(ctxIn context.Context, bucket string, policy *iam.Policy) error {
	panic("implement me")
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

func (g *stubGcsClient) DoesBucketExist(c context.Context, project string, bucket string) (bool, error) {
	return true, nil
}

func (*stubGcsClient) CreateBucket(c context.Context, bucket gcs.CloudStorageBucket) error {
	panic("implement me")
}

func (*stubGcsClient) UpdateBucket(ctxIn context.Context, bucket string, lifetimeInDays uint, archiveTTM uint, labels gcs.LabelsProvider) error {
	panic("implement me")
}

func (*stubGcsClient) DeleteBucket(c context.Context, bucket string) error {
	panic("implement me")
}

func (*stubGcsClient) ReadObject(c context.Context, bucketName, objectName string) ([]byte, error) {
	panic("implement me")
}

func (*stubGcsClient) GetBucketDetails(ctxIn context.Context, bucket string) (*storage.BucketAttrs, error) {
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
