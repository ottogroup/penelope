package gcs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/iam"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/storage"
	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
	gimpersonate "google.golang.org/api/impersonate"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	providerLabel = "provider"
	purposeLabel  = "purpose"
	typeLabel     = "type"
)

type LabelsProvider interface {
	Labels() map[string]string
}

type labels struct {
	backupType string
}

func (l labels) Labels() map[string]string {
	return map[string]string{
		providerLabel: "penelope",
		purposeLabel:  "backup",
		typeLabel:     l.backupType,
	}
}

func NewLabels(backupType string) LabelsProvider {
	return labels{backupType: backupType}
}

type CloudStorageBucket struct {
	Project, Bucket, Location, DualLocation, StorageClass string
	LifetimeInDays, ArchiveTTM                            uint
	Labels                                                LabelsProvider
}

// CloudStorageClient define operations with the GCS
type CloudStorageClient interface {
	IsInitialized(ctxIn context.Context) bool
	DoesBucketExist(ctxIn context.Context, project string, bucket string) (bool, error)
	BucketUsageInBytes(ctxIn context.Context, project string, bucket string) (float64, error)
	CreateBucket(ctxIn context.Context, bucket CloudStorageBucket) error
	GetProject(ctxIn context.Context, projectID string) (*resourcemanagerpb.Project, error)
	SetBucketIAMPolicy(ctxIn context.Context, bucket string, policy *iam.Policy) error
	CreateObject(ctxIn context.Context, bucketName, objectName, content string) error
	DeleteBucket(ctxIn context.Context, bucket string) error
	DeleteObject(ctxIn context.Context, bucketName string, objectName string) error
	DeleteObjectsWithObjectMatch(ctxIn context.Context, bucketName string, prefix string, objectPattern *regexp.Regexp) (deleted int, err error)
	MoveObject(ctxIn context.Context, bucketName, oldObjectName, newObjectName string) error
	ReadObject(ctxIn context.Context, bucketName, objectName string) ([]byte, error)
	GetBuckets(ctxIn context.Context, project string) ([]string, error)
	Close(ctxIn context.Context)
	UpdateBucket(ctxIn context.Context, bucket string, lifetimeInDays uint, archiveTTM uint) error
	GetBucketDetails(ctxIn context.Context, bucket string) (*storage.BucketAttrs, error)
	DeleteObjectWithPrefix(ctxIn context.Context, bucket string, objectPrefixName string) error
}

// CloudStorageClientFactory creates a CloudStorageClient with the credentails for a specified project
type CloudStorageClientFactory interface {
	NewCloudStorageClient(targetProjectID string) (CloudStorageClient, error)
}

// defaultGcsClient defines client to interact with the GCS
type defaultGcsClient struct {
	client        *storage.Client
	metricClient  *monitoring.MetricClient
	projectClient *resourcemanager.ProjectsClient
}

func (c *defaultGcsClient) DeleteObjectWithPrefix(ctxIn context.Context, bucket string, objectPrefixName string) error {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).DeleteObjectWithPrefix")
	defer span.End()

	objectIterator := c.client.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: objectPrefixName})
	for {
		objAttrs, err := objectIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("DeleteObjectWithPrefix failed for bucket %s, prefix %s", bucket, objectPrefixName))
		}
		if err := c.client.Bucket(bucket).Object(objAttrs.Name).Delete(ctx); err != nil {
			return errors.Wrap(err, fmt.Sprintf("DeleteObjectWithPrefix failed for bucket %s, prefix %s", bucket, objectPrefixName))
		}
	}

	return nil
}

func (c *defaultGcsClient) GetProject(ctxIn context.Context, projectID string) (*resourcemanagerpb.Project, error) {
	_, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).GetProject")
	defer span.End()

	return c.projectClient.GetProject(ctxIn, &resourcemanagerpb.GetProjectRequest{Name: fmt.Sprintf("projects/%s", projectID)})
}

// Close terminates all resources in use
func (c *defaultGcsClient) Close(ctxIn context.Context) {
	_, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).Close")
	defer span.End()

	c.client.Close()
	c.metricClient.Close()
	c.projectClient.Close()
}

// NewCloudStorageClient create a new CloudStorageClient
func NewCloudStorageClient(ctxIn context.Context, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, targetProjectID string) (CloudStorageClient, error) {
	ctx, span := trace.StartSpan(ctxIn, "NewCloudStorageClient")
	defer span.End()

	if strings.EqualFold(targetProjectID, config.GCPProjectId.GetOrDefault("")) {
		return createCloudStorageClient(ctx)
	}
	return createImpersonatedCloudStorageClient(ctx, tokenSourceProvider, targetProjectID)
}

