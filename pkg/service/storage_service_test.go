package service

import (
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/ottogroup/penelope/pkg/secret"
    "os"
    "testing"
)

func TestCredentialsReader_NoEnv_POSTGRES_HOST(t *testing.T) {
    os.Setenv("POSTGRES_HOST", "")
    os.Setenv("POSTGRES_USER", "sql_user")
    os.Setenv("POSTGRES_DB", "sql_db")

    _, err := NewStorageService(context.Background(), secret.NewEnvSecretProvider())
    assert.Error(t, err)
}

func TestCredentialsReader_NoEnv_For_User(t *testing.T) {
    os.Setenv("POSTGRES_HOST", "127.0.0.1")
    os.Setenv("POSTGRES_USER", "")
    os.Setenv("POSTGRES_DB", "sql_db")

    _, err := NewStorageService(context.Background(), secret.NewEnvSecretProvider())
    assert.Error(t, err)
}

func TestCredentialsReader_NoEnv_For_DB(t *testing.T) {
    os.Setenv("POSTGRES_HOST", "127.0.0.1")
    os.Setenv("POSTGRES_USER", "sql_user")
    os.Setenv("POSTGRES_DB", "")

    _, err := NewStorageService(context.Background(), secret.NewEnvSecretProvider())
    assert.Error(t, err)
}
