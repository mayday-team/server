package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mayday-team/server/internal/config"
	"github.com/mayday-team/server/internal/game"
	"github.com/mayday-team/server/internal/protocol"
	"github.com/mayday-team/server/internal/storage"
)

type capturingSender struct {
	mu   sync.Mutex
	msgs []protocol.ServerMessage
}

func (s *capturingSender) Send(m protocol.ServerMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, m)
}

func (s *capturingSender) Snapshot() []protocol.ServerMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]protocol.ServerMessage, len(s.msgs))
	copy(out, s.msgs)
	return out
}

func (s *capturingSender) HasType(want string) bool {
	for _, m := range s.Snapshot() {
		if m.Type == want {
			return true
		}
	}
	return false
}

func testConfig() config.Config {
	c := config.Load() // populates with defaults from env if not set
	c.TickRate = 60
	c.SnapshotRate = 30
	c.InitialTroopCount = 1
	c.MaxTroopCount = 4
	c.SessionMaxDuration = 5 * time.Second
	c.FinalStandAfter = 800 * time.Millisecond
	c.ForceDefeatAfter = 1500 * time.Millisecond
	c.SessionEventBufferSize = 64
	c.ClientSendBufferSize = 64
	return c
}

func TestSessionStartsAndStopsLoop(t *testing.T) {
	cfg := testConfig()
	send := &capturingSender{}
	events := storage.NewMemoryEventRepository()
	sessions := storage.NewMemorySessionRepository()

	s := game.NewSession(game.SessionParams{
		PlayerName:  "tester",
		Cfg:         cfg,
		Sender:      send,
		EventRepo:   events,
		SessionRepo: sessions,
		Now:         time.Now(),
	})
	require.NotEmpty(t, s.ID())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	// Run for a few hundred ms; the director will eventually trigger a
	// scripted defeat because cfg.ForceDefeatAfter is short.
	select {
	case <-s.Done():
	case <-time.After(4 * time.Second):
		t.Fatal("session did not finish within 4s")
	}

	assert.True(t, send.HasType(protocol.ServerMsgWelcome), "expected welcome")
	assert.True(t, send.HasType(protocol.ServerMsgSessionStarted), "expected session_started")
	assert.True(t, send.HasType(protocol.ServerMsgSessionEnded), "expected session_ended")
	assert.True(t, send.HasType(protocol.ServerMsgDefeatTriggered), "expected defeat_triggered")

	// Memory repo should have at least the SESSION_STARTED and SESSION_ENDED rows.
	assert.GreaterOrEqual(t, len(events.All()), 2)
	assert.Len(t, sessions.Starts(), 1)
	assert.Len(t, sessions.Ends(), 1)
	end := sessions.Ends()[0]
	assert.Equal(t, s.ID(), end.ID)
	assert.NotEmpty(t, end.DefeatReason)
}

func TestSessionStopAfterStart(t *testing.T) {
	cfg := testConfig()
	cfg.FinalStandAfter = 5 * time.Second
	cfg.ForceDefeatAfter = 10 * time.Second
	send := &capturingSender{}

	s := game.NewSession(game.SessionParams{
		PlayerName: "tester",
		Cfg:        cfg,
		Sender:     send,
	})
	s.Start(context.Background())

	time.Sleep(100 * time.Millisecond)
	s.Stop()

	select {
	case <-s.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("session did not stop after Stop()")
	}
}
