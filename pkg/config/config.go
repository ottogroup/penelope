package config

import (
	"github.com/golang/glog"
	"os"
	"strconv"
)

type EnvKey string

const (
	LocalPort                                         EnvKey = "PENELOPE_PORT"
	PprofActiveEnv                                    EnvKey = "PPROF_ACTIVE"
	DefaultBucketStorageClass                         EnvKey = "DEFAULT_BUCKET_STORAGE_CLASS"
	EnableTracingEnv                                  EnvKey = "PENELOPE_TRACING"
	TracingMetricsPrefixEnv                           EnvKey = "PENELOPE_TRACING_METRICS_PREFIX"
	UseDefaultHttpClient                              EnvKey = "PENELOPE_USE_DEFAULT_HTTP_CLIENT"
	UseGrpcWithoutAuthentication                      EnvKey = "PENELOPE_USE_GRPC_WITHOUT_AUTHENTICATION"
	GCPProjectId                                      EnvKey = "GCP_PROJECT_ID"
	AppJwtAudienceEnv                                 EnvKey = "APP_JWT_AUDIENCE"
	PgSocket                                          EnvKey = "POSTGRES_SOCKET"
	PgHostEnv                                         EnvKey = "POSTGRES_HOST"
	PgPortEnv                                         EnvKey = "POSTGRES_PORT"
	PgUserEnv                                         EnvKey = "POSTGRES_USER"
	PgDbEnv                                           EnvKey = "POSTGRES_DB"
	PgPasswordEnv                                     EnvKey = "POSTGRES_PASSWORD"
	PgDebugQueriesEnv                                 EnvKey = "POSTGRES_DEBUG_QUERIES"
	SetTestUser                                       EnvKey = "SET_TEST_USER"
	IsProviderLocal                                   EnvKey = "IS_PROVIDER_LOCAL"
	DefaultProviderBucketEnv                          EnvKey = "DEFAULT_PROVIDER_BUCKET"
	DefaultProviderSinkForProjectPathEnv              EnvKey = "DEFAULT_BACKUP_SINK_PROVIDER_FOR_PROJECT_FILE_PATH"
	DefaultProviderPrincipalForUserPathEnv            EnvKey = "DEFAULT_USER_PRINCIPAL_PROVIDER_FILE_PATH"
	DefaultProviderImpersonateGoogleServiceAccountEnv EnvKey = "DEFAULT_PROVIDER_IMPERSONATE_GOOGLE_SERVICE_ACCOUNT"
	StaticFilesPath                                   EnvKey = "STATIC_FILES_PATH"
	TokenHeaderKey                                    EnvKey = "TOKEN_HEADER_KEY"
	CompanyDomains                                    EnvKey = "COMPANY_DOMAINS"
	CorsAllowedMethods                                EnvKey = "CORS_ALLOWED_METHODS"
	CorsAllowedOrigin                                 EnvKey = "CORS_ALLOWED_ORIGIN"
	CorsAllowedHeaders                                EnvKey = "CORS_ALLOWED_HEADERS"
	TasksValidationHTTPHeaderName                     EnvKey = "TASKS_VALIDATION_HTTP_HEADER_NAME"
	TasksValidationHTTPHeaderValue                    EnvKey = "TASKS_VALIDATION_HTTP_HEADER_VALUE"
	TasksValidationAllowedIPAddresses                 EnvKey = "TASKS_VALIDATION_ALLOWED_IP_ADDRESSES"
)

func (e EnvKey) String() string {
	return string(e)
}

func (e EnvKey) GetOrDefault(defaultValue string) string {
	val, exist := os.LookupEnv(e.String())
	if exist {
		return val
	}
	return defaultValue
}

func (e EnvKey) MustGet() string {
	val := os.Getenv(e.String())
	if val == "" {
		glog.Errorf("Environment variable %s is not provided", e)
		os.Exit(1)
	}
	return val
}

func (e EnvKey) GetBoolOrDefault(defaultValue bool) bool {
	raw, exist := os.LookupEnv(e.String())
	if exist {
		val, _ := strconv.ParseBool(raw)
		return val
	}
	return defaultValue
}

func (e EnvKey) Exist() bool {
	return os.Getenv(e.String()) != ""
}
