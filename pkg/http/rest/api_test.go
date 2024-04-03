package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ottogroup/penelope/pkg/builder"
	"github.com/ottogroup/penelope/pkg/http/auth"
	"github.com/ottogroup/penelope/pkg/http/auth/model"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/processor"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/ottogroup/penelope/pkg/service"
	"github.com/ottogroup/penelope/pkg/tasks"
	"github.com/stretchr/testify/assert"
)

func createBuilder(backupProvider provider.SinkGCPProjectProvider, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialProvider secret.SecretProvider) *builder.ProcessorBuilder {
	return builder.NewProcessorBuilder(
		processor.NewCreatingProcessorFactory(backupProvider, tokenSourceProvider, credentialProvider),
		processor.NewGettingProcessorFactory(tokenSourceProvider, credentialProvider),
		processor.NewListingProcessorFactory(tokenSourceProvider, credentialProvider),
		processor.NewUpdatingProcessorFactory(tokenSourceProvider, credentialProvider),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
}

func restAPIFactoryWithStubFactory(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialProvider secret.SecretProvider) *httptest.Server {
	emptyTokenValidator := auth.NewEmptyTokenValidator()
	authenticationMiddleware, err := auth.NewAuthenticationMiddleware(emptyTokenValidator, givenDefaultPrincipalRetrieverWithoutRoles())
	if err != nil {
		panic(fmt.Errorf("error creating AuthenticationMiddleware: %s", err))
	}
	app := NewRestAPI(builder.NewProcessorBuilder(
		&StubFactory[requestobjects.CreateRequest, requestobjects.BackupResponse]{DefaultValue: requestobjects.BackupResponse{}},
		&StubFactory[requestobjects.GetRequest, requestobjects.BackupResponse]{DefaultValue: requestobjects.BackupResponse{}},
		&StubFactory[requestobjects.ListRequest, requestobjects.ListingResponse]{DefaultValue: requestobjects.ListingResponse{}},
		&StubFactory[requestobjects.UpdateRequest, requestobjects.UpdateResponse]{DefaultValue: requestobjects.UpdateResponse{}},
		&StubFactory[requestobjects.RestoreRequest, requestobjects.RestoreResponse]{DefaultValue: requestobjects.RestoreResponse{}},
		&StubFactory[requestobjects.CalculateRequest, requestobjects.CalculatedResponse]{DefaultValue: requestobjects.CalculatedResponse{}},
		&StubFactory[requestobjects.ComplianceRequest, requestobjects.ComplianceResponse]{DefaultValue: requestobjects.ComplianceResponse{}},
		&StubFactory[requestobjects.BucketListRequest, requestobjects.BucketListResponse]{DefaultValue: requestobjects.BucketListResponse{}},
		&StubFactory[requestobjects.DatasetListRequest, requestobjects.DatasetListResponse]{DefaultValue: requestobjects.DatasetListResponse{}},
		&StubFactory[requestobjects.EmptyRequest, requestobjects.RegionsListResponse]{DefaultValue: requestobjects.RegionsListResponse{}},
		&StubFactory[requestobjects.EmptyRequest, requestobjects.StorageClassListResponse]{DefaultValue: requestobjects.StorageClassListResponse{}},
	), authenticationMiddleware, tokenSourceProvider, credentialProvider)
	return httptest.NewServer(authenticationMiddleware.AddAuthentication(app.ServeHTTP))
}

func TestKnownEndpointKnowMethod(t *testing.T) {
	defer httpMockHandler.Stop()
	httpMockHandler.Start()

	s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
	defer s.Close()
	httpMockHandler.RegisterLocalServer(s.URL)

	var body interface{}
	resp, _ := post(t, s, "/api/backups", body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp, _ = get(t, s, "/api/backups")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestKnownEndpointWithTrailingPath(t *testing.T) {
	defer httpMockHandler.Stop()
	httpMockHandler.Start()

	s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
	defer s.Close()
	httpMockHandler.RegisterLocalServer(s.URL)

	var body interface{}
	resp, _ := post(t, s, "/api/backups", body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUnknownEndpoint(t *testing.T) {
	defer httpMockHandler.Stop()
	httpMockHandler.Start()

	s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
	defer s.Close()
	httpMockHandler.RegisterLocalServer(s.URL)

	resp, _ := get(t, s, "/unknown")
	assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)
}

func TestCronEndpointCallable(t *testing.T) {
	defer httpMockHandler.Stop()
	httpMockHandler.Start()

	s := restAPIFactoryWithStubFactory(nil, secret.NewEnvSecretProvider())
	defer s.Close()
	httpMockHandler.RegisterLocalServer(s.URL)

	defaultRegisteredTasks := []string{tasks.RunNewJobs, tasks.CheckJobsStatus, tasks.CleanupExpiredSinks, tasks.PrepareBackupJobs}
	for _, k := range defaultRegisteredTasks {
		resp, _ := get(t, s, fmt.Sprintf("/api/tasks/%s", k))
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	}
}

type emptyPrincipalRetriever struct {
	email string
	roles []model.ProjectRoleBinding
}

func givenDefaultPrincipalRetrieverWithoutRoles() auth.PrincipalRetriever {
	return &emptyPrincipalRetriever{email: "test@user.com"}
}

func givenDefaultPrincipalRetrieverWithRoles(p []model.ProjectRoleBinding) auth.PrincipalRetriever {
	return &emptyPrincipalRetriever{email: "test@user.com", roles: p}
}

func (p *emptyPrincipalRetriever) RetrieveCurrentPrincipal(context.Context, *http.Request) (*model.Principal, error) {
	if p.email != "" {
		return &model.Principal{
			User: model.User{
				Email: p.email,
			},
			RoleBindings: p.roles,
		}, nil
	}

	return nil, fmt.Errorf("user does not exists")
}

type StubProcessor[T, R any] struct {
	DefaultValue R
}

func (p *StubProcessor[T, R]) Process(ctxIn context.Context, args *processor.Argument[T]) (R, error) {
	return p.DefaultValue, nil
}

type StubFactory[T, R any] struct {
	DefaultValue R
}

func (s *StubFactory[T, R]) CreateProcessor(ctxIn context.Context) (processor.Operation[T, R], error) {
	return &StubProcessor[T, R]{DefaultValue: s.DefaultValue}, nil
}

func deleteBackup(backupID string) error {
	storageService, err := service.NewStorageService(context.Background(), secret.NewEnvSecretProvider())
	if err != nil {
		panic(err)
	}

	_, err = storageService.DB().Model(&repository.Backup{ID: backupID}).WherePK().Delete()
	return err

}
