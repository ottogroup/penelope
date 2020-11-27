package processor

import (
    "context"
    "fmt"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/service/bigquery"
    "go.opencensus.io/trace"
    "time"
)

// TrashcanEntry moved objects into trashcan
type TrashcanEntry struct {
    BackupID string
    Source   string
}

// ScheduleProcessor defines operation for scheduling
type ScheduleProcessor interface {
    CreateBigQueryJobCreator(ctxIn context.Context, client bigquery.Client) *BigQueryJobCreator
    CreateCloudStorageJobCreator(ctxIn context.Context) *CloudStorageJobCreator
    GetNextBackupJobs(context.Context, repository.BackupType) ([]*repository.Job, error)
    GetScheduledBackupJobs(context.Context, repository.BackupType) ([]*repository.Job, error)
    GetExpired(context.Context, repository.BackupType) ([]*repository.Backup, error)
    GetExpiredBigQueryMirrorRevisions(ctxIn context.Context, maxRevisionLifetimeInWeeks int) ([]*repository.MirrorRevision, error)
    GetScheduledBackups(context.Context, repository.BackupType) ([]*repository.Backup, error)
    GetByStatusAndAfter(context.Context, []repository.JobStatus, int) ([]*repository.Job, error)
    GetJobsForBackupID(ctxIn context.Context, backupID string, jobPage repository.JobPage) ([]*repository.Job, error)
    UpdateJob(ctxIn context.Context, backupType repository.BackupType, jobID string, status repository.JobStatus, externalID string) error
    UpdateBackupStatus(ctxIn context.Context, id string, status repository.BackupStatus) error
    UpdateLastCleanupTime(ctxIn context.Context, backupID string, lastCleanupTime time.Time) error
    MarkBackupDeleted(ctxIn context.Context, id string) error
    MarkSourceMetadataDeleted(ctxIn context.Context, id int) error
    MarkJobDeleted(ctxIn context.Context, id string) error
    GetBackupForID(ctxIn context.Context, id string) (*repository.Backup, error)
    AddTrashcanEntry(ctxIn context.Context, backupID string, source string, timestamp time.Time) error
    DeleteTrashcanEntry(ctxIn context.Context, backupID string, source string) error
    FilterExistingTrashcanEntries(context.Context, []TrashcanEntry) ([]TrashcanEntry, error)
    GetEntriesInTrashcanBefore(ctxIn context.Context, deltaWeeks int) ([]*repository.SourceTrashcan, error)
}

type defaultScheduleProcessor struct {
    backupRepository            repository.BackupRepository
    jobRepository               repository.JobRepository
    sourceMetadataRepository    repository.SourceMetadataRepository
    sourceMetadataJobRepository repository.SourceMetadataJobRepository
    sourceTrashcanRepository    repository.SourceTrashcanRepository
}

// NewScheduleProcessor create new instance of ScheduleProcessor
func NewScheduleProcessor(ctxIn context.Context, credentialsProvider secret.SecretProvider) (ScheduleProcessor, error) {
    ctx, span := trace.StartSpan(ctxIn, "NewScheduleProcessor")
    defer span.End()

    backupRepository, err := repository.NewBackupRepository(ctx, credentialsProvider)
    if err != nil {
        return nil, err
    }

    jobRepository, err := repository.NewJobRepository(ctx, credentialsProvider)
    if err != nil {
        return nil, err
    }

    sourceMetadataRepository, err := repository.NewSourceMetadataRepository(ctx, credentialsProvider)
    if err != nil {
        return nil, err
    }

    sourceMetadataJobRepository, err := repository.NewSourceMetadataJobRepository(ctx, credentialsProvider)
    if err != nil {
        return nil, err
    }

    sourceTrashcanRepository, err := repository.NewSourceTrashcanRepository(ctx, credentialsProvider)
    if err != nil {
        return nil, err
    }

    return &defaultScheduleProcessor{
        backupRepository:            backupRepository,
        jobRepository:               jobRepository,
        sourceMetadataRepository:    sourceMetadataRepository,
        sourceMetadataJobRepository: sourceMetadataJobRepository,
        sourceTrashcanRepository:    sourceTrashcanRepository,
    }, nil
}

func (d *defaultScheduleProcessor) CreateBigQueryJobCreator(ctxIn context.Context, bigQueryClient bigquery.Client) *BigQueryJobCreator {
    ctx, span := trace.StartSpan(ctxIn, "(*defaultScheduleProcessor).CreateBigQueryJobCreator")
    defer span.End()

    return NewBigQueryJobCreator(ctx, d.backupRepository, d.jobRepository, bigQueryClient, d.sourceMetadataRepository, d.sourceMetadataJobRepository)
}

func (d *defaultScheduleProcessor) CreateCloudStorageJobCreator(ctxIn context.Context, ) *CloudStorageJobCreator {
    ctx, span := trace.StartSpan(ctxIn, "(*defaultScheduleProcessor).CreateCloudStorageJobCreator")
    defer span.End()

    return NewCloudStorageJobCreator(ctx, d.backupRepository, d.jobRepository)
}

