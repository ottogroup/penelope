package provider

import (
	"context"
	"fmt"
	"github.com/ottogroup/penelope/pkg/config"
	authmodel "github.com/ottogroup/penelope/pkg/http/auth/model"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"go.opencensus.io/trace"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
)

type PrincipalProvider interface {
	GetPrincipalForEmail(ctxIn context.Context, email string) (*authmodel.Principal, error)
}

type defaultUserProvider struct {
	client gcs.CloudStorageClient
}

func NewDefaultUserProvider(ctxIn context.Context, gcsClient gcs.CloudStorageClient) (PrincipalProvider, error) {
	ctx, span := trace.StartSpan(ctxIn, "NewDefaultGCPBackupProvider")
	defer span.End()

	if gcsClient == nil || !gcsClient.IsInitialized(ctx) {
		return &defaultUserProvider{}, fmt.Errorf("can not create instance of defaultGCSBackupProvider with unititialized GcsClient")
	}

	return &defaultUserProvider{
		client: gcsClient,
	}, nil
}

func (p *defaultUserProvider) GetPrincipalForEmail(ctxIn context.Context, email string) (*authmodel.Principal, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultUserProvider).GetSinkGCPProjectID")
	defer span.End()

	bucketName := config.DefaultProviderBucketEnv.MustGet()
	objectName := config.DefaultProviderPrincipalForUserPathEnv.MustGet()

	var object []byte
	var err error

	if config.IsProviderLocal.GetBoolOrDefault(false) {
		filePath := filepath.Join(bucketName, objectName)
		object, err = ioutil.ReadFile(filePath)
	} else {
		object, err = p.client.ReadObject(ctx, bucketName, objectName)
	}

	if err != nil {
		return nil, err
	}

	var principal []*authmodel.Principal

	if err = yaml.Unmarshal(object, &principal); err != nil {
		return nil, fmt.Errorf("can not parse yaml file %s", err)
	}

	for _, p := range principal {
		if p.User.Email == email {
			return p, nil
		}
	}

	return nil, fmt.Errorf("could not find user '%s' in provided path %s", email, config.DefaultProviderPrincipalForUserPathEnv.MustGet())
}
