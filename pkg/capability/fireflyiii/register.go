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
	return capability.Register(buildSpec(app, svc))
}

// CatalogSpec returns capability metadata for documentation (handlers may close over a nil service and must not be invoked).
func CatalogSpec() capability.Spec {
	return buildSpec("", nil)
}

func buildSpec(app string, svc Service) capability.Spec {
	return capability.Spec{
		Type:        hub.CapFireflyiii,
		App:         app,
		Description: "Finance capability for Firefly III",
		Instance:    svc,
		Ops: []capability.OpDef{
			{
				Name: OpCreateTransaction, Description: "Create a transaction for the signed-in caller. Creates or updates data.\nExample: {\"type\":\"example\"}", Scopes: []string{auth.ScopeServiceFireflyiiiWrite}, Mutation: true,
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
					{Name: "budget_id", Type: "string", Required: false, Description: "Budget ID"},
					{Name: "budget_name", Type: "string", Required: false, Description: "Budget name"},
					{Name: "bill_id", Type: "string", Required: false, Description: "Bill ID"},
					{Name: "bill_name", Type: "string", Required: false, Description: "Bill name"},
					{Name: "notes", Type: "string", Required: false, Description: "Notes"},
					{Name: "tags", Type: "[]string", Required: false, Description: "Tags"},
				},
				Handler: invokeCreateTransaction(svc, app),
			},
			{
				Name: OpListTransactions, Description: "List transactions for the signed-in caller. Read-only.\nExample: {\"limit\":20}", Scopes: []string{auth.ScopeServiceFireflyiiiRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Opaque pagination cursor"},
					{Name: "type", Type: "string", Required: false, Description: "Transaction type filter"},
					{Name: "start", Type: "string", Required: false, Description: "Start date (YYYY-MM-DD)"},
					{Name: "end", Type: "string", Required: false, Description: "End date (YYYY-MM-DD)"},
					{Name: "account_id", Type: "string", Required: false, Description: "Filter by account ID"},
				},
				Handler: invokeListTransactions(svc),
			},
			{
				Name: OpGetTransaction, Description: "Get a transaction by group ID for the signed-in caller. Read-only.\nExample: {\"id\":\"example\"}", Scopes: []string{auth.ScopeServiceFireflyiiiRead},
				Input: []hub.ParamDef{
					{Name: "id", Type: "string", Required: true, Description: "Transaction group ID"},
				},
				Handler: invokeGetTransaction(svc),
			},
			{
				Name: OpUpdateTransaction, Description: "Update a transaction for the signed-in caller. Creates or updates data.\nExample: {\"id\":\"example\"}", Scopes: []string{auth.ScopeServiceFireflyiiiWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "id", Type: "string", Required: true, Description: "Transaction group ID"},
					{Name: "type", Type: "string", Required: false, Description: "Transaction type (withdrawal, deposit, transfer)"},
					{Name: "date", Type: "string", Required: false, Description: "Transaction date (YYYY-MM-DD)"},
					{Name: "amount", Type: "string", Required: false, Description: "Transaction amount"},
					{Name: "description", Type: "string", Required: false, Description: "Transaction description"},
					{Name: "source_id", Type: "string", Required: false, Description: "Source account ID"},
					{Name: "source_name", Type: "string", Required: false, Description: "Source account name"},
					{Name: "destination_id", Type: "string", Required: false, Description: "Destination account ID"},
					{Name: "destination_name", Type: "string", Required: false, Description: "Destination account name"},
					{Name: "category_name", Type: "string", Required: false, Description: "Category name"},
					{Name: "budget_id", Type: "string", Required: false, Description: "Budget ID"},
					{Name: "budget_name", Type: "string", Required: false, Description: "Budget name"},
					{Name: "bill_id", Type: "string", Required: false, Description: "Bill ID"},
					{Name: "bill_name", Type: "string", Required: false, Description: "Bill name"},
					{Name: "notes", Type: "string", Required: false, Description: "Notes"},
					{Name: "tags", Type: "[]string", Required: false, Description: "Tags"},
				},
				Handler: invokeUpdateTransaction(svc, app),
			},
			{
				Name: OpDeleteTransaction, Description: "Delete a transaction for the signed-in caller. Deletes or removes data.\nExample: {\"id\":\"example\"}", Scopes: []string{auth.ScopeServiceFireflyiiiWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "id", Type: "string", Required: true, Description: "Transaction group ID"},
				},
				Handler: invokeDeleteTransaction(svc),
			},
			{
				Name: OpListAccounts, Description: "List accounts for the signed-in caller. Read-only.\nExample: {\"limit\":20}", Scopes: []string{auth.ScopeServiceFireflyiiiRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Opaque pagination cursor"},
					{Name: "type", Type: "string", Required: false, Description: "Account type filter (asset, expense, revenue, ...)"},
				},
				Handler: invokeListAccounts(svc),
			},
			{
				Name: OpListCategories, Description: "List categories for the signed-in caller. Read-only.\nExample: {\"limit\":20}", Scopes: []string{auth.ScopeServiceFireflyiiiRead},
				Input: paginationParams(),
				Handler: invokeListCategories(svc),
			},
			{
				Name: OpListBudgets, Description: "List budgets for the signed-in caller. Read-only.\nExample: {\"limit\":20}", Scopes: []string{auth.ScopeServiceFireflyiiiRead},
				Input: paginationParams(),
				Handler: invokeListBudgets(svc),
			},
			{
				Name: OpListBills, Description: "List bills for the signed-in caller. Read-only.\nExample: {\"limit\":20}", Scopes: []string{auth.ScopeServiceFireflyiiiRead},
				Input: paginationParams(),
				Handler: invokeListBills(svc),
			},
			{
				Name: OpListTags, Description: "List tags for the signed-in caller. Read-only.\nExample: {\"limit\":20}", Scopes: []string{auth.ScopeServiceFireflyiiiRead},
				Input: paginationParams(),
				Handler: invokeListTags(svc),
			},
			{
				Name: OpAbout, Description: "Get Firefly III about info for the signed-in caller. Read-only.\nExample: {}", Scopes: []string{auth.ScopeServiceFireflyiiiRead},
				Handler: invokeAbout(svc),
			},
			{
				Name: OpCurrentUser, Description: "Get current Firefly III user for the signed-in caller. Read-only.\nExample: {}", Scopes: []string{auth.ScopeServiceFireflyiiiRead},
				Handler: invokeCurrentUser(svc),
			},
			{
				Name: OpHealth, Description: "Health check for the signed-in caller. Read-only.\nExample: {}", Scopes: []string{auth.ScopeServiceFireflyiiiRead},
				Handler: invokeHealth(svc),
			},
		},
	}
}

