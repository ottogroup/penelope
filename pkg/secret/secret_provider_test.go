package secret

import (
    "context"
    "github.com/stretchr/testify/assert"
    "os"
    "testing"
)

func TestDefaultEnvSecretProvider_GetSecret_Success(t *testing.T) {
    os.Setenv("POSTGRES_PASSWORD", "secret")

    provider := NewEnvSecretProvider()
    actual, err := provider.GetSecret(context.Background(), "")

    assert.NoError(t, err)
    assert.Equal(t, "secret", actual)
}

func TestDefaultEnvSecretProvider_GetSecret_Failed(t *testing.T) {
    os.Setenv("POSTGRES_PASSWORD", "")

    provider := NewEnvSecretProvider()
    actual, err := provider.GetSecret(context.Background(), "")

    assert.Error(t, err)
    assert.Equal(t, "", actual)
}
