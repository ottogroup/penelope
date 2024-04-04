package memory

import (
	"context"
	"github.com/ottogroup/penelope/pkg/repository"
)

type ComplianceRepository struct {
	SinkProjects []string
	InMemory     []*repository.SinkComplianceCheck
}

func (r *ComplianceRepository) ListActiveSinkProjects(_ context.Context) ([]string, error) {
	return r.SinkProjects, nil
}

func (r *ComplianceRepository) UpsertSinkComplianceCheck(_ context.Context, sinkComplianceCheck *repository.SinkComplianceCheck) error {
	found := false
	for i, datum := range r.InMemory {
		if datum.ProjectSink == sinkComplianceCheck.ProjectSink {
			r.InMemory[i].LastCheck = sinkComplianceCheck.LastCheck
			r.InMemory[i].BackupOnly = sinkComplianceCheck.BackupOnly
			found = true
			return nil
		}
	}

	if !found {
		r.InMemory = append(r.InMemory, sinkComplianceCheck)
	}
	return nil
}
