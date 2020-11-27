package service

import (
    "context"
    "fmt"
    "github.com/go-pg/pg/v10"
    "github.com/ottogroup/penelope/pkg/config"
    "github.com/ottogroup/penelope/pkg/secret"
    "github.com/ottogroup/penelope/pkg/service/sql"
    "go.opencensus.io/trace"
)

// Service represent operation with PostgresSQL
type Service struct {
    sqlClient sql.CloudSQLClient
}

//DefaultConnectionOptions returns default ConnectOptions
func DefaultConnectionOptions(ctxIn context.Context, credentialsProvider secret.SecretProvider) (sql.ConnectOptions, error) {
    ctx, span := trace.StartSpan(ctxIn, "DefaultConnectionOptions")
    defer span.End()

    requiredEnvKeys := []config.EnvKey{config.PgUserEnv, config.PgDbEnv}

    socket := ""
    if config.PgSocket.Exist() {
        socket = config.PgSocket.MustGet()
    } else {
        requiredEnvKeys = append(requiredEnvKeys, config.PgHostEnv)
    }

    var notSet []config.EnvKey
    for _, envKey := range requiredEnvKeys {
        if !envKey.Exist() {
            notSet = append(notSet, envKey)
        }
    }
    if len(notSet) > 0 {
        return sql.ConnectOptions{}, fmt.Errorf("required env are not set: %s", notSet)
    }

    user := config.PgUserEnv.MustGet()

    password, err := credentialsProvider.GetSecret(ctx, user)
    if err != nil {
        return sql.ConnectOptions{}, err
    }

    return sql.ConnectOptions{
        Host:         config.PgHostEnv.GetOrDefault(""),
        Port:         config.PgPortEnv.GetOrDefault("5432"),
        Socket:       socket,
        User:         user,
        Password:     password,
        Database:     config.PgDbEnv.MustGet(),
        DebugQueries: config.PgDebugQueriesEnv.GetOrDefault(""),
    }, nil
}

// NewStorageService create new instance of Service
func NewStorageService(ctxIn context.Context, credentialsProvider secret.SecretProvider) (*Service, error) {
    ctx, span := trace.StartSpan(ctxIn, "NewStorageService")
    defer span.End()

    options, err := DefaultConnectionOptions(ctx, credentialsProvider)
    if err != nil {
        return nil, err
    }

    return NewStorageServiceWithConnectionOptions(ctx, options)
}

// NewStorageServiceWithConnectionOptions create new instance of Service with connection options
func NewStorageServiceWithConnectionOptions(ctxIn context.Context, options sql.ConnectOptions) (*Service, error) {
    _, span := trace.StartSpan(ctxIn, "NewStorageServiceWithConnectionOptions")
    defer span.End()

    sqlClient := sql.NewCloudSQLClient(options)
    if !sqlClient.IsInitialized() {
        return &Service{}, fmt.Errorf("could not instantiate CloudSQLClient")
    }

    return &Service{sqlClient: sqlClient}, nil
}

// DB will create a new connection
func (c *Service) DB() *pg.DB {
    return c.sqlClient.DB()
}

// Close closes db connection
func (c *Service) Close() error {
    return c.sqlClient.Close()
}