func createCloudStorageClient(ctxIn context.Context) (CloudStorageClient, error) {
	ctx, span := trace.StartSpan(ctxIn, "createCloudStorageClient")
	defer span.End()

	var storageOptions []option.ClientOption
	if config.UseDefaultHttpClient.GetBoolOrDefault(false) {
		storageOptions = append(storageOptions, option.WithHTTPClient(http.DefaultClient))
	}

	var monitoringOptions = []option.ClientOption{option.WithScopes(metricAPIScope)}
	if config.UseGrpcWithoutAuthentication.GetBoolOrDefault(false) {
		monitoringOptions = append(monitoringOptions, option.WithoutAuthentication(), option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	}

	var projectsClientOptions = []option.ClientOption{option.WithScopes(metricAPIScope)}
	if config.UseGrpcWithoutAuthentication.GetBoolOrDefault(false) {
		projectsClientOptions = append(projectsClientOptions, option.WithoutAuthentication(), option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	}

	client, err := storage.NewClient(ctx, storageOptions...)
	if err != nil {
		return &defaultGcsClient{}, fmt.Errorf("failed to create storage.Client: %v", err)
	}

	metricClient, err := monitoring.NewMetricClient(ctx, monitoringOptions...)
	if err != nil {
		return &defaultGcsClient{}, fmt.Errorf("failed to create monitoring.MetricClient: %v", err)
	}

	projectClient, err := resourcemanager.NewProjectsClient(ctx, projectsClientOptions...)
	if err != nil {
		return &defaultGcsClient{}, fmt.Errorf("failed to create resourcemanager.ProjectsClient: %v", err)
	}

	return &defaultGcsClient{client: client, metricClient: metricClient, projectClient: projectClient}, nil
}

func createImpersonatedCloudStorageClient(ctxIn context.Context, targetPrincipalProvider impersonate.TargetPrincipalForProjectProvider, targetProjectID string) (CloudStorageClient, error) {
	ctx, span := trace.StartSpan(ctxIn, "createImpersonatedCloudStorageClient")
	defer span.End()

	target, delegates, err := targetPrincipalProvider.GetTargetPrincipalForProject(ctx, targetProjectID)
	if err != nil {
		return nil, err
	}

	var storageOptions []option.ClientOption
	if config.UseDefaultHttpClient.GetBoolOrDefault(false) {
		storageOptions = []option.ClientOption{
			option.WithHTTPClient(http.DefaultClient),
		}
	} else {
		tokenSource, err := gimpersonate.CredentialsTokenSource(ctx, gimpersonate.CredentialsConfig{
			TargetPrincipal: target,
			Scopes:          []string{cloudPlatformAPIScope, defaultAPIScope},
			Delegates:       delegates,
		})
		if err != nil {
			return nil, err
		}

		storageOptions = []option.ClientOption{
			option.WithTokenSource(tokenSource),
		}
	}
	client, err := storage.NewClient(ctx, storageOptions...)
	if err != nil {
		return &defaultGcsClient{}, fmt.Errorf("failed to create storage.Client: %v", err)
	}

	var monitoringOptions []option.ClientOption
	if config.UseGrpcWithoutAuthentication.GetBoolOrDefault(false) {
		monitoringOptions = []option.ClientOption{
			option.WithoutAuthentication(),
			option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		}
	} else {
		tokenSource, err := gimpersonate.CredentialsTokenSource(ctx, gimpersonate.CredentialsConfig{
			TargetPrincipal: target,
			Scopes:          []string{cloudPlatformAPIScope, defaultAPIScope, metricAPIScope},
		})
		if err != nil {
			return nil, err
		}

		monitoringOptions = []option.ClientOption{
			option.WithTokenSource(tokenSource),
		}
	}
	metricClient, err := monitoring.NewMetricClient(ctx, monitoringOptions...)
	if err != nil {
		return &defaultGcsClient{}, fmt.Errorf("failed to create monitoring.MetricClient: %v", err)
	}

	var projectsClientOptions []option.ClientOption
	if config.UseGrpcWithoutAuthentication.GetBoolOrDefault(false) {
		projectsClientOptions = []option.ClientOption{
			option.WithoutAuthentication(),
			option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		}
	} else {
		tokenSource, err := gimpersonate.CredentialsTokenSource(ctx, gimpersonate.CredentialsConfig{
			TargetPrincipal: target,
			Scopes:          []string{cloudPlatformAPIScope, defaultAPIScope},
		})
		if err != nil {
			return nil, err
		}

		projectsClientOptions = []option.ClientOption{
			option.WithTokenSource(tokenSource),
		}
	}
	projectsClient, err := resourcemanager.NewProjectsClient(ctx, projectsClientOptions...)
	if err != nil {
		return &defaultGcsClient{}, fmt.Errorf("failed to create resourcemanager.NewProjectsClient: %v", err)
	}

	return &defaultGcsClient{client: client, metricClient: metricClient, projectClient: projectsClient}, nil
}

func (c *defaultGcsClient) SetBucketIAMPolicy(ctxIn context.Context, bucket string, policy *iam.Policy) error {
	_, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).SetBucketIAMPolicy")
	defer span.End()

	return c.client.Bucket(bucket).IAM().SetPolicy(ctxIn, policy)
}

// IsInitialized check if client is initialized
func (c *defaultGcsClient) IsInitialized(ctxIn context.Context) bool {
	_, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).IsInitialized")
	defer span.End()

	return c.client != nil
}

