package repository

import (
    "github.com/stretchr/testify/assert"
    "testing"
    "time"
)

func TestDefaultJobRepository_AddJob_Simple(t *testing.T) {
    const backupID = "backup-id-1202"
    const jobID = "job-id-1202"

    ctx, storageService := prepareTest(t)

    setBackupWithIDs(t, storageService, backupID)
    repository := &defaultJobRepository{storageService: storageService}

    err := repository.AddJob(ctx, &Job{
        ID: jobID,
        BackupID: backupID,
        Status: NotScheduled,
    })
    assert.NoError(t, err)

    count, err := storageService.DB().Model(&Job{}).Count()
    assert.NoError(t, err)
    assert.Equal(t, count, 1)
}

func TestDefaultJobRepository_AddJobs_Simple(t *testing.T) {
    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3"}
    jobs := []*Job{
        {ID: "job-id-1", BackupID: "backup-id-1", Status: NotScheduled},
        {ID: "job-id-2", BackupID: "backup-id-2", Status: NotScheduled},
        {ID: "job-id-3", BackupID: "backup-id-3", Status: NotScheduled},
    }

    ctx, storageService := prepareTest(t)

    setBackupWithIDs(t, storageService, backupIDs...)
    repository := &defaultJobRepository{storageService: storageService}


    err := repository.AddJobs(ctx, jobs)
    assert.NoError(t, err)

    count, err := storageService.DB().Model(&Job{}).Count()
    assert.NoError(t, err)
    assert.Equal(t, count, 3)
}

func TestDefaultJobRepository_DeleteJob_Simple(t *testing.T) {
    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3"}
    jobs := []*Job{
        {ID: "job-id-1", BackupID: "backup-id-1", Status: NotScheduled},
        {ID: "job-id-2", BackupID: "backup-id-2", Status: NotScheduled}, // Deleted
        {ID: "job-id-3", BackupID: "backup-id-3", Status: NotScheduled},
    }

    ctx, storageService := prepareTest(t)

    setBackupWithIDs(t, storageService, backupIDs...)
    repository := &defaultJobRepository{storageService: storageService}

    err := repository.AddJobs(ctx, jobs)
    assert.NoError(t, err)

    err = repository.DeleteJob(ctx, "job-id-2")
    assert.NoError(t, err)

    count, err := storageService.DB().Model(&Job{}).Count()
    assert.NoError(t, err)
    assert.Equal(t, count, 2)

    count, err = storageService.DB().Model(&Job{ID: "job-id-2"}).WherePK().Count()
    assert.NoError(t, err)
    assert.Equal(t, count, 0)
}

func TestDefaultJobRepository_GetJob_Simple(t *testing.T) {
    const backupID = "backup-id-1202"
    const jobID = "job-id-1202"

    ctx, storageService := prepareTest(t)

    setBackupWithIDs(t, storageService, backupID)
    repository := &defaultJobRepository{storageService: storageService}

    _ = repository.AddJob(ctx, &Job{
        ID: jobID,
        BackupID: backupID,
        Status: JobDeleted,
    })

    job, err := repository.GetJob(ctx, jobID)
    assert.NoError(t, err)
    assert.NotNil(t, job)
    assert.Equal(t, jobID, job.ID)
    assert.Equal(t, backupID, job.BackupID)
    assert.Equal(t, JobDeleted, job.Status)
}

func TestDefaultJobRepository_GetJob_DeletedJob(t *testing.T) {
    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3"}
    jobs := []*Job{
        {ID: "job-id-1", BackupID: "backup-id-1", Status: NotScheduled, Type: BigQuery},
        {ID: "job-id-2", BackupID: "backup-id-2", Status: NotScheduled, Type: CloudStorage}, // Deleted
        {
            ID: "job-id-3",
            BackupID: "backup-id-3",
            Status: JobDeleted,
            Type: BigQuery,
            EntityAudit: EntityAudit{
                DeletedTimestamp: time.Now(),
            },
        },
    }

    ctx, storageService := prepareTest(t)

    setBackupWithIDs(t, storageService, backupIDs...)
    repository := &defaultJobRepository{storageService: storageService}

    err := repository.AddJobs(ctx, jobs)
    assert.NoError(t, err)

    job, err := repository.GetJob(ctx, "job-id-3")
    assert.NoError(t, err)
    assert.Nil(t, job)
}

