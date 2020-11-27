package builder

import (
    "context"
    "github.com/ottogroup/penelope/pkg/processor"
    "github.com/ottogroup/penelope/pkg/requestobjects"
)

// ProcessorFactory defines common operations for creating Operations
type ProcessorFactory interface {
    DoMatchRequestType(requestType requestobjects.RequestType) bool
    CreateProcessor(ctxIn context.Context) (processor.Operations, error)
}
