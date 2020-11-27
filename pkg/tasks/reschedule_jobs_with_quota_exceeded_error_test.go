package tasks

import (
    "context"
    "fmt"
    "github.com/stretchr/testify/assert"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/repository/memory"
    "testing"
    "time"
)

const (
    JobWithOK         = "job-ok"
    JobWithError      = "job-error"
    JobWithQuotaError = "job-quota-error"
)

func TestRescheduleJobsWithQuotaErrorService_Run_NoJobs(t *testing.T) {
    testUseCase := GivenATestRescheduleJobsWithQuotaErrorServiceContext()
    testUseCase.Service.Run(context.Background())
}

func TestRescheduleJobsWithQuotaErrorService_Run_JobsStatusNotChange(t *testing.T) {
    // given
    ctx := context.Background()
    testUseCase := GivenATestRescheduleJobsWithQuotaErrorServiceContext()

    jobOK := GivenAJobWithStatus(testUseCase, JobWithOK, repository.FinishedOk)
    err := testUseCase.JobRepository.AddJob(ctx, &jobOK)
    assert.NoError(t, err)

    jobError := GivenAJobWithStatus(testUseCase, JobWithError, repository.FinishedError)
    err = testUseCase.JobRepository.AddJob(ctx, &jobError)
    assert.NoError(t, err)

    jobQuotaError := GivenAJobWithStatus(testUseCase, JobWithQuotaError, repository.FinishedQuotaError)
    err = testUseCase.JobRepository.AddJob(ctx, &jobQuotaError)
    assert.NoError(t, err)
    // when
    testUseCase.Service.Run(ctx)
    // then
    job, err := testUseCase.JobRepository.GetJob(ctx, JobWithOK)
    assert.NoError(t, err, "GetJob", JobWithOK)
    assert.Equal(t, repository.FinishedOk, job.Status)

    job, err = testUseCase.JobRepository.GetJob(ctx, JobWithError)
    assert.NoError(t, err, "GetJob", JobWithError)
    assert.Equal(t, repository.FinishedError, job.Status)

    job, err = testUseCase.JobRepository.GetJob(ctx, JobWithQuotaError)
    assert.NoError(t, err, "GetJob", JobWithQuotaError)
    assert.Equal(t, repository.FinishedQuotaError, job.Status)
}

func TestRescheduleJobsWithQuotaErrorService_Run_JobRescheduled(t *testing.T) {
    // given
    ctx := context.Background()
    testUseCase := GivenATestRescheduleJobsWithQuotaErrorServiceContext()

    jobQuotaError := GivenAJobWithStatus(testUseCase, JobWithQuotaError, repository.FinishedQuotaError)
    jobQuotaError.UpdatedTimestamp = jobQuotaError.UpdatedTimestamp.AddDate(0, 0, -1)
    err := testUseCase.JobRepository.AddJob(ctx, &jobQuotaError)
    assert.NoError(t, err)
    // when
    testUseCase.Service.Run(ctx)
    // expect
    job, err := testUseCase.JobRepository.GetJob(ctx, JobWithQuotaError)
    assert.NoError(t, err, "GetJob", JobWithQuotaError)
    assert.Equal(t, repository.NotScheduled, job.Status)
    assert.Equal(t, repository.NotScheduled, job.Status)
}

func TestRescheduleJobsWithQuotaErrorService_hasQuotaRenewedForJob_False(t *testing.T) {
    // given
    testUseCase := GivenATestRescheduleJobsWithQuotaErrorServiceContext()
    jobQuotaError := GivenAJobWithStatus(testUseCase, JobWithQuotaError, repository.FinishedQuotaError)
    // when
    isQuotaRenewedForJob, err := hasQuotaRenewedForJob(&jobQuotaError)
    // expect
    assert.NoError(t, err, "hasQuotaRenewedForJob", JobWithQuotaError)
    assert.False(t, isQuotaRenewedForJob)
}

func TestRescheduleJobsWithQuotaErrorService_hasQuotaRenewedForJob_True(t *testing.T) {
    // given
    testUseCase := GivenATestRescheduleJobsWithQuotaErrorServiceContext()
    jobQuotaError := GivenAJobWithStatus(testUseCase, JobWithQuotaError, repository.FinishedQuotaError)
    jobQuotaError.UpdatedTimestamp = jobQuotaError.UpdatedTimestamp.AddDate(0, 0, -1)
    // when
    isQuotaRenewedForJob, err := hasQuotaRenewedForJob(&jobQuotaError)
    // expect
    assert.NoError(t, err, "hasQuotaRenewedForJob", JobWithQuotaError)
    assert.True(t, isQuotaRenewedForJob)
}

func GivenAJobWithStatus(ctxIn TestRescheduleJobsWithQuotaErrorServiceContext, id string, status repository.JobStatus) repository.Job {
    return repository.Job{
        ID:     id,
        Status: status,
        Type:   repository.BigQuery,
        EntityAudit: repository.EntityAudit{
            CreatedTimestamp: time.Now().In(ctxIn.PacificTimeLocation),
            UpdatedTimestamp: time.Now().In(ctxIn.PacificTimeLocation),
        },
    }
}

func GivenATestRescheduleJobsWithQuotaErrorServiceContext() TestRescheduleJobsWithQuotaErrorServiceContext {
    pacificTimeLocation, err := time.LoadLocation(pacificTimeLocation)
    if err != nil {
        panic(fmt.Sprintf("Load dLocation failed for %s: %v", pacificTimeLocation, err))
    }
    jobRepository := memory.JobRepository{}
    return TestRescheduleJobsWithQuotaErrorServiceContext{
        JobRepository:       &jobRepository,
        PacificTimeLocation: pacificTimeLocation,
        Service:             rescheduleJobsWithQuotaErrorService{jobRepository: &jobRepository},
    }
}

type TestRescheduleJobsWithQuotaErrorServiceContext struct {
    JobRepository       repository.JobRepository
    PacificTimeLocation *time.Location
    Service             rescheduleJobsWithQuotaErrorService
}
