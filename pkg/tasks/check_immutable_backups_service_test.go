package tasks

import (
	"context"
	"github.com/ottogroup/penelope/pkg/http/mock"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/repository"
	"github.com/ottogroup/penelope/pkg/repository/memory"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheckImmutableBackupsService_Run_Unsafe(t *testing.T) {
	httpMockHandler.Start()
	defer httpMockHandler.Stop()

	httpMockHandler.Register(mock.ListPoliciesUnsafeHTTPMock, mock.ListServiceUsageHTTPMock)

	ctx := context.Background()
	complianceRepository := &memory.ComplianceRepository{
		SinkProjects: []string{"test-example-unsafe"},
		InMemory:     []*repository.SinkComplianceCheck{},
	}
	impersonatedTokenConfigProvider := provider.NewDefaultImpersonatedTokenConfigProvider()
	service, err := newSinkProjectComplianceCheckService(complianceRepository, impersonatedTokenConfigProvider)
	assert.NoError(t, err)

	service.Run(ctx)
	assert.Len(t, complianceRepository.InMemory, 1, "should have one sink compliance check")
	assert.False(t, complianceRepository.InMemory[0].SingleWriter, "target sink should be safe: %s", complianceRepository.InMemory[0].ProjectSink)
}

func TestCheckImmutableBackupsService_Run_Safe(t *testing.T) {
	httpMockHandler.Start()
	defer httpMockHandler.Stop()

	httpMockHandler.Register(mock.ListPoliciesSafeHTTPMock, mock.ListServiceUsageHTTPMock)

	ctx := context.Background()
	complianceRepository := &memory.ComplianceRepository{
		SinkProjects: []string{"test-example-safe"},
	}
	impersonatedTokenConfigProvider := provider.NewDefaultImpersonatedTokenConfigProvider()
	service, err := newSinkProjectComplianceCheckService(complianceRepository, impersonatedTokenConfigProvider)
	assert.NoError(t, err)

	service.Run(ctx)
	assert.Len(t, complianceRepository.InMemory, 1, "should have one sink compliance check")
	assert.True(t, complianceRepository.InMemory[0].SingleWriter, "target sink should be safe: %s", complianceRepository.InMemory[0].ProjectSink)
}
