package application

import (
	"sync"

	"github.com/google/uuid"
)

type InFlightRegistry struct {
	mu     sync.Mutex
	active map[uuid.UUID]struct{}
}

func NewInFlightRegistry() *InFlightRegistry {
	return &InFlightRegistry{active: make(map[uuid.UUID]struct{})}
}

func (r *InFlightRegistry) TryStart(id uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.active[id]; exists {
		return false
	}

	r.active[id] = struct{}{}
	return true
}

func (r *InFlightRegistry) Release(id uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.active, id)
}
