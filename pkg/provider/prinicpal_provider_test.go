package provider

import (
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/ottogroup/penelope/pkg/config"
    "github.com/ottogroup/penelope/pkg/http/auth/model"
    "github.com/ottogroup/penelope/pkg/service/gcs"
    "os"
    "testing"
)

func TestDefaultUserProvider_GetPrincipalForUser_Found(t *testing.T) {
    _ = os.Setenv(config.DefaultProviderBucketEnv.String(), "local-xyz-dev.appspot.com")
    _ = os.Setenv(config.DefaultProviderPrincipalForUserPathEnv.String(), "principal.yaml")

    content := `
- user:
    email: 'some@email.de'
  role_bindings:
    - role: owner
      project: 'local-account'
    - role: viewer
      project: 'local-ability'
`
    provider, err := NewDefaultUserProvider(context.Background(), &gcs.MockGcsClient{
        ClientInitialized: true,
        ShouldFail:        false,
        ObjectContent:     []byte(content),
    })
    assert.NoError(t, err)

    principal, err := provider.GetPrincipalForEmail(context.Background(), "some@email.de")

    assert.NoError(t, err)
    assert.Equal(t, "some@email.de", principal.User.Email)
    assert.Len(t, principal.RoleBindings, 2)
    assert.Equal(t, "local-account", principal.RoleBindings[0].Project)
    assert.Equal(t, model.Owner, principal.RoleBindings[0].Role)
    assert.Equal(t, "local-ability", principal.RoleBindings[1].Project)
    assert.Equal(t, model.Viewer, principal.RoleBindings[1].Role)
}

func TestDefaultUserProvider_GetPrincipalForUser_NotFound(t *testing.T) {
    _ = os.Setenv(config.DefaultProviderBucketEnv.String(), "local-xyz-dev.appspot.com")
    _ = os.Setenv(config.DefaultProviderPrincipalForUserPathEnv.String(), "principal.yaml")

    content := `
- user:
    email: 'some@email.de'
  role_bindings:
    - role: owner
      project: 'local-account'
    - role: viewer
      project: 'local-ability'
`
    provider, err := NewDefaultUserProvider(context.Background(), &gcs.MockGcsClient{
        ClientInitialized: true,
        ShouldFail:        false,
        ObjectContent:     []byte(content),
    })
    assert.NoError(t, err)

    principal, err := provider.GetPrincipalForEmail(context.Background(), "notFound@email.de")
    assert.Error(t, err)
    assert.Nil(t, principal)
}
