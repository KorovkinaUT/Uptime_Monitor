package application

import (
	"sync"

	"github.com/google/uuid"
)

// InFlightController coordinates active monitor checks
type InFlightController interface {
	TryStart(id uuid.UUID) bool
	Release(id uuid.UUID)
}

// InFlightRegistry prevents concurrent checks for the same monitor
type InFlightRegistry struct {
	mu     sync.Mutex
	active map[uuid.UUID]struct{}
}

// NewInFlightRegistry creates an empty in-flight registry
func NewInFlightRegistry() *InFlightRegistry {
	return &InFlightRegistry{active: make(map[uuid.UUID]struct{})}
}

// TryStart reserves a monitor for a new check
func (r *InFlightRegistry) TryStart(id uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.active[id]; exists {
		return false
	}

	r.active[id] = struct{}{}
	return true
}

// Release removes a monitor reservation
func (r *InFlightRegistry) Release(id uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.active, id)
}
