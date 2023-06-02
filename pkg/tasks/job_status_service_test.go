package tasks

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ottogroup/penelope/pkg/http/mock"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"
)

const (
	targetProjectID                = "local-ability-backup"
	statusServiceJobID             = "status-uuid-1234"
	statusServiceBackupID          = "status-uuid-5678"
	jobHandlerTransferJobName      = "transferJobs/1234567890123456789"
	jobHandlerTransferOperationID1 = "transferOperations/transferJobs-1234567890123456789-8765432109876543210"
	jobHandlerTransferOperationID2 = "transferOperations/transferJobs-1234567890123456789-7654321098765432109"
	jobHandlerTransferOperationID3 = "transferOperations/transferJobs-1234567890123456789-6543210987654321098"
	jobHandlerTransferOperationID4 = "transferOperations/transferJobs-1234567890123456789-54321098765432109876"
)

func TestJobStatusService_WithoutValidJob(t *testing.T) {
	ctx := context.Background()
	service, err := newJobStatusService(ctx, nil, secret.NewEnvSecretProvider())
	require.NoError(t, err)

	service.scheduleProcessor = MockScheduleProcessor{
		shouldReturnValidJob:    false,
		shouldReturnValidBackup: false,
		ctx:                     ctx,
	}

	_, stdErr, err := captureStderr(func() {
		service.Run(ctx)
	})

	require.NoError(t, err)
	logMsg := "could not get scheduled backup jobs for backup type BigQuery"
	if !strings.Contains(strings.TrimSpace(stdErr), logMsg) {
		t.Errorf("Run should write log message %q but it logged\n\t%s", logMsg, stdErr)
	}
}

