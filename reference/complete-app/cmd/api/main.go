package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/miuratouya/go-rest-api/internal/api"
	"github.com/miuratouya/go-rest-api/internal/config"
	"github.com/miuratouya/go-rest-api/internal/store/sqlite"
	"github.com/miuratouya/go-rest-api/internal/task"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := ensureDirectory(cfg.DBPath); err != nil {
		return fmt.Errorf("prepare database directory: %w", err)
	}

	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open sqlite database: %w", err)
	}
	defer db.Close()

	migrationCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := sqlite.Migrate(migrationCtx, db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	taskService := task.NewService(sqlite.NewRepository(db))
	handler := api.NewRouter(api.RouterDependencies{
		Logger:      logger,
		TaskService: taskService,
	})

	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  30 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("starting api server", "addr", cfg.Addr, "db_path", cfg.DBPath)
		serverErrors <- server.ListenAndServe()
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("listen and serve: %w", err)
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	return nil
}

func ensureDirectory(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir == "." {
		return nil
	}

	return os.MkdirAll(dir, 0o755)
}
