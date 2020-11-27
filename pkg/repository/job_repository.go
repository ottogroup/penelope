package repository

import (
    "context"
    "fmt"
    "github.com/go-pg/pg/v10"
    "github.com/go-pg/pg/v10/orm"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/service"
    "go.opencensus.io/trace"
    "time"
)

// JobStatistics for a job
type JobStatistics map[JobStatus]uint64

// AllJobs will fetch all jobs
const AllJobs = -101

// JobPage represent what subset of jobs to fetch
type JobPage struct {
    // Size how many elements to fetch, value AllJobs will fetch all
    Size   int
    Number int
}

// JobRepository defines operation with backup job
type JobRepository interface {
    AddJob(context.Context, *Job) error
    AddJobs(ctxIn context.Context, jobs []*Job) error
    DeleteJob(context.Context, string) error
    GetJob(context.Context, string) (*Job, error)
    MarkDeleted(context.Context, string) error
    GetByJobTypeAndStatus(context.Context, BackupType, ...JobStatus) ([]*Job, error)
    GetByStatusAndBefore(context.Context, []JobStatus, int) ([]*Job, error)
    PatchJobStatus(ctx context.Context, patch JobPatch) error
    GetJobsForBackupID(ctx context.Context, backupID string, jobPage JobPage) ([]*Job, error)
    GetBackupRestoreJobs(ctx context.Context, backupID, jobID string) ([]*Job, error)
    GetStatisticsForBackupID(ctx context.Context, backupID string) (JobStatistics, error)
}

// defaultJobRepository implements JobRepository
type defaultJobRepository struct {
    storageService *service.Service
}

// NewJobRepository create new instance of JobRepository
func NewJobRepository(ctxIn context.Context, credentialsProvider secret.SecretProvider) (JobRepository, error) {
    ctx, span := trace.StartSpan(ctxIn, "NewJobRepository")
    defer span.End()

    storageService, err := service.NewStorageService(ctx, credentialsProvider)
    if err != nil {
        return nil, err
    }
    return &defaultJobRepository{storageService: storageService}, nil
}

// AddJob add new backup job
func (d *defaultJobRepository) AddJob(ctxIn context.Context, job *Job) error {
    _, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).AddJob")
    defer span.End()

    _, err := d.storageService.DB().Model(job).Insert()
    if err != nil {
        return fmt.Errorf("error during executing add job statement: %s", err)
    }
    return nil
}

// AddJobs add new backup jobs
func (d *defaultJobRepository) AddJobs(ctxIn context.Context, jobs []*Job) error {
    ctx, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).AddJobs")
    defer span.End()

    return d.storageService.DB().RunInTransaction(ctx, func(tx *pg.Tx) error {
        for _, job := range jobs {
            _, err := tx.Model(job).Insert()
            if err != nil {
                return err
            }
        }
        return nil
    })
}

// DeleteJob remove job
func (d *defaultJobRepository) DeleteJob(ctxIn context.Context, jobID string) error {
    _, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).DeleteJob")
    defer span.End()

    _, err := d.storageService.DB().Model(&Job{ID: jobID}).WherePK().Delete()
    if err != nil {
        return fmt.Errorf("delete job with id %s failed: %s", jobID, err)
    }
    return nil
}

// GetJob get backup job details
func (d *defaultJobRepository) GetJob(ctxIn context.Context, jobID string) (*Job, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).GetJob")
    defer span.End()

    job := &Job{ID: jobID}
    err := d.storageService.DB().
        Model(job).
        Where("id = ?", jobID).
        Where("audit_deleted_timestamp is null").
        Select()

    if err == pg.ErrNoRows {
        return nil, nil
    } else if err != nil {
        return nil, fmt.Errorf("get job with id %s failed: %s", jobID, err)
    }

    return job, err
}

// GetJobsForBackupID get all backup jobs
func (d *defaultJobRepository) GetJobsForBackupID(ctxIn context.Context, backupID string, jobPage JobPage) ([]*Job, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).GetJobsForBackupID")
    defer span.End()

    var jobs []*Job
    db := d.storageService.DB()
    
    query := db.Model(&jobs).Where("backup_id = ?", backupID)
    if jobPage.Size != AllJobs {
        offset := jobPage.Number * jobPage.Size
        query = query.Offset(offset).Limit(jobPage.Size)
    }
    err := query.Select()
    if err != nil {
        return nil, fmt.Errorf("error during executing GetJobsForBackupID statement: %s", err)
    }
    return jobs, err
}

