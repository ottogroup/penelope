package main

import (
	"context"
	"os"

	"github.com/golang/glog"
	app "github.com/ottogroup/penelope/cmd"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"github.com/ottogroup/penelope/pkg/tracing"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	bgContext := context.Background()

	appProjectID := os.Getenv(config.GCPProjectId.String())

	targetPrincipalForProjectProvider := provider.NewDefaultImpersonatedTokenConfigProvider()

	gcsClient, err := gcs.NewCloudStorageClient(bgContext, targetPrincipalForProjectProvider, appProjectID)
	if err != nil {
		glog.Errorf("could not create CloudStorageClient: %s", err)
		os.Exit(1)
	}

	principalProvider, err := provider.NewDefaultUserProvider(bgContext, gcsClient)
	if err != nil {
		glog.Errorf("could not create PrincipalProvider: %s", err)
		os.Exit(1)
	}

	sinkGCPProjectProvider, err := provider.NewDefaultGCPBackupProvider(bgContext, gcsClient)
	if err != nil {
		glog.Errorf("could not create SinkGCPProjectProvider: %s", err)
		os.Exit(1)
	}

	secretProvider := secret.NewEnvSecretProvider()

	var tracerProvider trace.TracerProvider
	if config.EnableTracingEnv.GetBoolOrDefault(false) {
		tracerProvider, err = tracing.NewGCPTracerProvider(bgContext, appProjectID)
		if err != nil {
			glog.Errorf("could not create GCP TracerProvider: %s", err)
			os.Exit(1)
		}
	}

	appStartArguments := app.AppStartArguments{
		PrincipalProvider:                 principalProvider,
		SinkGCPProjectProvider:            sinkGCPProjectProvider,
		TargetPrincipalForProjectProvider: targetPrincipalForProjectProvider,
		SecretProvider:                    secretProvider,
		TracerProvider:                    tracerProvider,
	}

	app.Run(appStartArguments)
}
