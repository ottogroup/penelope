package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/ottogroup/penelope/pkg/repository"
	"go.opencensus.io/trace"
)

// JobRepository is a client to a backup job
type JobRepository struct {
	jobs []*repository.Job
}

func (r *JobRepository) GetMostRecentJobForBackupID(ctxIn context.Context, backupID string, status ...repository.JobStatus) (*repository.Job, error) {
	return nil, nil
}

// GetStatisticsForBackupID prepare stats for a backup
func (r *JobRepository) GetStatisticsForBackupID(ctxIn context.Context, backupID string) (repository.JobStatistics, error) {
	_, span := trace.StartSpan(ctxIn, "(*JobRepository).GetStatisticsForBackupID")
	defer span.End()

	jobStatistics := make(repository.JobStatistics)
	for _, job := range r.jobs {
		jobStatistics[job.Status]++
	}
	return jobStatistics, nil
}

// GetBackupRestoreJobs is not implemented
func (r *JobRepository) GetBackupRestoreJobs(ctxIn context.Context, backupID, jobID string) ([]*repository.Job, error) {
	_, span := trace.StartSpan(ctxIn, "(*JobRepository).GetBackupRestoreJobs")
	defer span.End()

	panic("implement me")
}

// GetByJobTypeAndStatusAndLimit filter backup jobs by status and type with limit
func (r *JobRepository) ListByTypeAndStatusWithLimit(ctxIn context.Context, backupType repository.BackupType, jobStatus repository.JobStatus, limit uint) (jobs []*repository.Job, err error) {
	_, span := trace.StartSpan(ctxIn, "(*JobRepository).ListByTypeAndStatusWithLimit")
	defer span.End()
	for _, j := range r.jobs {
		if j.Type == backupType && j.Status == jobStatus {
			jobs = append(jobs, j)
		}
	}
	if len(jobs) == 0 {
		return jobs, err
	} else if uint(len(jobs)) < limit {
		limit = uint(len(jobs))
	}
	return jobs[:limit], err
}

// GetByJobTypeAndStatus filter backup jobs by status and type
func (r *JobRepository) GetByJobTypeAndStatus(ctxIn context.Context, backupType repository.BackupType, jobStatus ...repository.JobStatus) (jobs []*repository.Job, err error) {
	_, span := trace.StartSpan(ctxIn, "(*JobRepository).GetByJobTypeAndStatus")
	defer span.End()

	for _, status := range jobStatus {
		for _, j := range r.jobs {
			if j.Type == backupType && j.Status == status {
				jobs = append(jobs, j)
			}
		}
	}
	return jobs, err
}

func (r *JobRepository) GetByBackupIdAndSourceAndStatus(ctx context.Context, backupId string, source string, status ...repository.JobStatus) (rs []*repository.Job, err error) {
	for _, job := range r.jobs {
		if job.BackupID == backupId && job.Source == source {
			for _, jobStatus := range status {
				if job.Status == jobStatus {
					rs = append(rs, job)
					break
				}
			}
		}
	}
	return
}

// GetByStatusAndBefore is not implemented
func (r *JobRepository) GetByStatusAndBefore(ctxIn context.Context, status []repository.JobStatus, deltaHours int) ([]*repository.Job, error) {
	_, span := trace.StartSpan(ctxIn, "(*JobRepository).GetByStatusAndBefore")
	defer span.End()

	panic("implement me")
}

// GetJobsForBackupID get all backup jobs
func (r *JobRepository) GetJobsForBackupID(ctxIn context.Context, backupID string, jobPage repository.Page, status ...repository.JobStatus) (jobs []*repository.Job, err error) {
	_, span := trace.StartSpan(ctxIn, "(*JobRepository).GetJobsForBackupID")
	defer span.End()

	for _, j := range r.jobs {
		if backupID == j.BackupID {
			jobs = append(jobs, j)
		}
	}
	return jobs, err
}

// AddJob add new backup job
func (r *JobRepository) AddJob(ctxIn context.Context, job *repository.Job) error {
	ctx, span := trace.StartSpan(ctxIn, "(*JobRepository).AddJob")
	defer span.End()

	j, _ := r.GetJob(ctx, job.ID)
	if j != nil {
		return fmt.Errorf("job already exist %s", job.ID)
	}
	r.jobs = append(r.jobs, job)
	return nil
}

// AddJobs add new backup jobs
func (r *JobRepository) AddJobs(ctxIn context.Context, jobs []*repository.Job) error {
	ctx, span := trace.StartSpan(ctxIn, "(*JobRepository).AddJobs")
	defer span.End()

	for _, input := range jobs {
		err := r.AddJob(ctx, input)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteJob remove job
func (r *JobRepository) DeleteJob(ctxIn context.Context, jobID string) error {
	ctx, span := trace.StartSpan(ctxIn, "(*JobRepository).DeleteJob")
	defer span.End()

	_, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}
	for i, j := range r.jobs {
		if jobID == j.ID {
			r.jobs[i] = r.jobs[len(r.jobs)-1] // Replace it with the last one.
			r.jobs = r.jobs[:len(r.jobs)-1]   // Chop off the last one.
			break
		}
	}
	return nil
}

// GetJob get backup job details
func (r *JobRepository) GetJob(ctxIn context.Context, jobID string) (*repository.Job, error) {
	_, span := trace.StartSpan(ctxIn, "(*JobRepository).GetJob")
	defer span.End()

	for _, j := range r.jobs {
		if jobID == j.ID {
			return j, nil
		}
	}
	return nil, fmt.Errorf("job not found %s", jobID)
}

// MarkDeleted mark BigQuery job as deleted
func (r *JobRepository) MarkDeleted(ctxIn context.Context, jobID string) error {
	ctx, span := trace.StartSpan(ctxIn, "(*JobRepository).MarkDeleted")
	defer span.End()

	j, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}
	j.DeletedTimestamp = time.Now().UTC()
	j.Status = repository.JobDeleted
	return nil
}

// ListNotScheduledJobsForBackup get backup jos that weren't scheduled
func (r *JobRepository) ListNotScheduledJobsForBackup(ctxIn context.Context, backupID string) (jobs []*repository.Job, err error) {
	_, span := trace.StartSpan(ctxIn, "(*JobRepository).ListNotScheduledJobsForBackup")
	defer span.End()

	for _, j := range r.jobs {
		if j.BackupID == backupID && j.Status == repository.NotScheduled {
			jobs = append(jobs, j)
		}
	}
	return jobs, err
}

// PatchJobStatus change job status
func (r *JobRepository) PatchJobStatus(ctxIn context.Context, patch repository.JobPatch) error {
	ctx, span := trace.StartSpan(ctxIn, "(*JobRepository).PatchJobStatus")
	defer span.End()

	j, err := r.GetJob(ctx, patch.ID)
	if err != nil {
		return err
	}
	j.Status = patch.Status
	j.CloudStorageID = patch.CloudStorageID
	j.BigQueryID = patch.BigQueryID
	j.ForeignJobID = patch.ForeignJobID
	return nil
}

func (r *JobRepository) GetJobCountForBackupID(ctxIn context.Context, backupID string) (int, error) {
	_, span := trace.StartSpan(ctxIn, "(*JobRepository).GetJobCountForBackupID")
	defer span.End()

	panic("implement me")

}

func (r *JobRepository) GetRecoverableJobCountForBackupID(ctxIn context.Context, backupID string) (int, error) {
	_, span := trace.StartSpan(ctxIn, "(*JobRepository).GetRecoverableJobCountForBackupID")
	defer span.End()

	panic("implement me")
}