func TestJobStatusService_WithValidCloudStorageJob(t *testing.T) {

	// On the fake response for the transferOperations-List method alter one out of many transfer operations statuses.
	// The other transferOperations returned are in the success state.
	tests := []struct {
		// TransferOperationStatus is the status of one out of many Transfer
		TransferOperationInputValues map[string]string
		expectedJobStatus            repository.JobStatus
		expectedBackupStatus         repository.BackupStatus
	}{
		// All operations successful. The job status should be FinishedOk, whereas the Backup status remains in Prepared
		// state because the test case uses a Snapshot with an interval != 0
		{map[string]string{"OperationStatus": "SUCCESS", "OperationDone": "true"},
			repository.FinishedOk, repository.Prepared},
		// An operation is not Done yet and contains an error. This should flag the job as FinishedError.
		{map[string]string{"OperationStatus": "FAILED", "OperationDone": "false", "Error": errorMsg},
			repository.FinishedError, repository.Prepared},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("JobStatus on job with one run status %s done %s ", test.TransferOperationInputValues["OperationStatus"], test.TransferOperationInputValues["OperationDone"]), func(t *testing.T) {

			// ARRANGE
			// Simulate a job with three transferOperations
			mockResponse, _ := SimpleResponseBodyFromTemplate(mockResponseBody, test.TransferOperationInputValues, http.StatusOK)
			httpMockHandler.Register(
				mock.NewMockedHTTPRequestWithQuery("GET", "/v1/transferOperations", mockResponse, map[string]string{
					"fields":       "operations.done,operations.response,operations.error",
					"filter":       `{ "project_id": "` + targetProjectID + `", "jobNames": [ "` + jobHandlerTransferJobName + `" ] }`,
					"pretty_print": "false",
					"alt":          "json",
				}))

			httpMockHandler.Start()
			defer httpMockHandler.Stop()

			ctx := context.Background()
			backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
			require.NoErrorf(t, err, "BackupRepository should be instantiate")

			jobRepository, err := repository.NewJobRepository(ctx, secret.NewEnvSecretProvider())
			require.NoErrorf(t, err, "JobRepository should be instantiate")

			configProvider := &MockImpersonatedTokenConfigProvider{
				TargetPrincipal: "backup-tooling@local-test-prod.iam.gserviceaccount.com",
				Error:           nil,
			}

			service, err := newJobStatusService(ctx, configProvider, secret.NewEnvSecretProvider())
			require.NoErrorf(t, err, "JobStatusService should be instantiate")

			_, err = backupRepository.AddBackup(ctx, &repository.Backup{
				ID:            statusServiceBackupID,
				Status:        repository.Prepared,
				SourceProject: "local-ability",
				Strategy:      repository.Snapshot,
				Type:          repository.CloudStorage,
				SinkOptions: repository.SinkOptions{
					TargetProject: targetProjectID,
					Sink:          "uuid-5678-123456",
					Region:        "europe-west1",
					StorageClass:  repository.Coldline.String(),
				},
				SnapshotOptions: repository.SnapshotOptions{
					FrequencyInHours: 24,
				},
				BackupOptions: repository.BackupOptions{
					CloudStorageOptions: repository.CloudStorageOptions{
						Bucket:      "demo_delete_me_backup_target",
						IncludePath: nil,
						ExcludePath: nil,
					},
				},
				EntityAudit: repository.EntityAudit{
					CreatedTimestamp: time.Now(),
				},
			})
			require.NoError(t, err)
			defer func() { deleteBackup(statusServiceBackupID) }()

			job := repository.Job{
				ID:       statusServiceJobID,
				Source:   "amount_budget_plan",
				Status:   repository.NotScheduled,
				BackupID: statusServiceBackupID,
				Type:     repository.CloudStorage,
				ForeignJobID: repository.ForeignJobID{
					CloudStorageID: "transferJobs/ruzzelzuzzel",
				},
			}
			err = jobRepository.AddJob(ctx, &job)
			require.NoError(t, err, "should add new job")
			err = jobRepository.PatchJobStatus(ctx, repository.JobPatch{ID: statusServiceJobID, Status: repository.Scheduled, ForeignJobID: repository.ForeignJobID{CloudStorageID: "transferJobs/ruzzelzuzzel"}})
			require.NoError(t, err)
			defer func() { jobRepository.DeleteJob(ctx, statusServiceJobID) }()

			_, stdErr, err := captureStderr(func() {
				service.Run(ctx)
			})

			require.NoError(t, err)
			logMsg := "Checking status of 1 jobs"
			assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)

			updatedJob, err := jobRepository.GetJob(ctx, statusServiceJobID)
			require.NoError(t, err)
			assert.Equal(t, test.expectedJobStatus, updatedJob.Status)

			updatedBackup, err := backupRepository.GetBackup(ctx, statusServiceBackupID)
			require.NoError(t, err)
			assert.Equal(t, test.expectedBackupStatus, updatedBackup.Status)
		})
	}
}

