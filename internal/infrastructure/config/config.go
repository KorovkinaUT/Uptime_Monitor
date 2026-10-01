package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr                 string
	DatabaseURL              string
	WorkerCount              int
	JobQueueCapacity         int
	SchedulerIntervalSeconds int
}

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
	schedulerInterval, err := positiveIntEnv("SCHEDULER_INTERVAL_SECONDS", 1)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:                 envOrDefault("HTTP_ADDR", ":8080"),
		DatabaseURL:              databaseURL,
		WorkerCount:              workerCount,
		JobQueueCapacity:         queueCapacity,
		SchedulerIntervalSeconds: schedulerInterval,
	}, nil
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

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
