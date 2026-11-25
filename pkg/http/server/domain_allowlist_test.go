package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ottogroup/penelope/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDomainAllowlist(t *testing.T) {
	for _, test := range []struct {
		name, host, path, method string
		allowed                  bool
	}{
		{"normal", "prod.example.com", "/api/backups", "GET", true},
		{"case and port", "PROD.EXAMPLE.COM.:443", "/", "GET", true},
		{"static", "prod.example.com", "/static/ui/js/app.js", "GET", true},
		{"health", "tasks.example.com", "/_ah/health", "GET", true},
		{"health trailing slash", "tasks.example.com", "/_ah/health/", "GET", true},
		{"liveness", "tasks.example.com", "/liveness_check", "GET", true},
		{"readiness", "tasks.example.com", "/readiness_check", "GET", true},
		{"health no normal fallback", "prod.example.com", "/_ah/health", "GET", false},
		{"liveness no normal fallback", "prod.example.com", "/liveness_check", "GET", false},
		{"readiness no normal fallback", "prod.example.com", "/readiness_check", "GET", false},
		{"health similar prefix", "tasks.example.com", "/_ah/health-other", "GET", false},
		{"health child path", "tasks.example.com", "/_ah/health/other", "GET", false},
		{"other appengine path", "tasks.example.com", "/_ah/start", "GET", false},
		{"unknown route", "prod.example.com", "/unknown", "GET", true},
		{"tasks", "tasks.example.com", "/api/tasks/run_new_jobs", "GET", true},
		{"tasks root", "tasks.example.com", "/api/tasks", "GET", true},
		{"task trailing slash", "prod.example.com", "/api/tasks/run_new_jobs/", "GET", false},
		{"no normal fallback", "prod.example.com", "/api/tasks/run_new_jobs", "GET", false},
		{"no task fallback", "tasks.example.com", "/api/backups", "GET", false},
		{"similar prefix", "tasks.example.com", "/api/tasks-other", "GET", false},
		{"deleted dev", "dev-dot-project.appspot.com", "/api/backups", "POST", false},
		{"blocked static", "dev-dot-project.appspot.com", "/static/ui/js/app.js", "GET", false},
		{"blocked root", "dev-dot-project.appspot.com", "/", "GET", false},
		{"blocked preflight", "dev-dot-project.appspot.com", "/api/backups", "OPTIONS", false},
		{"blocked health", "dev-dot-project.appspot.com", "/_ah/health", "GET", false},
		{"blocked unknown", "dev-dot-project.appspot.com", "/unknown", "GET", false},
		{"suffix", "prod.example.com.evil.com", "/", "GET", false},
		{"subdomain", "dev.prod.example.com", "/", "GET", false},
		{"empty", "", "/", "GET", false},
		{"invalid port", "prod.example.com:abc", "/", "GET", false},
		{"port out of range", "prod.example.com:65536", "/", "GET", false},
		{"url", "https://prod.example.com", "/", "GET", false},
		{"multiple hosts", "prod.example.com,evil.com", "/", "GET", false},
		{"ipv4", "127.0.0.1:8080", "/", "GET", true},
		{"ipv6", "[::1]:8080", "/", "GET", true},
		{"ipv6 without port", "[::1]", "/", "GET", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			handler, err := newDomainAllowlist(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			}), " prod.example.com ;127.0.0.1;::1", "tasks.example.com")
			require.NoError(t, err)
			request := httptest.NewRequest(test.method, test.path, nil)
			request.Host = test.host
			request.Header.Set("Origin", "https://prod.example.com")
			request.Header.Set("X-Forwarded-Host", "prod.example.com")
			request.Header.Set("Forwarded", "host=prod.example.com")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, test.allowed, called)
			if test.allowed {
				require.Equal(t, http.StatusNoContent, response.Code)
			} else {
				require.Equal(t, http.StatusForbidden, response.Code)
			}
		})
	}
}

func TestDomainAllowlistInvalidConfiguration(t *testing.T) {
	for _, invalid := range []string{"", " ", "prod.example.com;", "*.example.com", "https://example.com", "example.com:443", "example.com/path", "example..com", "-example.com"} {
		t.Run(invalid, func(t *testing.T) {
			_, err := newDomainAllowlist(http.NotFoundHandler(), invalid, "tasks.example.com")
			require.ErrorContains(t, err, "ALLOWED_DOMAINS")
			_, err = newDomainAllowlist(http.NotFoundHandler(), "prod.example.com", invalid)
			require.ErrorContains(t, err, "TASKS_ALLOWED_DOMAINS")
		})
	}
}

func TestServerRejectsMissingDomainConfiguration(t *testing.T) {
	t.Setenv(config.AllowedDomains.String(), "")
	t.Setenv(config.TasksAllowedDomains.String(), "")
	require.ErrorContains(t, CreateServer(http.NotFoundHandler()).Run(), "ALLOWED_DOMAINS")
	t.Setenv(config.AllowedDomains.String(), "prod.example.com")
	require.ErrorContains(t, CreateServer(http.NotFoundHandler()).Run(), "TASKS_ALLOWED_DOMAINS")
}
