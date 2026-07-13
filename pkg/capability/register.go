package capability

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// OpDef declares a single capability operation for registration.
type OpDef struct {
	Name        string
	Description string
	Scopes      []string
	Mutation    bool
	Input       []hub.ParamDef
	Handler     Invoker
}

// Spec describes a capability for hub + invoker registration.
type Spec struct {
	Type        hub.CapabilityType
	App         string
	Description string
	Events      []hub.EventDef
	Instance    any
	Ops         []OpDef
}

// Register registers hub metadata and invokers from Spec.
// It skips registration and logs a warning when Instance is nil.
func Register(s Spec) error {
	if s.Type == "" {
		return types.Errorf(types.ErrInvalidArgument, "capability type is required")
	}
	if s.Instance == nil {
		flog.Warn("%s capability: service is nil, skipping registration for app %s", s.Type, s.App)
		return nil
	}

	ops := make([]hub.Operation, 0, len(s.Ops))
	opIndex := make(map[string]string, len(s.Ops))
	for _, op := range s.Ops {
		if op.Name == "" {
			return types.Errorf(types.ErrInvalidArgument, "operation name is required")
		}
		if op.Handler == nil {
			return types.Errorf(types.ErrInvalidArgument, "handler required for operation %s", op.Name)
		}
		ops = append(ops, hub.Operation{
			Name:        op.Name,
			Description: op.Description,
			Scopes:      op.Scopes,
			Input:       op.Input,
		})
		opIndex[op.Name] = op.Name
		if op.Mutation {
			registerMutation(op.Name)
		}
	}

	desc := hub.Descriptor{
		Type:        s.Type,
		App:         s.App,
		Description: s.Description,
		Operations:  ops,
		Events:      s.Events,
		Instance:    s.Instance,
		Healthy:     true,
	}
	if err := hub.Default.Register(desc); err != nil {
		return err
	}

	RegisterOperations(s.Type, opIndex)

	for _, op := range s.Ops {
		if err := RegisterInvoker(s.Type, op.Name, op.Handler); err != nil {
			return err
		}
	}
	return nil
}
