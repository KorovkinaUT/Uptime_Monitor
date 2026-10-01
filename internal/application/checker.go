package application

import (
	"context"

	"github.com/ulyana/uptime-monitor/internal/domain"
)

// CheckResult contains the outcome of an HTTP check
type CheckResult struct {
	Status     domain.CheckStatus
	StatusCode *int
	LatencyMS  int64
	Error      *string
}

// Checker performs a monitor check
type Checker interface {
	Check(ctx context.Context, monitor domain.Monitor) (CheckResult, error)
}
