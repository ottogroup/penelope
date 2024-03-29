package gcs

import (
	"context"
	"errors"
	"fmt"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/repository"
	"go.opencensus.io/trace"
	"google.golang.org/api/googleapi"
	gimpersonate "google.golang.org/api/impersonate"
	"google.golang.org/api/option"
	"google.golang.org/api/storagetransfer/v1"
	"net/http"
	"reflect"
	"time"
)

// TransferJobHandler represent api to deal with transfer jobs
type TransferJobHandler struct {
	client                  CloudStorageClient
	targetPrincipalProvider impersonate.TargetPrincipalForProjectProvider
}

// NewTransferJobHandler create new TransferJobHandler
func NewTransferJobHandler(ctxIn context.Context, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, targetProjectID string) (*TransferJobHandler, error) {
	ctx, span := trace.StartSpan(ctxIn, "NewTransferJobHandler")
	defer span.End()

	storageClient, err := NewCloudStorageClient(ctx, tokenSourceProvider, targetProjectID)
	if err != nil || storageClient == nil || !storageClient.IsInitialized(ctx) {
		return &TransferJobHandler{}, fmt.Errorf("can not create instance of TransferJobHandler with unititialized Client")
	}

	return &TransferJobHandler{client: storageClient, targetPrincipalProvider: tokenSourceProvider}, nil
}

// Close terminates terminates all resources in use
func (t *TransferJobHandler) Close(ctxIn context.Context) {
	t.client.Close(ctxIn)
}

func (t *TransferJobHandler) createClient(ctxIn context.Context, targetProjectID string) (*storagetransfer.Service, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*TransferJobHandler).createClient")
	defer span.End()

	var options []option.ClientOption

	target, delegates, err := t.targetPrincipalProvider.GetTargetPrincipalForProject(ctx, targetProjectID)
	if err != nil {
		return nil, err
	}

	ts, err := gimpersonate.CredentialsTokenSource(ctx, gimpersonate.CredentialsConfig{
		TargetPrincipal: target,
		Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
		Delegates:       delegates,
	})
	if err != nil {
		return nil, err
	}
	options = append(options, option.WithTokenSource(ts))

	if config.UseDefaultHttpClient.GetBoolOrDefault(false) {
		options = append(options, option.WithHTTPClient(http.DefaultClient))
	}
	storageTransferService, err := storagetransfer.NewService(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create new oauth2 client: %s", err)
	}

	return storageTransferService, nil
}

// ReuseTransferJob takes a StorageTransferJob with the id transferJobID, patches (changes) the configuration so it meets
// the given backup specification and then starts a run.
func (t *TransferJobHandler) ReuseTransferJob(ctxIn context.Context, srcProjectID, targetProjectID, srcBucket, targetBucket string, includePath, excludePath []string, transferJobID repository.TransferJobID) (string, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*TransferJobHandler).ReuseTransferJob")
	defer span.End()

	storageTransferService, err := t.createClient(ctx, targetProjectID)
	if err != nil {
		return "", fmt.Errorf("failed to create new oauth2 client: %s", err)
	}

	// Check if the job for reuse exists.
	reusableJob, err := storageTransferService.TransferJobs.Get(transferJobID.String(), targetProjectID).Do()
	if err != nil {
		return "", fmt.Errorf("error reusing transfer job: %s", err)
	}
	// If the projectID does not match, it is not possible to patch the job, because changing the projectID is not allowed.
	if reusableJob.ProjectId != targetProjectID {
		return "", fmt.Errorf("error reusing transfer job because its target project id %s differs from the configured %s", reusableJob.ProjectId, targetProjectID)
	}

	// Leave the targetProjectID blank, otherwise there is an API error.
	rb := newTransferJobObject(srcProjectID, srcBucket, "", targetBucket, includePath, excludePath)

	resp, err := storageTransferService.TransferJobs.Patch(transferJobID.String(), &storagetransfer.UpdateTransferJobRequest{
		ProjectId:                  targetProjectID,
		TransferJob:                rb,
		UpdateTransferJobFieldMask: "transfer_spec,status,description",
	}).Context(ctx).Do()

	if err != nil {
		return "", fmt.Errorf("error reusing transfer job: %s", err)
	}

	_, err = storageTransferService.TransferJobs.Run(transferJobID.String(), &storagetransfer.RunTransferJobRequest{
		ProjectId: targetProjectID,
	}).Context(ctx).Do()

	if err != nil {
		return "", fmt.Errorf("error starting run for re-used transfer job: %s", err)
	}

	return resp.Name, nil
}

