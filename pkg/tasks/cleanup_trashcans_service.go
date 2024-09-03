package tasks

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"go.opencensus.io/trace"
	"strings"
	"time"
)

func newCleanupTrashcansService(ctxIn context.Context, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) (*cleanupTrashcansService, error) {
	ctx, span := trace.StartSpan(ctxIn, "newPrepareBackupJobsService")
	defer span.End()

	backupRepository, err := repository.NewBackupRepository(ctx, credentialsProvider)
	if err != nil {
		return nil, err
	}

	return &cleanupTrashcansService{tokenSourceProvider: tokenSourceProvider, backupRepository: backupRepository}, nil
}

type cleanupTrashcansService struct {
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
	backupRepository    repository.BackupRepository
}

func (s *cleanupTrashcansService) Run(ctxIn context.Context) {
	ctx, span := trace.StartSpan(ctxIn, "(*cleanupTrashcansService).Run")
	defer span.End()

	backups, err := s.backupRepository.GetBackupsByCleanupTrashcanStatus(ctx, repository.ScheduledTrashcanCleanupStatus)
	if err != nil {
		glog.Errorf("could not get list of backups scheduled to cleanup trashcan: %s", err)
	}

	for _, backup := range backups {
		gcsClient, err := gcs.NewCloudStorageClient(ctx, s.tokenSourceProvider, backup.TargetProject)
		if err != nil {
			glog.Errorf("could not create new CloudStorageClient: %s", err)
			return
		}
		defer gcsClient.Close(ctx)

		err = s.backupRepository.MarkTrashcanCleanup(ctx, backup.ID, repository.TrashcanCleanup{
			Status:                   repository.InProgressCleanupTrashcanCleanupStatus,
			StartInProgressTimestamp: time.Now(),
		})
		if err != nil {
			return
		}

		trashcanPath := backup.GetTrashcanPath()
		if strings.EqualFold(trashcanPath, "") {
			errMsg := fmt.Sprintf("trashcan path is empty for backup with id %s", backup.ID)
			glog.Errorf(errMsg)
			err = s.backupRepository.MarkTrashcanCleanup(ctx, backup.ID, repository.TrashcanCleanup{
				Status:       repository.ErrorCleanupTrashcanCleanupStatus,
				ErrorMessage: errMsg,
			})
			if err != nil {
				glog.Errorf("could not mark trashcan cleanup status to %s: %s", repository.ErrorCleanupTrashcanCleanupStatus, err)
			}
			return
		}

		err = gcsClient.DeleteObjectWithPrefix(ctx, backup.Sink, trashcanPath)
		if err != nil {
			errMsg := fmt.Sprintf("could not delete objects in trashcan for backup with id %s: %s", backup.ID, err)
			glog.Errorf(errMsg)
			err = s.backupRepository.MarkTrashcanCleanup(ctx, backup.ID, repository.TrashcanCleanup{
				Status:       repository.ErrorCleanupTrashcanCleanupStatus,
				ErrorMessage: errMsg,
			})
			if err != nil {
				glog.Errorf("could not mark trashcan cleanup status to %s: %s", repository.ErrorCleanupTrashcanCleanupStatus, err)
			}
			return
		}

		err = gcsClient.CreateObject(ctx, backup.Sink, fmt.Sprintf("%s/THIS_TRASHCAN_CONTAINS_DELETED_OBJECTS_FROM_SOURCE", trashcanPath), "")
		if err != nil {
			glog.Errorf("could not create THIS_TRASHCAN_CONTAINS_DELETED_OBJECTS_FROM_SOURCE object in trashcan: %s", err)
			return
		}

		err = s.backupRepository.MarkTrashcanCleanup(ctx, backup.ID, repository.TrashcanCleanup{
			Status:        repository.NoopCleanupTrashcanCleanupStatus,
			LastScheduled: time.Now(),
		})
		if err != nil {
			glog.Errorf("could not mark trashcan cleanup status to %s: %s", repository.NoopCleanupTrashcanCleanupStatus, err)
			return
		}

		glog.Infof("trashcan cleanup for backup completed: %s", backup.ID)
	}
}
