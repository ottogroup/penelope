package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/ottogroup/penelope/pkg/service"
	"go.opencensus.io/trace"
)

// JobStatistics for a job
type JobStatistics map[JobStatus]uint64

// AllJobs will fetch all jobs
const AllJobs = -101

// Page represent what subset of rows to fetch
type Page struct {
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
	GetByJobTypeAndStatusAndLimit(context.Context, BackupType, JobStatus, uint) ([]*Job, error)
	GetByJobTypeAndStatus(context.Context, BackupType, ...JobStatus) ([]*Job, error)
	GetByStatusAndBefore(context.Context, []JobStatus, int) ([]*Job, error)
	PatchJobStatus(ctx context.Context, patch JobPatch) error
	GetJobsForBackupID(ctx context.Context, backupID string, jobPage Page, status ...JobStatus) ([]*Job, error)
	GetMostRecentJobForBackupID(ctxIn context.Context, backupID string, status ...JobStatus) (*Job, error)
	GetBackupRestoreJobs(ctx context.Context, backupID, jobID string) ([]*Job, error)
	GetStatisticsForBackupID(ctx context.Context, backupID string) (JobStatistics, error)
	GetJobCountForBackupID(ctx context.Context, backupID string) (int, error)
	GetRecoverableJobCountForBackupID(ctx context.Context, backupID string) (int, error)
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
func (d *defaultJobRepository) GetJobsForBackupID(ctxIn context.Context, backupID string, jobPage Page, status ...JobStatus) ([]*Job, error) {
	_, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).GetJobsForBackupID")
	defer span.End()

	var jobs []*Job
	db := d.storageService.DB()

	query := db.Model(&jobs).Where("backup_id = ?", backupID).Order("audit_created_timestamp DESC")
	if jobPage.Size != AllJobs {
		offset := jobPage.Number * jobPage.Size
		query = query.Offset(offset).Limit(jobPage.Size)
	}
	if len(status) > 0 {
		query = query.Where("status in (?)", pg.In(status))
	}
	err := query.Select()
	if err != nil {
		return nil, fmt.Errorf("error during executing GetJobsForBackupID statement: %s", err)
	}
	return jobs, err
}

func (d *defaultJobRepository) GetMostRecentJobForBackupID(ctxIn context.Context, backupID string, status ...JobStatus) (*Job, error) {
	_, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).GetMostRecentJobForBackupID")
	defer span.End()

	var jobs []*Job
	db := d.storageService.DB()

	query := db.Model(&jobs).
		Where("backup_id = ?", backupID).
		Order("audit_created_timestamp DESC").
		Order("audit_updated_timestamp DESC").
		Where("status in (?)", pg.In(status)).
		Where("audit_deleted_timestamp IS NULL").
		Limit(1)

	err := query.Select()
	if err != nil {
		return nil, fmt.Errorf("error during executing GetMostRecentJobForBackupID statement: %s", err)
	}
	if len(jobs) == 0 {
		return nil, err
	}
	return jobs[0], err
}

