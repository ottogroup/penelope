package tasks

import (
	"context"
	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/compliance"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/repository"
	"go.opencensus.io/trace"
	"time"
)

// sink_project_compliance_check_service
type sinkProjectComplianceCheckService struct {
	complianceRepository repository.ComplianceRepository
	compliance           compliance.Compliance
}

func newSinkProjectComplianceCheckService(complianceRepository repository.ComplianceRepository, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) (*sinkProjectComplianceCheckService, error) {
	return &sinkProjectComplianceCheckService{
		compliance:           compliance.NewCompliance(tokenSourceProvider),
		complianceRepository: complianceRepository,
	}, nil
}

func (c *sinkProjectComplianceCheckService) Run(ctxIn context.Context) {
	ctx, span := trace.StartSpan(ctxIn, "(*sinkProjectComplianceCheckService).Run")
	defer span.End()

	sinkProjects, err := c.complianceRepository.ListActiveSinkProjects(ctx)
	if err != nil {
		glog.Error("could not get list of backups: %s", err)
		return
	}

	for _, sink := range sinkProjects {
		sinkComplianceCheck, err := c.compliance.CheckCompliance(ctx, sink)
		if err != nil {
			glog.Errorf("could not check compliance for sink %s: %s", sink, err)
			continue
		}

		err = c.complianceRepository.UpsertSinkComplianceCheck(ctx, &repository.SinkComplianceCheck{
			ProjectSink: sink,
			Compliant:   sinkComplianceCheck.Compliant,
			Reasons:     sinkComplianceCheck.Reasons,
			LastCheck:   time.Time{},
		})
		if err != nil {
			glog.Errorf("could not upsert target sink %s: %s", sink, err)
		}
	}

	glog.Infof("Finished checking compliance for sink projects")
}
