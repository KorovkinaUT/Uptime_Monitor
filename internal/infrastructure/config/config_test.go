package config

import (
	"testing"
	"time"
)

func TestLoadParsesDurationUnits(t *testing.T) {
	//arrange
	t.Setenv("DATABASE_URL", "postgres://uptime:uptime@localhost:5432/uptime")
	t.Setenv("SCHEDULER_INTERVAL", "250ms")
	t.Setenv("DATABASE_PING_TIMEOUT", "5s")
	t.Setenv("SHUTDOWN_TIMEOUT", "2m")

	//act
	cfg, err := Load()

	//assert
	if err != nil {
		t.Errorf("Load() error = %v", err)
		return
	}
	if cfg.SchedulerInterval != 250*time.Millisecond {
		t.Errorf("SchedulerInterval = %s, want 250ms", cfg.SchedulerInterval)
	}
	if cfg.DatabasePingTimeout != 5*time.Second {
		t.Errorf("DatabasePingTimeout = %s, want 5s", cfg.DatabasePingTimeout)
	}
	if cfg.ShutdownTimeout != 2*time.Minute {
		t.Errorf("ShutdownTimeout = %s, want 2m", cfg.ShutdownTimeout)
	}
}

func TestLoadRejectsNonPositiveDuration(t *testing.T) {
	//arrange
	t.Setenv("DATABASE_URL", "postgres://uptime:uptime@localhost:5432/uptime")
	t.Setenv("SCHEDULER_INTERVAL", "0s")

	//act
	_, err := Load()

	//assert
	if err == nil {
		t.Errorf("Load() error = nil, want validation error")
	}
}