// CreateTransferJob create new transfer job
func (t *TransferJobHandler) CreateTransferJob(ctxIn context.Context, srcProjectID, targetProjectID, srcBucket, targetBucket string, includePath, excludePath []string) (string, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*TransferJobHandler).CreateTransferJob")
	defer span.End()

	storageTransferService, err := t.createClient(ctx, targetProjectID)
	if err != nil {
		return "", fmt.Errorf("failed to create new oauth2 client: %s", err)
	}

	rb := newTransferJobObject(srcProjectID, srcBucket, targetProjectID, targetBucket, includePath, excludePath)

	resp, err := storageTransferService.TransferJobs.Create(rb).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("error creation transfer job: %s", err)
	}

	return resp.Name, nil
}

func newTransferJobObject(srcProjectID string, srcBucket string, targetProjectID string, targetBucket string, includePath []string, excludePath []string) *storagetransfer.TransferJob {
	appProjectID := config.GCPProjectId.GetOrDefault("")
	description := fmt.Sprintf("Job to transfer %s:%s to %s:%s. Triggered by BackupApp in project %s", srcProjectID, srcBucket, targetProjectID, targetBucket, appProjectID)

	now := time.Now()
	rb := &storagetransfer.TransferJob{
		ProjectId:   targetProjectID,
		Description: description,
		Status:      "ENABLED",
		Schedule: &storagetransfer.Schedule{
			ScheduleStartDate: &storagetransfer.Date{Year: int64(now.Year()), Month: int64(now.Month()), Day: int64(now.Day())},
			ScheduleEndDate:   &storagetransfer.Date{Year: int64(now.Year()), Month: int64(now.Month()), Day: int64(now.Day())},
		},
		TransferSpec: &storagetransfer.TransferSpec{
			GcsDataSink: &storagetransfer.GcsData{
				BucketName: targetBucket,
			},
			GcsDataSource: &storagetransfer.GcsData{
				BucketName: srcBucket,
			},
		},
	}

	objectConditions := &storagetransfer.ObjectConditions{}
	if len(includePath) > 0 {
		objectConditions.IncludePrefixes = includePath
	}
	if len(excludePath) > 0 {
		objectConditions.ExcludePrefixes = excludePath
	}
	rb.TransferSpec.ObjectConditions = objectConditions
	return rb
}

// GetStatusOfJob return actual status of transfer job
func (t *TransferJobHandler) GetStatusOfJob(ctxIn context.Context, targetProjectID, name string) (TransferJobState, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*TransferJobHandler).GetStatusOfJob")
	defer span.End()

	storageTransferService, err := t.createClient(ctx, targetProjectID)
	if err != nil {
		return StateUnspecified, err
	}

	filterValue := fmt.Sprintf(`{"project_id" : "%s", "job_names" : ["%s"]}`, targetProjectID, name)
	fields := []googleapi.Field{"operations.done", "operations.response", "operations.error"}
	operations, err := storageTransferService.TransferOperations.List("transferOperations", filterValue).Fields(fields...).Do()
	if err != nil {
		return StateUnspecified, fmt.Errorf("error listing transfer operations: %s", err)
	}

	if operations == nil || reflect.ValueOf(operations).IsNil() {
		return Pending, nil
	}

	for _, operation := range operations.Operations {
		if !operation.Done {
			if operation.Error != nil && operation.Error.Message != "" {
				return Failed, fmt.Errorf("transfer operation finished in failed state: %s", operation.Error.Message)
			}
			return Pending, nil
		}
	}

	return Done, nil
}

// DeleteTransferJob delete transfer job
// If job does not exist, it returns nil
func (t *TransferJobHandler) DeleteTransferJob(ctxIn context.Context, targetProjectID, transferJobID string) error {
	ctx, span := trace.StartSpan(ctxIn, "(*TransferJobHandler).DeleteTransferJob")
	defer span.End()

	storageTransferService, err := t.createClient(ctx, targetProjectID)
	if err != nil {
		return fmt.Errorf("failed to create new oauth2 client: %s", err)
	}

	_, err = storageTransferService.TransferJobs.Delete(transferJobID, targetProjectID).Do()
	var googleAPIErr *googleapi.Error
	if errors.As(err, &googleAPIErr) && googleAPIErr.Code == http.StatusNotFound {
		return nil
	}

	return err
}
