package processor

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/iam"

	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/http/auth"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/ottogroup/penelope/pkg/service/bigquery"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

const sinkSTSAccountScheme = "project-%s@storage-transfer-service.iam.gserviceaccount.com"

type CreatingProcessorFactory interface {
	CreateProcessor(ctxIn context.Context) (Operation[requestobjects.CreateRequest, requestobjects.BackupResponse], error)
}

// creatingProcessorFactory create Process for Creating
type creatingProcessorFactory struct {
	backupProvider           provider.SinkGCPProjectProvider
	tokenSourceProvider      impersonate.TargetPrincipalForProjectProvider
	credentialsProvider      secret.SecretProvider
	sourceGCPProjectProvider provider.SourceGCPProjectProvider
}

func NewCreatingProcessorFactory(backupProvider provider.SinkGCPProjectProvider, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider, sourceGCPProjectProvider provider.SourceGCPProjectProvider) CreatingProcessorFactory {
	return &creatingProcessorFactory{
		backupProvider:           backupProvider,
		tokenSourceProvider:      tokenSourceProvider,
		credentialsProvider:      credentialsProvider,
		sourceGCPProjectProvider: sourceGCPProjectProvider,
	}
}

// CreateProcessor return instance of Operations for Creating
func (c *creatingProcessorFactory) CreateProcessor(ctxIn context.Context) (Operation[requestobjects.CreateRequest, requestobjects.BackupResponse], error) {
	ctx, span := trace.StartSpan(ctxIn, "(*CreatingProcessorFactory).CreateProcessor")
	defer span.End()

	backupRepository, err := repository.NewBackupRepository(ctx, c.credentialsProvider)
	if err != nil {
		return nil, fmt.Errorf("could not create backup repository: %s", err)
	}

	jobRepository, err := repository.NewJobRepository(ctx, c.credentialsProvider)
	if err != nil {
		glog.Error(err)
		return &creatingProcessor{}, err
	}

	return &creatingProcessor{
		BackupRepository:         backupRepository,
		JobRepository:            jobRepository,
		backupProvider:           c.backupProvider,
		tokenSourceProvider:      c.tokenSourceProvider,
		sourceGCPProjectProvider: c.sourceGCPProjectProvider,
	}, nil
}

type creatingProcessor struct {
	BackupRepository         repository.BackupRepository
	JobRepository            repository.JobRepository
	backupProvider           provider.SinkGCPProjectProvider
	sourceGCPProjectProvider provider.SourceGCPProjectProvider
	tokenSourceProvider      impersonate.TargetPrincipalForProjectProvider
}

func (b *creatingProcessor) Process(ctxIn context.Context, args *Argument[requestobjects.CreateRequest]) (requestobjects.BackupResponse, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*creatingProcessor).Process")
	defer span.End()

	var request requestobjects.CreateRequest = args.Request

	if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.Creating, request.Project) {
		return requestobjects.BackupResponse{}, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.Creating.String(), args.Principal.User.Email, request.Project)
	}

	backup, err := b.prepareBackupFromRequest(ctx, request)
	if err != nil {
		return requestobjects.BackupResponse{}, err
	}
	var impl creatingProcessorImpl
	if repository.BigQuery.EqualTo(request.Type) {
		impl, err = b.createBigQueryImpl(ctx, request)
		if err != nil {
			return requestobjects.BackupResponse{}, err
		}
	}
	if repository.CloudStorage.EqualTo(request.Type) {
		impl, err = b.createCloudStorageImpl(ctx, request)
		if err != nil {
			return requestobjects.BackupResponse{}, err
		}
	}
	defer impl.close(ctx)

	processedBackup, err := impl.process(ctx, backup)
	if err != nil {
		return requestobjects.BackupResponse{}, err
	}
	sourceGCPProject, err := b.sourceGCPProjectProvider.GetSourceGCPProject(ctx, request.Project)
	if err != nil {
		return requestobjects.BackupResponse{}, err
	}
	return mapBackupToResponse(processedBackup, nil, sourceGCPProject), nil
}