// DoesBucketExist check if bucket exist
func (c *defaultGcsClient) DoesBucketExist(ctxIn context.Context, project string, bucket string) (bool, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).DoesBucketExist")
	defer span.End()

	bucketsIterator := c.client.Buckets(ctx, project)
	bucketsIterator.Prefix = bucket
	for {
		// error or not found
		b, err := bucketsIterator.Next()
		if err == iterator.Done {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		// bucket already exist
		if b.Name == bucket {
			return true, nil
		}
	}
}

// BucketUsageInBytes report how many data are stored in the bucket
func (c *defaultGcsClient) BucketUsageInBytes(ctxIn context.Context, project string, bucket string) (totalSize float64, err error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).BucketUsageInBytes")
	defer span.End()

	startTime := time.Now().UTC().Add(time.Minute * -11) // storage/total_bytes metric is written every 5 minutes
	endTime := time.Now().UTC()
	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + project,
		Filter: fmt.Sprintf(`metric.type="storage.googleapis.com/storage/total_bytes" resource.type="gcs_bucket" resource.label.bucket_name="%s"`, bucket),
		Interval: &monitoringpb.TimeInterval{
			StartTime: &timestamppb.Timestamp{
				Seconds: startTime.Unix(),
			},
			EndTime: &timestamppb.Timestamp{
				Seconds: endTime.Unix(),
			},
		},
		View: monitoringpb.ListTimeSeriesRequest_FULL,
	}
	it := c.metricClient.ListTimeSeries(ctx, req)
	storageClassCounted := make(map[string]bool)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return totalSize, errors.Wrap(err, "ListTimeSeries failed")
		}
		for key, value := range resp.GetMetric().GetLabels() {
			if key == "storage_class" && !storageClassCounted[value] {
				for _, point := range resp.GetPoints() {
					storageClassCounted[value] = true
					totalSize += point.Value.GetDoubleValue()
					break
				}
				break
			}
		}
	}
	return totalSize, nil
}

