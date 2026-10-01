package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ulyana/uptime-monitor/internal/domain"
)

func TestMonitorRepositoryCreateAndGetByID(t *testing.T) {
	//arrange
	pool := testPool(t)
	repository := NewMonitorRepository(pool)
	monitor := domain.Monitor{
		ID:              newTestUUID(),
		Name:            "API",
		URL:             "https://example.com",
		IntervalSeconds: 30,
		TimeoutSeconds:  5,
		Enabled:         true,
	}

	//act
	saved, createErr := repository.Create(context.Background(), monitor)
	found, getErr := repository.GetByID(context.Background(), monitor.ID)

	//assert
	if createErr != nil {
		t.Errorf("Create() error = %v", createErr)
		return
	}
	if getErr != nil {
		t.Errorf("GetByID() error = %v", getErr)
		return
	}
	if saved != monitor {
		t.Errorf("saved monitor = %+v, want %+v", saved, monitor)
	}
	if found != monitor {
		t.Errorf("found monitor = %+v, want %+v", found, monitor)
	}
}

func TestMonitorRepositoryListWithLastCheck(t *testing.T) {
	//arrange
	pool := testPool(t)
	repository := NewMonitorRepository(pool)
	monitor := createTestMonitor(t, pool)
	checkedAt := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	statusCode := 200
	insertTestCheck(t, pool, domain.Check{
		ID:         newTestUUID(),
		MonitorID:  monitor.ID,
		Status:     domain.CheckStatusUp,
		StatusCode: &statusCode,
		LatencyMS:  18,
		CheckedAt:  checkedAt,
	})

	//act
	views, err := repository.ListWithLastCheck(context.Background())

	//assert
	if err != nil {
		t.Errorf("ListWithLastCheck() error = %v", err)
		return
	}
	if len(views) != 1 {
		t.Errorf("views length = %d, want 1", len(views))
		return
	}
	if views[0].Monitor != monitor {
		t.Errorf("monitor = %+v, want %+v", views[0].Monitor, monitor)
	}
	if views[0].LastCheck == nil {
		t.Errorf("LastCheck = nil, want latest check")
		return
	}
	if views[0].LastCheck.Status != domain.CheckStatusUp || views[0].LastCheck.StatusCode == nil {
		t.Errorf("latest check = %+v, want up with status code", views[0].LastCheck)
	}
	if views[0].LastCheck.StatusCode != nil && *views[0].LastCheck.StatusCode != statusCode {
		t.Errorf("status code = %d, want %d", *views[0].LastCheck.StatusCode, statusCode)
	}
}

func TestMonitorRepositoryListDue(t *testing.T) {
	//arrange
	pool := testPool(t)
	repository := NewMonitorRepository(pool)
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	due := createTestMonitor(t, pool)
	disabled := createTestMonitor(t, pool)
	disabled.Enabled = false
	if err := repository.Update(context.Background(), disabled); err != nil {
		t.Fatalf("disable test monitor: %v", err)
	}

	notDue := createTestMonitor(t, pool)
	insertTestCheck(t, pool, domain.Check{
		ID:        newTestUUID(),
		MonitorID: notDue.ID,
		Status:    domain.CheckStatusUp,
		CheckedAt: now.Add(-time.Second),
	})

	//act
	monitors, err := repository.ListDue(context.Background(), now)

	//assert
	if err != nil {
		t.Errorf("ListDue() error = %v", err)
		return
	}
	if len(monitors) != 1 {
		t.Errorf("monitors length = %d, want 1", len(monitors))
		return
	}
	if monitors[0] != due {
		t.Errorf("monitors = %+v, want only %+v", monitors, due)
	}
}

func TestMonitorRepositoryUpdateAndDelete(t *testing.T) {
	//arrange
	pool := testPool(t)
	repository := NewMonitorRepository(pool)
	monitor := createTestMonitor(t, pool)
	monitor.Name = "Updated API"
	monitor.Enabled = false

	//act
	updateErr := repository.Update(context.Background(), monitor)
	found, getErr := repository.GetByID(context.Background(), monitor.ID)
	deleteErr := repository.Delete(context.Background(), monitor.ID)
	_, notFoundErr := repository.GetByID(context.Background(), monitor.ID)

	//assert
	if updateErr != nil {
		t.Errorf("Update() error = %v", updateErr)
	}
	if getErr != nil {
		t.Errorf("GetByID() after update error = %v", getErr)
	} else if found != monitor {
		t.Errorf("updated monitor = %+v, want %+v", found, monitor)
	}
	if deleteErr != nil {
		t.Errorf("Delete() error = %v", deleteErr)
	}
	if !errors.Is(notFoundErr, domain.ErrMonitorNotFound) {
		t.Errorf("GetByID() after delete error = %v, want ErrMonitorNotFound", notFoundErr)
	}
}
