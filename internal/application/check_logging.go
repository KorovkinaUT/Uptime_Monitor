package application

import (
	"log/slog"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

func logDownCheck(logger *slog.Logger, monitorID uuid.UUID, check domain.Check) {
	var statusCode any
	if check.StatusCode != nil {
		statusCode = *check.StatusCode
	}

	var checkError any
	if check.Error != nil {
		checkError = *check.Error
	}

	logger.Warn("monitor check is down", "monitor_id", monitorID, "status_code", statusCode, "error", checkError)
}