func (c *defaultGcsClient) UpdateBucket(ctxIn context.Context, bucket string, lifetimeInDays uint, archiveTTM uint) error {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).UpdateBucket")
	defer span.End()

	attributes, err := c.client.Bucket(bucket).Attrs(ctx)
	if err != nil {
		return fmt.Errorf("UpdateBucket get Bucket attrs failed for %s: %v", bucket, err)
	}

	var DeleteRuleSet *storage.LifecycleRule
	var MoveToArchiveRuleSet *storage.LifecycleRule
	for _, rule := range attributes.Lifecycle.Rules {
		if rule.Action.Type == "Delete" {
			DeleteRuleSet = &rule
		}
		if rule.Action.Type == "SetStorageClass" && rule.Action.StorageClass == "ARCHIVE" {
			MoveToArchiveRuleSet = &rule
		}
	}

	AttrsToUpdate := storage.BucketAttrsToUpdate{
		Lifecycle: &storage.Lifecycle{
			Rules: []storage.LifecycleRule{},
		},
	}
	changed := false
	// create
	if DeleteRuleSet == nil && lifetimeInDays > 0 {
		ruleTTL := storage.LifecycleRule{
			Action:    storage.LifecycleAction{Type: "Delete"},
			Condition: storage.LifecycleCondition{AgeInDays: int64(lifetimeInDays)},
		}
		AttrsToUpdate.Lifecycle.Rules = append(AttrsToUpdate.Lifecycle.Rules, ruleTTL)
		changed = true
	}
	if MoveToArchiveRuleSet == nil && archiveTTM > 0 {
		ruleTTM := storage.LifecycleRule{
			Action:    storage.LifecycleAction{Type: "SetStorageClass", StorageClass: "ARCHIVE"},
			Condition: storage.LifecycleCondition{AgeInDays: int64(archiveTTM)},
		}
		AttrsToUpdate.Lifecycle.Rules = append(AttrsToUpdate.Lifecycle.Rules, ruleTTM)
		changed = true
	}

	// keep or update
	if lifetimeInDays > 0 && DeleteRuleSet != nil {
		ruleTTL := storage.LifecycleRule{
			Action:    storage.LifecycleAction{Type: "Delete"},
			Condition: storage.LifecycleCondition{AgeInDays: int64(lifetimeInDays)},
		}
		AttrsToUpdate.Lifecycle.Rules = append(AttrsToUpdate.Lifecycle.Rules, ruleTTL)
		changed = true
	}
	if archiveTTM > 0 && MoveToArchiveRuleSet != nil {
		ruleTTM := storage.LifecycleRule{
			Action:    storage.LifecycleAction{Type: "SetStorageClass", StorageClass: "ARCHIVE"},
			Condition: storage.LifecycleCondition{AgeInDays: int64(archiveTTM)},
		}
		AttrsToUpdate.Lifecycle.Rules = append(AttrsToUpdate.Lifecycle.Rules, ruleTTM)
		changed = true
	}
	// delete
	if lifetimeInDays == 0 || archiveTTM == 0 {
		changed = true
	}
	// delete all: workaround for case when all lifecycle rules are deleted
	// GRPC API doesn't send empty rules struct - that's why we need to create meaningless lifecycle rule
	if lifetimeInDays == 0 && archiveTTM == 0 {
		ruleTTM := storage.LifecycleRule{
			Action:    storage.LifecycleAction{Type: "SetStorageClass", StorageClass: "ARCHIVE"},
			Condition: storage.LifecycleCondition{CreatedBefore: time.Date(2000, 1, 1, 12, 12, 12, 12, time.UTC)},
		}
		AttrsToUpdate.Lifecycle.Rules = append(AttrsToUpdate.Lifecycle.Rules, ruleTTM)
	}
	if changed {
		glog.Infof("updating Bucket %s with Lifecycle attributes %v", bucket, AttrsToUpdate.Lifecycle)
		_, err := c.client.Bucket(bucket).Update(ctx, AttrsToUpdate)
		if err != nil {
			return fmt.Errorf("Bucket.Update failed with lifecycle rules %v: %v", AttrsToUpdate.Lifecycle.Rules, err)
		}
	}
	return nil
}

// CreateBucket create new bucket in a given project
func (c *defaultGcsClient) CreateBucket(ctxIn context.Context, bucket CloudStorageBucket) error {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).CreateBucket")
	defer span.End()

	var bucketAttrs = storage.BucketAttrs{
		StorageClass:     bucket.StorageClass,
		BucketPolicyOnly: storage.BucketPolicyOnly{Enabled: true},
		Labels:           bucket.Labels.Labels(),
		UniformBucketLevelAccess: storage.UniformBucketLevelAccess{
			Enabled: config.UniformBucketLevelAccess.GetBoolOrDefault(false),
		},
	}
	bucketAttrs.Lifecycle = storage.Lifecycle{}

	if bucket.DualLocation == "" {
		bucketAttrs.Location = bucket.Location
	} else {
		bucketAttrs.Location = "eu"
		bucketAttrs.CustomPlacementConfig = &storage.CustomPlacementConfig{
			DataLocations: []string{bucket.Location, bucket.DualLocation},
		}
	}

	if bucket.ArchiveTTM > 0 {
		ruleTTM := storage.LifecycleRule{
			Action:    storage.LifecycleAction{Type: "SetStorageClass", StorageClass: "ARCHIVE"},
			Condition: storage.LifecycleCondition{AgeInDays: int64(bucket.ArchiveTTM)},
		}
		bucketAttrs.Lifecycle.Rules = append(bucketAttrs.Lifecycle.Rules, ruleTTM)
	}

	if bucket.LifetimeInDays > 0 {
		ruleTTL := storage.LifecycleRule{
			Action:    storage.LifecycleAction{Type: "Delete"},
			Condition: storage.LifecycleCondition{AgeInDays: int64(bucket.LifetimeInDays)},
		}
		bucketAttrs.Lifecycle.Rules = append(bucketAttrs.Lifecycle.Rules, ruleTTL)
	}

	err := c.client.Bucket(bucket.Bucket).Create(ctx, bucket.Project, &bucketAttrs)
	if err != nil {
		return err
	}

	return nil
}

