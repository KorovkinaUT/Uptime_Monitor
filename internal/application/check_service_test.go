package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

func TestCheckServiceCheckPersistsResult(t *testing.T) {
	//arrange
	monitorID := uuid.New()
	statusCode := 200
	checkedAt := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	monitor := domain.Monitor{ID: monitorID, URL: "https://example.com", TimeoutSeconds: 5, IntervalSeconds: 30}
	monitors := &fakeMonitorRepository{monitor: monitor}
	checks := &fakeCheckRepository{}
	inFlight := &fakeInFlightController{allowed: true}
	checker := &fakeChecker{result: applicationCheckResult{status: domain.CheckStatusUp, statusCode: &statusCode, latencyMS: 42}}
	service := NewCheckService(monitors, checks, checker, inFlight, nil)
	service.now = func() time.Time { return checkedAt }

	//act
	check, err := service.Check(context.Background(), monitorID)

	//assert
	if err != nil {
		t.Errorf("Check() error = %v", err)
		return
	}
	if check.MonitorID != monitorID {
		t.Errorf("MonitorID = %s, want %s", check.MonitorID, monitorID)
	}
	if check.Status != domain.CheckStatusUp || check.StatusCode == nil || *check.StatusCode != statusCode {
		t.Errorf("check result = %+v, want up with status code %d", check, statusCode)
	}
	if check.LatencyMS != 42 || !check.CheckedAt.Equal(checkedAt) {
		t.Errorf("check timing = %d/%s, want 42/%s", check.LatencyMS, check.CheckedAt, checkedAt)
	}
	if checks.created != check {
		t.Errorf("saved check = %+v, want %+v", checks.created, check)
	}
	if inFlight.releaseCalls != 1 {
		t.Errorf("Release calls = %d, want 1", inFlight.releaseCalls)
	}
}

func TestCheckServiceRejectsConcurrentCheck(t *testing.T) {
	//arrange
	inFlight := &fakeInFlightController{allowed: false}
	service := NewCheckService(&fakeMonitorRepository{}, &fakeCheckRepository{}, &fakeChecker{}, inFlight, nil)

	//act
	_, err := service.Check(context.Background(), uuid.New())

	//assert
	if !errors.Is(err, domain.ErrCheckAlreadyRunning) {
		t.Errorf("Check() error = %v, want ErrCheckAlreadyRunning", err)
	}
	if inFlight.releaseCalls != 0 {
		t.Errorf("Release calls = %d, want 0", inFlight.releaseCalls)
	}
}
