package main

import (
	"context"
	"github.com/golang/glog"
	app "github.com/ottogroup/penelope/cmd"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/ottogroup/penelope/pkg/service/gcs"
	"os"
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

	appStartArguments := app.AppStartArguments{
		PrincipalProvider:                 principalProvider,
		SinkGCPProjectProvider:            sinkGCPProjectProvider,
		TargetPrincipalForProjectProvider: targetPrincipalForProjectProvider,
		SecretProvider:                    secretProvider,
	}

	app.Run(appStartArguments)
}
