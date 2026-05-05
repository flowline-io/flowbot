package finance

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

func Descriptor(backend, app string, svc Service) hub.Descriptor {
	return hub.Descriptor{
		Type:        hub.CapFinance,
		Backend:     backend,
		App:         app,
		Description: "Finance capability",
		Instance:    svc,
		Healthy:     svc != nil,
		Operations: []hub.Operation{
			{Name: ability.OpFinanceCreateTransaction, Description: "Create a transaction", Scopes: []string{}},
		},
	}
}

func RegisterService(backend, app string, svc Service) error {
	if svc == nil {
		return types.Errorf(types.ErrInvalidArgument, "finance service is required")
	}
	if err := hub.Default.Register(Descriptor(backend, app, svc)); err != nil {
		return err
	}
	for _, item := range []struct {
		operation string
		invoker   ability.Invoker
	}{
		{operation: ability.OpFinanceCreateTransaction, invoker: invokeCreateTransaction(svc)},
	} {
		if err := ability.RegisterInvoker(hub.CapFinance, item.operation, item.invoker); err != nil {
			return err
		}
	}
	return nil
}

func invokeCreateTransaction(svc Service) ability.Invoker {
	return func(ctx context.Context, params map[string]any) (*ability.InvokeResult, error) {
		req := CreateTransactionRequest{}
		req.Description, _ = ability.StringParam(params, "description")
		req.Amount, _ = ability.StringParam(params, "amount")
		req.Date, _ = ability.StringParam(params, "date")
		req.SourceID, _ = ability.StringParam(params, "source_id")
		if v, ok := ability.IntParam(params, "destination_id"); ok {
			req.DestinationID = v
		}
		result, err := svc.CreateTransaction(ctx, req)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: result, Text: "transaction created"}, nil
	}
}
