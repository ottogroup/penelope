package tasks

import (
	iam "cloud.google.com/go/iam/apiv2"
	"cloud.google.com/go/iam/apiv2/iampb"
	"context"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/secret"
	"go.opencensus.io/trace"
	gimpersonate "google.golang.org/api/impersonate"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"net/http"
	"strings"
)

type checkImmutableBackupsService struct {
	backupRepository    repository.BackupRepository
	tokenSourceProvider impersonate.TargetPrincipalForProjectProvider
}

var cloudStorageEditPermissions = []string{
	"storage.googleapis.com/objects.update",
	"storage.googleapis.com/objects.delete",
	"storage.googleapis.com/objects.create",
}

const (
	allPrincipals = "principalSet://goog/public:all"
)

func newCheckImmutableBackupsService(ctxIn context.Context, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialsProvider secret.SecretProvider) (*checkImmutableBackupsService, error) {
	ctx, span := trace.StartSpan(ctxIn, "newCheckImmutableBackupsService")
	defer span.End()

	backupRepository, err := repository.NewBackupRepository(ctx, credentialsProvider)
	if err != nil {
		return nil, err
	}

	return &checkImmutableBackupsService{
		backupRepository:    backupRepository,
		tokenSourceProvider: tokenSourceProvider,
	}, nil
}

func (c *checkImmutableBackupsService) Run(ctxIn context.Context) {
	ctx, span := trace.StartSpan(ctxIn, "(*checkImmutableBackupsService).Run")
	defer span.End()

	sinkProjects, err := c.backupRepository.ListBackupSinkProjects(ctx)
	if err != nil {
		glog.Error("could not get list of backups: %s", err)
		return
	}

	for _, sink := range sinkProjects {
		var isImmutable = false

		targetPrincipal, delegates, err := c.tokenSourceProvider.GetTargetPrincipalForProject(ctx, sink)
		if err != nil {
			glog.Errorf("could not get target principal for project %s: %s", sink, err)
			continue
		}

		tokenSource, err := gimpersonate.CredentialsTokenSource(ctx, gimpersonate.CredentialsConfig{
			TargetPrincipal: targetPrincipal,
			Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
			Delegates:       delegates,
		})
		if err != nil {
			glog.Errorf("could not create token source: %s", err)
			return
		}

		options := []option.ClientOption{
			option.WithTokenSource(tokenSource),
			option.WithHTTPClient(http.DefaultClient),
		}

		policiesClient, err := iam.NewPoliciesRESTClient(ctx, options...)
		if err != nil {
			glog.Errorf("could not create new IAM policies client: %s", err)
		}

		// FIXME: does not show inherited deny policies
		// bucket level deny policy is not supported: https://cloud.google.com/iam/docs/deny-access#attachment-point
		attachmentPoint := fmt.Sprintf("cloudresourcemanager.googleapis.com%%2Fprojects%%2F%s", sink)

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

			/**
			* We need to check if deny edit permission for cloud storage is set for all principals except for the
			* target backup service account.
			 */
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

				isImmutable = true
			}
		}

		if err := policiesClient.Close(); err != nil {
			glog.Errorf("could not close IAM policies client: %s", err)
		}

		if isImmutable {
			err = c.backupRepository.MarkTargetSinksAsImmutable(ctx, sink)
		} else {
			err = c.backupRepository.MarkTargetSinksAsMutable(ctx, sink)
		}

		if err != nil {
			glog.Errorf("could not mark target sink %s as safe: %s", sink, err)
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
