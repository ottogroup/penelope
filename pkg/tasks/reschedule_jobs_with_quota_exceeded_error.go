package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/secret"
	"go.opentelemetry.io/otel"
)

const (
	pacificTimeLocation = "America/Los_Angeles"
	// bigQueryBatchLimit is the maximum number of jobs to fetch from the database in one batch. BigQuery allows up to 100k jobs to be scheduled on a daily basis
	bigQueryBatchLimit = 10_000
)

type rescheduleJobsWithQuotaErrorService struct {
	jobRepository       repository.JobRepository
	pacificTimeLocation *time.Location
}

func newRescheduleJobsWithQuotaError(ctxIn context.Context, credentialsProvider secret.SecretProvider) (*rescheduleJobsWithQuotaErrorService, error) {
	ctx, span := otel.Tracer("").Start(ctxIn, "newRescheduleJobsWithQuotaError")
	defer span.End()

	jobRepository, err := repository.NewJobRepository(ctx, credentialsProvider)
	if err != nil {
		return &rescheduleJobsWithQuotaErrorService{}, fmt.Errorf("could not instantiate new JobRepository: %s", err)
	}
	timeLocation, err := time.LoadLocation(pacificTimeLocation)
	if err != nil {
		return &rescheduleJobsWithQuotaErrorService{}, fmt.Errorf("could not load location for %s: %s", pacificTimeLocation, err)
	}

	return &rescheduleJobsWithQuotaErrorService{jobRepository: jobRepository, pacificTimeLocation: timeLocation}, nil
}

func (r *rescheduleJobsWithQuotaErrorService) Run(ctxIn context.Context) {
	ctx, span := otel.Tracer("").Start(ctxIn, "(*rescheduleJobsWithQuotaErrorService).Run")
	defer span.End()

	glog.Infof("[START] Reschedule Jobs With Quota Error")
	jobs, err := r.jobRepository.ListByTypeAndStatusWithLimit(ctx, repository.BigQuery, repository.FinishedQuotaError, bigQueryBatchLimit)
	if err != nil {
		glog.Infof("[FAIL] GetByJobTypeAndStatus failed: %s", err)
		return
	}
	if len(jobs) == 0 {
		glog.Infof("[SUCCESS] No jobs has Quota exceeded")
		return
	}

	glog.Infof("Rescheduling %d jobs with Quota error", len(jobs))

	failedToRescheduleCount := 0
	for _, job := range jobs {
		isQuotaRenewed, err := hasQuotaRenewedForJob(job)
		if err != nil {
			glog.Warningf("[FAIL] not able to calculate job next quota time with ID: %s", job.ID)
			continue
		}
		if !isQuotaRenewed {
			continue
		}
		patch := repository.JobPatch{ID: job.ID, Status: repository.NotScheduled}
		err = r.jobRepository.PatchJobStatus(ctx, patch)
		if err != nil {
			glog.Warningf("[FAIL] not able to reschedule job with ID: %s", job.ID)
			failedToRescheduleCount++
		}
	}
	if failedToRescheduleCount != 0 {
		glog.Infof("[FAIL] %d jobs where not rescheduled", failedToRescheduleCount)
		return
	}
	glog.Infof("[SUCCESS] All jobs where rescheduled")
}

func hasQuotaRenewedForJob(job *repository.Job) (bool, error) {
	pacificTimeLocation, err := time.LoadLocation(pacificTimeLocation)
	if err != nil {
		return false, err
	}
	jobUpdateTimeInPT := job.UpdatedTimestamp.In(pacificTimeLocation)
	nextQuotaTimeResetForJob := jobUpdateTimeInPT.AddDate(0, 0, 1)
	nextQuotaTimeResetForJob = time.Date(
		nextQuotaTimeResetForJob.Year(),
		nextQuotaTimeResetForJob.Month(),
		nextQuotaTimeResetForJob.Day(),
		0,
		0,
		0,
		0,
		pacificTimeLocation,
	)
	return time.Now().In(pacificTimeLocation).After(nextQuotaTimeResetForJob), nil
}
