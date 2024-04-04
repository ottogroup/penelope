package compliance

import (
	"context"
	"fmt"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/repository"
	"go.opencensus.io/trace"
	gimpersonate "google.golang.org/api/impersonate"
	"google.golang.org/api/option"
	"google.golang.org/api/serviceusage/v1"
	"net/http"
)

var allowedServices = []string{
	"bigquery.googleapis.com",
	"storage.googleapis.com",
	"storagetransfer.googleapis.com",
}

func NewSinkProjectOnlyForBackupCheck(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) Check {
	return &sinkProjectOnlyForBackupCheck{
		TokenSourceProvider: tokenSourceProvider,
	}
}

// sinkProjectOnlyForBackupCheck checks if the project is used only for backup
type sinkProjectOnlyForBackupCheck struct {
	TokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

func (c *sinkProjectOnlyForBackupCheck) Check(ctxIn context.Context, check *repository.SinkComplianceCheck) error {
	ctx, span := trace.StartSpan(ctxIn, "(*sinkProjectOnlyForBackupCheck).Check")
	defer span.End()

	targetPrincipal, delegates, err := c.TokenSourceProvider.GetTargetPrincipalForProject(ctx, check.ProjectSink)
	if err != nil {
		return fmt.Errorf("could not get target principal for project %s: %s", check.ProjectSink, err)
	}

	tokenSource, err := gimpersonate.CredentialsTokenSource(ctx, gimpersonate.CredentialsConfig{
		TargetPrincipal: targetPrincipal,
		Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform.read-only"},
		Delegates:       delegates,
	})
	if err != nil {
		return fmt.Errorf("could not create token source: %s", err)
	}

	options := []option.ClientOption{
		option.WithTokenSource(tokenSource),
		option.WithHTTPClient(http.DefaultClient),
	}

	service, err := serviceusage.NewService(ctx, options...)
	if err != nil {
		return err
	}

	project := fmt.Sprintf("projects/%s", check.ProjectSink)
	listServicesResponse, err := service.Services.List(project).Do()
	if err != nil {
		return err
	}

	var enabledServices []string
	for _, s := range listServicesResponse.Services {
		if s.State != "ENABLED" {
			return fmt.Errorf("project %s has enabled services", check.ProjectSink)
		}
	}

	var invalidServices []string
	for _, enabledService := range enabledServices {
		if !contains(allowedServices, enabledService) {
			invalidServices = append(invalidServices, enabledService)
		}
	}

	if len(invalidServices) == 0 {
		check.BackupOnly = true
	}

	return nil
}

func contains(services []string, service string) bool {
	for _, s := range services {
		if s == service {
			return true
		}
	}
	return false
}
