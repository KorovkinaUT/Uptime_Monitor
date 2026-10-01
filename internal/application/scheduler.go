package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/ulyana/uptime-monitor/internal/domain"
)

type Scheduler struct {
	monitors MonitorRepository
	inFlight InFlightController
	queue    JobQueue
	interval time.Duration
	logger   *slog.Logger
	now      func() time.Time
	done     chan struct{}
}

func NewScheduler(
	monitors MonitorRepository,
	inFlight InFlightController,
	queue JobQueue,
	interval time.Duration,
	logger *slog.Logger,
) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Scheduler{
		monitors: monitors,
		inFlight: inFlight,
		queue:    queue,
		interval: interval,
		logger:   logger,
		now:      time.Now,
		done:     make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	go s.run(ctx)
	s.logger.Info("scheduler started", "interval", s.interval)
}

func (s *Scheduler) Wait() {
	<-s.done
}

func (s *Scheduler) run(ctx context.Context) {
	defer close(s.done)

	s.schedule(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.schedule(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) schedule(ctx context.Context) {
	dueMonitors, err := s.monitors.ListDue(ctx, s.now().UTC())
	if err != nil {
		if ctx.Err() == nil {
			s.logger.Error("list due monitors failed", "error", err)
		}
		return
	}

	for _, monitor := range dueMonitors {
		if !s.inFlight.TryStart(monitor.ID) {
			continue
		}

		err := s.queue.Enqueue(ctx, CheckJob{MonitorID: monitor.ID})
		if err == nil {
			continue
		}

		s.inFlight.Release(monitor.ID)
		if ctx.Err() != nil {
			return
		}
		if err != domain.ErrCheckAlreadyRunning {
			s.logger.Error("enqueue monitor check failed", "monitor_id", monitor.ID, "error", err)
		}
	}
}
