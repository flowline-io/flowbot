package hub

import (
	"cmp"
	"slices"
)

type Binding struct {
	Capability CapabilityType `json:"capability"`
	Backend    string         `json:"backend"`
	App        string         `json:"app"`
	Healthy    bool           `json:"healthy"`
}

func (r *Registry) Bindings() []Binding {
	r.mu.RLock()
	defer r.mu.RUnlock()
	bindings := make([]Binding, 0, len(r.descriptors))
	for _, desc := range r.descriptors {
		bindings = append(bindings, Binding{
			Capability: desc.Type,
			Backend:    desc.Backend,
			App:        desc.App,
			Healthy:    desc.Healthy,
		})
	}
	slices.SortFunc(bindings, func(a, b Binding) int {
		return cmp.Compare(a.Capability, b.Capability)
	})
	return bindings
}