func TestDefaultJobRepository_MarkDeleted(t *testing.T) {
    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3"}
    jobs := []*Job{
        {ID: "job-id-1", BackupID: "backup-id-1", Status: NotScheduled, Type: BigQuery},
        {ID: "job-id-2", BackupID: "backup-id-2", Status: NotScheduled, Type: CloudStorage}, // Deleted
        {ID: "job-id-3", BackupID: "backup-id-3", Status: Scheduled, Type: BigQuery},
    }

    ctx, storageService := prepareTest(t)

    setBackupWithIDs(t, storageService, backupIDs...)
    repository := &defaultJobRepository{storageService: storageService}

    err := repository.AddJobs(ctx, jobs)
    assert.NoError(t, err)

    err = repository.MarkDeleted(ctx, "job-id-2")
    assert.NoError(t, err)

    count, err := repository.storageService.DB().Model(&Job{}).Where("status = ?", JobDeleted).Count()
    assert.NoError(t, err)
    assert.Equal(t, 1, count)

    err = repository.MarkDeleted(ctx, "job-id-1")
    assert.NoError(t, err)

    count, err = repository.storageService.DB().Model(&Job{}).Where("status = ?", JobDeleted).Count()
    assert.NoError(t, err)
    assert.Equal(t, 2, count)
}

func TestDefaultJobRepository_GetByJobTypeAndStatus(t *testing.T) {
    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3"}
    jobs := []*Job{
        {ID: "job-id-1", BackupID: "backup-id-1", Status: NotScheduled, Type: BigQuery},
        {ID: "job-id-2", BackupID: "backup-id-2", Status: NotScheduled, Type: CloudStorage}, // Deleted
        {ID: "job-id-3", BackupID: "backup-id-3", Status: Scheduled, Type: BigQuery},
    }

    ctx, storageService := prepareTest(t)

    setBackupWithIDs(t, storageService, backupIDs...)
    repository := &defaultJobRepository{storageService: storageService}

    err := repository.AddJobs(ctx, jobs)
    assert.NoError(t, err)

    status, err := repository.GetByJobTypeAndStatus(ctx, CloudStorage, NotScheduled)
    assert.NoError(t, err)
    assert.Len(t, status, 1)

    status, err = repository.GetByJobTypeAndStatus(ctx, BigQuery, Scheduled)
    assert.NoError(t, err)
    assert.Len(t, status, 1)

    status, err = repository.GetByJobTypeAndStatus(ctx, BigQuery, Scheduled, NotScheduled)
    assert.NoError(t, err)
    assert.Len(t, status, 2)

    status, err = repository.GetByJobTypeAndStatus(ctx, CloudStorage, Scheduled)
    assert.NoError(t, err)
    assert.Len(t, status, 0)
}

func TestDefaultJobRepository_GetJobsForBackupID(t *testing.T) {
    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3","backup-id-4","backup-id-5"}
    jobs := []*Job{
        {ID: "job-id-1", BackupID: "backup-id-1", Status: NotScheduled, Type: BigQuery},
        {ID: "job-id-2", BackupID: "backup-id-2", Status: NotScheduled, Type: CloudStorage},
        {ID: "job-id-3", BackupID: "backup-id-2", Status: NotScheduled, Type: CloudStorage},
        {ID: "job-id-4", BackupID: "backup-id-3", Status: Scheduled, Type: BigQuery},
        {ID: "job-id-5", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery},
        {ID: "job-id-6", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery},
        {ID: "job-id-7", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery},
        {ID: "job-id-8", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery},
    }

    ctx, storageService := prepareTest(t)

    setBackupWithIDs(t, storageService, backupIDs...)
    repository := &defaultJobRepository{storageService: storageService}

    err := repository.AddJobs(ctx, jobs)
    assert.NoError(t, err)

    page := JobPage{
        Size:   5,
        Number: 0,
    }

    jobsForBackupID, err := repository.GetJobsForBackupID(ctx, "backup-id-1", page)
    assert.NoError(t, err)
    assert.Len(t, jobsForBackupID, 1)

    jobsForBackupID, err = repository.GetJobsForBackupID(ctx, "backup-id-2", page)
    assert.NoError(t, err)
    assert.Len(t, jobsForBackupID, 2)

    jobsForBackupID, err = repository.GetJobsForBackupID(ctx, "backup-id-5", page)
    assert.NoError(t, err)
    assert.Len(t, jobsForBackupID, 0)
}

