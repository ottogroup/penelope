package tasks

import (
    "context"
    "fmt"
    "github.com/golang/glog"
    "github.com/ottogroup/penelope/pkg/processor"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/secret"
    "go.opencensus.io/trace"
    "strings"
)

type jobStuckService struct {
    scheduleProcessor processor.ScheduleProcessor
    ctx               context.Context
}

func newJobsStuckService(ctxIn context.Context, credentialsProvider secret.SecretProvider) (*jobStuckService, error) {
    ctx, span := trace.StartSpan(ctxIn, "newJobsStuckService")
    defer span.End()

    scheduleProcessor, err := processor.NewScheduleProcessor(ctx, credentialsProvider)
    if err != nil {
        return &jobStuckService{}, fmt.Errorf("could not instantiate new ScheduleProcessor: %s", err)
    }

    return &jobStuckService{scheduleProcessor: scheduleProcessor, ctx: ctx}, nil
}

func (j *jobStuckService) Run(ctxIn context.Context) {
    ctx, span := trace.StartSpan(ctxIn, "(*jobStuckService).Run")
    defer span.End()

    deltaHours := 1
    statuses := []repository.JobStatus{repository.NotScheduled, repository.Scheduled, repository.Pending}
    jobs, err := j.scheduleProcessor.GetByStatusAndAfter(ctx, statuses, deltaHours)
    if err != nil {
        glog.Errorf("could not get list of jobs with status %v before %d hours: %s", statuses, deltaHours, err)
        return
    }
    glog.Infof("[START] Checking stucked jobs")
    if len(jobs) == 0 {
        glog.Infof("[SUCCESS] No jobs stuck with status %v before %d hours", statuses, deltaHours)
        return
    }

    glog.Infof("Alerting on %d stuck jobs in status %v before %d hours:", len(jobs), statuses, deltaHours)
    logMessage := "[FAIL]"
    logMessage += strings.Join(toString(jobs), "|")
    glog.Info(logMessage)
}

func toString(jobs []*repository.Job) (result []string) {
    for _, job := range jobs {
        result = append(result, job.String())
    }
    return result
}