func TestJobStatusService_WithValidJobValidBackup(t *testing.T) {
	httpMockHandler.Start()
	defer httpMockHandler.Stop()

	ctx := context.Background()
	backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
	require.NoErrorf(t, err, "BackupRepository should be instantiate")

	jobRepository, err := repository.NewJobRepository(ctx, secret.NewEnvSecretProvider())
	require.NoErrorf(t, err, "JobRepository should be instantiate")

	configProvider := &MockImpersonatedTokenConfigProvider{
		TargetPrincipal: "backup-tooling@local-test-prod.iam.gserviceaccount.com",
		Error:           nil,
	}

	service, err := newJobStatusService(ctx, configProvider, secret.NewEnvSecretProvider())
	require.NoErrorf(t, err, "JobStatusService should be instantiate")

	_, err = backupRepository.AddBackup(ctx, &repository.Backup{
		ID:            statusServiceBackupID,
		Status:        repository.Prepared,
		SourceProject: "local-ability",
		Strategy:      repository.Snapshot,
		Type:          repository.BigQuery,
		SinkOptions: repository.SinkOptions{
			TargetProject: targetProjectID,
			Sink:          "uuid-5678-123456",
			Region:        "europe-west1",
			StorageClass:  repository.Nearline.String(),
		},
		BackupOptions: repository.BackupOptions{
			BigQueryOptions: repository.BigQueryOptions{"demo_delete_me_backup_target", []string{"gcp_billing_budget_amount_plan"}, []string{}},
		},
		EntityAudit: repository.EntityAudit{
			CreatedTimestamp: time.Now(),
		},
	})
	require.NoError(t, err)
	defer func() { deleteBackup(statusServiceBackupID) }()

	job := repository.Job{
		ID:       statusServiceJobID,
		Source:   "amount_budget_plan",
		Status:   repository.NotScheduled,
		BackupID: statusServiceBackupID,
		Type:     repository.BigQuery,
	}
	err = jobRepository.AddJob(ctx, &job)
	require.NoError(t, err, "should add new job")
	err = jobRepository.PatchJobStatus(ctx, repository.JobPatch{ID: statusServiceJobID, Status: repository.Scheduled, ForeignJobID: repository.ForeignJobID{BigQueryID: "extractJobId"}})
	require.NoError(t, err)
	defer func() { jobRepository.DeleteJob(ctx, statusServiceJobID) }()

	_, stdErr, err := captureStderr(func() {
		service.Run(ctx)
	})

	require.NoError(t, err)
	logMsg := "Checking status of 1 jobs"
	assert.Containsf(t, strings.TrimSpace(stdErr), logMsg, "Run should write log message %q but it logged\n\t%s", logMsg, stdErr)

	updatedJob, err := jobRepository.GetJob(ctx, statusServiceJobID)
	require.NoError(t, err)
	assert.Equal(t, repository.FinishedOk, updatedJob.Status)

	updatedBackup, err := backupRepository.GetBackup(ctx, statusServiceBackupID)
	require.NoError(t, err)
	assert.Equal(t, repository.Prepared, updatedBackup.Status)
}

func SimpleResponseBodyFromTemplate(bodyTemplate string, values map[string]string, statusCode int) (string, error) {
	bodyTmpl, err := template.New("MockResponseBody").Parse(bodyTemplate)
	if err != nil {
		return "", err
	}

	var bodyBuf bytes.Buffer
	err = bodyTmpl.Execute(&bodyBuf, values)
	if err != nil {
		return "", err
	}

	envelopeTmpl, err := template.New("MockResponseEnvelope").Parse(`HTTP/1.0 {{ .StatusCode }} {{ .StatusText }}
Content-Length: {{ .ContentLength }}
Content-Type: application/json; charset=UTF-8

{{ .Body }}
`)
	body := bodyBuf.String()
	templateData := map[string]string{
		"StatusCode":    strconv.Itoa(statusCode),
		"StatusText":    http.StatusText(statusCode),
		"ContentLength": strconv.Itoa(len(body)),
		"Body":          body,
	}

	var responseBuf bytes.Buffer
	envelopeTmpl.Execute(&responseBuf, templateData)

	return responseBuf.String(), err
}