func (b *creatingProcessor) prepareBackupFromRequest(ctxIn context.Context, request requestobjects.CreateRequest) (*repository.Backup, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*creatingProcessor).prepareBackupFromRequest")
	defer span.End()

	id := generateNewID()

	storageClass := request.TargetOptions.StorageClass
	if storageClass == "" {
		storageClass = config.DefaultBucketStorageClass.MustGet()
	} else {
		for _, r := range StorageClasses {
			if strings.EqualFold(r.String(), storageClass) {
				storageClass = r.String()
				break
			}
		}
	}

	strategy := request.Strategy
	for _, r := range repository.Strategies {
		if strings.EqualFold(r.String(), strategy) {
			strategy = r.String()
			break
		}
	}

	region := request.TargetOptions.Region
	for _, r := range Regions {
		if strings.EqualFold(r.String(), region) {
			region = r.String()
			break
		}
	}

	dualRegion := request.TargetOptions.DualRegion
	for _, r := range Regions {
		if strings.EqualFold(r.String(), dualRegion) {
			dualRegion = r.String()
			break
		}
	}

	sourceProject := request.Project
	targetProject, err := b.backupProvider.GetSinkGCPProjectID(ctx, sourceProject)
	if err != nil {
		return nil, err
	}

	if !(repository.BigQuery.EqualTo(request.Type) || repository.CloudStorage.EqualTo(request.Type)) {
		return nil, fmt.Errorf("can not process request for type %s", request.Type)
	}
	var suffix string
	if repository.CloudStorage.EqualTo(request.Type) {
		suffix = "gcs"
	}
	if repository.BigQuery.EqualTo(request.Type) {
		suffix = "bq"
	} // little smell
	sinkName := fmt.Sprintf("bkp_%s_%s", suffix, id)
	backup := repository.Backup{
		ID:            id,
		Description:   request.Description,
		Status:        repository.NotStarted,
		Type:          repository.BackupType(request.Type),
		Strategy:      repository.Strategy(strategy),
		SourceProject: sourceProject,
		SinkOptions: repository.SinkOptions{
			TargetProject: targetProject,
			Region:        region,
			DualRegion:    dualRegion,
			Sink:          sinkName,
			StorageClass:  storageClass,
			ArchiveTTM:    request.TargetOptions.ArchiveTTM,
		},
		SnapshotOptions: repository.SnapshotOptions{
			LifetimeInDays:   request.SnapshotOptions.LifetimeInDays,
			FrequencyInHours: request.SnapshotOptions.FrequencyInHours,
		},
		MirrorOptions: repository.MirrorOptions{
			LifetimeInDays: request.MirrorOptions.LifetimeInDays,
		},
		BackupOptions: repository.BackupOptions{
			BigQueryOptions: repository.BigQueryOptions{
				Dataset:        request.BigQueryOptions.Dataset,
				Table:          request.BigQueryOptions.Table,
				ExcludedTables: request.BigQueryOptions.ExcludedTables,
			},
			CloudStorageOptions: repository.CloudStorageOptions{
				Bucket:      request.GCSOptions.Bucket,
				ExcludePath: normalizePath(request.GCSOptions.ExcludePath),
				IncludePath: normalizePath(request.GCSOptions.IncludePath),
			},
		},
		EntityAudit: repository.EntityAudit{
			CreatedTimestamp: time.Now(),
		},
		RecoveryPointObjective: request.RecoveryPointObjective,
		RecoveryTimeObjective:  request.RecoveryTimeObjective,
	}

	return &backup, nil
}

func normalizePath(pathList []string) []string {
	var updatedPathList []string

	for _, path := range pathList {
		if len(path) == 0 {
			continue
		}
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
		updatedPathList = append(updatedPathList, strings.TrimSpace(path))
	}

	return updatedPathList
}

func (b *creatingProcessor) createBigQueryImpl(ctxIn context.Context, request requestobjects.CreateRequest) (creatingProcessorImpl, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*creatingProcessor).createBigQueryImpl")
	defer span.End()

	sourceProject := request.Project
	targetProject, err := b.backupProvider.GetSinkGCPProjectID(ctx, request.Project)
	if err != nil {
		return nil, err
	}

	bq, err := bigquery.NewBigQueryClient(ctx, b.tokenSourceProvider, sourceProject, targetProject)
	if err != nil {
		return nil, err
	}

	gcsClient, err := gcs.NewCloudStorageClient(ctx, b.tokenSourceProvider, targetProject)
	if err != nil {
		return nil, err
	}

	bigQueryProcessor := &bigQueryProcessorImpl{
		BackupRepository: b.BackupRepository,
		JobRepository:    b.JobRepository,
		BigQuery:         bq,
		CloudStorage:     gcsClient,
	}
	return bigQueryProcessor, nil
}

