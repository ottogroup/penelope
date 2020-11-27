package builder

import (
    "context"
    "fmt"
    "github.com/ottogroup/penelope/pkg/http/auth/model"
    "github.com/ottogroup/penelope/pkg/processor"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "go.opencensus.io/trace"
)

// Service define how request are processed
type Service interface {
    ProcessorForRequestType(requestType requestobjects.RequestType) (processor.Operations, error)
    ProcessorArgumentsForRequest(request *requestobjects.CreateRequest) processor.Arguments
}

// ProcessorBuilder is responsible for creating Operations for each request type
type ProcessorBuilder struct {
    factories []ProcessorFactory
}

//NewProcessorBuilder created a new ProcessorBuilder
func NewProcessorBuilder(factories []ProcessorFactory) *ProcessorBuilder {
    return &ProcessorBuilder{factories: factories}
}

// ProcessorForRequestType create Operations for specified RequestType
func (s *ProcessorBuilder) ProcessorForRequestType(ctxIn context.Context, requestType requestobjects.RequestType) (processor.Operations, error) {
    ctx, span := trace.StartSpan(ctxIn, "(*ProcessorBuilder).ProcessorForRequestType")
    defer span.End()

    for _, factory := range s.factories {
        if factory.DoMatchRequestType(requestType) {
            return factory.CreateProcessor(ctx)
        }
    }
    return nil, fmt.Errorf("factory not found")
}

// ProcessorArgumentsForRequest create Arguments for a request
func (s *ProcessorBuilder) ProcessorArgumentsForRequest(request interface{}, principal *model.Principal) processor.Arguments {
    return processor.Arguments{Request: request, Principal: principal}
}
