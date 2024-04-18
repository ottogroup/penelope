package processor

import (
	"context"
	"fmt"

	"github.com/ottogroup/penelope/pkg/http/auth"
	"github.com/ottogroup/penelope/pkg/http/impersonate"
	"github.com/ottogroup/penelope/pkg/provider"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
)

type SourceProjectGetProcessorFactory interface {
	CreateProcessor(ctxIn context.Context) (Operation[requestobjects.SourceProjectGetRequest, requestobjects.SourceProjectGetResponse], error)
}

// SourceProjectGetProcessorFactory create Process for SourceProjectGet
type sourceProjectGetProcessorFactory struct {
	sourceGCPProjectProvider provider.SourceGCPProjectProvider
	tokenSourceProvider      impersonate.TargetPrincipalForProjectProvider
}

func NewSourceProjectGetProcessorFactory(sourceGCPProjectProvider provider.SourceGCPProjectProvider, tokenSourceProvider impersonate.TargetPrincipalForProjectProvider) SourceProjectGetProcessorFactory {
	return &sourceProjectGetProcessorFactory{
		sourceGCPProjectProvider: sourceGCPProjectProvider,
		tokenSourceProvider:      tokenSourceProvider,
	}
}

// CreateProcessor return instance of Operations for SourceProjectGet
func (c *sourceProjectGetProcessorFactory) CreateProcessor(ctxIn context.Context) (Operation[requestobjects.SourceProjectGetRequest, requestobjects.SourceProjectGetResponse], error) {
	return &sourceProjectGetProcessor{
		sourceGCPProjectProvider: c.sourceGCPProjectProvider,
		tokenSourceProvider:      c.tokenSourceProvider,
	}, nil
}

type sourceProjectGetProcessor struct {
	sourceGCPProjectProvider provider.SourceGCPProjectProvider
	tokenSourceProvider      impersonate.TargetPrincipalForProjectProvider
}

// Process request
func (l sourceProjectGetProcessor) Process(ctxIn context.Context, args *Argument[requestobjects.SourceProjectGetRequest]) (requestobjects.SourceProjectGetResponse, error) {
	ctx, span := trace.StartSpan(ctxIn, "(sourceProjectGetProcessor).Process")
	defer span.End()

	var request = &args.Request

	sourceProject, err := l.sourceGCPProjectProvider.GetSourceGCPProject(ctx, request.Project)
	if err != nil {
		return requestobjects.SourceProjectGetResponse{}, err
	}

	if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.SourceProjectGet, request.Project) {
		return requestobjects.SourceProjectGetResponse{}, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.SourceProjectGet.String(), args.Principal.User.Email, sourceProject)
	}

	var sourceProjectGetResponse = requestobjects.SourceProjectGetResponse{
		SourceProject: provider.SourceGCPProject{
			AvailabilityClass: sourceProject.AvailabilityClass,
			DataOwner:         sourceProject.DataOwner,
		},
	}

	return sourceProjectGetResponse, err
}
