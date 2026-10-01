package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

// MonitorWithLastCheck combines a monitor with its latest check
type MonitorWithLastCheck struct {
	Monitor   domain.Monitor
	LastCheck *domain.Check
}

// MonitorRepository stores and retrieves monitors
type MonitorRepository interface {
	Create(ctx context.Context, monitor domain.Monitor) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Monitor, error)
	ListWithLastCheck(ctx context.Context) ([]MonitorWithLastCheck, error)
	Update(ctx context.Context, monitor domain.Monitor) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListDue(ctx context.Context, now time.Time) ([]domain.Monitor, error)
}

// CheckRepository stores and retrieves check results
type CheckRepository interface {
	Create(ctx context.Context, check domain.Check) error
	ListByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]domain.Check, error)
}

// CheckResult contains the outcome of an HTTP check
type CheckResult struct {
	Status     domain.CheckStatus
	StatusCode *int
	LatencyMS  int64
	Error      *string
}

// Checker performs a monitor check
type Checker interface {
	Check(ctx context.Context, monitor domain.Monitor) (CheckResult, error)
}

// JobQueue accepts monitor check jobs
type JobQueue interface {
	Enqueue(ctx context.Context, job CheckJob) error
}

// CheckJob identifies a monitor to check
type CheckJob struct {
	MonitorID uuid.UUID
}

// InFlightController coordinates active monitor checks
type InFlightController interface {
	TryStart(id uuid.UUID) bool
	Release(id uuid.UUID)
}
