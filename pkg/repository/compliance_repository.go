package repository

import (
	"context"
	"fmt"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/ottogroup/penelope/pkg/service"
	"go.opencensus.io/trace"
)

type ComplianceRepository interface {
	ListActiveSinkProjects(ctx context.Context) ([]string, error)
	UpsertSinkComplianceCheck(ctx context.Context, sinkComplianceCheck *SinkComplianceCheck) error
}

// NewComplianceRepository return instance of ComplianceRepository
func NewComplianceRepository(ctxIn context.Context, credentialsProvider secret.SecretProvider) (ComplianceRepository, error) {
	ctx, span := trace.StartSpan(ctxIn, "NewComplianceRepository")
	defer span.End()
	storageService, err := service.NewStorageService(ctx, credentialsProvider)
	if err != nil {
		return nil, err
	}

	return &defaultComplianceRepository{storageService: storageService}, nil
}

type defaultComplianceRepository struct {
	storageService *service.Service
}

func (r *defaultComplianceRepository) UpsertSinkComplianceCheck(ctx context.Context, sinkComplianceCheck *SinkComplianceCheck) error {
	_, span := trace.StartSpan(ctx, "(*defaultComplianceRepository).UpsertSinkComplianceCheck")
	defer span.End()

	_, err := r.storageService.
		DB().
		Model(sinkComplianceCheck).
		OnConflict("(project_sink) DO UPDATE").
		Insert()
	if err != nil {
		logQueryError("UpsertSinkComplianceCheck", err)
		return fmt.Errorf("error during executing upsert sink compliance check statement: %s", err)
	}
	return nil
}

func (r *defaultComplianceRepository) ListActiveSinkProjects(ctx context.Context) ([]string, error) {
	_, span := trace.StartSpan(ctx, "(*defaultComplianceRepository).ListBackupSinkProjects")
	defer span.End()

	var projects []string
	err := r.storageService.
		DB().
		Model(&Backup{}).
		ColumnExpr("DISTINCT target_sink").
		Where("audit_deleted_timestamp IS NULL").
		Select(&projects)
	if err != nil {
		logQueryError("ListBackupSinkProjects", err)
		return nil, fmt.Errorf("error during executing get backup by status statement: %s", err)
	}
	return projects, nil
}
