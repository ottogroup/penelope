package impersonate

import (
	"context"
)

type TargetPrincipalForProjectProvider interface {
	GetTargetPrincipalForProject(ctxIn context.Context, projectID string) (target string, delegates []string, err error)
}
