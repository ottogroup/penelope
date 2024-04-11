package compliance

import (
	"context"
	"fmt"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"go.opencensus.io/trace"
	gimpersonate "google.golang.org/api/impersonate"
	"google.golang.org/api/option"
	"google.golang.org/api/serviceusage/v1"
	"net/http"
	"strings"
)

var allowedServices = []string{
	"bigquery.googleapis.com",
	"storage.googleapis.com",
	"storagetransfer.googleapis.com",
}

func NewSinkProjectOnlyForBackupCheckFunc(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) CheckFunc {
	return func(ctxIn context.Context, sinkProject string) error {
		ctx, span := trace.StartSpan(ctxIn, "(*sinkProjectOnlyForBackupCheck).Checkfunc")
		defer span.End()

		targetPrincipal, delegates, err := tokenSourceProvider.GetTargetPrincipalForProject(ctx, sinkProject)
		if err != nil {
			return fmt.Errorf("could not get target principal for project %s: %s", sinkProject, err)
		}

		tokenSource, err := gimpersonate.CredentialsTokenSource(ctx, gimpersonate.CredentialsConfig{
			TargetPrincipal: targetPrincipal,
			Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
			Delegates:       delegates,
		})
		if err != nil {
			return fmt.Errorf("could not create token source: %s", err)
		}

		options := []option.ClientOption{
			option.WithTokenSource(tokenSource),
		}

		if config.UseDefaultHttpClient.GetBoolOrDefault(false) {
			options = append(options, option.WithHTTPClient(http.DefaultClient))
		}

		service, err := serviceusage.NewService(ctx, options...)
		if err != nil {
			return err
		}

		project := fmt.Sprintf("projects/%s", sinkProject)
		listServicesResponse, err := service.Services.List(project).Do()
		if err != nil {
			return fmt.Errorf("could not list services for project %s: %s", sinkProject, err)
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
			return &CheckError{
				CheckName: backupOnly,
				Reason:    fmt.Sprintf("project %s has invalid services enabled: [%v]", sinkProject, strings.Join(invalidServices, ", ")),
			}
		}

		return nil
	}
}

func contains(services []string, service string) bool {
	for _, s := range services {
		if s == service {
			return true
		}
	}
	return false
}