func (b *creatingProcessor) createCloudStorageImpl(ctxIn context.Context, request requestobjects.CreateRequest) (creatingProcessorImpl, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*creatingProcessor).createCloudStorageImpl")
	defer span.End()

	targetProject, err := b.backupProvider.GetSinkGCPProjectID(ctx, request.Project)
	if err != nil {
		return nil, err
	}

	gcsClient, err := gcs.NewCloudStorageClient(ctx, b.tokenSourceProvider, targetProject)
	if err != nil {
		return nil, err
	}

	cloudStorageProcessor := &cloudStorageProcessorImpl{
		BackupRepository: b.BackupRepository,
		CloudStorage:     gcsClient,
	}
	return cloudStorageProcessor, nil
}

type creatingProcessorImpl interface {
	process(ctxIn context.Context, backup *repository.Backup) (*repository.Backup, error)
	close(context.Context)
}

type bigQueryProcessorImpl struct {
	BackupRepository repository.BackupRepository
	JobRepository    repository.JobRepository
	BigQuery         bigquery.Client
	CloudStorage     gcs.CloudStorageClient
}

type cloudStorageProcessorImpl struct {
	BackupRepository repository.BackupRepository
	CloudStorage     gcs.CloudStorageClient
}

func (b *bigQueryProcessorImpl) close(ctxIn context.Context) {
	ctx, span := trace.StartSpan(ctxIn, "(*bigQueryProcessorImpl).close")
	defer span.End()

	b.CloudStorage.Close(ctx)
}

func (b *bigQueryProcessorImpl) process(ctxIn context.Context, backup *repository.Backup) (*repository.Backup, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*bigQueryProcessorImpl).process")
	defer span.End()

	err := b.validateSource(ctx, backup)
	if err != nil {
		return nil, err
	}
	err = validateIntersection(ctx, backup)
	if err != nil {
		return nil, err
	}
	backup, err = b.BackupRepository.AddBackup(ctx, backup)
	if err != nil {
		return nil, err
	}

	err = prepareSink(ctx, b.CloudStorage, backup)
	return backup, err
}

func (b *bigQueryProcessorImpl) validateSource(ctxIn context.Context, backup *repository.Backup) error {
	ctx, span := trace.StartSpan(ctxIn, "(*bigQueryProcessorImpl).validateSource")
	defer span.End()

	exists, err := b.BigQuery.DoesDatasetExists(ctx, backup.SourceProject, backup.Dataset)
	if err != nil {
		return err
	}

	if !exists {
		glog.Errorf("dataset %s not found in project %s", backup.Dataset, backup.SourceProject)
		return fmt.Errorf("dataset %s not found in project %s", backup.Dataset, backup.SourceProject)
	}

	for _, table := range backup.Table {
		table, err := b.BigQuery.GetTable(ctx, backup.SourceProject, backup.Dataset, table)
		if err != nil {
			return err
		}

		if !exists {
			glog.Errorf("table %s not found in dataset %s and project %s", table.Name, backup.Dataset, backup.SourceProject)
			return fmt.Errorf("table %s not found in dataset %s and project %s", table.Name, backup.Dataset, backup.SourceProject)
		}
	}

	return nil
}

func (c *cloudStorageProcessorImpl) close(ctxIn context.Context) {
	ctx, span := trace.StartSpan(ctxIn, "(*bigQueryProcessorImpl).close")
	defer span.End()

	c.CloudStorage.Close(ctx)
}

func (c *cloudStorageProcessorImpl) process(ctxIn context.Context, backup *repository.Backup) (*repository.Backup, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*cloudStorageProcessorImpl).process")
	defer span.End()

	err := c.validateSource(ctx, backup)
	if err != nil {
		return nil, err
	}
	err = validateIntersection(ctx, backup)
	if err != nil {
		return nil, err
	}
	backup, err = c.BackupRepository.AddBackup(ctx, backup)
	if err != nil {
		return nil, err
	}

	err = prepareSink(ctx, c.CloudStorage, backup)
	return backup, err
}

