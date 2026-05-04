package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mayday-team/server/internal/config"
	"github.com/mayday-team/server/internal/game"
	"github.com/mayday-team/server/internal/logger"
	"github.com/mayday-team/server/internal/observability"
	"github.com/mayday-team/server/internal/storage"
	httpx "github.com/mayday-team/server/internal/transport/http"
)

func main() {
	log := logger.New()

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Error("invalid config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	metrics := observability.New()

	var (
		eventRepo   storage.EventRepository = storage.NoopEventRepository{}
		sessionRepo storage.SessionRepository = storage.NoopSessionRepository{}
	)
	if cfg.DatabaseURL != "" {
		pool, err := storage.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Warn("database unavailable; running with no-op repositories", "err", err)
		} else {
			defer pool.Close()
			eventRepo = storage.NewPostgresEventRepository(pool)
			sessionRepo = storage.NewPostgresSessionRepository(pool)
			log.Info("database connected")
		}
	}

	sessions := game.NewSessionManager(cfg, log, eventRepo, sessionRepo, metrics)

	srv := httpx.New(":"+cfg.Port, log, sessions, metrics)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err, ok := <-errCh:
		if ok && err != nil {
			log.Error("http server error", "err", err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Warn("http shutdown error", "err", err)
	}
	sessions.StopAll()
	log.Info("mayday-server stopped")
}
