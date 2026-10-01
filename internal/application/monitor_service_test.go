package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

func TestMonitorServiceCreateUsesSavedMonitor(t *testing.T) {
	//arrange
	saved := domain.Monitor{
		ID:              uuid.New(),
		Name:            "API",
		URL:             "https://example.com",
		IntervalSeconds: 30,
		TimeoutSeconds:  5,
		Enabled:         true,
	}
	repository := &fakeMonitorRepository{created: saved}
	service := NewMonitorService(repository, &fakeCheckRepository{})
	input := saved
	input.ID = uuid.Nil

	//act
	view, err := service.Create(context.Background(), input)

	//assert
	if err != nil {
		t.Errorf("Create() error = %v", err)
		return
	}
	if view.Monitor != saved {
		t.Errorf("created monitor = %+v, want %+v", view.Monitor, saved)
	}
	if view.LastCheck != nil {
		t.Errorf("LastCheck = %+v, want nil", view.LastCheck)
	}
	if repository.createCalls != 1 {
		t.Errorf("Create calls = %d, want 1", repository.createCalls)
	}
	if repository.getCalls != 0 {
		t.Errorf("GetByID calls = %d, want 0", repository.getCalls)
	}
}

func TestMonitorServiceGetIncludesLatestCheck(t *testing.T) {
	//arrange
	monitorID := uuid.New()
	latest := domain.Check{ID: uuid.New(), MonitorID: monitorID, Status: domain.CheckStatusUp}
	repository := &fakeMonitorRepository{monitor: domain.Monitor{ID: monitorID}}
	checks := &fakeCheckRepository{history: []domain.Check{latest}}
	service := NewMonitorService(repository, checks)

	//act
	view, err := service.Get(context.Background(), monitorID)

	//assert
	if err != nil {
		t.Errorf("Get() error = %v", err)
		return
	}
	if view.LastCheck == nil {
		t.Errorf("LastCheck = nil, want latest check")
		return
	}
	if *view.LastCheck != latest {
		t.Errorf("LastCheck = %+v, want %+v", *view.LastCheck, latest)
	}
}
