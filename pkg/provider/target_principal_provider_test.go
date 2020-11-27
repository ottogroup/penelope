package provider

import (
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/ottogroup/penelope/pkg/config"
    "os"
    "testing"
)

func TestDefaultImpersonatedTokenConfigProvider_GetTargetPrincipalForProject_ProvideDefault(t *testing.T) {
    os.Setenv(config.DefaultProviderPrincipalForProjectPathEnv.String(), "principal@gsa.google.de")
    defer os.Setenv(config.DefaultProviderPrincipalForProjectPathEnv.String(), "")

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
