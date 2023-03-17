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

func (ip *defaultImpersonatedTokenConfigProvider) GetTargetPrincipalForProject(context.Context, string) (target string, delegate []string, err error) {
	if config.DefaultProviderImpersonateGoogleServiceAccountEnv.Exist() {
		return config.DefaultProviderImpersonateGoogleServiceAccountEnv.MustGet(), delegate, nil
	}
	return target, delegate, fmt.Errorf("no default target principal provided")
}
