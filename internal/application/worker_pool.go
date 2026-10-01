package application

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

var ErrWorkerPoolStopped = errors.New("worker pool stopped")

type JobHandler func(ctx context.Context, job CheckJob) error

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

func (p *WorkerPool) Start(workerCount int) {
	for i := 0; i < workerCount; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	p.logger.Info("worker pool started", "workers", workerCount, "queue_capacity", cap(p.jobs))
}

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
