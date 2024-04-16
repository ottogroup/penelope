package provider

import (
	"context"
	"fmt"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"go.opencensus.io/trace"
	"gopkg.in/yaml.v3"
	"time"
)

type AvailabilityClass string

const (
	A0Invalid    AvailabilityClass = ""
	A1Irrelevant AvailabilityClass = "A1"
	A2Aimed      AvailabilityClass = "A2"
	A3Guaranteed AvailabilityClass = "A3"
	A4Resilient  AvailabilityClass = "A4"
)

func (AvailabilityClass) ValidValues() []AvailabilityClass {
	return []AvailabilityClass{A1Irrelevant, A2Aimed, A3Guaranteed, A4Resilient}
}

type SourceGCPProject struct {
	AvailabilityClass AvailabilityClass
	DataOwner         string
}

type SourceGCPProjectProvider interface {
	GetSourceGCPProject(ctxIn context.Context, gcpProjectID string) (SourceGCPProject, error)
}

type defaultSourceGCPProjectProvider struct {
	client          gcs.CloudStorageClient
	lastFetch       time.Time
	refreshDuration time.Duration
	cache           []gcpSourceProject
}

type gcpSourceProject struct {
	Project           string            `yaml:"project"`
	AvailabilityClass AvailabilityClass `yaml:"availability_class"`
	DataOwner         string            `yaml:"data_owner"`
}

func (d *defaultSourceGCPProjectProvider) GetSourceGCPProject(ctxIn context.Context, gcpProjectID string) (SourceGCPProject, error) {
	ctx, span := trace.StartSpan(ctxIn, "GetSourceGCPProject")
	defer span.End()

	if len(d.cache) == 0 || time.Since(d.lastFetch) > d.refreshDuration {

		bucketName := config.DefaultProviderBucketEnv.MustGet()
		objectName := config.DefaultProviderGCPSourceProjectPathEnv.MustGet()

		object, err := d.client.ReadObject(ctx, bucketName, objectName)
		if err != nil {
			return SourceGCPProject{}, err
		}

		if err = yaml.Unmarshal(object, &d.cache); err != nil {
			return SourceGCPProject{}, fmt.Errorf("can not parse yaml file %s", err)
		}
		d.lastFetch = time.Now()
	}

	for _, entry := range d.cache {
		if entry.Project == gcpProjectID {
			return SourceGCPProject{
				AvailabilityClass: entry.AvailabilityClass,
				DataOwner:         entry.DataOwner,
			}, nil
		}
	}

	return SourceGCPProject{}, fmt.Errorf("could not find GCP source project for %s in backupProjectsPath %s", gcpProjectID, config.DefaultProviderGCPSourceProjectPathEnv.MustGet())
}

func NewDefaultSourceGCPBackupProvider(ctxIn context.Context, gcsClient gcs.CloudStorageClient) (SourceGCPProjectProvider, error) {
	ctx, span := trace.StartSpan(ctxIn, "NewDefaultSourceGCPBackupProvider")
	defer span.End()

	if gcsClient == nil || !gcsClient.IsInitialized(ctx) {
		return &defaultSourceGCPProjectProvider{}, fmt.Errorf("can not create instance of defaultGCSBackupProvider with unititialized GcsClient")
	}

	ttl, err := defaultProviderCacheTTL()
	if err != nil {
		return &defaultSourceGCPProjectProvider{}, fmt.Errorf("can not create instance of defaultGCSBackupProvider %s", err)
	}

	return &defaultSourceGCPProjectProvider{gcsClient, time.Now().Add(ttl * -2), ttl, []gcpSourceProject{}}, nil
}
