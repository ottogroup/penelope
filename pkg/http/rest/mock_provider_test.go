package rest

import (
    "context"
)

type mockBackupProvider struct {
    Backup string
    Error  error
}

func (mb *mockBackupProvider) GetSinkGCPProjectID(_ context.Context, _ string) (string, error) {
    return mb.Backup, mb.Error
}

type MockImpersonatedTokenConfigProvider struct {
    TargetPrincipal string
    Error           error
}

func (mi *MockImpersonatedTokenConfigProvider) GetTargetPrincipalForProject(_ context.Context, _ string) (string, error) {
    return mi.TargetPrincipal, mi.Error
}