func paginationParams() []hub.ParamDef {
	return []hub.ParamDef{
		{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
		{Name: "cursor", Type: "string", Required: false, Description: "Opaque pagination cursor"},
	}
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
		budgetID, _ := capability.StringParam(params, "budget_id")
		budgetName, _ := capability.StringParam(params, "budget_name")
		billID, _ := capability.StringParam(params, "bill_id")
		billName, _ := capability.StringParam(params, "bill_name")
		notes, _ := capability.StringParam(params, "notes")
		tags, _ := stringListParam(params, "tags")

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
			BudgetID:        budgetID,
			BudgetName:      budgetName,
			BillID:          billID,
			BillName:        billName,
			Notes:           notes,
			Tags:            tags,
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

func invokeListTransactions(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		limit, _ := capability.IntParam(params, "limit")
		cursor, _ := capability.StringParam(params, "cursor")
		typ, _ := capability.StringParam(params, "type")
		start, _ := capability.StringParam(params, "start")
		end, _ := capability.StringParam(params, "end")
		accountID, _ := capability.StringParam(params, "account_id")
		result, err := svc.ListTransactions(ctx, TransactionListQuery{
			Page:      capability.PageRequest{Limit: limit, Cursor: cursor},
			Type:      typ,
			Start:     start,
			End:       end,
			AccountID: accountID,
		})
		if err != nil {
			return nil, err
		}
		return listInvokeResult(result), nil
	}
}

func invokeGetTransaction(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetTransaction(ctx, id)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item}, nil
	}
}

