package cmd

import (
	"flag"
	"fmt"
	"log"
	"os"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/builder"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/http/auth"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/http/rest"
	"github.com/ottogroup/penelope/pkg/http/server"
	"github.com/ottogroup/penelope/pkg/processor"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/secret"
	"go.opencensus.io/trace"
)

var envKeys = []config.EnvKey{
	config.GCPProjectId,
	config.PgUserEnv,
	config.PgDbEnv,
	config.DefaultBucketStorageClass,
}

// AppStartArguments holds the necessary arguments to start the app
type AppStartArguments struct {
	SourceGCPProjectProvider          provider.SourceGCPProjectProvider
	SinkGCPProjectProvider            provider.SinkGCPProjectProvider
	TargetPrincipalForProjectProvider impersonate.TargetPrincipalForProjectProvider
	SecretProvider                    secret.SecretProvider
	PrincipalProvider                 provider.PrincipalProvider
}

// Run penelope app and starts rest api
func Run(args AppStartArguments) {
	glog.Infoln("Starting penelope...")

	if config.EnableTracingEnv.GetBoolOrDefault(false) {
		createAndRegisterExporters()
	}

	flag.Parse()

	if err := flag.Lookup("logtostderr").Value.Set("true"); err != nil {
		glog.Errorf("error on set logtostderr to true: %s", err)
		os.Exit(1)
	}

	validateEnvironmentVariables()

	tokenValidator, err := newTokenValidator()
	if err != nil {
		glog.Errorf("could not create token validator: %s", err)
		os.Exit(1)
	}

	principalRetriever, err := auth.NewPrincipalRetriever(args.PrincipalProvider)
	if err != nil {
		glog.Errorf("could not create principalRetriever: %s", err)
		os.Exit(1)
	}

	authenticationMiddleware, err := auth.NewAuthenticationMiddleware(tokenValidator, principalRetriever)
	if err != nil {
		glog.Errorf("could not create AuthenticationMiddleware: %s", err)
		os.Exit(1)
	}

	api := rest.NewAPI(rest.NewAPIArgs{
		ProcessorBuilder:    createBuilder(args),
		AuthMiddleware:      authenticationMiddleware,
		TokenSourceProvider: args.TargetPrincipalForProjectProvider,
		CredentialsProvider: args.SecretProvider,
	})

	api.Register()

	s := server.CreateServer(api)

	err = s.Run()

	if err != nil {
		glog.Errorf("error could not start server: %s", err)
		os.Exit(1)
	}
}

func validateEnvironmentVariables() {
	for _, envKey := range envKeys {
		if !envKey.Exist() {
			glog.Errorf("error environment variable %s is not set", envKey)
			os.Exit(1)
		}
	}
}

func createBuilder(provider AppStartArguments) *builder.ProcessorBuilder {
	return builder.NewProcessorBuilder(
		processor.NewCreatingProcessorFactory(provider.SinkGCPProjectProvider, provider.TargetPrincipalForProjectProvider, provider.SecretProvider, provider.SourceGCPProjectProvider),
		processor.NewGettingProcessorFactory(provider.TargetPrincipalForProjectProvider, provider.SecretProvider, provider.SourceGCPProjectProvider),
		processor.NewListingProcessorFactory(provider.TargetPrincipalForProjectProvider, provider.SecretProvider, provider.SourceGCPProjectProvider),
		processor.NewUpdatingProcessorFactory(provider.TargetPrincipalForProjectProvider, provider.SecretProvider),
		processor.NewRestoringProcessorFactory(provider.TargetPrincipalForProjectProvider, provider.SecretProvider),
		processor.NewCalculatingProcessorFactory(provider.SinkGCPProjectProvider, provider.TargetPrincipalForProjectProvider),
		processor.NewComplianceProcessorFactory(provider.TargetPrincipalForProjectProvider, provider.SinkGCPProjectProvider),
		processor.NewBucketListingProcessorFactory(provider.SinkGCPProjectProvider, provider.TargetPrincipalForProjectProvider),
		processor.NewDatasetListingProcessorFactory(provider.SinkGCPProjectProvider, provider.TargetPrincipalForProjectProvider),
		processor.NewConfigRegionsProcessorFactory(),
		processor.NewConfigStorageClassesProcessorFactory(),
		processor.NewSourceProjectGetProcessorFactory(provider.SourceGCPProjectProvider, provider.TargetPrincipalForProjectProvider),
		processor.NewTrashcanCleanUpProcessorFactory(provider.TargetPrincipalForProjectProvider, provider.SecretProvider),
	)
}

func createAndRegisterExporters() {
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	prefix := config.TracingMetricsPrefixEnv.GetOrDefault("penelope-server")
	se, err := stackdriver.NewExporter(stackdriver.Options{
		MetricPrefix: prefix,
		ProjectID:    config.GCPProjectId.MustGet(),
	})
	if err != nil {
		log.Fatalf("Failed to create Stackdriver exporter: %v", err)
	}
	trace.RegisterExporter(se)
}

func newTokenValidator() (auth.TokenValidator, error) {
	if config.DevMode.GetBoolOrDefault(false) {
		return auth.NewEmptyTokenValidator(), nil
	}

	requiredEnvs := []config.EnvKey{config.TokenHeaderKey, config.AppJwtAudienceEnv}
	var missingEnvs []config.EnvKey
	for _, env := range requiredEnvs {
		if !env.Exist() {
			missingEnvs = append(missingEnvs, env)
		}
	}

	if len(missingEnvs) > 0 {
		return nil, fmt.Errorf("required environment variables are missing: %s", requiredEnvs)
	}

	keyForTokenHeader := config.TokenHeaderKey.MustGet()
	appJwtAudience := config.AppJwtAudienceEnv.MustGet()

	tokenValidator, err := auth.NewTokenValidator(keyForTokenHeader, appJwtAudience)
	if err != nil {
		return nil, fmt.Errorf("could not create jwtTokenValidator: %s", err)
	}
	return tokenValidator, nil
}
