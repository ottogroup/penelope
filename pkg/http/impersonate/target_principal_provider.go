package impersonate

import (
    "context"
)

type TargetPrincipalForProjectProvider interface {
    GetTargetPrincipalForProject(ctxIn context.Context, projectID string) (string, error)
}
