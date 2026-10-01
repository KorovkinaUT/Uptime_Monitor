package application

import (
	"testing"

	"github.com/google/uuid"
)

func TestInFlightRegistry(t *testing.T) {
	//arrange
	registry := NewInFlightRegistry()
	monitorID := uuid.New()

	//act
	first := registry.TryStart(monitorID)
	second := registry.TryStart(monitorID)
	registry.Release(monitorID)
	third := registry.TryStart(monitorID)

	//assert
	if !first {
		t.Errorf("first TryStart() = false, want true")
	}
	if second {
		t.Errorf("second TryStart() = true, want false")
	}
	if !third {
		t.Errorf("TryStart() after Release() = false, want true")
	}
}
