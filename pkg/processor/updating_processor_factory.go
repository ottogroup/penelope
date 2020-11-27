package processor

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/http/auth"
    "github.com/ottogroup/penelope/pkg/http/impersonate"
    "github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/requestobjects"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/service/bigquery"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"go.opencensus.io/trace"
)

// UpdatingProcessorFactory factory for operation Updating
type UpdatingProcessorFactory struct {
    tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
    credentialsProvider secret.SecretProvider
}

func NewUpdatingProcessorFactory(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) *UpdatingProcessorFactory {
    return &UpdatingProcessorFactory{tokenSourceProvider, credentialsProvider}
}

// DoMatchRequestType compare request string type to Updating
func (c UpdatingProcessorFactory) DoMatchRequestType(requestType requestobjects.RequestType) bool {
	return requestobjects.Updating.EqualTo(requestType.String())
}

// CreateProcessor create instance of Operations
func (c UpdatingProcessorFactory) CreateProcessor(ctxIn context.Context) (Operations, error) {
	processor, err := c.newUpdatingProcessor(ctxIn)
	if err != nil {
		return nil, err
	}

	return processor, nil
}

type updatingProcessor struct {
	BackupRepository repository.BackupRepository
	JobRepository    repository.JobRepository
	Context          context.Context

	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func (c UpdatingProcessorFactory) newUpdatingProcessor(ctxIn context.Context) (*updatingProcessor, error) {
	ctx, span := trace.StartSpan(ctxIn, "newUpdatingProcessor")
	defer span.End()

	backupRepository, err := repository.NewBackupRepository(ctx, c.credentialsProvider)
	if err != nil {
		glog.Error(err)
		return &updatingProcessor{}, err
	}

	jobRepository, err := repository.NewJobRepository(ctx, c.credentialsProvider)
	if err != nil {
		glog.Error(err)
		return &updatingProcessor{}, err
	}

	return &updatingProcessor{
	    BackupRepository: backupRepository,
	    JobRepository: jobRepository,
	    tokenSourceProvider: c.tokenSourceProvider,
	}, nil
}

func (c updatingProcessor) Process(ctxIn context.Context, args *Arguments) (*Result, error) {
	ctx, span := trace.StartSpan(ctxIn, "(updatingProcessor).Process")
	defer span.End()

	var request *requestobjects.UpdateRequest
	if args.Request == nil {
		return nil, fmt.Errorf("nil request object for processing updating request")
	}
	request, ok := args.Request.(*requestobjects.UpdateRequest)
	if !ok {
		return nil, fmt.Errorf("wrong request object for processing updating request")
	}
	backup, err := c.BackupRepository.GetBackup(ctx, request.BackupID)
	if err != nil {
		return &Result{backups: []*repository.Backup{backup}}, fmt.Errorf("backup with id %s not found: %s", request.BackupID, err)
	}

	if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.Updating, backup.SourceProject) {
		return &Result{backups: []*repository.Backup{backup}}, fmt.Errorf("%s is not allowed for user '%s' on project '%s'", requestobjects.Updating.String(), args.Principal.User.Email, backup.TargetProject)
	}

	// handle status change
	if request.Status != "" && !backup.Status.EqualTo(request.Status) {
		if !isBackupStatusTransitionValid(backup.Status, repository.BackupStatus(request.Status)) {
			return &Result{backups: []*repository.Backup{backup}}, fmt.Errorf("backup status update not allowed from %s to %s", request.BackupID, request.Status)
		}
		// make a shortcut from NotStarted -> ToDelete to NotStarted -> BackupDeleted
		if repository.NotStarted == backup.Status && repository.ToDelete.EqualTo(request.Status) {
			err = c.BackupRepository.MarkDeleted(ctx, backup.ID)
			if err != nil {
				return &Result{backups: []*repository.Backup{backup}}, err
			}
			backup, err = c.BackupRepository.GetBackup(ctx, request.BackupID)
			return &Result{backups: []*repository.Backup{backup}}, err
		}
	}
	// handle other fields
	if repository.BigQuery == backup.Type && (0 < len(request.Table)) {
		bigqueryClient, err := bigquery.NewBigQueryClient(ctx, c.tokenSourceProvider, backup.SourceProject, backup.TargetProject)
		if err != nil {
			return &Result{backups: []*repository.Backup{backup}}, fmt.Errorf("failed to create BigQuery client: %s", err)
		}
		for _, tableName := range request.Table {
			_, err := bigqueryClient.GetTable(ctx, backup.SourceProject, backup.Dataset, tableName)
			if err != nil {
				return &Result{backups: []*repository.Backup{backup}}, fmt.Errorf("failed to get tableName %s: %s", tableName, err)
			}
		}
		if hasIntersection(request.Table, request.ExcludedTables) {
			return &Result{backups: []*repository.Backup{backup}}, fmt.Errorf("bigQuery request has intersection in tables: %s, %s", request.Table, request.ExcludedTables)
		}
	}
	fields := repository.UpdateFields{
		BackupID:       request.BackupID,
		Status:         repository.BackupStatus(request.Status),
		IncludePath:    request.IncludePath,
		ExcludePath:    request.ExcludePath,
		Table:          request.Table,
		ExcludedTables: request.ExcludedTables,
		MirrorTTL:      request.MirrorTTL,
		SnapshotTTL:    request.SnapshotTTL,
		ArchiveTTM:     request.ArchiveTTM,
	}
	err = c.BackupRepository.UpdateBackup(ctx, fields)

	if err != nil {
		return &Result{backups: []*repository.Backup{backup}}, err
	}

	client, err := gcs.NewCloudStorageClient(ctx, c.tokenSourceProvider, backup.SourceProject)
	if err != nil {
		return &Result{backups: []*repository.Backup{backup}}, fmt.Errorf("updatingProcessor.Process NewCloudStorageClient failed: %v", err)
	}

    // if backup was deleted, create bucket sink again
	if repository.BackupDeleted.EqualTo(backup.Status.String()) && repository.NotStarted.EqualTo(request.Status) {
        exist, err := client.DoesBucketExist(ctx, backup.TargetProject, backup.Sink)
        if err != nil {
            return &Result{backups: []*repository.Backup{backup}}, fmt.Errorf("couldn't check if bucket sink exist for backup: %v", backup)
        }
        if !exist {
            glog.Infof("recreating bucket for backup: %v", backup)
            err := prepareSink(ctx, client, backup)
            if err != nil {
                return &Result{backups: []*repository.Backup{backup}}, fmt.Errorf("sink couldn't be prepared: %v", backup)
            }
        }
    }

	if backup.Strategy == "Mirror" {
		err = client.UpdateBucket(ctx, backup.Sink, request.MirrorTTL, request.ArchiveTTM)
		if err != nil {
			return &Result{backups: []*repository.Backup{backup}}, fmt.Errorf("updatingProcessor.Process UpdateBucket for Mirror failed: %v", err)
		}
	}
	if backup.Strategy == "Snapshot" {
		err = client.UpdateBucket(ctx, backup.Sink, request.SnapshotTTL, request.ArchiveTTM)
		if err != nil {
			return &Result{backups: []*repository.Backup{backup}}, fmt.Errorf("updatingProcessor.Process UpdateBucket for Snapshot failed: %v", err)
		}
	}

	backup, err = c.BackupRepository.GetBackup(ctx, request.BackupID)
	return &Result{backups: []*repository.Backup{backup}}, err
}
