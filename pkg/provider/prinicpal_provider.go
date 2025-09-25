package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ottogroup/penelope/pkg/config"
	authmodel "github.com/ottogroup/penelope/pkg/http/auth/model"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"go.opencensus.io/trace"
	"gopkg.in/yaml.v2"
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
		object, err = os.ReadFile(filePath)
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

	var principalCache = make(map[string][]authmodel.ProjectRoleBinding)

	// Populate the cache and deduplicate role bindings by project using the highest role only
	// First, collect all role bindings for each user across all principal entries
	userRoleBindings := make(map[string][]authmodel.ProjectRoleBinding)

	for _, p := range principal {
		// Append all role bindings for this user
		userRoleBindings[p.User.Email] = append(userRoleBindings[p.User.Email], p.RoleBindings...)
	}

	// Now consolidate role bindings for each user by keeping highest role per project
	for email, allBindings := range userRoleBindings {
		// Create a map to track the highest role for each project for this user
		projectRoleMap := make(map[string]authmodel.Role)

		// Process all role bindings for this user
		for _, roleBinding := range allBindings {
			project := roleBinding.Project
			newRole := roleBinding.Role

			// Check if we already have a role for this project
			if existingRole, exists := projectRoleMap[project]; exists {
				// Keep the highest role
				if newRole.IsHigher(existingRole) {
					projectRoleMap[project] = newRole
				}
			} else {
				// No existing role for this project, add the new one
				projectRoleMap[project] = newRole
			}
		}

		// Convert the map back to a slice of ProjectRoleBinding
		var consolidatedBindings []authmodel.ProjectRoleBinding
		for project, role := range projectRoleMap {
			consolidatedBindings = append(consolidatedBindings, authmodel.ProjectRoleBinding{
				Role:    role,
				Project: project,
			})
		}

		// Update the cache with the consolidated bindings
		principalCache[email] = consolidatedBindings
	}

	if roleBindings, ok := principalCache[email]; ok {
		return &authmodel.Principal{
			User:         authmodel.User{Email: email},
			RoleBindings: roleBindings,
		}, nil
	}

	return nil, fmt.Errorf("could not find user '%s' in provided path %s", email, config.DefaultProviderPrincipalForUserPathEnv.MustGet())
}
