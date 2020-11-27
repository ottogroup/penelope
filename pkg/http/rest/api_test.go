package rest

import (
    "context"
    "fmt"
    "github.com/stretchr/testify/assert"
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
    "net/http"
    "net/http/httptest"
    "testing"
)


func createBuilder(backupProvider provider.SinkGCPProjectProvider, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialProvider secret.SecretProvider) *builder.ProcessorBuilder {
    var factories []builder.ProcessorFactory
    factories = append(factories, processor.NewCreatingProcessorFactory(backupProvider, tokenSourceProvider, credentialProvider))
    factories = append(factories, processor.NewGettingProcessorFactory(tokenSourceProvider, credentialProvider))
    factories = append(factories, processor.NewListingProcessorFactory(tokenSourceProvider, credentialProvider))
    factories = append(factories, processor.NewUpdatingProcessorFactory(tokenSourceProvider, credentialProvider))
    return builder.NewProcessorBuilder(factories)
}

func restAPIFactoryWithStubFactory(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider, credentialProvider secret.SecretProvider) *httptest.Server {
    var factories []builder.ProcessorFactory
    factories = append(factories, &StubFactory{requestobjects.Creating})
    factories = append(factories, &StubFactory{requestobjects.Listing})
    factories = append(factories, &StubFactory{requestobjects.Getting})
    factories = append(factories, &StubFactory{requestobjects.Updating})
    emptyTokenValidator := auth.NewEmptyTokenValidator()
    authenticationMiddleware, err := auth.NewAuthenticationMiddleware(emptyTokenValidator, givenDefaultPrincipalRetrieverWithoutRoles())
    if err != nil {
        panic(fmt.Errorf("error creating AuthenticationMiddleware: %s", err))
    }
    app := NewRestAPI(builder.NewProcessorBuilder(factories), authenticationMiddleware, tokenSourceProvider, credentialProvider)
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

type StubProcessor struct {
}

func (*StubProcessor) Process(ctxIn context.Context, p *processor.Arguments) (*processor.Result, error) {
    return &processor.Result{}, nil
}

type StubFactory struct {
    Type requestobjects.RequestType
}

func (s *StubFactory) DoMatchRequestType(requestType requestobjects.RequestType) bool {
    return s.Type.EqualTo(requestType.String())
}

func (s *StubFactory) CreateProcessor(ctxIn context.Context) (processor.Operations, error) {
    return &StubProcessor{}, nil
}

func deleteBackup(backupID string) error {
    storageService, err := service.NewStorageService(context.Background(), secret.NewEnvSecretProvider())
    if err != nil {
        panic(err)
    }

    _, err = storageService.DB().Model(&repository.Backup{ID: backupID}).WherePK().Delete()
    return err

}
