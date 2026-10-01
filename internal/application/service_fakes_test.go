package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

// fakeMonitorRepository stores monitor data for service tests
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

// fakeCheckRepository stores check data for service tests
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

// fakeChecker returns a configured check result
type fakeChecker struct {
	result applicationCheckResult
	err    error
}

// applicationCheckResult describes a fake check result
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

// fakeInFlightController controls concurrent check tests
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
