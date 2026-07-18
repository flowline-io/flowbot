package nocodb

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// Register registers the nocodb capability with hub and invoker registry.
// When svc is nil the provider is not configured and registration is skipped.
func Register(app string, svc Service) error {
	return capability.Register(capability.Spec{
		Type:        hub.CapNocodb,
		App:         app,
		Description: "NocoDB bases, tables, and records",
		Instance:    svc,
		Ops: []capability.OpDef{
			{
				Name: OpListBases, Description: "List bases", Scopes: []string{auth.ScopeServiceNocodbRead},
				Handler: invokeListBases(svc),
			},
			{
				Name: OpListTables, Description: "List tables in a base", Scopes: []string{auth.ScopeServiceNocodbRead},
				Input: []hub.ParamDef{
					{Name: "base_id", Type: "string", Required: true, Description: "Base ID"},
				},
				Handler: invokeListTables(svc),
			},
			{
				Name: OpGetTable, Description: "Get table metadata", Scopes: []string{auth.ScopeServiceNocodbRead},
				Input: []hub.ParamDef{
					{Name: "table_id", Type: "string", Required: true, Description: "Table ID"},
				},
				Handler: invokeGetTable(svc),
			},
			{
				Name: OpListRecords, Description: "List records in a table", Scopes: []string{auth.ScopeServiceNocodbRead},
				Input: []hub.ParamDef{
					{Name: "table_id", Type: "string", Required: true, Description: "Table ID"},
					{Name: "limit", Type: "number", Description: "Max records to return"},
					{Name: "offset", Type: "number", Description: "Record offset"},
					{Name: "where", Type: "string", Description: "NocoDB where filter"},
					{Name: "sort", Type: "string", Description: "Sort expression"},
					{Name: "fields", Type: "string", Description: "Comma-separated field names"},
				},
				Handler: invokeListRecords(svc),
			},
			{
				Name: OpGetRecord, Description: "Get a record by ID", Scopes: []string{auth.ScopeServiceNocodbRead},
				Input: []hub.ParamDef{
					{Name: "table_id", Type: "string", Required: true, Description: "Table ID"},
					{Name: "record_id", Type: "string", Required: true, Description: "Record ID"},
				},
				Handler: invokeGetRecord(svc),
			},
			{
				Name: OpCreateRecord, Description: "Create a record", Scopes: []string{auth.ScopeServiceNocodbWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "table_id", Type: "string", Required: true, Description: "Table ID"},
					{Name: "fields", Type: "object", Required: true, Description: "Field values"},
				},
				Handler: invokeCreateRecord(svc, app),
			},
			{
				Name: OpUpdateRecord, Description: "Update a record", Scopes: []string{auth.ScopeServiceNocodbWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "table_id", Type: "string", Required: true, Description: "Table ID"},
					{Name: "record_id", Type: "string", Required: true, Description: "Record ID"},
					{Name: "fields", Type: "object", Required: true, Description: "Field values"},
				},
				Handler: invokeUpdateRecord(svc, app),
			},
			{
				Name: OpDeleteRecord, Description: "Delete a record", Scopes: []string{auth.ScopeServiceNocodbWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "table_id", Type: "string", Required: true, Description: "Table ID"},
					{Name: "record_id", Type: "string", Required: true, Description: "Record ID"},
				},
				Handler: invokeDeleteRecord(svc),
			},
			{
				Name: OpHealth, Description: "Health check", Scopes: []string{auth.ScopeServiceNocodbRead},
				Handler: invokeHealth(svc),
			},
		},
	})
}

func invokeListBases(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		result, err := svc.ListBases(ctx)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.NocoBase]{Items: []*capability.NocoBase{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeListTables(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		baseID, err := capability.RequiredString(params, "base_id")
		if err != nil {
			return nil, err
		}
		result, err := svc.ListTables(ctx, ListTablesInput{BaseID: baseID})
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.NocoTable]{Items: []*capability.NocoTable{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGetTable(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		tableID, err := capability.RequiredString(params, "table_id")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetTable(ctx, GetTableInput{TableID: tableID})
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item}, nil
	}
}

func invokeListRecords(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		tableID, err := capability.RequiredString(params, "table_id")
		if err != nil {
			return nil, err
		}
		in := ListRecordsInput{TableID: tableID}
		if where, ok := capability.StringParam(params, "where"); ok {
			in.Where = where
		}
		if sort, ok := capability.StringParam(params, "sort"); ok {
			in.Sort = sort
		}
		if fields, ok := capability.StringParam(params, "fields"); ok {
			in.Fields = fields
		}
		if limit, ok := capability.IntParam(params, "limit"); ok {
			in.Limit = limit
		}
		if offset, ok := capability.IntParam(params, "offset"); ok {
			in.Offset = offset
		}
		items, err := svc.ListRecords(ctx, in)
		if err != nil {
			return nil, err
		}
		if items == nil {
			items = &capability.ListResult[capability.NocoRecord]{Items: []*capability.NocoRecord{}}
		}
		return &capability.InvokeResult{Data: items.Items, Page: items.Page}, nil
	}
}

func invokeGetRecord(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		tableID, err := capability.RequiredString(params, "table_id")
		if err != nil {
			return nil, err
		}
		recordID, err := capability.RequiredString(params, "record_id")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetRecord(ctx, GetRecordInput{TableID: tableID, RecordID: recordID})
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item}, nil
	}
}

func invokeCreateRecord(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		tableID, err := capability.RequiredString(params, "table_id")
		if err != nil {
			return nil, err
		}
		fields, err := requiredFields(params, "fields")
		if err != nil {
			return nil, err
		}
		item, err := svc.CreateRecord(ctx, CreateRecordInput{TableID: tableID, Fields: fields})
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

func invokeUpdateRecord(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		tableID, err := capability.RequiredString(params, "table_id")
		if err != nil {
			return nil, err
		}
		recordID, err := capability.RequiredString(params, "record_id")
		if err != nil {
			return nil, err
		}
		fields, err := requiredFields(params, "fields")
		if err != nil {
			return nil, err
		}
		item, err := svc.UpdateRecord(ctx, UpdateRecordInput{
			TableID: tableID, RecordID: recordID, Fields: fields,
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

func invokeDeleteRecord(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		tableID, err := capability.RequiredString(params, "table_id")
		if err != nil {
			return nil, err
		}
		recordID, err := capability.RequiredString(params, "record_id")
		if err != nil {
			return nil, err
		}
		if err := svc.DeleteRecord(ctx, DeleteRecordInput{TableID: tableID, RecordID: recordID}); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: map[string]any{"deleted": recordID}}, nil
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
