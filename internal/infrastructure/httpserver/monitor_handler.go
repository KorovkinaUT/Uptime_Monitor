package httpserver

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/ulyana/uptime-monitor/internal/application"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

type MonitorService interface {
	Create(ctx context.Context, monitor domain.Monitor) (application.MonitorWithLastCheck, error)
	Get(ctx context.Context, id uuid.UUID) (application.MonitorWithLastCheck, error)
	List(ctx context.Context) ([]application.MonitorWithLastCheck, error)
	Update(ctx context.Context, monitor domain.Monitor) (application.MonitorWithLastCheck, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type MonitorHandler struct {
	service MonitorService
	logger  *slog.Logger
}

func NewMonitorHandler(service MonitorService, logger *slog.Logger) *MonitorHandler {
	return &MonitorHandler{service: service, logger: logger}
}

func (h *MonitorHandler) List(writer http.ResponseWriter, request *http.Request) {
	monitors, err := h.service.List(request.Context())
	if err != nil {
		writeError(writer, h.logger, err)
		return
	}

	response := make([]monitorResponse, 0, len(monitors))
	for _, monitor := range monitors {
		response = append(response, newMonitorResponse(monitor))
	}
	writeJSON(writer, http.StatusOK, response)
}

func (h *MonitorHandler) Create(writer http.ResponseWriter, request *http.Request) {
	var input monitorRequest
	if err := decodeJSON(request, &input); err != nil {
		writeBadRequest(writer, err)
		return
	}

	monitor, err := h.service.Create(request.Context(), input.monitor(uuid.New()))
	if err != nil {
		writeError(writer, h.logger, err)
		return
	}
	writeJSON(writer, http.StatusCreated, newMonitorResponse(monitor))
}

func (h *MonitorHandler) Get(writer http.ResponseWriter, request *http.Request) {
	id, err := parseMonitorID(request)
	if err != nil {
		writeBadRequest(writer, err)
		return
	}

	monitor, err := h.service.Get(request.Context(), id)
	if err != nil {
		writeError(writer, h.logger, err)
		return
	}
	writeJSON(writer, http.StatusOK, newMonitorResponse(monitor))
}

func (h *MonitorHandler) Update(writer http.ResponseWriter, request *http.Request) {
	id, err := parseMonitorID(request)
	if err != nil {
		writeBadRequest(writer, err)
		return
	}

	var input monitorRequest
	if err := decodeJSON(request, &input); err != nil {
		writeBadRequest(writer, err)
		return
	}

	monitor, err := h.service.Update(request.Context(), input.monitor(id))
	if err != nil {
		writeError(writer, h.logger, err)
		return
	}
	writeJSON(writer, http.StatusOK, newMonitorResponse(monitor))
}

func (h *MonitorHandler) Delete(writer http.ResponseWriter, request *http.Request) {
	id, err := parseMonitorID(request)
	if err != nil {
		writeBadRequest(writer, err)
		return
	}

	if err := h.service.Delete(request.Context(), id); err != nil {
		writeError(writer, h.logger, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func parseMonitorID(request *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(requestPathParam(request, "id"))
	if err != nil {
		return uuid.Nil, domain.ErrInvalidMonitor
	}
	return id, nil
}
