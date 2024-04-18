package rest

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ottogroup/penelope/pkg/http/auth/model"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetting_WithUnknownResponse(t *testing.T) {
	defer httpMockHandler.Stop()
	httpMockHandler.Start()

	mockBackupProvider := &mockBackupProvider{
		Backup: "gcp-project-backup",
		Error:  nil,
	}

	mockTokenConfigProvider := &MockImpersonatedTokenConfigProvider{
		TargetPrincipal: "backup-project@local-test-prod.iam.gserviceaccount.com",
		Error:           nil,
	}

	s := restAPIFactoryWithRealFactory(t, []model.ProjectRoleBinding{{
		Role:    model.Viewer,
		Project: defaultProjectID,
	}}, mockBackupProvider, mockTokenConfigProvider, mockSourceTokenProvider)
	defer s.Close()

	httpMockHandler.RegisterLocalServer(s.URL)

	resp, respString := get(t, s, buildBackupRequestPath()+"/noid")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, `could not handle request because of: no backup with id "noid" found`, respString)
}

func TestGetting_WithKnownResponse(t *testing.T) {
	defer httpMockHandler.Stop()
	httpMockHandler.Start()

	mockBackupProvider := &mockBackupProvider{
		Backup: "gcp-project-backup",
		Error:  nil,
	}

	mockTokenConfigProvider := &MockImpersonatedTokenConfigProvider{
		TargetPrincipal: "backup-project@local-test-prod.iam.gserviceaccount.com",
		Error:           nil,
	}

	s := restAPIFactoryWithRealFactory(t, []model.ProjectRoleBinding{{
		Role:    model.Viewer,
		Project: defaultProjectID,
	}}, mockBackupProvider, mockTokenConfigProvider, mockSourceTokenProvider)
	defer s.Close()

	httpMockHandler.RegisterLocalServer(s.URL)

	ctx := context.Background()
	backupRepository, err := repository.NewBackupRepository(ctx, secret.NewEnvSecretProvider())
	require.NoError(t, err, "BackupRepository should be instantiate")

	jobRepository, err := repository.NewJobRepository(ctx, secret.NewEnvSecretProvider())
	require.NoErrorf(t, err, "JobRepository should be instantiate")

	backupID := "test-backup-id"
	_, err = backupRepository.AddBackup(ctx, &repository.Backup{
		ID:            backupID,
		Status:        repository.Prepared,
		SourceProject: defaultProjectID,
		Strategy:      repository.Snapshot,
		Type:          repository.BigQuery,
		SinkOptions: repository.SinkOptions{
			TargetProject: "local-ability-backup",
			Sink:          "uuid-5678-123456",
			Region:        "europe-west1",
			StorageClass:  "NEARLINE",
			ArchiveTTM:    10203,
		},
		BackupOptions: repository.BackupOptions{
			BigQueryOptions: repository.BigQueryOptions{"demo_delete_me_backup_target", []string{"gcp_billing_budget_amount_plan"}, []string{}},
		},
		EntityAudit: repository.EntityAudit{
			CreatedTimestamp: time.Now(),
		},
	})
	require.NoError(t, err, "should add new backup")

	defer func() { deleteBackup(backupID) }()

	jobID := "test-job-id"
	job := repository.Job{
		ID:       jobID,
		Source:   "amount_budget_plan",
		Status:   repository.NotScheduled,
		BackupID: backupID,
		Type:     repository.BigQuery,
	}
	err = jobRepository.AddJob(ctx, &job)
	require.NoError(t, err, "should add new job")
	defer func() { jobRepository.DeleteJob(ctx, jobID) }()

	resp, respString := get(t, s, buildBackupRequestPath()+"/"+backupID)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, respString)
	assert.Contains(t, respString, backupID)
	assert.Contains(t, respString, jobID)
}
