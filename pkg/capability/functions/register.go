package functions

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	pkgfunctions "github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// serviceMarker is a non-nil instance used for hub registration.
type serviceMarker struct{}

// Register registers hub.CapFunctions with invoke/get/health operations.
func Register() error {
	return capability.Register(buildSpec())
}

// CatalogSpec returns capability metadata for documentation (handlers must not be invoked).
func CatalogSpec() capability.Spec {
	return buildSpec()
}

func buildSpec() capability.Spec {
	return capability.Spec{
		Type:        hub.CapFunctions,
		Description: "Named functions (FaaS): pure transform invoke of published function versions. HTTP token/hmac on POST /service/functions/call only; Pipeline and capability.Invoke do not validate function HTTP secrets.",
		Instance:    serviceMarker{},
		Ops: []capability.OpDef{
			{
				Name: OpInvoke, Description: "Invoke a published named function version (platform path; does not check function HTTP token/hmac)", Mutation: true,
				Scopes: []string{auth.ScopeFunctionRun},
				Input: []hub.ParamDef{
					{Name: "name", Type: "string", Required: true, Description: "Function name"},
					{Name: "version", Type: "number", Required: true, Description: "Published version to invoke"},
					{Name: "event", Type: "any", Required: false, Description: "Event payload passed to the function"},
				},
				Handler: invokeInvoker,
			},
			{
				Name: OpGet, Description: "Get published function metadata without secrets",
				Scopes: []string{auth.ScopeFunctionRead},
				Input: []hub.ParamDef{
					{Name: "name", Type: "string", Required: true, Description: "Function name"},
					{Name: "version", Type: "number", Required: false, Description: "Optional published version; latest when omitted"},
				},
				Handler: getInvoker,
			},
			{
				Name: OpHealth, Description: "Functions subsystem health",
				Scopes:  []string{auth.ScopeFunctionRead},
				Handler: healthInvoker,
			},
		},
	}
}

func invokeInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	svc := pkgfunctions.ActiveService()
	if svc == nil || !svc.Ready() {
		return nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	name, err := capability.RequiredString(params, "name")
	if err != nil {
		return nil, err
	}
	version, err := capability.RequiredInt(params, "version")
	if err != nil {
		return nil, err
	}
	result, err := svc.Invoke(ctx, pkgfunctions.InvokeRequest{
		Name:           name,
		Version:        &version,
		Event:          params["event"],
		RequireVersion: true,
	})
	if result != nil && result.Replayed {
		return &capability.InvokeResult{Data: result}, nil
	}
	if err != nil {
		return nil, err
	}
	return &capability.InvokeResult{Data: result}, nil
}

func getInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	svc := pkgfunctions.ActiveService()
	if svc == nil {
		return nil, types.Errorf(types.ErrUnavailable, "function service not ready")
	}
	name, err := capability.RequiredString(params, "name")
	if err != nil {
		return nil, err
	}
	var version *int
	if v, ok := capability.IntParam(params, "version"); ok {
		version = &v
	}
	data, err := svc.GetPublic(ctx, name, version)
	if err != nil {
		return nil, err
	}
	return &capability.InvokeResult{Data: data}, nil
}

func healthInvoker(_ context.Context, _ map[string]any) (*capability.InvokeResult, error) {
	svc := pkgfunctions.ActiveService()
	return &capability.InvokeResult{Data: svc != nil && svc.Ready()}, nil
}
