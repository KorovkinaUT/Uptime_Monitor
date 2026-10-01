package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ulyana/uptime-monitor/internal/application"
	"github.com/ulyana/uptime-monitor/internal/infrastructure/config"
	httpclient "github.com/ulyana/uptime-monitor/internal/infrastructure/http/client"
	httpserver "github.com/ulyana/uptime-monitor/internal/infrastructure/http/server"
	"github.com/ulyana/uptime-monitor/internal/infrastructure/postgres"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("application stopped with error", "error", err)
	}
}

func run(logger *slog.Logger) error {
	// Build all application dependencies
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databasePool, err := postgres.NewPool(appCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer databasePool.Close()
	logger.Info("database connected")

	monitorRepository := postgres.NewMonitorRepository(databasePool)
	checkRepository := postgres.NewCheckRepository(databasePool)
	monitorService := application.NewMonitorService(monitorRepository, checkRepository)
	inFlight := application.NewInFlightRegistry()
	httpChecker := httpclient.NewChecker()
	checkService := application.NewCheckService(monitorRepository, checkRepository, httpChecker, inFlight, logger)

	workerPool, err := application.NewWorkerPool(
		cfg.WorkerCount,
		cfg.JobQueueCapacity,
		func(ctx context.Context, job application.CheckJob) error {
			_, err := checkService.RunClaimed(ctx, job.MonitorID)
			return err
		},
		logger,
	)
	if err != nil {
		return fmt.Errorf("create worker pool: %w", err)
	}
	workerPool.Start(cfg.WorkerCount)

	scheduler := application.NewScheduler(
		monitorRepository,
		inFlight,
		workerPool,
		time.Duration(cfg.SchedulerIntervalSeconds)*time.Second,
		logger,
	)
	scheduler.Start(appCtx)

	server := httpserver.NewServer(cfg.HTTPAddr, httpserver.Dependencies{
		Monitors: monitorService,
		Checks:   checkService,
		WebDir:   "web",
		Logger:   logger,
	})
	server.Start(logger, stop)
	logger.Info("application started")

	<-appCtx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Stop HTTP intake before draining scheduler and worker activity
	var shutdownErr error
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown failed", "error", err)
		shutdownErr = err
	}

	if err := scheduler.Wait(shutdownCtx); err != nil {
		logger.Error("scheduler shutdown failed", "error", err)
		if shutdownErr == nil {
			shutdownErr = err
		}
	}
	if err := workerPool.Stop(shutdownCtx); err != nil {
		logger.Error("worker pool shutdown failed", "error", err)
		if shutdownErr == nil {
			shutdownErr = err
		}
	}

	logger.Info("application shutdown complete")
	return shutdownErr
}
