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
			{Name: "create_transaction", Description: "Create a transaction", Scopes: []string{}},
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
		{operation: "create_transaction", invoker: invokeCreateTransaction(svc)},
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
		req.Description, _ = stringParam(params, "description")
		req.Amount, _ = stringParam(params, "amount")
		req.Date, _ = stringParam(params, "date")
		req.SourceID, _ = stringParam(params, "source_id")
		if v, ok := intParam(params, "destination_id"); ok {
			req.DestinationID = v
		}
		result, err := svc.CreateTransaction(ctx, req)
		if err != nil {
			return nil, err
		}
		return &ability.InvokeResult{Data: result, Text: "transaction created"}, nil
	}
}

func stringParam(params map[string]any, key string) (string, bool) {
	value, ok := params[key]
	if !ok || value == nil {
		return "", false
	}
	s, ok := value.(string)
	if !ok {
		return "", false
	}
	return s, true
}

func intParam(params map[string]any, key string) (int, bool) {
	value, ok := params[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	}
	return 0, false
}
