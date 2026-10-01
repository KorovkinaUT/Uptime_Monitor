package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

type MonitorWithLastCheck struct {
	Monitor   domain.Monitor
	LastCheck *domain.Check
}

type MonitorRepository interface {
	Create(ctx context.Context, monitor domain.Monitor) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Monitor, error)
	ListWithLastCheck(ctx context.Context) ([]MonitorWithLastCheck, error)
	Update(ctx context.Context, monitor domain.Monitor) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListDue(ctx context.Context, now time.Time) ([]domain.Monitor, error)
}

type CheckRepository interface {
	Create(ctx context.Context, check domain.Check) error
	ListByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]domain.Check, error)
}

type CheckResult struct {
	Status     domain.CheckStatus
	StatusCode *int
	LatencyMS  int64
	Error      *string
}

type Checker interface {
	Check(ctx context.Context, monitor domain.Monitor) (CheckResult, error)
}

type JobQueue interface {
	Enqueue(ctx context.Context, job CheckJob) error
}

type CheckJob struct {
	MonitorID uuid.UUID
}

type InFlightController interface {
	TryStart(id uuid.UUID) bool
	Release(id uuid.UUID)
}
