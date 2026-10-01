package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

type CheckService interface {
	Check(ctx context.Context, monitorID uuid.UUID) (domain.Check, error)
	ListHistory(ctx context.Context, monitorID uuid.UUID, limit int) ([]domain.Check, error)
}

type CheckHandler struct {
	service CheckService
	logger  *slog.Logger
}

func NewCheckHandler(service CheckService, logger *slog.Logger) *CheckHandler {
	return &CheckHandler{service: service, logger: logger}
}

func (h *CheckHandler) Run(writer http.ResponseWriter, request *http.Request) {
	id, err := parseMonitorID(request)
	if err != nil {
		writeBadRequest(writer, err)
		return
	}

	check, err := h.service.Check(request.Context(), id)
	if err != nil {
		writeError(writer, h.logger, err)
		return
	}
	writeJSON(writer, http.StatusOK, newCheckResponse(check))
}

func (h *CheckHandler) History(writer http.ResponseWriter, request *http.Request) {
	id, err := parseMonitorID(request)
	if err != nil {
		writeBadRequest(writer, err)
		return
	}

	limit, err := parseLimit(request)
	if err != nil {
		writeError(writer, h.logger, err)
		return
	}

	checks, err := h.service.ListHistory(request.Context(), id, limit)
	if err != nil {
		writeError(writer, h.logger, err)
		return
	}

	response := make([]checkResponse, 0, len(checks))
	for _, check := range checks {
		response = append(response, newCheckResponse(check))
	}
	writeJSON(writer, http.StatusOK, response)
}

func parseLimit(request *http.Request) (int, error) {
	value := request.URL.Query().Get("limit")
	if value == "" {
		return 100, nil
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > 1000 {
		return 0, domain.ErrInvalidLimit
	}
	return limit, nil
}
