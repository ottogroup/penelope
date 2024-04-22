package gcs

import (
	"cloud.google.com/go/iam"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"context"
	"fmt"
	"regexp"

	"cloud.google.com/go/storage"
)

type MockGcsClient struct {
	ClientInitialized bool
	ShouldFail        bool
	ObjectContent     []byte
}

func (c *MockGcsClient) GetProject(ctxIn context.Context, projectID string) (*resourcemanagerpb.Project, error) {
	panic("implement me")
}

func (c *MockGcsClient) SetBucketIAMPolicy(ctxIn context.Context, bucket string, policy *iam.Policy) error {
	panic("implement me")
}

func (c *MockGcsClient) Close(ctxIn context.Context) {
	panic("implement me")
}

func (c *MockGcsClient) MoveObject(ctxIn context.Context, bucketName, oldObjectName, newObjectName string) error {
	panic("implement me")
}

func (c *MockGcsClient) CreateObject(ctxIn context.Context, bucketName, objectName, content string) error {
	panic("implement me")
}
func (c *MockGcsClient) DeleteObject(ctxIn context.Context, bucketName string, objectName string) error {
	panic("implement me")
}
func (c *MockGcsClient) GetBuckets(ctxIn context.Context, project string) ([]string, error) {
	panic("implement me")
}

func (c *MockGcsClient) BucketUsageInBytes(ctxIn context.Context, project string, bucket string) (float64, error) {
	panic("implement me")
}

func (c *MockGcsClient) DeleteObjectsWithObjectMatch(ctxIn context.Context, bucketName string, prefix string, objectPattern *regexp.Regexp) (deleted int, err error) {
	panic("implement me")
}

func (c *MockGcsClient) DoesBucketExist(ctxIn context.Context, project string, bucket string) (bool, error) {
	panic("implement me")
}

func (c *MockGcsClient) CreateBucket(ctxIn context.Context, project, bucket, location, dualLocation, storageClass string, lifetimeInDays uint, archiveTTM uint) error {
	panic("implement me")
}

func (c *MockGcsClient) UpdateBucket(ctxIn context.Context, bucket string, lifetimeInDays uint, archiveTTM uint) error {
	panic("implement me")
}

func (c *MockGcsClient) DeleteBucket(ctxIn context.Context, bucket string) error {
	panic("implement me")
}

func (c *MockGcsClient) GetBucketDetails(ctxIn context.Context, bucket string) (*storage.BucketAttrs, error) {
	panic("implement me")
}

func (c *MockGcsClient) IsInitialized(ctxIn context.Context) bool {
	return c.ClientInitialized
}

func (c *MockGcsClient) ReadObject(ctxIn context.Context, bucketName, objectName string) ([]byte, error) {
	if c.ShouldFail {
		return nil, fmt.Errorf("failed")
	}
	return c.ObjectContent, nil
}

func NewMockGcsClient(initialized bool, shouldFail bool) CloudStorageClient {
	return &MockGcsClient{ClientInitialized: initialized, ShouldFail: shouldFail}
}
