package websocket

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/mayday-team/server/internal/game"
	"github.com/mayday-team/server/internal/observability"
	"github.com/mayday-team/server/internal/protocol"
)

// Handler accepts /ws connections and brokers them to the SessionManager.
type Handler struct {
	log      *slog.Logger
	sessions *game.SessionManager
	metrics  *observability.Metrics
	upgrader websocket.Upgrader
}

func NewHandler(log *slog.Logger, sessions *game.SessionManager, metrics *observability.Metrics) *Handler {
	return &Handler{
		log:      log,
		sessions: sessions,
		metrics:  metrics,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(_ *http.Request) bool { return true },
		},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Warn("ws upgrade failed", "err", err)
		return
	}

	client := newClient(conn, h.log, 64)
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
			session = h.sessions.Create(r.Context(), name, client)
			return
		}
		session.EnqueueInput(msg)
	})

	if session != nil {
		session.MarkDisconnected()
	}
	h.log.Info("ws client disconnected", "remote", r.RemoteAddr)
}
