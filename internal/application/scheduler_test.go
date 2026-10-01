package application

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSchedulerWaitReturnsContextError(t *testing.T) {
	//arrange
	scheduler := NewScheduler(nil, nil, nil, time.Second, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	//act
	err := scheduler.Wait(ctx)

	//assert
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Wait() error = %v, want %v", err, context.DeadlineExceeded)
	}
}
