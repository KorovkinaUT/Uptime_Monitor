package application

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWorkerPoolEnqueue(t *testing.T) {
	//arrange
	handled := make(chan CheckJob, 1)
	pool, err := NewWorkerPool(1, 1, func(_ context.Context, job CheckJob) error {
		handled <- job
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("NewWorkerPool() error = %v", err)
	}
	pool.Start()

	//act
	enqueueErr := pool.Enqueue(context.Background(), CheckJob{})

	//assert
	if enqueueErr != nil {
		t.Errorf("Enqueue() error = %v, want nil", enqueueErr)
	}
	select {
	case <-handled:
	case <-time.After(time.Second):
		t.Errorf("worker did not handle the enqueued job")
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := pool.Stop(stopCtx); err != nil {
		t.Errorf("Stop() error = %v, want nil", err)
	}
}

func TestWorkerPoolStopUnblocksWaitingEnqueue(t *testing.T) {
	//arrange
	started := make(chan struct{}, 2)
	release := make(chan struct{}, 2)
	pool, err := NewWorkerPool(1, 1, func(ctx context.Context, _ CheckJob) error {
		started <- struct{}{}
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, nil)
	if err != nil {
		t.Fatalf("NewWorkerPool() error = %v", err)
	}
	pool.Start()
	if err := pool.Enqueue(context.Background(), CheckJob{}); err != nil {
		t.Fatalf("first Enqueue() error = %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatalf("worker did not start the first job")
	}
	if err := pool.Enqueue(context.Background(), CheckJob{}); err != nil {
		t.Fatalf("second Enqueue() error = %v", err)
	}

	blockedResult := make(chan error, 1)
	go func() {
		blockedResult <- pool.Enqueue(context.Background(), CheckJob{})
	}()

	select {
	case err := <-blockedResult:
		t.Fatalf("Enqueue() returned before shutdown: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	stopResult := make(chan error, 1)
	go func() {
		stopResult <- pool.Stop(stopCtx)
	}()

	//act
	select {
	case err := <-blockedResult:
		if !errors.Is(err, ErrWorkerPoolStopped) {
			t.Errorf("blocked Enqueue() error = %v, want ErrWorkerPoolStopped", err)
		}
	case <-time.After(time.Second):
		t.Errorf("Stop() did not unblock the waiting Enqueue()")
	}

	//assert
	release <- struct{}{}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Errorf("worker did not start the queued job")
	}
	release <- struct{}{}
	select {
	case err := <-stopResult:
		if err != nil {
			t.Errorf("Stop() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Errorf("Stop() did not finish after workers were released")
	}
}

func TestWorkerPoolGracefulShutdownDrainsQueue(t *testing.T) {
	//arrange
	handled := make(chan CheckJob, 2)
	pool, err := NewWorkerPool(1, 2, func(_ context.Context, job CheckJob) error {
		handled <- job
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("NewWorkerPool() error = %v", err)
	}
	pool.Start()
	for i := 0; i < 2; i++ {
		if err := pool.Enqueue(context.Background(), CheckJob{}); err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}
	}

	//act
	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	stopErr := pool.Stop(stopCtx)

	//assert
	if stopErr != nil {
		t.Errorf("Stop() error = %v, want nil", stopErr)
	}
	if got := len(handled); got != 2 {
		t.Errorf("handled jobs = %d, want 2", got)
	}
}

func TestWorkerPoolStopIsIdempotent(t *testing.T) {
	//arrange
	pool, err := NewWorkerPool(1, 1, func(context.Context, CheckJob) error {
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("NewWorkerPool() error = %v", err)
	}
	pool.Start()
	firstCtx, firstCancel := context.WithTimeout(context.Background(), time.Second)
	defer firstCancel()

	//act
	firstErr := pool.Stop(firstCtx)
	secondCtx, secondCancel := context.WithTimeout(context.Background(), time.Second)
	defer secondCancel()
	secondErr := pool.Stop(secondCtx)

	//assert
	if firstErr != nil {
		t.Errorf("first Stop() error = %v, want nil", firstErr)
	}
	if secondErr != nil {
		t.Errorf("second Stop() error = %v, want nil", secondErr)
	}
}

func TestWorkerPoolStopTimeoutCancelsHandler(t *testing.T) {
	//arrange
	started := make(chan struct{})
	canceled := make(chan struct{})
	pool, err := NewWorkerPool(1, 1, func(ctx context.Context, _ CheckJob) error {
		close(started)
		<-ctx.Done()
		close(canceled)
		return ctx.Err()
	}, nil)
	if err != nil {
		t.Fatalf("NewWorkerPool() error = %v", err)
	}
	pool.Start()
	if err := pool.Enqueue(context.Background(), CheckJob{}); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatalf("worker did not start the handler")
	}

	//act
	stopCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	stopErr := pool.Stop(stopCtx)

	//assert
	if !errors.Is(stopErr, context.DeadlineExceeded) {
		t.Errorf("Stop() error = %v, want context deadline exceeded", stopErr)
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Errorf("handler did not receive cancellation")
	}

	finishCtx, finishCancel := context.WithTimeout(context.Background(), time.Second)
	defer finishCancel()
	if err := pool.Stop(finishCtx); err != nil {
		t.Errorf("final Stop() error = %v, want nil", err)
	}
}
