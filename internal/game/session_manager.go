package game

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mayday-team/server/internal/config"
	"github.com/mayday-team/server/internal/observability"
	"github.com/mayday-team/server/internal/storage"
)

// ErrAtCapacity is returned by SessionManager.Create when the configured
// MaxSessions limit has already been reached.
var ErrAtCapacity = errors.New("session manager at capacity")

// SessionManager owns active sessions and brokers WebSocket clients to
// them. Mayday is a single-player game, so each connected client gets its
// own session keyed by the client's transport identifier.
type SessionManager struct {
	cfg         config.Config
	log         *slog.Logger
	eventRepo   storage.EventRepository
	sessionRepo storage.SessionRepository
	metrics     *observability.Metrics

	mu       sync.Mutex
	sessions map[string]*Session
}

func NewSessionManager(
	cfg config.Config,
	log *slog.Logger,
	eventRepo storage.EventRepository,
	sessionRepo storage.SessionRepository,
	metrics *observability.Metrics,
) *SessionManager {
	if eventRepo == nil {
		eventRepo = storage.NoopEventRepository{}
	}
	if sessionRepo == nil {
		sessionRepo = storage.NoopSessionRepository{}
	}
	return &SessionManager{
		cfg:         cfg,
		log:         log,
		eventRepo:   eventRepo,
		sessionRepo: sessionRepo,
		metrics:     metrics,
		sessions:    make(map[string]*Session),
	}
}

// Create constructs and starts a new session for the supplied transport
// client. The caller is responsible for routing inbound messages to the
// session via session.EnqueueInput. Returns ErrAtCapacity if MaxSessions
// would be exceeded.
func (m *SessionManager) Create(ctx context.Context, playerName string, sender Sender) (*Session, error) {
	id := uuid.NewString()
	s := NewSession(SessionParams{
		ID:          id,
		PlayerName:  playerName,
		Cfg:         m.cfg,
		Logger:      m.log,
		Sender:      sender,
		EventRepo:   m.eventRepo,
		SessionRepo: m.sessionRepo,
		Now:         time.Now(),
	})

	m.mu.Lock()
	if m.cfg.MaxSessions > 0 && len(m.sessions) >= m.cfg.MaxSessions {
		m.mu.Unlock()
		return nil, ErrAtCapacity
	}
	m.sessions[id] = s
	m.mu.Unlock()

	if m.metrics != nil {
		m.metrics.ActiveSessions.Add(1)
		m.metrics.TotalSessions.Add(1)
	}

	s.Start(ctx)

	go func() {
		<-s.Done()
		m.mu.Lock()
		delete(m.sessions, id)
		m.mu.Unlock()
		if m.metrics != nil {
			m.metrics.ActiveSessions.Add(-1)
		}
	}()

	return s, nil
}

// ActiveCount returns the number of currently running sessions.
func (m *SessionManager) ActiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

// StopAll signals every active session to terminate. Used during graceful
// server shutdown.
func (m *SessionManager) StopAll() {
	m.mu.Lock()
	for _, s := range m.sessions {
		s.MarkDisconnected()
	}
	m.mu.Unlock()
}
