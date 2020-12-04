package provider

import (
	"context"
	"fmt"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
)

type defaultImpersonatedTokenConfigProvider struct {
}

func NewDefaultImpersonatedTokenConfigProvider() impersonate.TargetPrincipalForProjectProvider {
	return &defaultImpersonatedTokenConfigProvider{}
}

func (ip *defaultImpersonatedTokenConfigProvider) GetTargetPrincipalForProject(context.Context, string) (string, error) {
	if config.DefaultProviderImpersonateGoogleServiceAccountEnv.Exist() {
		return config.DefaultProviderImpersonateGoogleServiceAccountEnv.MustGet(), nil
	}
	return "", fmt.Errorf("no default target principal provided")
}
