package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config contains application settings loaded from the environment
type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	WorkerCount         int
	JobQueueCapacity    int
	SchedulerInterval   int
	DatabasePingTimeout int
	ShutdownTimeout     int
}

// Load reads and validates application settings
func Load() (Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	workerCount, err := positiveIntEnv("WORKER_COUNT", 5)
	if err != nil {
		return Config{}, err
	}
	queueCapacity, err := positiveIntEnv("JOB_QUEUE_CAPACITY", 100)
	if err != nil {
		return Config{}, err
	}
	schedulerInterval, err := positiveIntEnv("SCHEDULER_INTERVAL", 1)
	if err != nil {
		return Config{}, err
	}
	databasePingTimeout, err := positiveIntEnv("DATABASE_PING_TIMEOUT", 5)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := positiveIntEnv("SHUTDOWN_TIMEOUT", 10)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:            envOrDefault("HTTP_ADDR", ":8080"),
		DatabaseURL:         databaseURL,
		WorkerCount:         workerCount,
		JobQueueCapacity:    queueCapacity,
		SchedulerInterval:   schedulerInterval,
		DatabasePingTimeout: databasePingTimeout,
		ShutdownTimeout:     shutdownTimeout,
	}, nil
}

// envOrDefault returns an environment value or fallback
func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

// positiveIntEnv reads a positive integer setting
func positiveIntEnv(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return parsed, nil
}
