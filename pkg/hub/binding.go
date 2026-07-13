package hub

import (
	"cmp"
	"slices"
)

// Binding is a projection of a registered capability for hub APIs.
type Binding struct {
	Capability CapabilityType `json:"capability"`
	App        string         `json:"app"`
	Healthy    bool           `json:"healthy"`
}

// Bindings returns all registered capability bindings sorted by type.
func (r *Registry) Bindings() []Binding {
	r.mu.RLock()
	defer r.mu.RUnlock()
	bindings := make([]Binding, 0, len(r.descriptors))
	for _, desc := range r.descriptors {
		bindings = append(bindings, Binding{
			Capability: desc.Type,
			App:        desc.App,
			Healthy:    desc.Healthy,
		})
	}
	slices.SortFunc(bindings, func(a, b Binding) int {
		return cmp.Compare(a.Capability, b.Capability)
	})
	return bindings
}
