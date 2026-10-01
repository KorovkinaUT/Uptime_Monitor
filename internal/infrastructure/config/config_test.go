package config

import "testing"

func TestLoadTimeoutDefaults(t *testing.T) {
	//arrange
	t.Setenv("DATABASE_URL", "postgres://uptime:uptime@localhost:5432/uptime")
	t.Setenv("DATABASE_PING_TIMEOUT", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")

	//act
	cfg, err := Load()

	//assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabasePingTimeout != 5 {
		t.Errorf("DatabasePingTimeout = %d, want 5", cfg.DatabasePingTimeout)
	}
	if cfg.ShutdownTimeout != 10 {
		t.Errorf("ShutdownTimeout = %d, want 10", cfg.ShutdownTimeout)
	}
}

func TestLoadRejectsInvalidShutdownTimeout(t *testing.T) {
	//arrange
	t.Setenv("DATABASE_URL", "postgres://uptime:uptime@localhost:5432/uptime")
	t.Setenv("SHUTDOWN_TIMEOUT", "0")

	//act
	_, err := Load()

	//assert
	if err == nil {
		t.Errorf("Load() error = nil, want invalid timeout error")
	}
}