func (d *defaultScheduleProcessor) GetNextBackupJobs(ctxIn context.Context, backupType repository.BackupType) ([]*repository.Job, error) {
    return d.jobRepository.GetByJobTypeAndStatus(ctxIn, backupType, repository.NotScheduled)
}

func (d *defaultScheduleProcessor) GetScheduledBackupJobs(ctxIn context.Context, backupType repository.BackupType) ([]*repository.Job, error) {
    return d.jobRepository.GetByJobTypeAndStatus(ctxIn, backupType, repository.Scheduled, repository.Pending)
}

func (d *defaultScheduleProcessor) UpdateJob(ctxIn context.Context, backupType repository.BackupType, jobID string, status repository.JobStatus, externalID string) error {
    patch := repository.JobPatch{ID: jobID, Status: status, ForeignJobID: repository.ForeignJobID{}}

    switch backupType.String() {
    case repository.BigQuery.String():
        patch.ForeignJobID.BigQueryID = repository.ExtractJobID(externalID)
    case repository.CloudStorage.String():
        patch.ForeignJobID.CloudStorageID = repository.TransferJobID(externalID)
    default:
        return fmt.Errorf("unknown job type %v", backupType.String())
    }

    return d.jobRepository.PatchJobStatus(ctxIn, patch)
}

func (d *defaultScheduleProcessor) UpdateBackupStatus(ctxIn context.Context, id string, status repository.BackupStatus) error {
    return d.backupRepository.MarkStatus(ctxIn, id, status)
}

func (d *defaultScheduleProcessor) UpdateLastCleanupTime(ctxIn context.Context, backupID string, lastCleanupTime time.Time) error {
    return d.backupRepository.UpdateLastCleanupTime(ctxIn, backupID, lastCleanupTime)
}

func (d *defaultScheduleProcessor) MarkBackupDeleted(ctxIn context.Context, id string) error {
    return d.backupRepository.MarkDeleted(ctxIn, id)
}

func (d *defaultScheduleProcessor) MarkSourceMetadataDeleted(ctxIn context.Context, id int) error {
    return d.sourceMetadataRepository.MarkDeleted(ctxIn, id)
}

func (d *defaultScheduleProcessor) MarkJobDeleted(ctxIn context.Context, jobID string) error {
    return d.jobRepository.MarkDeleted(ctxIn, jobID)
}

func (d *defaultScheduleProcessor) GetBackupForID(ctxIn context.Context, backupID string) (*repository.Backup, error) {
    return d.backupRepository.GetBackup(ctxIn, backupID)
}

func (d *defaultScheduleProcessor) GetExpired(ctxIn context.Context, backupType repository.BackupType) ([]*repository.Backup, error) {
    return d.backupRepository.GetExpired(ctxIn, backupType)
}

func (d *defaultScheduleProcessor) GetExpiredBigQueryMirrorRevisions(ctxIn context.Context, maxRevisionLifetimeInWeeks int) ([]*repository.MirrorRevision, error) {
    return d.backupRepository.GetExpiredBigQueryMirrorRevisions(ctxIn, maxRevisionLifetimeInWeeks)
}

func (d *defaultScheduleProcessor) GetScheduledBackups(ctxIn context.Context, backupType repository.BackupType) ([]*repository.Backup, error) {
    return d.backupRepository.GetScheduledBackups(ctxIn, backupType)
}

func (d *defaultScheduleProcessor) GetByStatusAndAfter(ctxIn context.Context, status []repository.JobStatus, deltaHours int) ([]*repository.Job, error) {
    return d.jobRepository.GetByStatusAndBefore(ctxIn, status, deltaHours)
}

func (d *defaultScheduleProcessor) GetJobsForBackupID(ctxIn context.Context, backupID string, page repository.JobPage) ([]*repository.Job, error) {
    return d.jobRepository.GetJobsForBackupID(ctxIn, backupID, page)
}

func (d *defaultScheduleProcessor) AddTrashcanEntry(ctxIn context.Context, backupID string, source string, timestamp time.Time) error {
    return d.sourceTrashcanRepository.Add(ctxIn, backupID, source, timestamp)
}

func (d *defaultScheduleProcessor) DeleteTrashcanEntry(ctxIn context.Context, backupID string, source string) error {
    return d.sourceTrashcanRepository.Delete(ctxIn, backupID, source)
}

func (d *defaultScheduleProcessor) GetEntriesInTrashcanBefore(ctx context.Context, deltaWeeks int) ([]*repository.SourceTrashcan, error) {
    return d.sourceTrashcanRepository.GetBefore(ctx, deltaWeeks)
}

func (d *defaultScheduleProcessor) FilterExistingTrashcanEntries(ctx context.Context, trashcanEntries []TrashcanEntry) (entries []TrashcanEntry, err error) {
    var sources []repository.SourceTrashcan
    for _, entry := range trashcanEntries {
        sources = append(sources, repository.SourceTrashcan{BackupID: entry.BackupID, Source: entry.Source})
    }
    sources, err = d.sourceTrashcanRepository.FilterExistingEntries(ctx, sources)
    if err != nil {
        return entries, err
    }
    for _, src := range sources {
        entries = append(entries, TrashcanEntry{BackupID: src.BackupID, Source: src.Source})
    }
    return entries, nil
}