func invokeUpdateTransaction(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		in := updateInputFromParams(id, params)
		item, err := svc.UpdateTransaction(ctx, in)
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

func updateInputFromParams(id string, params map[string]any) UpdateTransactionInput {
	in := UpdateTransactionInput{ID: id}
	setOptional := func(dst **string, key string) {
		if v, ok := capability.StringParam(params, key); ok {
			*dst = &v
		}
	}
	setOptional(&in.Type, "type")
	setOptional(&in.Date, "date")
	setOptional(&in.Amount, "amount")
	setOptional(&in.Description, "description")
	setOptional(&in.SourceID, "source_id")
	setOptional(&in.SourceName, "source_name")
	setOptional(&in.DestinationID, "destination_id")
	setOptional(&in.DestinationName, "destination_name")
	setOptional(&in.CategoryName, "category_name")
	setOptional(&in.BudgetID, "budget_id")
	setOptional(&in.BudgetName, "budget_name")
	setOptional(&in.BillID, "bill_id")
	setOptional(&in.BillName, "bill_name")
	setOptional(&in.Notes, "notes")
	if tags, ok := stringListParam(params, "tags"); ok {
		in.Tags = &tags
	}
	return in
}

func invokeDeleteTransaction(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		if err := svc.DeleteTransaction(ctx, id); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: true}, nil
	}
}

func invokeListAccounts(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		limit, _ := capability.IntParam(params, "limit")
		cursor, _ := capability.StringParam(params, "cursor")
		typ, _ := capability.StringParam(params, "type")
		result, err := svc.ListAccounts(ctx, AccountListQuery{
			Page: capability.PageRequest{Limit: limit, Cursor: cursor},
			Type: typ,
		})
		if err != nil {
			return nil, err
		}
		return listInvokeResult(result), nil
	}
}

func invokeListCategories(svc Service) capability.Invoker {
	return invokeResourceList(func(ctx context.Context, q ResourceListQuery) (*capability.ListResult[capability.FinanceCategory], error) {
		return svc.ListCategories(ctx, q)
	})
}

func invokeListBudgets(svc Service) capability.Invoker {
	return invokeResourceList(func(ctx context.Context, q ResourceListQuery) (*capability.ListResult[capability.FinanceBudget], error) {
		return svc.ListBudgets(ctx, q)
	})
}

func invokeListBills(svc Service) capability.Invoker {
	return invokeResourceList(func(ctx context.Context, q ResourceListQuery) (*capability.ListResult[capability.FinanceBill], error) {
		return svc.ListBills(ctx, q)
	})
}

func invokeListTags(svc Service) capability.Invoker {
	return invokeResourceList(func(ctx context.Context, q ResourceListQuery) (*capability.ListResult[capability.FinanceTag], error) {
		return svc.ListTags(ctx, q)
	})
}

func invokeResourceList[T any](fn func(context.Context, ResourceListQuery) (*capability.ListResult[T], error)) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		result, err := fn(ctx, resourceListQuery(params))
		if err != nil {
			return nil, err
		}
		return listInvokeResult(result), nil
	}
}

func listInvokeResult[T any](result *capability.ListResult[T]) *capability.InvokeResult {
	if result == nil {
		return &capability.InvokeResult{Data: []*T{}, Page: &capability.PageInfo{}}
	}
	return &capability.InvokeResult{Data: result.Items, Page: result.Page}
}

func resourceListQuery(params map[string]any) ResourceListQuery {
	limit, _ := capability.IntParam(params, "limit")
	cursor, _ := capability.StringParam(params, "cursor")
	return ResourceListQuery{Page: capability.PageRequest{Limit: limit, Cursor: cursor}}
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

func stringListParam(params map[string]any, key string) ([]string, bool) {
	value, ok := params[key]
	if !ok || value == nil {
		return nil, false
	}
	switch v := value.(type) {
	case []string:
		return v, true
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result, true
	}
	return nil, false
}
