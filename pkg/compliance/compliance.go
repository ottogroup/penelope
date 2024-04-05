package compliance

import (
	"context"
	"errors"
	"fmt"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
)

type Result struct {
	Compliant bool
	Reasons   []string
}

type CheckName string

const (
	singleWriter CheckName = "SingleWriter"
	backupOnly   CheckName = "BackupOnly"
)

type CheckError struct {
	CheckName CheckName `json:"checkName"`
	Reason    string    `json:"reason"`
}

func (e *CheckError) Error() string {
	return fmt.Sprintf("Compliance check failed: %s - %s", e.CheckName, e.Reason)
}

type CheckFunc func(ctxIn context.Context, sinkProject string) error

type Compliance interface {
	// CheckCompliance runs various checks on the sink project and returns the result
	CheckCompliance(ctx context.Context, sinkProject string) (Result, error)
}

func NewCompliance(tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) Compliance {
	return &defaultCompliance{
		checks: []CheckFunc{
			NewSinkProjectWithSinglerWriterCheckFunc(tokenSourceProvider),
			NewSinkProjectOnlyForBackupCheckFunc(tokenSourceProvider),
		},
	}
}

type defaultCompliance struct {
	checks []CheckFunc
}

func (c *defaultCompliance) CheckCompliance(ctx context.Context, sinkProject string) (Result, error) {
	var reasons []string
	for _, check := range c.checks {
		if err := check(ctx, sinkProject); err != nil {
			var checkError *CheckError
			if errors.As(err, &checkError) {
				reasons = append(reasons, checkError.Reason)
			} else {
				return Result{}, err
			}
		}
	}

	return Result{
		Compliant: len(reasons) == 0,
		Reasons:   reasons,
	}, nil
}
