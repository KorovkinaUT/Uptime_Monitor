package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

type MonitorService struct {
	monitors MonitorRepository
	checks   CheckRepository
}

func NewMonitorService(monitors MonitorRepository, checks CheckRepository) *MonitorService {
	return &MonitorService{monitors: monitors, checks: checks}
}

func (s *MonitorService) Create(ctx context.Context, monitor domain.Monitor) (MonitorWithLastCheck, error) {
	if err := monitor.Validate(); err != nil {
		return MonitorWithLastCheck{}, err
	}
	if monitor.ID == uuid.Nil {
		monitor.ID = uuid.New()
	}

	if err := s.monitors.Create(ctx, monitor); err != nil {
		return MonitorWithLastCheck{}, err
	}

	return s.Get(ctx, monitor.ID)
}

func (s *MonitorService) Get(ctx context.Context, id uuid.UUID) (MonitorWithLastCheck, error) {
	monitor, err := s.monitors.GetByID(ctx, id)
	if err != nil {
		return MonitorWithLastCheck{}, err
	}

	checks, err := s.checks.ListByMonitorID(ctx, id, 1)
	if err != nil {
		return MonitorWithLastCheck{}, err
	}

	view := MonitorWithLastCheck{Monitor: monitor}
	if len(checks) > 0 {
		view.LastCheck = &checks[0]
	}

	return view, nil
}

func (s *MonitorService) List(ctx context.Context) ([]MonitorWithLastCheck, error) {
	return s.monitors.ListWithLastCheck(ctx)
}

func (s *MonitorService) Update(ctx context.Context, monitor domain.Monitor) (MonitorWithLastCheck, error) {
	if err := monitor.Validate(); err != nil {
		return MonitorWithLastCheck{}, err
	}

	if err := s.monitors.Update(ctx, monitor); err != nil {
		return MonitorWithLastCheck{}, err
	}

	return s.Get(ctx, monitor.ID)
}

func (s *MonitorService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.monitors.Delete(ctx, id)
}