// GetBackupRestoreJobs get restore jobs for a backup
func (d *defaultJobRepository) GetBackupRestoreJobs(ctxIn context.Context, backupID, jobID string) ([]*Job, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).GetBackupRestoreJobs")
	defer span.End()
	
	var jobs []*Job 
	db := d.storageService.DB()

	subselect := db.Model().
		Table("source_metadata").
		Column("*").
		ColumnExpr("rank() over (partition by source order by audit_created_timestamp desc) as inner_rank").
		Where("backup_id = ?", backupID)


	if jobID != "" {
		auditCreatedTimestamp := db.Model(&Job{}).Column("audit_created_timestamp").Where("id = ?", jobID)
		subselect = subselect.Where("to_timestamp(to_char(audit_created_timestamp, 'YYYY-MM-DD HH24:MI'), 'YYYY-MM-DD HH24:MI') <= (?)", auditCreatedTimestamp)
	} else {
		subselect = subselect.Where("to_timestamp(to_char(audit_created_timestamp, 'YYYY-MM-DD HH24:MI'), 'YYYY-MM-DD HH24:MI') <= NOW()")
	}

    err := db.Model().TableExpr("(?) AS s", subselect).
        Column("j.*").
        Join("LEFT JOIN source_metadata_jobs as smj on s.id=smj.source_metadata_id").
        Join("LEFT JOIN jobs j on smj.job_id = j.id").
        Where("inner_rank = 1 AND j.backup_id is NOT NULL").
        Where("s.operation != 'Delete'").
		Select(&jobs)

	if err != nil {
		return jobs, fmt.Errorf("error during executing GetBackupRestoreJobs statement: %s", err)
	}

	return jobs, nil
}

// GetByJobTypeAndStatus filter backup jobs by status and type
func (d *defaultJobRepository) GetByJobTypeAndStatus(ctxIn context.Context, backupType BackupType, status ...JobStatus) ([]*Job, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).GetByJobTypeAndStatus")
    defer span.End()

    var jobs []*Job
    db := d.storageService.DB()
    
    err := db.Model(&jobs).
        Where("type = ?", backupType.String()).
        Where("audit_deleted_timestamp is null").
        Where("status in (?)", pg.In(status)).
        Select()

    if err != nil {
        return jobs, fmt.Errorf("error during executing get job by status statement: %s", err)
    }

    return jobs, nil
}

// GetByStatusAndBefore get job by status and before given time
func (d *defaultJobRepository) GetByStatusAndBefore(ctxIn context.Context, status []JobStatus, deltaHours int) ([]*Job, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).GetByStatusAndBefore")
    defer span.End()

    var jobs []*Job 
    db := d.storageService.DB()

    err := db.Model(&jobs).
        Where("audit_deleted_timestamp is null").
        WhereGroup(func(sub *orm.Query) (*orm.Query, error) {
            return sub.
                WhereGroup(func(sub *orm.Query) (*orm.Query, error) {
                    return sub.
                        Where("audit_updated_timestamp is null").
                        Where("audit_created_timestamp < NOW()-interval '1 hour'*?", deltaHours), nil
                }).
                WhereOrGroup(func(sub *orm.Query) (*orm.Query, error) {
                    return sub.
                        Where("audit_updated_timestamp is not null").
                        Where("audit_updated_timestamp < NOW()-interval '1 hour'*?", deltaHours), nil
                }), nil
        }).
        Where("status in (?)", pg.In(status)).
        Select()

    if err != nil {
        return jobs, fmt.Errorf("error during executing get job by status statement: %s", err)
    }

    return jobs, nil
}

// PatchJobStatus change job status
func (d *defaultJobRepository) PatchJobStatus(ctxIn context.Context, jobPatcher JobPatch) error {
    _, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).PatchJobStatus")
    defer span.End()

    job := &Job{
        Status: jobPatcher.Status,
        ForeignJobID: ForeignJobID{
            BigQueryID: jobPatcher.ForeignJobID.BigQueryID,
            CloudStorageID: jobPatcher.ForeignJobID.CloudStorageID,
        },
        EntityAudit: EntityAudit{
            UpdatedTimestamp: time.Now(),
        },
    }

    _, err := d.storageService.DB().Model(job).
        Column("status", "audit_updated_timestamp", "bigquery_extract_job_id", "cloudstorage_transfer_job_id").
        Where("audit_deleted_timestamp IS NULL").
        Where("id = ?", jobPatcher.ID).
        Update()

    if err != nil {
        return fmt.Errorf("error during executing updating job statement: %s", err)
    }

    return nil
}

// MarkDeleted mark BigQuery job as deleted
func (d *defaultJobRepository) MarkDeleted(ctxIn context.Context, id string) error {
    _, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).MarkDeleted")
    defer span.End()

    job := &Job{
        ID: id,
        Status: JobDeleted,
        EntityAudit: EntityAudit{
            UpdatedTimestamp: time.Now(),
            DeletedTimestamp: time.Now(),
        },
    }

    _, err := d.storageService.DB().
        Model(job).
        Column("status", "audit_updated_timestamp", "audit_deleted_timestamp").
        WherePK().
        Where("audit_deleted_timestamp IS NULL").
        Update()

    if err != nil {
        return fmt.Errorf("error during executing updating job statemant: %s", err)
    }

    return nil
}

// GetStatisticsForBackupID prepare stats for a backup
func (d *defaultJobRepository) GetStatisticsForBackupID(ctxIn context.Context, backupID string) (JobStatistics, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).GetStatisticsForBackupID")
    defer span.End()

    var results []struct {
        Count  uint64
        Status JobStatus
    }

    err := d.storageService.DB().
        Model((*Job)(nil)).
        ColumnExpr("count(*) AS count").
        Column("status").
        Where("backup_id = ?", backupID).
        Group("status").
        Select(&results)

    if err != nil {
        return nil, err
    }

    jobStatistics := make(JobStatistics)
    for _, result := range results {
        jobStatistics[result.Status] = result.Count
    }
    return jobStatistics, nil
}

