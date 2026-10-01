package httpserver

import (
	"time"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/application"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

type monitorRequest struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	IntervalSeconds int    `json:"interval_seconds"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	Enabled         bool   `json:"enabled"`
}

func (r monitorRequest) monitor(id uuid.UUID) domain.Monitor {
	return domain.Monitor{
		ID:              id,
		Name:            r.Name,
		URL:             r.URL,
		IntervalSeconds: r.IntervalSeconds,
		TimeoutSeconds:  r.TimeoutSeconds,
		Enabled:         r.Enabled,
	}
}

type monitorResponse struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	URL             string     `json:"url"`
	IntervalSeconds int        `json:"interval_seconds"`
	TimeoutSeconds  int        `json:"timeout_seconds"`
	Enabled         bool       `json:"enabled"`
	Status          string     `json:"status"`
	LastStatusCode  *int       `json:"last_status_code"`
	LastLatencyMS   *int64     `json:"last_latency_ms"`
	LastCheckedAt   *time.Time `json:"last_checked_at"`
}

func newMonitorResponse(view application.MonitorWithLastCheck) monitorResponse {
	response := monitorResponse{
		ID:              view.Monitor.ID,
		Name:            view.Monitor.Name,
		URL:             view.Monitor.URL,
		IntervalSeconds: view.Monitor.IntervalSeconds,
		TimeoutSeconds:  view.Monitor.TimeoutSeconds,
		Enabled:         view.Monitor.Enabled,
		Status:          "unknown",
	}
	if view.LastCheck == nil {
		return response
	}

	response.Status = string(view.LastCheck.Status)
	response.LastStatusCode = view.LastCheck.StatusCode
	latency := view.LastCheck.LatencyMS
	response.LastLatencyMS = &latency
	checkedAt := view.LastCheck.CheckedAt
	response.LastCheckedAt = &checkedAt
	return response
}

type checkResponse struct {
	ID         uuid.UUID `json:"id"`
	MonitorID  uuid.UUID `json:"monitor_id"`
	Status     string    `json:"status"`
	StatusCode *int      `json:"status_code"`
	LatencyMS  int64     `json:"latency_ms"`
	Error      *string   `json:"error"`
	CheckedAt  time.Time `json:"checked_at"`
}

func newCheckResponse(check domain.Check) checkResponse {
	return checkResponse{
		ID:         check.ID,
		MonitorID:  check.MonitorID,
		Status:     string(check.Status),
		StatusCode: check.StatusCode,
		LatencyMS:  check.LatencyMS,
		Error:      check.Error,
		CheckedAt:  check.CheckedAt,
	}
}

type errorResponse struct {
	Error string `json:"error"`
}