var mockResponseBody = `{
  "operations": [
    {
      "name": "` + jobHandlerTransferOperationID1 + `",
      "metadata": {
        "@type": "type.googleapis.com/google.storagetransfer.v1.TransferOperation",
        "name": "` + jobHandlerTransferOperationID1 + `",
        "projectId": "` + targetProjectID + `",
        "transferSpec": {
          "gcsDataSource": {
            "bucketName": "demo_deleteme"
          },
          "gcsDataSink": {
            "bucketName": "bkp_gcs_76efa173-e2cd-42db-be6a-c0a63ddcf215"
          },
          "objectConditions": {}
        },
        "startTime": "2023-06-01T11:01:06.037707214Z",
        "endTime": "2023-06-01T11:01:16.813138233Z",
        "status": "{{ .OperationStatus }}",
        "counters": {
          "objectsFromSourceSkippedBySync": "1",
          "bytesFromSourceSkippedBySync": "13"
        },
        "transferJobName": "` + jobHandlerTransferJobName + `"
      },
      "done": {{ .OperationDone }},
{{ if ne (index . "Error") "" }}
      "error": {{ .Error }}
{{ else }}
      "response": {
        "@type": "type.googleapis.com/google.protobuf.Empty"
      }
{{ end }}
    },
    {
      "name": "` + jobHandlerTransferOperationID2 + `",
      "metadata": {
        "@type": "type.googleapis.com/google.storagetransfer.v1.TransferOperation",
        "name": "` + jobHandlerTransferOperationID2 + `",
        "projectId": "` + targetProjectID + `",
        "transferSpec": {
          "gcsDataSource": {
            "bucketName": "demo_deleteme"
          },
          "gcsDataSink": {
            "bucketName": "bkp_gcs_76efa173-e2cd-42db-be6a-c0a63ddcf215"
          },
          "objectConditions": {}
        },
        "startTime": "2023-06-01T10:59:56.555350100Z",
        "endTime": "2023-06-01T11:00:07.627710108Z",
        "status": "SUCCESS",
        "counters": {
          "objectsFromSourceSkippedBySync": "1",
          "bytesFromSourceSkippedBySync": "13"
        },
        "transferJobName": "` + jobHandlerTransferJobName + `"
      },
      "done": true,
      "response": {
        "@type": "type.googleapis.com/google.protobuf.Empty"
      }
    },
    {
      "name": "` + jobHandlerTransferOperationID3 + `",
      "metadata": {
        "@type": "type.googleapis.com/google.storagetransfer.v1.TransferOperation",
        "name": "` + jobHandlerTransferOperationID3 + `",
        "projectId": "` + targetProjectID + `",
        "transferSpec": {
          "gcsDataSource": {
            "bucketName": "demo_deleteme"
          },
          "gcsDataSink": {
            "bucketName": "bkp_gcs_76efa173-e2cd-42db-be6a-c0a63ddcf215"
          },
          "objectConditions": {}
        },
        "startTime": "2023-06-01T10:53:16.149283504Z",
        "endTime": "2023-06-01T10:53:27.258066918Z",
        "status": "SUCCESS",
        "counters": {
          "objectsFromSourceSkippedBySync": "1",
          "bytesFromSourceSkippedBySync": "13"
        },
        "transferJobName": "` + jobHandlerTransferJobName + `"
      },
      "done": true,
      "response": {
        "@type": "type.googleapis.com/google.protobuf.Empty"
      }
    },
    {
      "name": "` + jobHandlerTransferOperationID4 + `",
      "metadata": {
        "@type": "type.googleapis.com/google.storagetransfer.v1.TransferOperation",
        "name": "` + jobHandlerTransferOperationID4 + `",
        "projectId": "` + targetProjectID + `",
        "transferSpec": {
          "gcsDataSource": {
            "bucketName": "demo_deleteme"
          },
          "gcsDataSink": {
            "bucketName": "bkp_gcs_76efa173-e2cd-42db-be6a-c0a63ddcf215"
          },
          "objectConditions": {}
        },
        "startTime": "2023-06-01T09:34:07.553370479Z",
        "endTime": "2023-06-01T09:34:29.339166829Z",
        "status": "SUCCESS",
        "counters": {
          "objectsFoundFromSource": "1",
          "bytesFoundFromSource": "13",
          "objectsCopiedToSink": "1",
          "bytesCopiedToSink": "13"
        },
        "transferJobName": "` + jobHandlerTransferJobName + `"
      },
      "done": true,
      "response": {
        "@type": "type.googleapis.com/google.protobuf.Empty"
      }
    }
  ]
}`

var errorMsg = `{ "code": 1234, "message": "An error occurred", "details": [ { "id": 1234, "@type": "types.example.com/standard/id" } ] }`
