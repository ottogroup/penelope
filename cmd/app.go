package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

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
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"
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
	ctx := context.Background()

	creds, err := oauth.NewApplicationDefault(ctx)
	if err != nil {
		log.Fatalf("Failed to load application default credentials: %v", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
		resource.WithFromEnv(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String("penelope"),
			attribute.String("gcp.project_id", config.GCPProjectId.MustGet()),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create OpenTelemetry resource: %v", err)
	}

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint("telemetry.googleapis.com:443"),
		otlptracegrpc.WithDialOption(grpc.WithPerRPCCredentials(creds)),
	)
	if err != nil {
		log.Fatalf("Failed to create OTLP trace exporter: %v", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
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