func (c *cloudStorageProcessorImpl) validateSource(ctxIn context.Context, backup *repository.Backup) error {
	ctx, span := trace.StartSpan(ctxIn, "(*cloudStorageProcessorImpl).validateSource")
	defer span.End()

	exists, err := c.CloudStorage.DoesBucketExist(ctx, backup.SourceProject, backup.CloudStorageOptions.Bucket)
	if err != nil {
		return errors.Wrap(err, "operation DoesBucketExist failed")
	}

	if !exists {
		glog.Errorf("bucket %s not found in project %s", backup.Bucket, backup.SourceProject)
		return fmt.Errorf("dataset %s not found in project %s", backup.Bucket, backup.SourceProject)
	}

	return nil
}

func prepareSink(ctxIn context.Context, cloudStorageClient gcs.CloudStorageClient, backup *repository.Backup) error {
	ctx, span := trace.StartSpan(ctxIn, "prepareSink")
	defer span.End()

	exists, err := cloudStorageClient.DoesBucketExist(ctx, backup.TargetProject, backup.Sink)
	if err != nil {
		return errors.Wrap(err, "operation DoesBucketExist failed")
	}

	if !exists {
		var lifetimeInDays uint = 0
		if backup.Strategy.EqualTo(repository.Mirror.String()) {
			lifetimeInDays = backup.MirrorOptions.LifetimeInDays
		}
		if backup.Strategy.EqualTo(repository.Snapshot.String()) {
			lifetimeInDays = backup.SnapshotOptions.LifetimeInDays
		}

		err = cloudStorageClient.CreateBucket(ctx, gcs.CloudStorageBucket{
			backup.TargetProject, backup.Sink, backup.Region, backup.DualRegion, backup.StorageClass, lifetimeInDays, backup.ArchiveTTM, gcs.NewLabels(pascalCaseToSnakeCase(backup.Type.String())),
		})
		if err != nil {
			return err
		}
		err = cloudStorageClient.CreateObject(ctx, backup.Sink, fmt.Sprintf("%s/THIS_TRASHCAN_CONTAINS_DELETED_OBJECTS_FROM_SOURCE", backup.GetTrashcanPath()), "")
		if err != nil {
			return err
		}
		if backup.Type != repository.CloudStorage {
			return nil
		}
		project, err := cloudStorageClient.GetProject(ctx, backup.TargetProject)
		if err != nil {
			return err
		}
		projectNumber := strings.ReplaceAll(project.Name, "projects/", "")
		bucketPolicy := &iam.Policy{}
		// Storage Transfer Service needs to write to and read from the sink bucket
		// based on https://cloud.google.com/storage-transfer/docs/sink-cloud-storage#required_permissions
		storageTransferGSABinding := "serviceAccount:" + fmt.Sprintf(sinkSTSAccountScheme, projectNumber)
		bucketPolicy.Add(storageTransferGSABinding, "roles/storage.legacyBucketWriter")
		bucketPolicy.Add(storageTransferGSABinding, "roles/storage.legacyBucketReader")
		err = cloudStorageClient.SetBucketIAMPolicy(ctx, backup.Sink, bucketPolicy)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateIntersection(ctxIn context.Context, backup *repository.Backup) error {
	if backup.Type == repository.BigQuery && hasIntersection(backup.Table, backup.ExcludedTables) {
		return fmt.Errorf("bigquery tables have intersections: %v, %v", backup.Table, backup.ExcludedTables)
	}
	if backup.Type == repository.CloudStorage && hasIntersection(backup.IncludePath, backup.ExcludePath) {
		return fmt.Errorf("bucket paths have intersections: %v, %v", backup.IncludePath, backup.ExcludePath)
	}
	return nil
}

func hasIntersection(a, b []string) bool {
	var c []string
	m := make(map[string]bool)

	for _, item := range a {
		m[item] = true
	}

	for _, item := range b {
		if _, ok := m[item]; ok {
			c = append(c, item)
		}
	}
	return len(c) > 0
}

func pascalCaseToSnakeCase(s string) string {
	re := regexp.MustCompile("([A-Z][a-z0-9]*)")
	snake := re.ReplaceAllStringFunc(s, func(sub string) string {
		return "_" + strings.ToLower(sub)
	})
	snake = strings.TrimLeft(snake, "_")
	return snake
}