func TestDefaultJobRepository_GetJobsForBackupID_PageSize(t *testing.T) {
    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3","backup-id-4","backup-id-5"}
    jobs := []*Job{
        {ID: "job-id-1", BackupID: "backup-id-1", Status: NotScheduled, Type: BigQuery},
        {ID: "job-id-2", BackupID: "backup-id-2", Status: NotScheduled, Type: CloudStorage},
        {ID: "job-id-3", BackupID: "backup-id-2", Status: NotScheduled, Type: CloudStorage},
        {ID: "job-id-4", BackupID: "backup-id-3", Status: Scheduled, Type: BigQuery},
        {ID: "job-id-5", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery},
        {ID: "job-id-6", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery},
        {ID: "job-id-7", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery},
        {ID: "job-id-8", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery},
    }

    ctx, storageService := prepareTest(t)

    setBackupWithIDs(t, storageService, backupIDs...)
    repository := &defaultJobRepository{storageService: storageService}

    err := repository.AddJobs(ctx, jobs)
    assert.NoError(t, err)

    page := JobPage{
        Size:   3,
        Number: 0,
    }

    jobsForBackupID, err := repository.GetJobsForBackupID(ctx, "backup-id-4", page)
    assert.NoError(t, err)
    assert.Len(t, jobsForBackupID, 3)

    page = JobPage{
        Size:   3,
        Number: 1,
    }

    jobsForBackupID, err = repository.GetJobsForBackupID(ctx, "backup-id-4", page)
    assert.NoError(t, err)
    assert.Len(t, jobsForBackupID, 1)

    page = JobPage{
        Size:   3,
        Number: 2,
    }

    jobsForBackupID, err = repository.GetJobsForBackupID(ctx, "backup-id-4", page)
    assert.NoError(t, err)
    assert.Len(t, jobsForBackupID, 0)

    page = JobPage{
        Size:   AllJobs,
        Number: 0,
    }

    jobsForBackupID, err = repository.GetJobsForBackupID(ctx, "backup-id-4", page)
    assert.NoError(t, err)
    assert.Len(t, jobsForBackupID, 4)
}

func TestDefaultJobRepository_GetByStatusAndBefore_OnlyCreatedTimestamp(t *testing.T) {
    yesterday := time.Now().AddDate(0,0,-1)
    oneWeekBefore := time.Now().AddDate(0,0,-7)
    now := time.Now()

    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3","backup-id-4","backup-id-5"}
    jobs := []*Job{
        {ID: "job-id-1", BackupID: "backup-id-1", Status: NotScheduled, Type: BigQuery, EntityAudit: EntityAudit{CreatedTimestamp: yesterday}},
        {ID: "job-id-2", BackupID: "backup-id-2", Status: NotScheduled, Type: CloudStorage, EntityAudit: EntityAudit{CreatedTimestamp: oneWeekBefore}},
        {ID: "job-id-3", BackupID: "backup-id-2", Status: NotScheduled, Type: CloudStorage, EntityAudit: EntityAudit{CreatedTimestamp: yesterday}},
        {ID: "job-id-4", BackupID: "backup-id-3", Status: Scheduled, Type: BigQuery, EntityAudit: EntityAudit{CreatedTimestamp: yesterday}},
        {ID: "job-id-5", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery, EntityAudit: EntityAudit{CreatedTimestamp: oneWeekBefore}},
        {ID: "job-id-6", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery, EntityAudit: EntityAudit{CreatedTimestamp: yesterday}},
        {ID: "job-id-7", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery, EntityAudit: EntityAudit{CreatedTimestamp: yesterday}},
        {ID: "job-id-8", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery, EntityAudit: EntityAudit{CreatedTimestamp: now}},
    }

    ctx, storageService := prepareTest(t)

    setBackupWithIDs(t, storageService, backupIDs...)
    repository := &defaultJobRepository{storageService: storageService}

    err := repository.AddJobs(ctx, jobs)
    assert.NoError(t, err)

    before, err := repository.GetByStatusAndBefore(ctx, []JobStatus{Scheduled}, 25)
    assert.NoError(t, err)
    assert.Len(t, before, 1)
    assert.Equal(t, "job-id-5", before[0].ID)

    before, err = repository.GetByStatusAndBefore(ctx, []JobStatus{Scheduled, NotScheduled}, 25)
    assert.NoError(t, err)
    assert.Len(t, before, 2)
}

