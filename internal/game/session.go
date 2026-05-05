package game

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mayday-team/server/internal/config"
	gmath "github.com/mayday-team/server/internal/game/math"
	"github.com/mayday-team/server/internal/game/scenario"
	"github.com/mayday-team/server/internal/game/state"
	"github.com/mayday-team/server/internal/game/systems"
	"github.com/mayday-team/server/internal/protocol"
	"github.com/mayday-team/server/internal/storage"
)

// Sender is the minimal contract the session expects of a transport-level
// client. The websocket package satisfies this; tests can plug in fakes.
type Sender interface {
	Send(msg protocol.ServerMessage)
}

// SessionParams collects everything a Session needs at construction.
type SessionParams struct {
	ID          string
	PlayerName  string
	Cfg         config.Config
	Logger      *slog.Logger
	Sender      Sender
	EventRepo   storage.EventRepository
	SessionRepo storage.SessionRepository
	Now         time.Time
}

// Session is the authoritative simulation for one player. Its tick loop is
// the only goroutine that mutates session state; all transport input is
// delivered through inputCh.
type Session struct {
	id  string
	log *slog.Logger
	cfg config.Config

	inputCh chan protocol.ClientMessage
	sender  Sender

	eventRepo   storage.EventRepository
	sessionRepo storage.SessionRepository

	director *scenario.Director

	player *state.CivilianPlayerState
	troops map[string]*state.MartialTroopState

	serverTick int64
	rng        *rand.Rand

	eventBuf      chan storage.EventRecord
	persisterDone chan struct{}

	stats systems.SessionStats

	ctx    context.Context
	cancel context.CancelFunc

	startedAt  time.Time
	lastTickAt time.Time

	onceStop sync.Once
	doneCh   chan struct{}
}

// NewSession constructs a Session. The tick loop is not started until Start
// is called.
func NewSession(p SessionParams) *Session {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.Logger == nil {
		p.Logger = slog.Default()
	}
	if p.EventRepo == nil {
		p.EventRepo = storage.NoopEventRepository{}
	}
	if p.SessionRepo == nil {
		p.SessionRepo = storage.NoopSessionRepository{}
	}
	if p.Now.IsZero() {
		p.Now = time.Now()
	}

	rng := rand.New(rand.NewSource(p.Now.UnixNano()))

	cfg := p.Cfg
	player := &state.CivilianPlayerState{
		ID:       uuid.NewString(),
		Name:     p.PlayerName,
		Position: gmath.Vector3{X: 0, Y: PlayerStartY, Z: 0},
		HP:       cfg.PlayerMaxHP,
		MaxHP:    cfg.PlayerMaxHP,
		Ammo:     cfg.PlayerMaxAmmo,
		MaxAmmo:  cfg.PlayerMaxAmmo,
		IsAlive:  true,
		JoinedAt: p.Now.UTC(),
		Morale:   1.0,
	}

	return &Session{
		id:          p.ID,
		log:         p.Logger.With("session_id", p.ID),
		cfg:         cfg,
		inputCh:     make(chan protocol.ClientMessage, cfg.SessionEventBufferSize),
		sender:      p.Sender,
		eventRepo:   p.EventRepo,
		sessionRepo: p.SessionRepo,
		director: scenario.NewDirector(p.Now, scenario.Config{
			FinalStandAfter:  cfg.FinalStandAfter,
			ForceDefeatAfter: cfg.ForceDefeatAfter,
			MaxTroops:        cfg.MaxTroopCount,
		}),
		player:        player,
		troops:        make(map[string]*state.MartialTroopState),
		rng:           rng,
		eventBuf:      make(chan storage.EventRecord, cfg.SessionEventBufferSize),
		persisterDone: make(chan struct{}),
		doneCh:        make(chan struct{}),
		startedAt:     p.Now,
	}
}

// ID returns the session UUID.
func (s *Session) ID() string { return s.id }

// Done returns a channel closed once the tick loop fully exits.
func (s *Session) Done() <-chan struct{} { return s.doneCh }

// EnqueueInput delivers a parsed client message to the session loop. The
// call is non-blocking: if the buffer is full, the message is dropped and
// a warning is logged.
func (s *Session) EnqueueInput(msg protocol.ClientMessage) {
	select {
	case s.inputCh <- msg:
	default:
		s.log.Warn("input dropped; buffer full", "type", msg.Type)
	}
}

// Start launches the tick loop and the asynchronous event persister.
func (s *Session) Start(parent context.Context) {
	if parent == nil {
		parent = context.Background()
	}
	s.ctx, s.cancel = context.WithCancel(parent)
	s.lastTickAt = s.startedAt
	s.persistSessionStart()
	go s.runEventPersister()
	go s.run()
}

// Stop signals the session loop to exit. Idempotent.
func (s *Session) Stop() {
	s.onceStop.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
	})
}

// MarkDisconnected forces the session into DEFEAT due to a transport-level
// disconnect. Called by the WebSocket handler when the connection drops.
func (s *Session) MarkDisconnected() {
	upd := s.director.MarkDisconnected(time.Now())
	if upd.TriggeredDefeat {
		s.log.Info("session marked as disconnected")
	}
	s.Stop()
}

func (s *Session) persistSessionStart() {
	rec := storage.SessionStartRecord{
		ID:         s.id,
		PlayerName: s.player.Name,
		StartedAt:  s.startedAt.UTC(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.sessionRepo.Start(ctx, rec); err != nil {
		s.log.Warn("session start persist failed", "err", err)
	}
}

func (s *Session) runEventPersister() {
	defer close(s.persisterDone)
	for ev := range s.eventBuf {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := s.eventRepo.Insert(ctx, ev); err != nil {
			s.log.Warn("event insert failed", "type", ev.Type, "err", err)
		}
		cancel()
	}
}

func (s *Session) sendType(msgType string, payload any) {
	if s.sender == nil {
		return
	}
	s.sender.Send(protocol.ServerMessage{Type: msgType, Payload: payload})
}

func aliveTroopCount(troops map[string]*state.MartialTroopState) int {
	n := 0
	for _, t := range troops {
		if t != nil && t.IsAlive {
			n++
		}
	}
	return n
}
