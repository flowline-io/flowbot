package fireflyiii

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// Register registers the fireflyiii capability with hub and invoker registry.
// When svc is nil the provider is not configured and registration is skipped.
func Register(app string, svc Service) error {
	return capability.Register(capability.Spec{
		Type:        hub.CapFireflyiii,
		App:         app,
		Description: "Finance capability for Firefly III",
		Instance:    svc,
		Ops: []capability.OpDef{
			{
				Name: OpCreateTransaction, Description: "Create a transaction", Scopes: []string{auth.ScopeServiceFireflyiiiWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "type", Type: "string", Required: true, Description: "Transaction type (withdrawal, deposit, transfer)"},
					{Name: "date", Type: "string", Required: true, Description: "Transaction date (YYYY-MM-DD)"},
					{Name: "amount", Type: "string", Required: true, Description: "Transaction amount"},
					{Name: "description", Type: "string", Required: true, Description: "Transaction description"},
					{Name: "source_id", Type: "string", Required: false, Description: "Source account ID"},
					{Name: "source_name", Type: "string", Required: false, Description: "Source account name"},
					{Name: "destination_id", Type: "string", Required: false, Description: "Destination account ID"},
					{Name: "destination_name", Type: "string", Required: false, Description: "Destination account name"},
					{Name: "category_name", Type: "string", Required: false, Description: "Category name"},
					{Name: "notes", Type: "string", Required: false, Description: "Notes"},
				},
				Handler: invokeCreateTransaction(svc, app),
			},
			{
				Name: OpAbout, Description: "Get Firefly III about info", Scopes: []string{auth.ScopeServiceFireflyiiiRead},
				Handler: invokeAbout(svc),
			},
			{
				Name: OpCurrentUser, Description: "Get current Firefly III user", Scopes: []string{auth.ScopeServiceFireflyiiiRead},
				Handler: invokeCurrentUser(svc),
			},
			{
				Name: OpHealth, Description: "Health check", Scopes: []string{auth.ScopeServiceFireflyiiiRead},
				Handler: invokeHealth(svc),
			},
		},
	})
}

func invokeCreateTransaction(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		typ, err := capability.RequiredString(params, "type")
		if err != nil {
			return nil, err
		}
		date, err := capability.RequiredString(params, "date")
		if err != nil {
			return nil, err
		}
		amount, err := capability.RequiredString(params, "amount")
		if err != nil {
			return nil, err
		}
		description, err := capability.RequiredString(params, "description")
		if err != nil {
			return nil, err
		}
		sourceID, _ := capability.StringParam(params, "source_id")
		sourceName, _ := capability.StringParam(params, "source_name")
		destinationID, _ := capability.StringParam(params, "destination_id")
		destinationName, _ := capability.StringParam(params, "destination_name")
		categoryName, _ := capability.StringParam(params, "category_name")
		notes, _ := capability.StringParam(params, "notes")

		item, err := svc.CreateTransaction(ctx, CreateTransactionInput{
			Type:            typ,
			Date:            date,
			Amount:          amount,
			Description:     description,
			SourceID:        sourceID,
			SourceName:      sourceName,
			DestinationID:   destinationID,
			DestinationName: destinationName,
			CategoryName:    categoryName,
			Notes:           notes,
		})
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{
			Data: item,
			Resource: &capability.ResourceMeta{
				EntityID: item.ID,
				App:      app,
			},
		}, nil
	}
}

func invokeAbout(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		info, err := svc.About(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: info}, nil
	}
}

func invokeCurrentUser(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		user, err := svc.CurrentUser(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: user}, nil
	}
}

func invokeHealth(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		ok, err := svc.HealthCheck(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: ok}, nil
	}
}
