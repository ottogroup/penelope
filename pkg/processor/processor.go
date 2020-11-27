package processor

import (
    "context"
    "github.com/google/uuid"
    "github.com/ottogroup/penelope/pkg/http/auth/model"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "go.opencensus.io/trace"
)

// Arguments for a Processor
type Arguments struct {
    Request   interface{}
    Principal *model.Principal
}

// Result for a request
type Result struct {
    backups             []*repository.Backup
    jobs                []*repository.Job
    JobsTotal           uint64
    BucketListResponse  *requestobjects.BucketListResponse
    CalculateResponse   *requestobjects.CalculatedResponse
    DatasetListResponse *requestobjects.DatasetListResponse
}

// GetJobs returns backup jobs
func (pr Result) GetJobs() []*repository.Job {
    return pr.jobs
}

// GetBackup returns backup
func (pr Result) GetBackup() *repository.Backup {
    if len(pr.backups) == 0 {
        return nil
    } else if len(pr.backups) == 1 {
        return pr.backups[0]
    } else {
        panic("result has more than one expected backup")
    }
}

// GetBackups returns backups
func (pr Result) GetBackups() []*repository.Backup {
    return pr.backups
}

// Operations define operations for processors
type Operations interface {
    Process(context.Context, *Arguments) (*Result, error)
}

// Processor handle request
type Processor struct {
    impl Operations
}

// Process triggers request processing
func (p *Processor) Process(ctxIn context.Context, processorArgs *Arguments) (*Result, error) {
    ctx, span := trace.StartSpan(ctxIn, "(*Processor).Process")
    defer span.End()

    return p.impl.Process(ctx, processorArgs)
}

func generateNewID() string {
    for {
        id, err := uuid.NewRandom()
        if err == nil {
            return id.String()
        }
    }
}

func isBackupStatusTransitionValid(current repository.BackupStatus, new repository.BackupStatus) (isValid bool) {
    switch current {
    case repository.NotStarted:
        switch new {
        case repository.Prepared, repository.Finished, repository.Paused, repository.ToDelete, repository.BackupDeleted:
            isValid = true
        }
    case repository.Prepared:
        switch new {
        case repository.NotStarted, repository.Finished, repository.Paused, repository.ToDelete, repository.BackupDeleted:
            isValid = true
        }
    case repository.Finished:
        switch new {
        case repository.NotStarted, repository.Paused, repository.ToDelete, repository.BackupDeleted:
            isValid = true
        }
    case repository.Paused:
        switch new {
        case repository.NotStarted, repository.ToDelete, repository.BackupDeleted:
            isValid = true
        }
    case repository.ToDelete:
        switch new {
        case repository.NotStarted, repository.BackupDeleted:
            isValid = true
        }
    case repository.BackupDeleted:
        switch new {
        case repository.NotStarted:
            isValid = true
        }
    }
    return isValid
}
