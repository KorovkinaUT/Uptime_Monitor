package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config contains application settings loaded from the environment
type Config struct {
	HTTPAddr            string        `envconfig:"HTTP_ADDR" default:":8080"`
	DatabaseURL         string        `envconfig:"DATABASE_URL" required:"true"`
	WorkerCount         int           `envconfig:"WORKER_COUNT" default:"5"`
	JobQueueCapacity    int           `envconfig:"JOB_QUEUE_CAPACITY" default:"100"`
	SchedulerInterval   time.Duration `envconfig:"SCHEDULER_INTERVAL" default:"1s"`
	DatabasePingTimeout time.Duration `envconfig:"DATABASE_PING_TIMEOUT" default:"5s"`
	ShutdownTimeout     time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"10s"`
}

// Load reads and validates application settings
func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, fmt.Errorf("load environment config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// validate checks numeric and duration settings loaded from the environment
func (c Config) validate() error {
	integerSettings := []struct {
		name  string
		value int
	}{
		{name: "WORKER_COUNT", value: c.WorkerCount},
		{name: "JOB_QUEUE_CAPACITY", value: c.JobQueueCapacity},
	}
	for _, setting := range integerSettings {
		if setting.value < 1 {
			return fmt.Errorf("%s must be a positive integer", setting.name)
		}
	}

	durationSettings := []struct {
		name  string
		value time.Duration
	}{
		{name: "SCHEDULER_INTERVAL", value: c.SchedulerInterval},
		{name: "DATABASE_PING_TIMEOUT", value: c.DatabasePingTimeout},
		{name: "SHUTDOWN_TIMEOUT", value: c.ShutdownTimeout},
	}
	for _, setting := range durationSettings {
		if setting.value < 1 {
			return fmt.Errorf("%s must be a positive duration", setting.name)
		}
	}
	return nil
}
