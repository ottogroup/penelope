package actions

import (
    "fmt"
    "github.com/golang/glog"
    "github.com/ottogroup/penelope/pkg/http/auth"
    "github.com/ottogroup/penelope/pkg/http/auth/model"
    "github.com/ottogroup/penelope/pkg/repository"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "net/http"
    "reflect"
    "time"
)

func formatTime(t time.Time) string {
    if t.IsZero() {
        return ""
    }
    return t.Format(time.RFC3339)
}

func mapBackupToResponse(backup *repository.Backup, jobs []*repository.Job) requestobjects.BackupResponse {
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
        ID:               backup.ID,
        Sink:             backup.SinkOptions.Sink,
        Status:           status.String(),
        SinkProject:      backup.SinkOptions.TargetProject,
        CreatedTimestamp: formatTime(backup.CreatedTimestamp),
        UpdatedTimestamp: formatTime(backup.UpdatedTimestamp),
        DeletedTimestamp: formatTime(backup.DeletedTimestamp),
        CreateRequest: requestobjects.CreateRequest{
            Type:     backup.Type.String(),
            Strategy: backup.Strategy.String(),
            Project:  backup.SourceProject,
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
                LifetimeInDays: backup.SnapshotOptions.LifetimeInDays,
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
            action += fmt.Sprintf(`bq --location=EU load --project_id "%s" --source_format=AVRO %s.%s "%s"`,
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

func checkRequestBodyIsValid(w http.ResponseWriter, err error) bool {
    if err != nil {
        logMsg := fmt.Sprintf("Error reading request body. Err: %s", err)
        respMsg := "Could not read body of request"
        prepareResponse(w, logMsg, respMsg, http.StatusUnprocessableEntity)
        return false
    }

    return true
}

func getPrincipalOrElsePrepareFailedResponse(w http.ResponseWriter, r *http.Request) (*model.Principal, bool) {
    principal, ok := r.Context().Value(auth.CtxPrincipalKey).(*model.Principal)
    if !ok || principal == nil {
        prepareResponse(w, "no principal found in context", "could not retrieve user-info", http.StatusInternalServerError)
        return nil, false
    }
    return principal, true
}

func checkBackupIsFound(w http.ResponseWriter, backup *repository.Backup, backupID string) bool {
    if backup == nil || reflect.ValueOf(backup).IsNil() {
        logMsg := fmt.Sprintf("no backup with id %q found", backupID)
        prepareResponse(w, logMsg, logMsg, http.StatusNotFound)
        return false
    }
    return true
}

func checkParsingBodyIsValid(w http.ResponseWriter, err error, body string) bool {
    if err != nil {
        logMsg := fmt.Sprintf("Error parsing json request body. Err: %s\n body: %s", err, body)
        respMsg := "Could not parsing json request body of request"
        prepareResponse(w, logMsg, respMsg, http.StatusUnprocessableEntity)
        return false
    }

    return true
}

func prepareResponse(w http.ResponseWriter, logMsg string, responseMsg string, responseCode int) {
    glog.Info(logMsg)
    w.WriteHeader(responseCode)
    if _, err := fmt.Fprint(w, responseMsg); err != nil {
        glog.Warningf("Error writing response: %s", err)
    }
}
