package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/mayday-team/server/internal/game"
	"github.com/mayday-team/server/internal/observability"
	"github.com/mayday-team/server/internal/transport/websocket"
)

// Server wires the HTTP listener for both /health and /ws.
type Server struct {
	srv     *http.Server
	log     *slog.Logger
	addr    string
}

// New constructs the HTTP server. The websocket handler is given the shared
// session manager so each accepted connection gets its own session.
func New(addr string, log *slog.Logger, sessions *game.SessionManager, metrics *observability.Metrics) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", NewHealthHandler(metrics))
	mux.Handle("/ws", websocket.NewHandler(log, sessions, metrics))

	return &Server{
		log:  log,
		addr: addr,
		srv: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// ListenAndServe blocks until the server returns an error.
func (s *Server) ListenAndServe() error {
	s.log.Info("http server listening", "addr", s.addr)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server with the supplied context deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
