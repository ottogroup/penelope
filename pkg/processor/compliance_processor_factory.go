package processor

import (
	"context"
	"fmt"

	"github.com/ottogroup/penelope/pkg/http/auth"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"github.com/ottogroup/penelope/pkg/service/bigquery"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"go.opencensus.io/trace"
)

type ComplianceProcessorFactory interface {
	CreateProcessor(ctxIn context.Context) (Operation[requestobjects.ComplianceRequest, requestobjects.ComplianceResponse], error)
}

// complianceProcessorFactory create Process for Compliance
type complianceProcessorFactory struct {
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
	backupProvider      provider.SinkGCPProjectProvider
}

func NewComplianceProcessorFactory(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, backupProvider provider.SinkGCPProjectProvider) ComplianceProcessorFactory {
	return &complianceProcessorFactory{
		tokenSourceProvider: tokenSourceProvider,
		backupProvider:      backupProvider,
	}
}

// CreateProcessor return instance of Operations for Calculating
func (c complianceProcessorFactory) CreateProcessor(ctxIn context.Context) (Operation[requestobjects.ComplianceRequest, requestobjects.ComplianceResponse], error) {
	_, span := trace.StartSpan(ctxIn, "(*ComplianceProcessorFactory).CreateProcessor")
	defer span.End()

	return &complianceProcessor{
		checks: []ComplianceCheck{
			&backupLocationCheck{
				tokenSourceProvider: c.tokenSourceProvider,
				backupProvider:      c.backupProvider,
			},
			&backupEncryptionCheck{},
			&backupProjectCheck{
				backupProvider: c.backupProvider,
			},
		},
	}, nil
}

type ComplianceCheck interface {
	Check(ctx context.Context, request requestobjects.ComplianceRequest) (requestobjects.ComplianceCheck, error)
}

type complianceProcessor struct {
	checks []ComplianceCheck
}

// Process request
func (c *complianceProcessor) Process(ctxIn context.Context, args *Argument[requestobjects.ComplianceRequest]) (requestobjects.ComplianceResponse, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*calculatingProcessor).Process")
	defer span.End()

	var request requestobjects.ComplianceRequest = args.Request

	if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.Calculating, request.Project) {
		return requestobjects.ComplianceResponse{}, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.Compliance.String(), args.Principal.User.Email, request.Project)
	}

	result := requestobjects.ComplianceResponse{}
	for _, check := range c.checks {
		res, err := check.Check(ctx, request)
		if err != nil {
			return requestobjects.ComplianceResponse{}, fmt.Errorf("some compliance check failed with err: %s", err)
		}
		result.Checks = append(result.Checks, res)
	}

	return result, nil
}

type backupLocationCheck struct {
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
	backupProvider      provider.SinkGCPProjectProvider
}

func (c *backupLocationCheck) Check(ctx context.Context, request requestobjects.ComplianceRequest) (requestobjects.ComplianceCheck, error) {
	sourceRegion := ""
	targetProject, err := c.backupProvider.GetSinkGCPProjectID(ctx, request.Project)
	if err != nil {
		return requestobjects.ComplianceCheck{}, err
	}

	if repository.BigQuery.EqualTo(request.Type) {
		bigQueryClient, err := bigquery.NewBigQueryClient(ctx, c.tokenSourceProvider, request.Project, targetProject)
		if err != nil {
			return requestobjects.ComplianceCheck{}, err
		}
		details, err := bigQueryClient.GetDatasetDetails(ctx, request.BigQueryOptions.Dataset)
		if err != nil {
			return requestobjects.ComplianceCheck{}, err
		}
		sourceRegion = details.Location
	} else if repository.CloudStorage.EqualTo(request.Type) {
		storageClient, err := gcs.NewCloudStorageClient(ctx, c.tokenSourceProvider, targetProject)
		if err != nil {
			return requestobjects.ComplianceCheck{}, err
		}

		details, err := storageClient.GetBucketDetails(ctx, request.GCSOptions.Bucket)
		if err != nil {
			return requestobjects.ComplianceCheck{}, err
		}
		sourceRegion = details.Location
	} else {
		return requestobjects.ComplianceCheck{}, fmt.Errorf("unknown request type `%s` for check backup location", request.Type)
	}

	sourceRegionConfig := getRegionConfiguration(sourceRegion)
	targetRegionConfig := getRegionConfiguration(request.TargetOptions.Region)
	targetDualRegionConfig := getRegionConfiguration(request.TargetOptions.DualRegion)

	distance := sourceRegionConfig.Location.Distance(targetRegionConfig.Location)
	dualDistance := sourceRegionConfig.Location.Distance(targetDualRegionConfig.Location)

	result := requestobjects.ComplianceCheck{
		Field:       "target.region",
		Passed:      false,
		Description: "Data and backup location should be at least 200km apart",
	}

	if sourceRegionConfig.MultiRegion || targetRegionConfig.MultiRegion {
		result.Details = "Data or backup location is multi-region"
		result.Passed = true
	} else if distance > 200 || (request.TargetOptions.DualRegion != "" && dualDistance > 200) {
		result.Details = fmt.Sprintf("Data and backup location is %.2fkm apart", distance)
		result.Passed = true
	}

	return result, nil
}

type backupProjectCheck struct {
	backupProvider provider.SinkGCPProjectProvider
}

func (c *backupProjectCheck) Check(ctx context.Context, request requestobjects.ComplianceRequest) (requestobjects.ComplianceCheck, error) {
	targetProject, err := c.backupProvider.GetSinkGCPProjectID(ctx, request.Project)
	if err != nil {
		return requestobjects.ComplianceCheck{}, err
	}

	return requestobjects.ComplianceCheck{
		Field:       "request.Project",
		Passed:      request.Project != targetProject,
		Description: "Backup and source project should be different",
	}, nil
}

type backupEncryptionCheck struct {
}

func (c *backupEncryptionCheck) Check(ctx context.Context, request requestobjects.ComplianceRequest) (requestobjects.ComplianceCheck, error) {
	return requestobjects.ComplianceCheck{
		Field:       "request.Project",
		Passed:      true, // Google Cloud Storage Buckets are encrypted by default
		Description: "Backup should be encrypted",
	}, nil
}
