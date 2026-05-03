package hub

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/homelab"
)

// DiscoveredBinding represents a capability that has been discovered but not
// yet registered in the hub as a full-capability descriptor.
type DiscoveredBinding struct {
	Capability CapabilityType
	Backend    string
	App        string
	Endpoint   *homelab.EndpointInfo
	Auth       *homelab.AuthInfo
	Bound      bool
}

// AutoBind reads the homelab registry and returns a list of discovered
// bindings together with their registration status in the hub.
func AutoBind() []DiscoveredBinding {
	apps := homelab.DefaultRegistry.List()
	bindings := make([]DiscoveredBinding, 0)

	for _, app := range apps {
		for _, cap := range app.Capabilities {
			capType := CapabilityType(cap.Capability)
			_, bound := Default.Get(capType)
			bindings = append(bindings, DiscoveredBinding{
				Capability: capType,
				Backend:    cap.Backend,
				App:        app.Name,
				Endpoint:   cap.Endpoint,
				Auth:       cap.Auth,
				Bound:      bound,
			})
		}
	}

	return bindings
}

// LogDiscovered reads the homelab registry and logs discovered capabilities
// that are not yet registered in the hub. It does not perform automatic
// binding, which requires provider configuration to be set up first.
func LogDiscovered() {
	discovered := AutoBind()
	for _, d := range discovered {
		if d.Bound {
			flog.Info("hub autobind: capability %s already bound for app %s",
				d.Capability, d.App)
			continue
		}
		flog.Info("hub autobind: discovered %s (%s) on app %s (not yet configured)",
			d.Capability, d.Backend, d.App)
	}
}
