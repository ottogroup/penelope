package processor

import (
	"fmt"
	"github.com/ottogroup/penelope/pkg/provider"
	"time"

	"github.com/google/uuid"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/requestobjects"
)

func generateNewID() string {
	for {
		id, err := uuid.NewRandom()
		if err == nil {
			return id.String()
		}
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func mapBackupToResponse(backup *repository.Backup, jobs []*repository.Job, sourceGCPProject provider.SourceGCPProject) requestobjects.BackupResponse {
	var jobResponse []requestobjects.JobResponse
	for _, job := range jobs {
		var foreignJobID string
		if backup.Type == repository.BigQuery {
			foreignJobID = string(job.ForeignJobID.BigQueryID)
		}
		if backup.Type == repository.CloudStorage {
			foreignJobID = string(job.ForeignJobID.CloudStorageID)
		}
		jobResponse = append(jobResponse, requestobjects.JobResponse{
			ID:               job.ID,
			BackupID:         job.BackupID,
			ForeignJobID:     foreignJobID,
			Status:           job.Status.String(),
			Source:           job.Source,
			CreatedTimestamp: formatTime(job.CreatedTimestamp),
			UpdatedTimestamp: formatTime(job.UpdatedTimestamp),
			DeletedTimestamp: formatTime(job.DeletedTimestamp),
		})
	}
	status := backup.Status
	if repository.Prepared == backup.Status {
		status = "Running" //rewording prepared status for frontend
	}

	return requestobjects.BackupResponse{
		ID:                               backup.ID,
		Sink:                             backup.SinkOptions.Sink,
		Status:                           status.String(),
		SinkProject:                      backup.SinkOptions.TargetProject,
		CreatedTimestamp:                 formatTime(backup.CreatedTimestamp),
		UpdatedTimestamp:                 formatTime(backup.UpdatedTimestamp),
		DeletedTimestamp:                 formatTime(backup.DeletedTimestamp),
		DataOwner:                        sourceGCPProject.DataOwner,
		DataAvailabilityClass:            sourceGCPProject.AvailabilityClass,
		TrashcanCleanupStatus:            backup.TrashcanCleanup.Status.String(),
		TrashcanCleanupErrorMessage:      backup.TrashcanCleanup.ErrorMessage,
		TrashcanCleanupLastScheduledTime: formatTime(backup.TrashcanCleanup.LastScheduled),
		CreateRequest: requestobjects.CreateRequest{
			Type:                   backup.Type.String(),
			Strategy:               backup.Strategy.String(),
			Project:                backup.SourceProject,
			RecoveryPointObjective: backup.RecoveryPointObjective,
			RecoveryTimeObjective:  backup.RecoveryTimeObjective,
			TargetOptions: requestobjects.TargetOptions{
				StorageClass: backup.StorageClass,
				Region:       backup.Region,
				ArchiveTTM:   backup.SinkOptions.ArchiveTTM,
			},
			SnapshotOptions: requestobjects.SnapshotOptions{
				FrequencyInHours: backup.FrequencyInHours,
				LifetimeInDays:   backup.SnapshotOptions.LifetimeInDays,
				LastScheduled:    formatTime(backup.LastScheduledTime),
			},
			MirrorOptions: requestobjects.MirrorOptions{
				LifetimeInDays: backup.MirrorOptions.LifetimeInDays,
			},
			BigQueryOptions: requestobjects.BigQueryOptions{
				Dataset:        backup.Dataset,
				Table:          backup.Table,
				ExcludedTables: backup.ExcludedTables,
			},
			GCSOptions: requestobjects.GCSOptions{
				Bucket:      backup.Bucket,
				ExcludePath: backup.ExcludePath,
				IncludePath: backup.IncludePath,
			},
		},
		Jobs: jobResponse,
	}
}

func mapToRestoreResponse(backup *repository.Backup, jobs []*repository.Job) (restoreResponse requestobjects.RestoreResponse) {
	restoreResponse.BackupID = backup.ID
	for _, job := range jobs {
		var action string
		var backupType string
		if backup.Type == repository.BigQuery {
			backupType = "bq"
			action += fmt.Sprintf(`bq --location=EU load --project_id "%s" --source_format=AVRO "%s.%s" "%s"`,
				backup.SourceProject,
				backup.BigQueryOptions.Dataset,
				job.Source,
				repository.BuildFullObjectStoragePath(backup.Sink, backup.BigQueryOptions.Dataset, job.Source, job.ID),
			)
		}
		if backup.Type == repository.CloudStorage {
			backupType = "gcs"
		}
		restoreResponse.RestoreActions = append(restoreResponse.RestoreActions, requestobjects.RestoreAction{
			Type:   backupType,
			Action: action,
		})
	}
	return restoreResponse
}
