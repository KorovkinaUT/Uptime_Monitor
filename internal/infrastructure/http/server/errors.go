package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/ulyana/uptime-monitor/internal/domain"
)

func decodeJSON(request *http.Request, destination any) error {
	// Reject unknown fields and trailing JSON values
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("request contains more than one JSON value")
	}
	return nil
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		slog.Error("write JSON response failed", "error", err)
	}
}

func writeError(writer http.ResponseWriter, logger *slog.Logger, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	switch {
	case errors.Is(err, domain.ErrInvalidMonitor), errors.Is(err, domain.ErrInvalidLimit):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, domain.ErrMonitorNotFound):
		status = http.StatusNotFound
		message = domain.ErrMonitorNotFound.Error()
	case errors.Is(err, domain.ErrCheckAlreadyRunning):
		status = http.StatusConflict
		message = domain.ErrCheckAlreadyRunning.Error()
	default:
		logger.Error("request failed", "error", err)
	}

	writeJSON(writer, status, errorResponse{Error: message})
}

func writeBadRequest(writer http.ResponseWriter, err error) {
	writeJSON(writer, http.StatusBadRequest, errorResponse{Error: err.Error()})
}
