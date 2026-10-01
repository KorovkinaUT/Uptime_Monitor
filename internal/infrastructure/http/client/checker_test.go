package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ulyana/uptime-monitor/internal/domain"
)

func TestHTTPCheckerStatusMapping(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantStatus domain.CheckStatus
	}{
		{name: "200 is up", statusCode: http.StatusOK, wantStatus: domain.CheckStatusUp},
		{name: "302 is up", statusCode: http.StatusFound, wantStatus: domain.CheckStatusUp},
		{name: "404 is down", statusCode: http.StatusNotFound, wantStatus: domain.CheckStatusDown},
		{name: "500 is down", statusCode: http.StatusInternalServerError, wantStatus: domain.CheckStatusDown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//arrange
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(tt.statusCode)
			}))
			defer server.Close()
			monitor := testMonitor(server.URL)
			client := NewChecker()

			//act
			result, err := client.Check(context.Background(), monitor)

			//assert
			if err != nil {
				t.Errorf("Check() error = %v", err)
			}
			if result.Status != tt.wantStatus {
				t.Errorf("Check() status = %q, want %q", result.Status, tt.wantStatus)
			}
			if result.StatusCode == nil {
				t.Errorf("Check() status code is nil")
			} else if *result.StatusCode != tt.statusCode {
				t.Errorf("Check() status code = %d, want %d", *result.StatusCode, tt.statusCode)
			}
		})
	}
}

func TestHTTPCheckerTimeout(t *testing.T) {
	//arrange
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	defer server.Close()
	monitor := testMonitor(server.URL)
	client := NewChecker()

	//act
	result, err := client.Check(context.Background(), monitor)

	//assert
	if err != nil {
		t.Errorf("Check() error = %v", err)
	}
	if result.Status != domain.CheckStatusDown {
		t.Errorf("Check() status = %q, want %q", result.Status, domain.CheckStatusDown)
	}
	if result.StatusCode != nil {
		t.Errorf("Check() status code = %v, want nil", *result.StatusCode)
	}
	if result.Error == nil || *result.Error == "" {
		t.Errorf("Check() error = %v, want non-empty error", result.Error)
	}
	if result.LatencyMS < 900 {
		t.Errorf("Check() latency = %d ms, want at least 900 ms", result.LatencyMS)
	}
}

func TestHTTPCheckerLatency(t *testing.T) {
	//arrange
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		time.Sleep(25 * time.Millisecond)
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	monitor := testMonitor(server.URL)
	client := NewChecker()

	//act
	result, err := client.Check(context.Background(), monitor)

	//assert
	if err != nil {
		t.Errorf("Check() error = %v", err)
	}
	if result.LatencyMS < 20 {
		t.Errorf("Check() latency = %d ms, want at least 20 ms", result.LatencyMS)
	}
}

func testMonitor(targetURL string) domain.Monitor {
	return domain.Monitor{
		Name:            "Test monitor",
		URL:             targetURL,
		IntervalSeconds: 2,
		TimeoutSeconds:  1,
	}
}