// DeleteBucket remove whole bucket
func (c *defaultGcsClient) DeleteBucket(ctxIn context.Context, bucket string) error {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).DeleteBucket")
	defer span.End()

	err := c.client.Bucket(bucket).Delete(ctx)
	if err != nil {
		return err
	}

	return nil
}

// ReadObject get content of a bucket object
func (c *defaultGcsClient) ReadObject(ctxIn context.Context, bucketName, objectName string) ([]byte, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).ReadObject")
	defer span.End()

	rc, err := c.client.Bucket(bucketName).Object(objectName).ReadCompressed(true).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to open file from bucket %q, file %q: %v", bucketName, objectName, err)
	}

	slurp, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("unable to read data from bucket %q, file %q: %v", bucketName, objectName, err)
	}

	return slurp, nil
}

// DeleteObjectsWithObjectMatch delete all bucket objects that have same prefix
func (c *defaultGcsClient) DeleteObjectsWithObjectMatch(ctxIn context.Context, bucketName string, prefix string, objectPattern *regexp.Regexp) (deleted int, err error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).DeleteObjectsWithObjectMatch")
	defer span.End()

	storageQuery := storage.Query{}
	storageQuery.Prefix = prefix
	bucket := c.client.Bucket(bucketName)
	objectsIterator := bucket.Objects(ctx, &storageQuery)
	for {
		objAttr, err := objectsIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if !errors.Is(err, iterator.Done) && err != nil {
			return deleted, err
		}
		if objAttr == nil {
			if objectPattern == nil {
				glog.Warningf("objAttr is nil bucket/prefix/pattern %s/%s", bucketName, prefix)
			} else {
				glog.Warningf("objAttr is nil bucket/prefix/pattern %s/%s/%v", bucketName, prefix, objectPattern.String())
			}
			continue
		}
		if objectPattern == nil || objectPattern.MatchString(objAttr.Name) {
			err = bucket.Object(objAttr.Name).Delete(ctx)
			if err != nil && !errors.Is(err, storage.ErrObjectNotExist) {
				return deleted, err
			}
			deleted++
		}
	}
	return deleted, nil
}

// DeleteObject delete bucket object
func (c *defaultGcsClient) DeleteObject(ctxIn context.Context, bucketName string, objectName string) error {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).DeleteObject")
	defer span.End()

	return c.client.Bucket(bucketName).Object(objectName).Delete(ctx)

}

// CreateObject create new bucket object
func (c *defaultGcsClient) CreateObject(ctxIn context.Context, bucketName, objectName, content string) error {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).CreateObject")
	defer span.End()

	w := c.client.Bucket(bucketName).Object(objectName).NewWriter(ctx)
	if _, err := fmt.Fprint(w, content); err != nil {
		return err
	}

	return w.Close()
}

// MoveObject move bucket object
func (c *defaultGcsClient) MoveObject(ctxIn context.Context, bucketName, oldObjectName, newObjectName string) error {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).MoveObject")
	defer span.End()

	bucket := c.client.Bucket(bucketName)
	_, err := bucket.Object(newObjectName).CopierFrom(bucket.Object(oldObjectName)).Run(ctx)
	if nil != err {
		return errors.Wrapf(err, "CopierFrom failed for object %s", oldObjectName)
	}
	return c.DeleteObject(ctx, bucketName, oldObjectName)
}

// GetBuckets list all buckets that belongs to a given project
func (c *defaultGcsClient) GetBuckets(ctxIn context.Context, project string) (buckets []string, err error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).GetBuckets")
	defer span.End()

	bucketsIterator := c.client.Buckets(ctx, project)
	for {
		// error or not found
		b, err := bucketsIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return []string{}, errors.Wrap(err, fmt.Sprintf("Buckets.Next() failed for project %s", project))
		}
		buckets = append(buckets, b.Name)
	}
	return buckets, err
}

// GetBucketDetails get details of a bucket
func (c *defaultGcsClient) GetBucketDetails(ctxIn context.Context, bucket string) (*storage.BucketAttrs, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultGcsClient).GetBucketDetails")
	defer span.End()

	return c.client.Bucket(bucket).Attrs(ctx)
}
