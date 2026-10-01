package domain

import (
	"time"

	"github.com/google/uuid"
)

// CheckStatus describes the result of a monitor check
type CheckStatus string

const (
	CheckStatusUp   CheckStatus = "up"
	CheckStatusDown CheckStatus = "down"
)

// Check stores one monitor check result
type Check struct {
	ID         uuid.UUID
	MonitorID  uuid.UUID
	Status     CheckStatus
	StatusCode *int
	LatencyMS  int64
	Error      *string
	CheckedAt  time.Time
}
