package builder

import (
    "context"
    "github.com/ottogroup/penelope/pkg/processor"
    "github.com/ottogroup/penelope/pkg/requestobjects"
    "testing"
)

func TestUninitialisedBuilder(t *testing.T) {
    // given
    builder := ProcessorBuilder{}
    // when
    processorObj, err := builder.ProcessorForRequestType(context.Background(), requestobjects.Updating)
    // expect t,hat dummy processor will be given
    if err == nil {
        t.Error("expected error")
    }
    if processorObj != nil {
        t.Error("expected processor to be empty")
    }
}

type ProcessorMock struct {
}

func (*ProcessorMock) Process(context.Context, *processor.Arguments) (*processor.Result, error) {
    panic("implement me")
}

type FactoryMock struct {
    Type                  requestobjects.RequestType
    createProcessorCalled bool
}

func (t *FactoryMock) DoMatchRequestType(requestType requestobjects.RequestType) bool {
    return t.Type.EqualTo(requestType.String())
}

func (t *FactoryMock) CreateProcessor(ctxIn context.Context) (processor.Operations, error) {
    t.createProcessorCalled = true
    return &ProcessorMock{}, nil
}

func TestInitialisedBuilder(t *testing.T) {
    // given
    ctx := context.Background()
    var factories []ProcessorFactory
    bqF := FactoryMock{Type: requestobjects.Updating}
    csF := FactoryMock{Type: requestobjects.Listing}
    factories = append(factories, &bqF)
    factories = append(factories, &csF)
    builder := NewProcessorBuilder(factories)

    // when
    _, createErr := builder.ProcessorForRequestType(ctx, requestobjects.Updating)
    _, listErr := builder.ProcessorForRequestType(ctx, requestobjects.Listing)

    // expect
    if !bqF.createProcessorCalled {
        t.Error("expected CreateProcessor called for canceling factory")
    }
    if createErr != nil {
        t.Error("expected not error for create processor")
    }
    if !csF.createProcessorCalled {
        t.Error("expected CreateProcessor called for list factory")
    }
    if listErr != nil {
        t.Error("expected not error for list processor")
    }
}
