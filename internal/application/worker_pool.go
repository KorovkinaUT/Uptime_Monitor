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
	jobs      chan CheckJob
	handler   JobHandler
	logger    *slog.Logger
	workerCtx context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closed    chan struct{}
	mu        sync.RWMutex
	stopped   bool
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
		jobs:      make(chan CheckJob, queueCapacity),
		handler:   handler,
		logger:    logger,
		workerCtx: workerCtx,
		cancel:    cancel,
		closed:    make(chan struct{}),
	}, nil
}

// Start launches the configured worker goroutines
func (p *WorkerPool) Start(workerCount int) {
	for i := 0; i < workerCount; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	p.logger.Info("worker pool started", "workers", workerCount, "queue_capacity", cap(p.jobs))
}

// Enqueue adds a job while the pool accepts new work
func (p *WorkerPool) Enqueue(ctx context.Context, job CheckJob) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.stopped {
		return ErrWorkerPoolStopped
	}

	select {
	case p.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-p.closed:
		return ErrWorkerPoolStopped
	}
}

// Stop closes intake and waits for workers to finish
func (p *WorkerPool) Stop(ctx context.Context) error {
	// Close intake first so workers can drain the bounded queue
	p.mu.Lock()
	if !p.stopped {
		p.stopped = true
		close(p.closed)
		close(p.jobs)
	}
	p.mu.Unlock()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.cancel()
		p.logger.Info("worker pool stopped")
		return nil
	case <-ctx.Done():
		p.cancel()
		return ctx.Err()
	}
}

// worker processes jobs from the queue
func (p *WorkerPool) worker() {
	// Keep one goroutine per configured worker slot
	defer p.wg.Done()

	for {
		select {
		case <-p.workerCtx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			if err := p.handler(p.workerCtx, job); err != nil {
				p.logger.Error("worker job failed", "monitor_id", job.MonitorID, "error", err)
			}
		}
	}
}