// GetBackupRestoreJobs get restore jobs for a backup
func (d *defaultJobRepository) GetBackupRestoreJobs(ctxIn context.Context, backupID, jobID string) ([]*Job, error) {
	_, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).GetBackupRestoreJobs")
	defer span.End()

	var jobs []*Job
	db := d.storageService.DB()

	backup := new(Backup)
	err := db.Model(backup).Where("id = ?", backupID).Select()
	if err != nil {
		logQueryError("GetBackup", err)
		return nil, err
	}

	// For snapshot jobs
	if backup.Strategy == Snapshot {
		// Get the reference job's timestamp based on whether jobID is provided
		var referenceTimestampQuery *orm.Query
		if jobID == "" {
			// When jobId is empty, get the most recent FinishedOk job's timestamp
			referenceTimestampQuery = db.ModelContext(ctxIn, (*Job)(nil)).
				Column("audit_created_timestamp").
				Where("backup_id = ?", backupID).
				Where("status = ?", FinishedOk).
				Order("audit_created_timestamp DESC").
				Limit(1)
		} else {
			// When jobId is provided, use its timestamp
			referenceTimestampQuery = db.ModelContext(ctxIn, (*Job)(nil)).
				Column("audit_created_timestamp").
				Where("id = ?", jobID)
		}

		// Create the job_latest CTE
		jobLatestQuery := db.ModelContext(ctxIn, (*Job)(nil)).
			Column("source").
			ColumnExpr("MAX(id) AS id").
			Where("backup_id = ?", backupID).
			Where("audit_created_timestamp <= (?)", referenceTimestampQuery).
			Group("source")

		// Main query with CTE and JOIN
		err = db.ModelContext(ctxIn).
			With("job_latest", jobLatestQuery).
			TableExpr("job_latest jl").
			Column("j.*").
			Join("INNER JOIN jobs j USING (id)").
			Where("j.audit_deleted_timestamp IS NULL").
			Select(&jobs)

		if err != nil {
			return jobs, fmt.Errorf("error during executing GetBackupRestoreJobs statement for snapshot: %s", err)
		}
	} else if backup.Strategy == Mirror {
		// To determine the jobs to restore for a backup we use a restore point which is defined by a jobID.
		// If jobID is empty, we take the most recent FinishedOk job for the backup as restore point.
		// If jobID is provided, we use it directly as restore point.
		// Based on the restore point we determine the source_metadata entries which are relevant for the restore.
		// We do this by first get the source_metadata associated with the job via a source_metadata_jobs lookup and selecting all preceding source_metadata entries for each source.
		// For each source of the selected source_metadata we take the latest source_metadata.
		// Finally, we get all jobs which are linked to those source_metadata entries and where the operation is not "Delete" so there is actually something to restore.

		// Create the job_of_interest CTE based on whether jobID is provided
		var jobOfInterestQuery *orm.Query
		if jobID == "" {
			// When jobId is empty, get the most recent FinishedOk job
			jobOfInterestQuery = db.ModelContext(ctxIn, (*Job)(nil)).
				Column("id").
				Where("backup_id = ?", backupID).
				Where("status = ?", FinishedOk).
				Order("id DESC").
				Limit(1)
		} else {
			// When jobId is provided, use it directly
			jobOfInterestQuery = db.ModelContext(ctxIn).ColumnExpr("? as id", jobID)
		}

		// Create the source_metadata_jobs to source_metadata CTE
		smjSmQuery := db.ModelContext(ctxIn).
			TableExpr("source_metadata_jobs smj").
			Column("sm.id", "sm.source").
			Join("JOIN source_metadata sm ON smj.source_metadata_id >= sm.id").
			Where("sm.backup_id = ?", backupID).
			Where("smj.job_id IN (?)", jobOfInterestQuery)

		// Create the latest source_metadata CTE
		smLatestQuery := db.ModelContext(ctxIn).
			TableExpr("smj_sm").
			ColumnExpr("MAX(smj_sm.id) AS id").
			Group("smj_sm.source")

		err = db.ModelContext(ctxIn).
			With("smj_sm", smjSmQuery).
			With("sm_latest", smLatestQuery).
			TableExpr("sm_latest sml").
			Column("j.*").
			Join("JOIN source_metadata sm ON sml.id = sm.id").
			Join("JOIN source_metadata_jobs smj ON smj.source_metadata_id = sml.id").
			Join("JOIN jobs j ON smj.job_id = j.id").
			Where("sm.operation != ?", "Delete").        // remove deleted table/partition when the last operation was delete
			Where("sm.audit_deleted_timestamp IS NULL"). // only keep recoverable entries
			Select(&jobs)

		if err != nil {
			return jobs, fmt.Errorf("error during executing GetBackupRestoreJobs statement: %s", err)
		}
	}

	return jobs, nil
}

// GetByJobTypeAndStatusAndLimit filter backup jobs by status and type with limit
func (d *defaultJobRepository) GetByJobTypeAndStatusAndLimit(ctxIn context.Context, backupType BackupType, status JobStatus, limit uint) ([]*Job, error) {
	_, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).GetByJobTypeAndStatusAndLimit")
	defer span.End()

	var jobs []*Job
	db := d.storageService.DB()

	err := db.Model(&jobs).
		Where("type = ?", backupType.String()).
		Where("audit_deleted_timestamp is null").
		Where("status in (?)", pg.In(status)).
		Limit(int(limit)).
		Select()

	if err != nil {
		return jobs, fmt.Errorf("error during executing get job by status statement: %s", err)
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
			BigQueryID:     jobPatcher.ForeignJobID.BigQueryID,
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
		ID:     id,
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

// GetJobCountForBackupID counts total job count for a backup
func (d *defaultJobRepository) GetJobCountForBackupID(ctxIn context.Context, backupID string) (int, error) {
	_, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).GetJobCountForBackupID")
	defer span.End()

	var result int

	err := d.storageService.DB().
		Model((*Job)(nil)).
		ColumnExpr("count(*) AS count").
		Where("backup_id = ?", backupID).
		Select(&result)

	if err != nil {
		return -1, err
	}

	return result, nil
}

// GetRecoverableJobCountForBackupID counts total recoverable job count for a backup
func (d *defaultJobRepository) GetRecoverableJobCountForBackupID(ctxIn context.Context, backupID string) (int, error) {
	_, span := trace.StartSpan(ctxIn, "(*defaultJobRepository).GetRecoverableJobCountForBackupID")
	defer span.End()

	var result int

	err := d.storageService.DB().
		Model((*Job)(nil)).
		ColumnExpr("count(*) AS count").
		Where("backup_id = ?", backupID).
		Where("status = ?", FinishedOk).
		Where("audit_deleted_timestamp IS NULL").
		Select(&result)

	if err != nil {
		return -1, err
	}

	return result, nil
}
