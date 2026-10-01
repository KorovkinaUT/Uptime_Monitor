package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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
	// Load configuration before opening external resources
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Propagate termination signals to every background component
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Keep the database open while services finish their work
	databasePool, err := postgres.NewPool(
		ctx,
		cfg.DatabaseURL,
		cfg.DatabasePingTimeout,
	)
	if err != nil {
		return err
	}
	defer databasePool.Close()
	logger.Info("database connected")

	// Build application services around shared repositories
	monitorRepository := postgres.NewMonitorRepository(databasePool)
	checkRepository := postgres.NewCheckRepository(databasePool)
	monitorService := application.NewMonitorService(monitorRepository, checkRepository)
	inFlight := application.NewInFlightRegistry()
	checker := httpclient.NewChecker()
	checkService := application.NewCheckService(monitorRepository, checkRepository, checker, inFlight, logger)

	// Execute checks through a bounded queue
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

	// Schedule only monitors whose next check is due
	scheduler := application.NewScheduler(
		monitorRepository,
		inFlight,
		workerPool,
		cfg.SchedulerInterval,
		logger,
	)
	scheduler.Start(ctx)

	// Expose the application services through the HTTP transport
	server := httpserver.NewServer(cfg.HTTPAddr, httpserver.Dependencies{
		Monitors: monitorService,
		Checks:   checkService,
		WebDir:   "web",
		Logger:   logger,
	})
	server.Start(logger, stop)
	logger.Info("application started")

	// Wait until a termination signal cancels the application context
	<-ctx.Done()
	logger.Info("starting shutdown")

	// Give every shutdown operation one shared deadline
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		cfg.ShutdownTimeout,
	)
	defer cancel()

	// Stop new requests before draining scheduled and queued checks
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

	// Return the first shutdown error after all components had a chance to stop
	logger.Info("application shutdown complete")
	return shutdownErr
}
