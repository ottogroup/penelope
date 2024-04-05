package repository

import (
	"context"
	"github.com/ottogroup/penelope/pkg/service"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestComplianceRepository_UpsertSinkComplianceCheck_Twice(t *testing.T) {
	repository := setUpAndGetComplianceRepository(t)
	ctx := context.Background()

	sinkComplianceCheck := &SinkComplianceCheck{
		ProjectSink: "test",
	}
	err := repository.UpsertSinkComplianceCheck(ctx, sinkComplianceCheck)
	assert.NoError(t, err)

	sinkComplianceCheck.Compliant = true
	err = repository.UpsertSinkComplianceCheck(ctx, sinkComplianceCheck)
	assert.NoError(t, err)

	_, err = repository.storageService.DB().Model(sinkComplianceCheck).WherePK().Delete()
	assert.NoError(t, err)
}

func TestComplianceRepository_UpsertSinkComplianceCheck_Reasons(t *testing.T) {
	repository := setUpAndGetComplianceRepository(t)
	ctx := context.Background()

	sinkComplianceCheck := &SinkComplianceCheck{
		ProjectSink: "test",
		Compliant:   false,
		Reasons:     []string{"reason1", "reason2"},
	}
	err := repository.UpsertSinkComplianceCheck(ctx, sinkComplianceCheck)
	assert.NoError(t, err)

	_, err = repository.storageService.DB().Model(sinkComplianceCheck).WherePK().Delete()
	assert.NoError(t, err)
}

func setUpAndGetComplianceRepository(t *testing.T) *defaultComplianceRepository {
	options := getTestConnectOptions()
	ctx := context.Background()

	storageService, err := service.NewStorageServiceWithConnectionOptions(ctx, options)
	assert.NoError(t, err)

	err = clearDatabase(storageService)
	assert.NoError(t, err)

	return &defaultComplianceRepository{storageService: storageService}
}
