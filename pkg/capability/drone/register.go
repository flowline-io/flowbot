package drone

import (
	"context"
	"fmt"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Register registers the drone capability with hub and invoker registry.
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
		Type:        hub.CapDrone,
		App:         app,
		Description: "Drone CI deploy",
		Instance:    svc,
		Ops: []capability.OpDef{
			{
				Name: OpDeploy, Description: "Trigger a build of the deploy repository for the signed-in caller. Creates or updates data.\nExample: {}", Scopes: []string{auth.ScopeServiceDroneWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "uid", Type: "string", Description: "User id for the deploy notification"},
				},
				Handler: invokeDeploy(svc, app),
			},
			{
				Name: OpHealth, Description: "Health check for the signed-in caller. Read-only.\nExample: {}", Scopes: []string{auth.ScopeServiceDroneRead},
				Handler: invokeHealth(svc),
			},
		},
	}
}

// invokeDeploy reads the notification target from params. The MCP server hides
// uid from the tool schema and replaces it with the token subject; pipeline and
// hub callers set it.
func invokeDeploy(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		uid, _ := capability.StringParam(params, "uid")
		build, err := svc.Deploy(ctx, DeployInput{UID: types.Uid(uid)})
		if err != nil {
			return nil, err
		}
		if build == nil {
			return nil, types.Errorf(types.ErrProvider, "drone deploy returned no build")
		}
		return &capability.InvokeResult{
			Data: build,
			Text: fmt.Sprintf("%s/%s #%d", build.Namespace, build.Repo, build.Number),
			Resource: &capability.ResourceMeta{
				EntityID: strconv.Itoa(build.Number),
				App:      app,
			},
		}, nil
	}
}

func invokeHealth(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		ok, err := svc.HealthCheck(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: ok, Text: "ok"}, nil
	}
}
