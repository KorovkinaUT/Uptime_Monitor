package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

func TestCheckRepositoryCreateAndListByMonitorID(t *testing.T) {
	//arrange
	pool := testPool(t)
	monitor := createTestMonitor(t, pool)
	statusCode := 500
	checkError := "server error"
	check := domain.Check{
		ID:         uuid.New(),
		MonitorID:  monitor.ID,
		Status:     domain.CheckStatusDown,
		StatusCode: &statusCode,
		LatencyMS:  42,
		Error:      &checkError,
		CheckedAt:  time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
	}
	repository := NewCheckRepository(pool)

	//act
	createErr := repository.Create(context.Background(), check)
	checks, listErr := repository.ListByMonitorID(context.Background(), monitor.ID, 100)

	//assert
	if createErr != nil {
		t.Errorf("Create() error = %v", createErr)
		return
	}
	if listErr != nil {
		t.Errorf("ListByMonitorID() error = %v", listErr)
		return
	}
	if len(checks) != 1 {
		t.Errorf("checks length = %d, want 1", len(checks))
		return
	}
	if checks[0].ID != check.ID || checks[0].MonitorID != check.MonitorID {
		t.Errorf("check identity = %+v, want %+v", checks[0], check)
	}
	if checks[0].Status != check.Status || checks[0].LatencyMS != check.LatencyMS {
		t.Errorf("check result = %+v, want %+v", checks[0], check)
	}
	if checks[0].StatusCode == nil || *checks[0].StatusCode != statusCode {
		t.Errorf("status code = %v, want %d", checks[0].StatusCode, statusCode)
	}
	if checks[0].Error == nil || *checks[0].Error != checkError {
		t.Errorf("error = %v, want %q", checks[0].Error, checkError)
	}
}
