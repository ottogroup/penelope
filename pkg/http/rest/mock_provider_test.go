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
	Delegates       []string
	Error           error
}

func (mi *MockImpersonatedTokenConfigProvider) GetTargetPrincipalForProject(ctxIn context.Context, projectID string) (string, []string, error) {
	return mi.TargetPrincipal, mi.Delegates, mi.Error
}
