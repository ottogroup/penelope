package compliance

import (
	iam "cloud.google.com/go/iam/apiv2"
	"cloud.google.com/go/iam/apiv2/iampb"
	"context"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"go.opencensus.io/trace"
	gimpersonate "google.golang.org/api/impersonate"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"net/http"
	"strings"
)

var cloudStorageEditPermissions = []string{
	"storage.googleapis.com/objects.update",
	"storage.googleapis.com/objects.delete",
	"storage.googleapis.com/objects.create",
}

const (
	allPrincipals = "principalSet://goog/public:all"
)

func NewSinkProjectWithSinglerWriterCheckFunc(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) CheckFunc {
	return func(ctxIn context.Context, sinkProject string) error {
		ctx, span := trace.StartSpan(ctxIn, "(*sinkProjectWithSinglerWriterCheck).Check")
		defer span.End()

		compliant := false

		targetPrincipal, delegates, err := tokenSourceProvider.GetTargetPrincipalForProject(ctx, sinkProject)
		if err != nil {
			return fmt.Errorf("could not get target principal for project %s: %s", sinkProject, err)
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

		policiesClient, err := iam.NewPoliciesRESTClient(ctx, options...)
		if err != nil {
			return fmt.Errorf("could not create new IAM policies client: %s", err)
		}

		attachmentPoint := fmt.Sprintf("cloudresourcemanager.googleapis.com%%2Fprojects%%2F%s", sinkProject)

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
			return nil
		}

		return &CheckError{
			CheckName: singleWriter,
			Reason:    "Sink project does not prevent all principals from editing cloud storage except for backup service account",
		}
	}
}

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
