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

// main starts the uptime monitor application
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("application stopped with error", "error", err)
	}
}

// run builds dependencies and manages the application lifecycle
func run(logger *slog.Logger) error {
	// Build all application dependencies
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databasePool, err := postgres.NewPool(
		ctx,
		cfg.DatabaseURL,
		time.Duration(cfg.DatabasePingTimeout)*time.Second,
	)
	if err != nil {
		return err
	}
	defer databasePool.Close()
	logger.Info("database connected")

	monitorRepository := postgres.NewMonitorRepository(databasePool)
	checkRepository := postgres.NewCheckRepository(databasePool)
	monitorService := application.NewMonitorService(monitorRepository, checkRepository)
	inFlight := application.NewInFlightRegistry()
	checker := httpclient.NewChecker()
	checkService := application.NewCheckService(monitorRepository, checkRepository, checker, inFlight, logger)

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
	workerPool.Start()

	scheduler := application.NewScheduler(
		monitorRepository,
		inFlight,
		workerPool,
		time.Duration(cfg.SchedulerInterval)*time.Second,
		logger,
	)
	scheduler.Start(ctx)

	server := httpserver.NewServer(cfg.HTTPAddr, httpserver.Dependencies{
		Monitors: monitorService,
		Checks:   checkService,
		WebDir:   "web",
		Logger:   logger,
	})
	server.Start(logger, stop)
	logger.Info("application started")

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(cfg.ShutdownTimeout)*time.Second,
	)
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
