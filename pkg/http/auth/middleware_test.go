package auth

import (
	"context"
	"flag"
	"fmt"
	"github.com/ottogroup/penelope/pkg/http/auth/model"
	"github.com/ottogroup/penelope/pkg/http/mock"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var httpMockHandler *mock.HTTPMockHandler

var tokenHeaderKey = "X-Goog-IAP-JWT-Assertion"

func init() {
	testing.Init()
	os.Setenv("GCP_PROJECT_ID", "local-project")

	if os.Getenv("POSTGRES_HOST") == "" {
		os.Setenv("POSTGRES_HOST", "127.0.0.1")
	}
	if os.Getenv("POSTGRES_USER") == "" {
		os.Setenv("POSTGRES_USER", "backupuser")
	}
	if os.Getenv("POSTGRES_DB") == "" {
		os.Setenv("POSTGRES_DB", "backupdatabase")
	}
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		os.Setenv("POSTGRES_PASSWORD", "backupuserpassword")
	}

	os.Setenv("COMPANY_DOMAINS", "@example.org")

	os.Setenv("DEFAULT_BUCKET_STORAGE_CLASS", "REGIONAL")
	os.Setenv("CLOUD_SQL_SECRETS_PATH", "path/to/secret1")
	os.Setenv("CLOUD_SQL_SECRETS_READING_STRATEGY", "ENV")

	os.Setenv("PENELOPE_USE_DEFAULT_HTTP_CLIENT", "true")

	os.Setenv("TOKEN_HEADER_KEY", tokenHeaderKey)

	flag.Lookup("logtostderr").Value.Set("true")
	flag.Parse()

	httpMockHandler = mock.NewHTTPMockHandler()
	httpMockHandler.Register(mock.NewMockedHTTPRequest("GET", "/local-kebab-database/"+os.Getenv("CLOUD_SQL_SECRETS_PATH"), mock.SQLPasswordStorageResponse))
}

type emptyPrincipalRetriever struct {
	email string
}

func givenDefaultPrincipalRetrieverWithoutRoles() PrincipalRetriever {
	return &emptyPrincipalRetriever{email: "test@user.com"}
}

func (p *emptyPrincipalRetriever) RetrieveCurrentPrincipal(context.Context, *http.Request) (*model.Principal, error) {
	if p.email != "" {
		return &model.Principal{
			User: model.User{
				Email: p.email,
			},
			RoleBindings: []model.ProjectRoleBinding{},
		}, nil
	}

	return nil, fmt.Errorf("user does not exists")
}

type stubUserProvider struct {
}

func (s *stubUserProvider) GetPrincipalForEmail(ctx context.Context, email string) (*model.Principal, error) {
	return nil, fmt.Errorf("no principal for user")
}

func TestRequestWithAuthMiddleware_WithoutCredentials(t *testing.T) {
	defer httpMockHandler.Stop()
	httpMockHandler.Start()
	emptyTokenValidator := NewEmptyTokenValidator()
	principal, _ := NewPrincipalRetriever(&stubUserProvider{})
	middleware, err := NewAuthenticationMiddleware(emptyTokenValidator, principal)
	if err != nil {
		t.Error("expected", "instance of AuthenticationMiddleware can be created", "got", fmt.Sprintf("error: %s", err))
		os.Exit(1)
	}

	ts := httptest.NewServer(middleware.AddAuthentication(func(w http.ResponseWriter, r *http.Request) {
		panic("test entered test handler, this should not happen")
	}))
	defer ts.Close()

	httpMockHandler.RegisterLocalServer(ts.URL)
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Error("expected", fmt.Sprintf("http %d", http.StatusUnauthorized), "got", resp.StatusCode)
	}
}

func TestRequestWithAuthMiddleware_WithJWTAssertion(t *testing.T) {
	defer httpMockHandler.Stop()
	httpMockHandler.Start()

	emptyTokenValidator := NewEmptyTokenValidator()
	principal, _ := NewPrincipalRetriever(&stubUserProvider{})
	middleware, err := NewAuthenticationMiddleware(emptyTokenValidator, principal)
	if err != nil {
		t.Error("expected", "instance of AuthenticationMiddleware can be created", "got", fmt.Sprintf("error: %s", err))
		os.Exit(1)
	}

	ts := httptest.NewServer(middleware.AddAuthentication(func(w http.ResponseWriter, r *http.Request) {
		panic("test entered test handler, this should not happen")
	}))
	defer ts.Close()

	httpMockHandler.RegisterLocalServer(ts.URL)
	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Error(err)
	}

	req.Header.Set(tokenHeaderKey, mock.DefaultJWTToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Error("expected", fmt.Sprintf("http %d", http.StatusUnauthorized), "got", resp.StatusCode)
	}
}

func TestRequestWithAuthMiddleware_WithCredentials(t *testing.T) {
	defer httpMockHandler.Stop()
	httpMockHandler.Start()

	emptyTokenValidator := NewEmptyTokenValidator()
	middleware, err := NewAuthenticationMiddleware(emptyTokenValidator, givenDefaultPrincipalRetrieverWithoutRoles())
	if err != nil {
		t.Error("expected", "instance of AuthenticationMiddleware can be created", "got", fmt.Sprintf("error: %s", err))
		os.Exit(1)
	}

	ts := httptest.NewServer(middleware.AddAuthentication(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer ts.Close()

	httpMockHandler.RegisterLocalServer(ts.URL)
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", ts.URL, "t1"), nil)
	if err != nil {
		t.Error(err)
	}

	req.Header.Set(tokenHeaderKey, mock.DefaultJWTToken+"-WithCredentials")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusTeapot {
		t.Error("expected", fmt.Sprintf("http %d", http.StatusTeapot), "got", resp.StatusCode)
	}
}

func TestRequestWithAuthMiddleware_WithCredentialsOfUnknownOrgDomain(t *testing.T) {
	emptyTokenValidator := NewEmptyTokenValidator()
	principal, _ := NewPrincipalRetriever(&stubUserProvider{})
	middleware, err := NewAuthenticationMiddleware(emptyTokenValidator, principal)
	if err != nil {
		t.Error("expected", "instance of AuthenticationMiddleware can be created", "got", fmt.Sprintf("error: %s", err))
		os.Exit(1)
	}

	ts := httptest.NewServer(middleware.AddAuthentication(func(w http.ResponseWriter, r *http.Request) {
		panic("test entered test handler, this should not happen")
	}))
	defer ts.Close()

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", ts.URL, "t2"), nil)
	if err != nil {
		t.Error(err)
	}

	req.Header.Set(tokenHeaderKey, mock.DefaultJWTToken+"-WithCredentialsOfUnknownOrgDomain")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Error("expected", fmt.Sprintf("http %d", http.StatusUnauthorized), "got", resp.StatusCode)
	}
}
