package secret

import (
    "context"
    "fmt"
    "github.com/ottogroup/penelope/pkg/config"
    "go.opencensus.io/trace"
)

type SecretProvider interface {
  GetSecret(ctxIn context.Context, user string) (string, error)
}

// defaultCloudSQLSecretProvider represent client to read secrets
type defaultEnvSecretProvider struct {

}

func NewEnvSecretProvider() SecretProvider {
    return &defaultEnvSecretProvider{}
}

func (p *defaultEnvSecretProvider) GetSecret(ctxIn context.Context, _ string) (string, error) {
    _, span := trace.StartSpan(ctxIn, "(*defaultEnvSecretProvider).GetSecret")
    defer span.End()

    if config.PgPasswordEnv.Exist() {
        return config.PgPasswordEnv.MustGet(), nil
    }

    return "", fmt.Errorf("required env variable is missing: %s", config.PgPasswordEnv)
}



