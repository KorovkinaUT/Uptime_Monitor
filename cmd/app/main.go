package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ulyana/uptime-monitor/internal/application"
	"github.com/ulyana/uptime-monitor/internal/infrastructure/checker"
	"github.com/ulyana/uptime-monitor/internal/infrastructure/config"
	"github.com/ulyana/uptime-monitor/internal/infrastructure/httpserver"
	"github.com/ulyana/uptime-monitor/internal/infrastructure/postgres"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("application stopped with error", "error", err)
	}
}

func run(logger *slog.Logger) error {
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
	httpChecker := checker.NewHTTPChecker(nil)
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

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: httpserver.NewRouter(monitorService, checkService, "web", logger),
	}
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()
	logger.Info("application started", "address", cfg.HTTPAddr)

	var serverErr error
	select {
	case <-appCtx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			serverErr = err
		}
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown failed", "error", err)
		if serverErr == nil {
			serverErr = err
		}
	}

	scheduler.Wait()
	if err := workerPool.Stop(shutdownCtx); err != nil {
		logger.Error("worker pool shutdown failed", "error", err)
		if serverErr == nil {
			serverErr = err
		}
	}

	logger.Info("application shutdown complete")
	return serverErr
}
