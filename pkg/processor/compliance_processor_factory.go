package processor

import (
	iam "cloud.google.com/go/iam/apiv2"
	"cloud.google.com/go/iam/apiv2/iampb"
	"context"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/http/auth"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"github.com/ottogroup/penelope/pkg/service/bigquery"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"go.opencensus.io/trace"
	gimpersonate "google.golang.org/api/impersonate"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/api/serviceusage/v1"
	"net/http"
	"strings"
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
			&backupOnlySinkProjectCheck{
				tokenSourceProvider: c.tokenSourceProvider,
				backupProvider:      c.backupProvider,
			},
			&backupWithSingleWriterCheck{
				backupProvider:      c.backupProvider,
				tokenSourceProvider: c.tokenSourceProvider,
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

var allowedServices = []string{
	"bigquery.googleapis.com",
	"storage.googleapis.com",
	"storagetransfer.googleapis.com",
}

type backupOnlySinkProjectCheck struct {
	backupProvider      provider.SinkGCPProjectProvider
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func (c *backupOnlySinkProjectCheck) Check(ctx context.Context, request requestobjects.ComplianceRequest) (requestobjects.ComplianceCheck, error) {
	targetProject, err := c.backupProvider.GetSinkGCPProjectID(ctx, request.Project)
	if err != nil {
		return requestobjects.ComplianceCheck{}, err
	}

	targetPrincipal, delegates, err := c.tokenSourceProvider.GetTargetPrincipalForProject(ctx, targetProject)
	if err != nil {
		return requestobjects.ComplianceCheck{}, fmt.Errorf("could not get target principal for project %s: %s", targetProject, err)
	}

	tokenSource, err := gimpersonate.CredentialsTokenSource(ctx, gimpersonate.CredentialsConfig{
		TargetPrincipal: targetPrincipal,
		Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform.read-only"},
		Delegates:       delegates,
	})
	if err != nil {
		return requestobjects.ComplianceCheck{}, fmt.Errorf("could not create token source: %s", err)
	}

	options := []option.ClientOption{
		option.WithTokenSource(tokenSource),
	}

	if config.UseDefaultHttpClient.GetBoolOrDefault(false) {
		options = append(options, option.WithHTTPClient(http.DefaultClient))
	}

	service, err := serviceusage.NewService(ctx, options...)
	if err != nil {
		return requestobjects.ComplianceCheck{}, err
	}

	project := fmt.Sprintf("projects/%s", targetProject)
	listServicesResponse, err := service.Services.List(project).Do()
	if err != nil {
		return requestobjects.ComplianceCheck{}, fmt.Errorf("could not list services for project %s: %s", targetProject, err)
	}

	var enabledServices []string
	for _, s := range listServicesResponse.Services {
		if s.State == "ENABLED" {
			enabledServices = append(enabledServices, s.Name)
		}
	}

	var invalidServices []string
	for _, enabledService := range enabledServices {
		if !contains(allowedServices, enabledService) {
			invalidServices = append(invalidServices, enabledService)
		}
	}

	if len(invalidServices) > 0 {
		return requestobjects.ComplianceCheck{
			Field:       "request.Target",
			Passed:      false,
			Description: "Backup project should have only allowed services enabled",
		}, nil
	}

	return requestobjects.ComplianceCheck{
		Field:       "request.Target",
		Passed:      true,
		Description: "Backup project is only used as sink project",
	}, nil
}

func contains(services []string, service string) bool {
	for _, s := range services {
		if s == service {
			return true
		}
	}
	return false
}

type backupWithSingleWriterCheck struct {
	backupProvider      provider.SinkGCPProjectProvider
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func (c *backupWithSingleWriterCheck) Check(ctx context.Context, request requestobjects.ComplianceRequest) (requestobjects.ComplianceCheck, error) {
	targetProject, err := c.backupProvider.GetSinkGCPProjectID(ctx, request.Project)
	if err != nil {
		return requestobjects.ComplianceCheck{}, err
	}

	compliant := false

	targetPrincipal, delegates, err := c.tokenSourceProvider.GetTargetPrincipalForProject(ctx, targetProject)
	if err != nil {
		return requestobjects.ComplianceCheck{}, fmt.Errorf("could not get target principal for project %s: %s", targetProject, err)
	}

	tokenSource, err := gimpersonate.CredentialsTokenSource(ctx, gimpersonate.CredentialsConfig{
		TargetPrincipal: targetPrincipal,
		Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
		Delegates:       delegates,
	})
	if err != nil {
		return requestobjects.ComplianceCheck{}, fmt.Errorf("could not create token source: %s", err)
	}

	options := []option.ClientOption{
		option.WithTokenSource(tokenSource),
	}

	if config.UseDefaultHttpClient.GetBoolOrDefault(false) {
		options = append(options, option.WithHTTPClient(http.DefaultClient))
	}

	policiesClient, err := iam.NewPoliciesRESTClient(ctx, options...)
	if err != nil {
		return requestobjects.ComplianceCheck{}, fmt.Errorf("could not create new IAM policies client: %s", err)
	}

	attachmentPoint := fmt.Sprintf("cloudresourcemanager.googleapis.com%%2Fprojects%%2F%s", targetProject)

	it := policiesClient.ListPolicies(ctx, &iampb.ListPoliciesRequest{
		Parent: fmt.Sprintf("policies/%s/denypolicies", attachmentPoint),
	})

	for {
		policy, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			glog.Errorf("could not get next policy: %s", err)
			break
		}

		// We need to check if deny edit permission for cloud storage is set for all principals except for the
		// target backup service account.
		for _, rule := range policy.Rules {
			deniedPermissions := rule.GetDenyRule().GetDeniedPermissions()
			deniedPrincipals := rule.GetDenyRule().GetDeniedPrincipals()
			exceptionPrincipals := rule.GetDenyRule().GetExceptionPrincipals()

			if !containsAllEditPermissions(deniedPermissions) {
				continue
			}

			if !containsAllPrincipals(deniedPrincipals) {
				continue
			}

			if !containsOnlyBackupServiceAccountAsException(targetPrincipal, exceptionPrincipals) {
				continue
			}

			compliant = true
		}
	}

	if err := policiesClient.Close(); err != nil {
		glog.Errorf("could not close IAM policies client: %s", err)
	}

	if compliant {
		return requestobjects.ComplianceCheck{
			Field:       "request.Target",
			Passed:      true,
			Description: "Backup project has single writer",
		}, nil
	}

	return requestobjects.ComplianceCheck{
		Field:       "request.Target",
		Passed:      false,
		Description: "Backup project should have single writer",
	}, nil
}

var cloudStorageEditPermissions = []string{
	"storage.googleapis.com/objects.update",
	"storage.googleapis.com/objects.delete",
	"storage.googleapis.com/objects.create",
}

const (
	allPrincipals = "principalSet://goog/public:all"
)

func containsOnlyBackupServiceAccountAsException(targetPrincipal string, principals []string) bool {
	return len(principals) == 1 && strings.EqualFold(principals[0], fmt.Sprintf("principal://iam.googleapis.com/projects/-/serviceAccounts/%s", targetPrincipal))
}

func containsAllPrincipals(principals []string) bool {
	for _, item := range principals {
		if strings.EqualFold(item, allPrincipals) {
			return true
		}
	}
	return false
}

func containsAllEditPermissions(permissions []string) bool {
	for _, item := range cloudStorageEditPermissions {
		found := false
		for _, element := range permissions {
			if item == element {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
