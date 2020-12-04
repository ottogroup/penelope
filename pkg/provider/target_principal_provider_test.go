package provider

import (
	"context"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestDefaultImpersonatedTokenConfigProvider_GetTargetPrincipalForProject_ProvideDefault(t *testing.T) {
	os.Setenv(config.DefaultProviderImpersonateGoogleServiceAccountEnv.String(), "principal@gsa.google.de")
	defer os.Setenv(config.DefaultProviderImpersonateGoogleServiceAccountEnv.String(), "")

	provider := NewDefaultImpersonatedTokenConfigProvider()
	target, err := provider.GetTargetPrincipalForProject(context.Background(), "")

	assert.NoError(t, err)
	assert.Equal(t, "principal@gsa.google.de", target)
}

func TestDefaultImpersonatedTokenConfigProvider_GetTargetPrincipalForProject_MissingEnv(t *testing.T) {
	provider := NewDefaultImpersonatedTokenConfigProvider()
	_, err := provider.GetTargetPrincipalForProject(context.Background(), "")
	assert.Error(t, err)
}
