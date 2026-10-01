package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

type fakeMonitorRepository struct {
	created     domain.Monitor
	createErr   error
	createCalls int
	monitor     domain.Monitor
	getErr      error
	getCalls    int
	list        []MonitorWithLastCheck
	listErr     error
	updateErr   error
	updateCalls int
	deleteErr   error
	due         []domain.Monitor
	dueErr      error
}

func (f *fakeMonitorRepository) Create(context.Context, domain.Monitor) (domain.Monitor, error) {
	f.createCalls++
	if f.createErr != nil {
		return domain.Monitor{}, f.createErr
	}
	return f.created, nil
}

func (f *fakeMonitorRepository) GetByID(context.Context, uuid.UUID) (domain.Monitor, error) {
	f.getCalls++
	if f.getErr != nil {
		return domain.Monitor{}, f.getErr
	}
	return f.monitor, nil
}

func (f *fakeMonitorRepository) ListWithLastCheck(context.Context) ([]MonitorWithLastCheck, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.list, nil
}

func (f *fakeMonitorRepository) Update(context.Context, domain.Monitor) error {
	f.updateCalls++
	return f.updateErr
}

func (f *fakeMonitorRepository) Delete(context.Context, uuid.UUID) error {
	return f.deleteErr
}

func (f *fakeMonitorRepository) ListDue(context.Context, time.Time) ([]domain.Monitor, error) {
	if f.dueErr != nil {
		return nil, f.dueErr
	}
	return f.due, nil
}

type fakeCheckRepository struct {
	created   domain.Check
	createErr error
	history   []domain.Check
	listErr   error
}

func (f *fakeCheckRepository) Create(_ context.Context, check domain.Check) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = check
	return nil
}

func (f *fakeCheckRepository) ListByMonitorID(context.Context, uuid.UUID, int) ([]domain.Check, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.history, nil
}

type fakeChecker struct {
	result applicationCheckResult
	err    error
}

type applicationCheckResult struct {
	status     domain.CheckStatus
	statusCode *int
	latencyMS  int64
	err        *string
}

func (f *fakeChecker) Check(context.Context, domain.Monitor) (CheckResult, error) {
	return CheckResult{
		Status:     f.result.status,
		StatusCode: f.result.statusCode,
		LatencyMS:  f.result.latencyMS,
		Error:      f.result.err,
	}, f.err
}

type fakeInFlightController struct {
	allowed       bool
	tryStartCalls int
	releaseCalls  int
}

func (f *fakeInFlightController) TryStart(uuid.UUID) bool {
	f.tryStartCalls++
	return f.allowed
}

func (f *fakeInFlightController) Release(uuid.UUID) {
	f.releaseCalls++
}

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
