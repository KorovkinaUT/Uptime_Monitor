package application

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

var ErrWorkerPoolStopped = errors.New("worker pool stopped")

// JobHandler processes one queued check job
type JobHandler func(ctx context.Context, job CheckJob) error

// JobQueue accepts monitor check jobs
type JobQueue interface {
	Enqueue(ctx context.Context, job CheckJob) error
}

// CheckJob identifies a monitor to check
type CheckJob struct {
	MonitorID uuid.UUID
}

// WorkerPool processes check jobs with bounded concurrency
type WorkerPool struct {
	jobs        chan CheckJob
	handler     JobHandler
	logger      *slog.Logger
	workerCount int
	workerCtx   context.Context
	cancel      context.CancelFunc
	workersWG   sync.WaitGroup
	enqueueWG   sync.WaitGroup
	mu          sync.Mutex
	stopping    bool
	started     bool
	stopCh      chan struct{}
	doneCh      chan struct{}
	stopOnce    sync.Once
}

// NewWorkerPool creates a bounded worker pool
func NewWorkerPool(workerCount, queueCapacity int, handler JobHandler, logger *slog.Logger) (*WorkerPool, error) {
	if workerCount < 1 || queueCapacity < 1 {
		return nil, errors.New("worker count and queue capacity must be positive")
	}
	if handler == nil {
		return nil, errors.New("worker handler is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	workerCtx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		jobs:        make(chan CheckJob, queueCapacity),
		handler:     handler,
		logger:      logger,
		workerCount: workerCount,
		workerCtx:   workerCtx,
		cancel:      cancel,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}, nil
}

// Start launches the configured worker goroutines once
func (p *WorkerPool) Start() {
	p.mu.Lock()
	if p.started || p.stopping {
		p.mu.Unlock()
		return
	}
	p.started = true
	for i := 0; i < p.workerCount; i++ {
		p.workersWG.Add(1)
		go p.worker()
	}
	p.mu.Unlock()

	p.logger.Info("worker pool started", "workers", p.workerCount, "queue_capacity", cap(p.jobs))
}

// Enqueue adds a job while the pool accepts new work
func (p *WorkerPool) Enqueue(ctx context.Context, job CheckJob) error {
	p.mu.Lock()
	if p.stopping {
		p.mu.Unlock()
		return ErrWorkerPoolStopped
	}
	p.enqueueWG.Add(1)
	p.mu.Unlock()
	defer p.enqueueWG.Done()

	select {
	case p.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-p.stopCh:
		return ErrWorkerPoolStopped
	}
}

// Stop closes intake and waits for workers to finish
func (p *WorkerPool) Stop(ctx context.Context) error {
	p.stopOnce.Do(func() {
		p.mu.Lock()
		p.stopping = true
		close(p.stopCh)
		p.mu.Unlock()

		go p.shutdown()
	})

	select {
	case <-p.doneCh:
		p.logger.Info("worker pool stopped")
		return nil
	case <-ctx.Done():
		p.cancel()
		return ctx.Err()
	}
}

// shutdown drains enqueues and workers in order
func (p *WorkerPool) shutdown() {
	p.enqueueWG.Wait()
	close(p.jobs)
	p.workersWG.Wait()
	p.cancel()
	close(p.doneCh)
}

// worker processes jobs from the queue
func (p *WorkerPool) worker() {
	defer p.workersWG.Done()

	for job := range p.jobs {
		if err := p.handler(p.workerCtx, job); err != nil {
			p.logger.Error("worker job failed", "monitor_id", job.MonitorID, "error", err)
		}
	}
}
