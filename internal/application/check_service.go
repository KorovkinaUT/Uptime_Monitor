package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

type CheckService struct {
	monitors MonitorRepository
	checks   CheckRepository
	checker  Checker
	inFlight InFlightController
	logger   *slog.Logger
	now      func() time.Time
}

func NewCheckService(
	monitors MonitorRepository,
	checks CheckRepository,
	checker Checker,
	inFlight InFlightController,
	logger *slog.Logger,
) *CheckService {
	if logger == nil {
		logger = slog.Default()
	}

	return &CheckService{
		monitors: monitors,
		checks:   checks,
		checker:  checker,
		inFlight: inFlight,
		logger:   logger,
		now:      time.Now,
	}
}

func (s *CheckService) Check(ctx context.Context, monitorID uuid.UUID) (domain.Check, error) {
	if !s.inFlight.TryStart(monitorID) {
		return domain.Check{}, domain.ErrCheckAlreadyRunning
	}

	return s.RunClaimed(ctx, monitorID)
}

func (s *CheckService) RunClaimed(ctx context.Context, monitorID uuid.UUID) (domain.Check, error) {
	// Run a check for a monitor already reserved by scheduler or manual request
	defer s.inFlight.Release(monitorID)

	monitor, err := s.monitors.GetByID(ctx, monitorID)
	if err != nil {
		return domain.Check{}, err
	}

	result, err := s.checker.Check(ctx, monitor)
	if err != nil {
		return domain.Check{}, fmt.Errorf("run monitor check: %w", err)
	}

	check := domain.Check{
		ID:         uuid.New(),
		MonitorID:  monitor.ID,
		Status:     result.Status,
		StatusCode: result.StatusCode,
		LatencyMS:  result.LatencyMS,
		Error:      result.Error,
		CheckedAt:  s.now().UTC(),
	}

	if err := s.checks.Create(ctx, check); err != nil {
		return domain.Check{}, fmt.Errorf("save monitor check: %w", err)
	}
	if check.Status == domain.CheckStatusDown {
		logDownCheck(s.logger, monitor.ID, check)
	}

	return check, nil
}

func (s *CheckService) ListHistory(ctx context.Context, monitorID uuid.UUID, limit int) ([]domain.Check, error) {
	if limit < 1 || limit > 1000 {
		return nil, domain.ErrInvalidLimit
	}
	if _, err := s.monitors.GetByID(ctx, monitorID); err != nil {
		return nil, err
	}

	return s.checks.ListByMonitorID(ctx, monitorID, limit)
}
