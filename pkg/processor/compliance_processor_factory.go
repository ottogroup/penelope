package processor

import (
	"context"
	"fmt"

	"github.com/ottogroup/penelope/pkg/http/auth"
	"github.com/ottogroup/penelope/pkg/requestobjects"
	"go.opencensus.io/trace"
)

type ComplianceProcessorFactory interface {
	CreateProcessor(ctxIn context.Context) (Operation[requestobjects.ComplianceRequest, requestobjects.ComplianceResponse], error)
}

// complianceProcessorFactory create Process for Compliance
type complianceProcessorFactory struct {
}

func NewComplianceProcessorFactory() ComplianceProcessorFactory {
	return &complianceProcessorFactory{}
}

// CreateProcessor return instance of Operations for Calculating
func (c complianceProcessorFactory) CreateProcessor(ctxIn context.Context) (Operation[requestobjects.ComplianceRequest, requestobjects.ComplianceResponse], error) {
	_, span := trace.StartSpan(ctxIn, "(*ComplianceProcessorFactory).CreateProcessor")
	defer span.End()

	return &complianceProcessor{}, nil
}

type complianceProcessor struct {
}

// Process request
func (c *complianceProcessor) Process(ctxIn context.Context, args *Argument[requestobjects.ComplianceRequest]) (requestobjects.ComplianceResponse, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*calculatingProcessor).Process")
	defer span.End()

	var request requestobjects.ComplianceRequest = args.Request

	if !auth.CheckRequestIsAllowed(args.Principal, requestobjects.Calculating, request.Project) {
		return requestobjects.ComplianceResponse{}, fmt.Errorf("%s is not allowed for user %q on project %q", requestobjects.Calculating.String(), args.Principal.User.Email, request.Project)
	}

	//TODO: Implement the logic for the processor
	_ = ctx
	result := requestobjects.ComplianceResponse{}

	return result, nil
}
