package websocket

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mayday-team/server/internal/config"
	"github.com/mayday-team/server/internal/game"
	"github.com/mayday-team/server/internal/observability"
	"github.com/mayday-team/server/internal/protocol"
)

// Handler accepts /ws connections and brokers them to the SessionManager.
type Handler struct {
	log              *slog.Logger
	sessions         *game.SessionManager
	metrics          *observability.Metrics
	upgrader         websocket.Upgrader
	clientSendBuffer int
	readTimeout      time.Duration
	pingInterval     time.Duration
	writeTimeout     time.Duration
}

func NewHandler(log *slog.Logger, sessions *game.SessionManager, metrics *observability.Metrics, cfg config.Config) *Handler {
	allowed := cfg.AllowedOrigins
	checkOrigin := func(r *http.Request) bool {
		if len(allowed) == 0 {
			return true
		}
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Native clients without an Origin header are accepted when an
			// allowlist is set; browsers always send Origin.
			return true
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		host := strings.ToLower(u.Host)
		for _, a := range allowed {
			if strings.EqualFold(a, origin) || strings.EqualFold(a, host) {
				return true
			}
		}
		log.Warn("ws origin rejected", "origin", origin)
		return false
	}
	return &Handler{
		log:              log,
		sessions:         sessions,
		metrics:          metrics,
		clientSendBuffer: cfg.ClientSendBufferSize,
		readTimeout:      cfg.ClientReadTimeout,
		pingInterval:     cfg.ClientPingInterval,
		writeTimeout:     cfg.ClientWriteTimeout,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     checkOrigin,
		},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Warn("ws upgrade failed", "err", err)
		return
	}

	client := newClient(conn, h.log, h.clientSendBuffer, h.readTimeout, h.writeTimeout)
	h.log.Info("ws client connected", "remote", r.RemoteAddr)

	var session *game.Session

	go h.runWriter(client)

	h.runReader(client, func(msg protocol.ClientMessage) {
		if session == nil {
			if msg.Type != protocol.ClientMsgStartSession {
				client.Send(protocol.ServerMessage{
					Type: protocol.ServerMsgError,
					Payload: protocol.ErrorPayload{
						Code:    "session_not_started",
						Message: "expected start_session as first message",
					},
				})
				return
			}
			name := "anonymous"
			if msg.StartSession != nil && msg.StartSession.PlayerName != "" {
				name = msg.StartSession.PlayerName
			}
			s, cerr := h.sessions.Create(r.Context(), name, client)
			if cerr != nil {
				if errors.Is(cerr, game.ErrAtCapacity) {
					client.Send(protocol.ServerMessage{
						Type: protocol.ServerMsgError,
						Payload: protocol.ErrorPayload{
							Code:    "at_capacity",
							Message: "server at session capacity; try again later",
						},
					})
				} else {
					h.log.Warn("session create failed", "err", cerr)
					client.Send(protocol.ServerMessage{
						Type: protocol.ServerMsgError,
						Payload: protocol.ErrorPayload{
							Code:    "session_create_failed",
							Message: cerr.Error(),
						},
					})
				}
				client.Close()
				return
			}
			session = s
			return
		}
		session.EnqueueInput(msg)
	})

	if session != nil {
		session.MarkDisconnected()
	}
	h.log.Info("ws client disconnected", "remote", r.RemoteAddr)
}
