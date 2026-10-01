package checker

import (
	"context"
	"net/http"
	"time"

	"github.com/ulyana/uptime-monitor/internal/application"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

type HTTPChecker struct {
	client *http.Client
}

func NewHTTPChecker(client *http.Client) *HTTPChecker {
	if client == nil {
		client = &http.Client{}
	}
	return &HTTPChecker{client: client}
}

func (c *HTTPChecker) Check(ctx context.Context, monitor domain.Monitor) (application.CheckResult, error) {
	// Apply the timeout configured for this monitor only
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(monitor.TimeoutSeconds)*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, monitor.URL, nil)
	if err != nil {
		return application.CheckResult{}, err
	}

	start := time.Now()
	response, err := c.client.Do(request)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		// Persist transport failures as down results without a status code
		errorText := err.Error()
		return application.CheckResult{
			Status:    domain.CheckStatusDown,
			LatencyMS: latency,
			Error:     &errorText,
		}, nil
	}
	defer response.Body.Close()

	status := domain.CheckStatusDown
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusBadRequest {
		status = domain.CheckStatusUp
	}

	statusCode := response.StatusCode
	return application.CheckResult{
		Status:     status,
		StatusCode: &statusCode,
		LatencyMS:  latency,
	}, nil
}
