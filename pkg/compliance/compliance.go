package compliance

import (
	"context"
	"github.com/golang/glog"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/repository"
	"time"
)

type Check interface {
	Check(ctx context.Context, check *repository.SinkComplianceCheck) error
}

type Compliance interface {
	// CheckSinkProject runs various checks on the sink project and returns the result
	CheckSinkProject(ctx context.Context, sinkProject string) *repository.SinkComplianceCheck
}

func NewCompliance(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) Compliance {
	return &defaultCompliance{
		checks: []Check{
			NewSinkProjectWithSinglerWriterCheck(tokenSourceProvider),
			NewSinkProjectOnlyForBackupCheck(tokenSourceProvider),
		},
	}
}

type defaultCompliance struct {
	checks []Check
}

func (c *defaultCompliance) CheckSinkProject(ctx context.Context, sinkProject string) *repository.SinkComplianceCheck {
	check := &repository.SinkComplianceCheck{
		ProjectSink:  sinkProject,
		BackupOnly:   false,
		SingleWriter: false,
		LastCheck:    time.Now(),
	}

	for _, checker := range c.checks {
		err := checker.Check(ctx, check)
		if err != nil {
			glog.Errorf("Error checking compliance for sink %s: %s", sinkProject, err)
		}
	}

	return check
}