func TestDefaultJobRepository_GetByStatusAndBefore_WithUpdatedTimestamp(t *testing.T) {
    yesterday := time.Now().Add(-24*time.Hour)
    oneHour := time.Now().Add(-1*time.Hour)
    threeHour := time.Now().Add(-3*time.Hour)

    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3","backup-id-4","backup-id-5"}
    jobs := []*Job{
        // Expected
        {ID: "job-id-1", BackupID: "backup-id-1", Status: NotScheduled, Type: BigQuery, EntityAudit: EntityAudit{CreatedTimestamp: yesterday}},
        {ID: "job-id-2", BackupID: "backup-id-2", Status: NotScheduled, Type: CloudStorage, EntityAudit: EntityAudit{CreatedTimestamp: yesterday}},

        {ID: "job-id-3", BackupID: "backup-id-2", Status: NotScheduled, Type: CloudStorage, EntityAudit: EntityAudit{CreatedTimestamp: yesterday, UpdatedTimestamp: threeHour}},

        // Expected
        {ID: "job-id-4", BackupID: "backup-id-3", Status: Scheduled, Type: BigQuery, EntityAudit: EntityAudit{CreatedTimestamp: yesterday}},
        {ID: "job-id-6", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery, EntityAudit: EntityAudit{CreatedTimestamp: yesterday, UpdatedTimestamp: threeHour}},
        {ID: "job-id-7", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery, EntityAudit: EntityAudit{CreatedTimestamp: yesterday}},

        {ID: "job-id-5", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery, EntityAudit: EntityAudit{CreatedTimestamp: yesterday, UpdatedTimestamp: oneHour}},
        {ID: "job-id-8", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery, EntityAudit: EntityAudit{CreatedTimestamp: yesterday, UpdatedTimestamp: oneHour}},
        {ID: "job-id-9", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery},
    }

    ctx, storageService := prepareTest(t)

    setBackupWithIDs(t, storageService, backupIDs...)
    repository := &defaultJobRepository{storageService: storageService}

    err := repository.AddJobs(ctx, jobs)
    assert.NoError(t, err)

    before, err := repository.GetByStatusAndBefore(ctx, []JobStatus{NotScheduled}, 4)
    assert.NoError(t, err)
    assert.Len(t, before, 2)

    before, err = repository.GetByStatusAndBefore(ctx, []JobStatus{Scheduled}, 2)
    assert.NoError(t, err)
    assert.Len(t, before, 3)
}

func TestDefaultJobRepository_PatchJobStatus(t *testing.T) {
    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3"}
    jobs := []*Job{
        {ID: "job-id-1", BackupID: "backup-id-1", Status: NotScheduled},
        {ID: "job-id-2", BackupID: "backup-id-2", Status: NotScheduled},
        {ID: "job-id-3", BackupID: "backup-id-3", Status: NotScheduled},
    }

    ctx, storageService := prepareTest(t)

    setBackupWithIDs(t, storageService, backupIDs...)
    repository := &defaultJobRepository{storageService: storageService}


    err := repository.AddJobs(ctx, jobs)
    assert.NoError(t, err)

    const bigQueryID = "bigquery-id-100"
    const cloudStorageID = "cloudstorage-id-101"

    err = repository.PatchJobStatus(ctx, JobPatch{
        ID:     "job-id-2",
        Status: Scheduled,
        ForeignJobID: ForeignJobID{
            BigQueryID:     bigQueryID,
            CloudStorageID: cloudStorageID,
        },
    })
    assert.NoError(t, err)

    job, err := repository.GetJob(ctx, "job-id-2")
    assert.NoError(t, err)
    assert.Equal(t, Scheduled, job.Status)
    assert.Equal(t, bigQueryID, job.BigQueryID.String())
    assert.Equal(t, cloudStorageID, job.CloudStorageID.String())

    count, _ := storageService.DB().
        Model(&Job{}).
        Where("status = ?", Scheduled).
        Count()
    assert.Equal(t, 1, count)

    err = repository.PatchJobStatus(ctx, JobPatch{
        ID:     "job-id-4",
        Status: Scheduled,
        ForeignJobID: ForeignJobID{
            BigQueryID:     bigQueryID,
            CloudStorageID: cloudStorageID,
        },
    })
    assert.Nil(t, err)
}

