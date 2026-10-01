package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

// mockMonitorRepository stores monitor data for service tests
type mockMonitorRepository struct {
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

func (f *mockMonitorRepository) Create(context.Context, domain.Monitor) (domain.Monitor, error) {
	f.createCalls++
	if f.createErr != nil {
		return domain.Monitor{}, f.createErr
	}
	return f.created, nil
}

func (f *mockMonitorRepository) GetByID(context.Context, uuid.UUID) (domain.Monitor, error) {
	f.getCalls++
	if f.getErr != nil {
		return domain.Monitor{}, f.getErr
	}
	return f.monitor, nil
}

func (f *mockMonitorRepository) ListWithLastCheck(context.Context) ([]MonitorWithLastCheck, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.list, nil
}

func (f *mockMonitorRepository) Update(context.Context, domain.Monitor) error {
	f.updateCalls++
	return f.updateErr
}

func (f *mockMonitorRepository) Delete(context.Context, uuid.UUID) error {
	return f.deleteErr
}

func (f *mockMonitorRepository) ListDue(context.Context, time.Time) ([]domain.Monitor, error) {
	if f.dueErr != nil {
		return nil, f.dueErr
	}
	return f.due, nil
}

// mockCheckRepository stores check data for service tests
type mockCheckRepository struct {
	created   domain.Check
	createErr error
	history   []domain.Check
	listErr   error
}

func (f *mockCheckRepository) Create(_ context.Context, check domain.Check) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = check
	return nil
}

func (f *mockCheckRepository) ListByMonitorID(context.Context, uuid.UUID, int) ([]domain.Check, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.history, nil
}

// mockChecker returns a configured check result
type mockChecker struct {
	result applicationCheckResult
	err    error
}

// applicationCheckResult describes a mock check result
type applicationCheckResult struct {
	status     domain.CheckStatus
	statusCode *int
	latencyMS  int64
	err        *string
}

func (f *mockChecker) Check(context.Context, domain.Monitor) (CheckResult, error) {
	return CheckResult{
		Status:     f.result.status,
		StatusCode: f.result.statusCode,
		LatencyMS:  f.result.latencyMS,
		Error:      f.result.err,
	}, f.err
}

// mockInFlightController controls concurrent check tests
type mockInFlightController struct {
	allowed       bool
	tryStartCalls int
	releaseCalls  int
}

func (f *mockInFlightController) TryStart(uuid.UUID) bool {
	f.tryStartCalls++
	return f.allowed
}

func (f *mockInFlightController) Release(uuid.UUID) {
	f.releaseCalls++
}
