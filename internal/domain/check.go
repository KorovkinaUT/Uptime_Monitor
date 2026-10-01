package domain

import (
	"time"

	"github.com/google/uuid"
)

type CheckStatus string

const (
	CheckStatusUp   CheckStatus = "up"
	CheckStatusDown CheckStatus = "down"
)

type Check struct {
	ID         uuid.UUID
	MonitorID  uuid.UUID
	Status     CheckStatus
	StatusCode *int
	LatencyMS  int64
	Error      *string
	CheckedAt  time.Time
}
