package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

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
