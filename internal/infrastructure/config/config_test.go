package config

import (
	"os"
	"testing"
)

func TestLoadTimeoutDefaults(t *testing.T) {
	//arrange
	t.Setenv("DATABASE_URL", "postgres://uptime:uptime@localhost:5432/uptime")
	unsetEnv(t, "DATABASE_PING_TIMEOUT")
	unsetEnv(t, "SHUTDOWN_TIMEOUT")

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

func TestLoadRequiresDatabaseURL(t *testing.T) {
	//arrange
	unsetEnv(t, "DATABASE_URL")

	//act
	_, err := Load()

	//assert
	if err == nil {
		t.Errorf("Load() error = nil, want missing database URL error")
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

// unsetEnv removes an environment variable and restores it after the test
func unsetEnv(t *testing.T, name string) {
	t.Helper()
	value, exists := os.LookupEnv(name)
	if err := os.Unsetenv(name); err != nil {
		t.Fatalf("Unsetenv(%q) error = %v", name, err)
	}
	t.Cleanup(func() {
		if exists {
			_ = os.Setenv(name, value)
			return
		}
		_ = os.Unsetenv(name)
	})
}
