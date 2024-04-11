package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/ottogroup/penelope/pkg/secret"
	"github.com/ottogroup/penelope/pkg/service"
	"go.opencensus.io/trace"
	"time"
)

type ComplianceRepository interface {
	ListActiveSinkProjects(ctx context.Context) ([]string, error)
	UpsertSinkComplianceCheck(ctx context.Context, sinkComplianceCheck *SinkComplianceCheck) error
	GetSinkComplianceCheck(ctx context.Context, projectSink string) (*SinkComplianceCheck, error)
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

func (r *defaultComplianceRepository) GetSinkComplianceCheck(ctx context.Context, projectSink string) (*SinkComplianceCheck, error) {
	_, span := trace.StartSpan(ctx, "(*defaultComplianceRepository).GetSinkComplianceCheck")
	defer span.End()

	sinkComplianceCheck := &SinkComplianceCheck{ProjectSink: projectSink}
	err := r.storageService.
		DB().
		Model(sinkComplianceCheck).
		Where("project_sink = ?", projectSink).
		Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, err
	} else if err != nil {
		logQueryError("GetSinkComplianceCheck", err)
		return nil, fmt.Errorf("error during executing get sink compliance check statement: %s", err)
	}
	return sinkComplianceCheck, nil
}

func (r *defaultComplianceRepository) UpsertSinkComplianceCheck(ctx context.Context, sinkComplianceCheck *SinkComplianceCheck) error {
	_, span := trace.StartSpan(ctx, "(*defaultComplianceRepository).UpsertSinkComplianceCheck")
	defer span.End()

	sinkComplianceCheck.LastCheck = time.Now()

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
		ColumnExpr("DISTINCT target_project").
		Where("audit_deleted_timestamp IS NULL").
		Select(&projects)
	if err != nil {
		logQueryError("ListBackupSinkProjects", err)
		return nil, fmt.Errorf("error during executing get backup by status statement: %s", err)
	}
	return projects, nil
}
