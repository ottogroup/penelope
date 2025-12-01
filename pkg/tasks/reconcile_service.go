package tasks

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"github.com/ottogroup/penelope/pkg/service/util"
	"go.opencensus.io/trace"
)

type reconcileService struct {
	ctx                               context.Context
	db                                repository.BackupRepository
	targetPrincipalForProjectProvider impersonate.TargetPrincipalForProjectProvider
	cloudStorageClients               map[string]gcs.CloudStorageClient
}

func newReconcileService(ctxIn context.Context, provider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) (*reconcileService, error) {
	ctx, span := trace.StartSpan(ctxIn, "newReconcileService")
	defer span.End()

	db, err := repository.NewBackupRepository(ctx, credentialsProvider)
	if err != nil {
		return &reconcileService{}, fmt.Errorf("could not instantiate new BackupRepository: %s", err)
	}

	return &reconcileService{db: db, ctx: ctx, targetPrincipalForProjectProvider: provider, cloudStorageClients: make(map[string]gcs.CloudStorageClient)}, nil
}

func (j *reconcileService) Run(ctxIn context.Context) {
	ctx, span := trace.StartSpan(ctxIn, "(*reconcileService).Run")
	defer span.End()

	backups, err := j.db.GetBackups(ctx, repository.BackupFilter{})
	if err != nil {
		glog.Errorf("could not get list of backups: %s", err)
	}
	glog.Infof("[START] Reconcile backup")
	var failedBackups []string
	var successBackups []string
	for _, backup := range backups {
		hasActiveStatus := backup.Status == repository.Prepared || backup.Status == repository.NotStarted || backup.Status == repository.Paused || backup.Status == repository.Finished
		if !hasActiveStatus {
			continue
		}
		err = j.syncSinkBucket(ctx, backup)
		if err != nil {
			glog.Errorf("could not sync bucket labels and lifecycle for backup %s: %s", backup.ID, err)
			failedBackups = append(failedBackups, backup.ID)
			continue
		}
		successBackups = append(successBackups, backup.ID)
	}

	// cleanup clients
	for _, cloudStorageClient := range j.cloudStorageClients {
		cloudStorageClient.Close(ctx)
	}

	if len(failedBackups) == 0 {
		logMessage := "[SUCCESS] Synced backups"
		logMessage += strings.Join(successBackups, "|")
		glog.Infof(logMessage)
		return
	}

	logMessage := "[FAIL] Reconcile backup:"
	logMessage += strings.Join(failedBackups, "|")
	glog.Info(logMessage)
}

func (j *reconcileService) syncSinkBucket(ctx context.Context, backup *repository.Backup) error {
	cloudStorageClient, err := j.prepareCloudStorageClient(ctx, backup)
	if err != nil {
		return fmt.Errorf("could not create cloud storage client for backup %s: %s", backup.ID, err)
	}

	var lifetimeInDays uint = 0
	if backup.Strategy.EqualTo(repository.Mirror.String()) {
		lifetimeInDays = backup.MirrorOptions.LifetimeInDays
	}
	if backup.Strategy.EqualTo(repository.Snapshot.String()) {
		lifetimeInDays = backup.SnapshotOptions.LifetimeInDays
	}

	bucketName := backup.SinkOptions.Sink
	exist, err := cloudStorageClient.DoesBucketExist(ctx, backup.TargetProject, bucketName)
	if err != nil {
		return fmt.Errorf("could not check if bucket %s exists: %s", bucketName, err)
	}

	if !exist {
		err = cloudStorageClient.CreateBucket(ctx, gcs.CloudStorageBucket{
			Project:        backup.TargetProject,
			Bucket:         bucketName,
			Location:       backup.Region,
			DualLocation:   backup.DualRegion,
			StorageClass:   backup.StorageClass,
			LifetimeInDays: lifetimeInDays,
			ArchiveTTM:     backup.ArchiveTTM,
			Labels:         gcs.NewLabels(util.PascalCaseToSnakeCase(backup.Type.String()), backup.ID, backup.SourceProject),
		})
		return fmt.Errorf("failed to create bucket %s: %s", bucketName, err)
	}

	err = cloudStorageClient.UpdateBucket(
		ctx,
		bucketName,
		lifetimeInDays,
		backup.ArchiveTTM,
		gcs.NewLabels(util.PascalCaseToSnakeCase(backup.Type.String()), backup.ID, backup.SourceProject),
	)
	if err != nil {
		return fmt.Errorf("could not update bucket %s lifecycle: %s", backup.Sink, err)
	}
	return nil
}

func (j *reconcileService) prepareCloudStorageClient(ctx context.Context, backup *repository.Backup) (gcs.CloudStorageClient, error) {
	if client, exists := j.cloudStorageClients[backup.TargetProject]; exists {
		return client, nil
	}
	cloudStorageClient, err := gcs.NewCloudStorageClient(ctx, j.targetPrincipalForProjectProvider, backup.TargetProject)
	if err != nil {
		return nil, fmt.Errorf("could not create cloud storage client for backup %s: %s", backup.ID, err)
	}
	j.cloudStorageClients[backup.TargetProject] = cloudStorageClient
	return cloudStorageClient, nil
}
