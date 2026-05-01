package hub

import (
	"cmp"
	"slices"
	"sync"

	"github.com/flowline-io/flowbot/pkg/types"
)

type Registry struct {
	mu          sync.RWMutex
	descriptors map[CapabilityType]Descriptor
}

var Default = NewRegistry()

func NewRegistry() *Registry {
	return &Registry{descriptors: make(map[CapabilityType]Descriptor)}
}

func (r *Registry) Register(desc Descriptor) error {
	if desc.Type == "" {
		return types.Errorf(types.ErrInvalidArgument, "capability type is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.descriptors[desc.Type] = desc
	return nil
}

func (r *Registry) Get(capability CapabilityType) (Descriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	desc, ok := r.descriptors[capability]
	return desc, ok
}

func (r *Registry) List() []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Descriptor, 0, len(r.descriptors))
	for _, desc := range r.descriptors {
		result = append(result, desc)
	}
	slices.SortFunc(result, func(a, b Descriptor) int {
		return cmp.Compare(a.Type, b.Type)
	})
	return result
}
