package domain

import (
	"errors"
	"testing"
)

func TestMonitorValidate(t *testing.T) {
	tests := []struct {
		name    string
		monitor Monitor
		wantErr bool
	}{
		{
			name: "valid monitor",
			monitor: Monitor{
				Name:            "My API",
				URL:             "https://example.com/health",
				IntervalSeconds: 30,
				TimeoutSeconds:  5,
			},
		},
		{
			name:    "empty name",
			monitor: Monitor{URL: "https://example.com", IntervalSeconds: 30, TimeoutSeconds: 5},
			wantErr: true,
		},
		{
			name:    "invalid url",
			monitor: Monitor{Name: "API", URL: "not a url", IntervalSeconds: 30, TimeoutSeconds: 5},
			wantErr: true,
		},
		{
			name:    "unsupported scheme",
			monitor: Monitor{Name: "API", URL: "ftp://example.com", IntervalSeconds: 30, TimeoutSeconds: 5},
			wantErr: true,
		},
		{
			name:    "non positive interval",
			monitor: Monitor{Name: "API", URL: "https://example.com", IntervalSeconds: 0, TimeoutSeconds: 5},
			wantErr: true,
		},
		{
			name:    "non positive timeout",
			monitor: Monitor{Name: "API", URL: "https://example.com", IntervalSeconds: 30, TimeoutSeconds: 0},
			wantErr: true,
		},
		{
			name:    "timeout equals interval",
			monitor: Monitor{Name: "API", URL: "https://example.com", IntervalSeconds: 5, TimeoutSeconds: 5},
			wantErr: true,
		},
		{
			name:    "timeout exceeds interval",
			monitor: Monitor{Name: "API", URL: "https://example.com", IntervalSeconds: 5, TimeoutSeconds: 6},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//arrange

			//act
			err := tt.monitor.Validate()

			//assert
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrInvalidMonitor) {
				t.Errorf("Validate() error = %v, want %v", err, ErrInvalidMonitor)
			}
		})
	}
}
