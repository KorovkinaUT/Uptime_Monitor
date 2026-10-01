package config

import "testing"

func TestLoadTimeoutDefaults(t *testing.T) {
	//arrange
	t.Setenv("DATABASE_URL", "postgres://uptime:uptime@localhost:5432/uptime")
	t.Setenv("DATABASE_PING_TIMEOUT_SECONDS", "")
	t.Setenv("SHUTDOWN_TIMEOUT_SECONDS", "")

	//act
	cfg, err := Load()

	//assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabasePingTimeoutSeconds != 5 {
		t.Errorf("DatabasePingTimeoutSeconds = %d, want 5", cfg.DatabasePingTimeoutSeconds)
	}
	if cfg.ShutdownTimeoutSeconds != 10 {
		t.Errorf("ShutdownTimeoutSeconds = %d, want 10", cfg.ShutdownTimeoutSeconds)
	}
}

func TestLoadRejectsInvalidShutdownTimeout(t *testing.T) {
	//arrange
	t.Setenv("DATABASE_URL", "postgres://uptime:uptime@localhost:5432/uptime")
	t.Setenv("SHUTDOWN_TIMEOUT_SECONDS", "0")

	//act
	_, err := Load()

	//assert
	if err == nil {
		t.Errorf("Load() error = nil, want invalid timeout error")
	}
}