func TestDefaultJobRepository_GetStatisticsForBackupID(t *testing.T) {
    backupIDs := []string{"backup-id-1","backup-id-2","backup-id-3","backup-id-4","backup-id-5"}
    jobs := []*Job{
        {ID: "job-id-2", BackupID: "backup-id-2", Status: NotScheduled, Type: CloudStorage},
        {ID: "job-id-3", BackupID: "backup-id-2", Status: NotScheduled, Type: CloudStorage},

        {ID: "job-id-5", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery},
        {ID: "job-id-6", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery},
        {ID: "job-id-7", BackupID: "backup-id-4", Status: Scheduled, Type: BigQuery},

        {ID: "job-id-8", BackupID: "backup-id-4", Status: NotScheduled, Type: BigQuery},

        {ID: "job-id-9", BackupID: "backup-id-4", Status: Error, Type: BigQuery},

        {ID: "job-id-10", BackupID: "backup-id-4", Status: FinishedOk, Type: BigQuery},
        {ID: "job-id-11", BackupID: "backup-id-4", Status: FinishedOk, Type: BigQuery},
    }

    ctx, storageService := prepareTest(t)

    setBackupWithIDs(t, storageService, backupIDs...)
    repository := &defaultJobRepository{storageService: storageService}

    err := repository.AddJobs(ctx, jobs)
    assert.NoError(t, err)

    jobStatistics, err := repository.GetStatisticsForBackupID(ctx, "backup-id-4")
    assert.NoError(t, err)
    assert.Len(t, jobStatistics, 4)
    assert.Equal(t, 3, int(jobStatistics[Scheduled]))
    assert.Equal(t, 1, int(jobStatistics[NotScheduled]))
    assert.Equal(t, 1, int(jobStatistics[Error]))
    assert.Equal(t, 2, int(jobStatistics[FinishedOk]))

    jobStatistics, err = repository.GetStatisticsForBackupID(ctx, "backup-id-notfound")
    assert.NoError(t, err)
    assert.Len(t, jobStatistics, 0)

    jobStatistics, err = repository.GetStatisticsForBackupID(ctx, "backup-id-2")
    assert.NoError(t, err)
    assert.Len(t, jobStatistics, 1)
    assert.Equal(t, 2, int(jobStatistics[NotScheduled]))
}

func TestDefaultJobRepository_GetBackupRestoreJobs(t *testing.T) {
    backups := []Backup{
        {ID: "backup-id-1"},
    }
    jobs := []Job{
        {ID: "job-id-1", BackupID: "backup-id-1", Status: Scheduled, Type: BigQuery, Source: "amount_budget_plan", EntityAudit: EntityAudit{CreatedTimestamp: time.Now()}},
        {ID: "job-id-2", BackupID: "backup-id-1", Status: Scheduled, Type: BigQuery, Source: "amount_budget_plan", EntityAudit: EntityAudit{CreatedTimestamp: time.Now()}},
    }
    metadata := []SourceMetadata{
        {ID: 1, BackupID: "backup-id-1", Source: "partition$20190102", SourceChecksum: "111", Operation: Add.String(), CreatedTimestamp: time.Now().AddDate(0,0,-1)},
        {ID: 2, BackupID: "backup-id-1", Source: "partition$20190101", SourceChecksum: "111", Operation: Delete.String(), CreatedTimestamp: time.Now().AddDate(0,0,-1)},
    }
    metadataJobs := []SourceMetadataJob{
        {SourceMetadataID: 1, JobId: "job-id-1"},
        {SourceMetadataID: 2, JobId: "job-id-2"},
    }

    ctx, storageService := prepareTest(t)

    repository := &defaultJobRepository{storageService: storageService}

    err := setDatabase(storageService, backups, jobs, metadata, metadataJobs)
    assert.NoError(t, err)

    restoreJobs, err := repository.GetBackupRestoreJobs(ctx, "backup-id-1", "job-id-1")
    assert.NoError(t, err)
    assert.Len(t, restoreJobs, 1)
}
