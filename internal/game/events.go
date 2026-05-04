package game

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType is the canonical set of session event types persisted to
// game_events for replay and analytics.
type EventType string

const (
	EventSessionStarted   EventType = "SESSION_STARTED"
	EventPhaseChanged     EventType = "PHASE_CHANGED"
	EventPressureChanged  EventType = "PRESSURE_CHANGED"
	EventTroopSpawned     EventType = "TROOP_SPAWNED"
	EventPlayerShot       EventType = "PLAYER_SHOT"
	EventPlayerHitTroop   EventType = "PLAYER_HIT_TROOP"
	EventPlayerDamaged    EventType = "PLAYER_DAMAGED"
	EventPlayerDied       EventType = "PLAYER_DIED"
	EventDefeatTriggered  EventType = "DEFEAT_TRIGGERED"
	EventSessionEnded     EventType = "SESSION_ENDED"
)

// Event is one row in the game_events log.
type Event struct {
	ID         string          `json:"id"`
	SessionID  string          `json:"session_id"`
	Type       EventType       `json:"type"`
	ServerTick int64           `json:"server_tick"`
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"created_at"`
}

// NewEvent builds an Event with a fresh UUID and the supplied payload
// already marshaled. Marshal errors are silently turned into "{}" so the
// game loop never panics on a logging failure.
func NewEvent(sessionID string, t EventType, tick int64, payload any) Event {
	raw := encodePayload(payload)
	return Event{
		ID:         uuid.NewString(),
		SessionID:  sessionID,
		Type:       t,
		ServerTick: tick,
		Payload:    raw,
		CreatedAt:  time.Now().UTC(),
	}
}

func encodePayload(payload any) json.RawMessage {
	if payload == nil {
		return json.RawMessage(`{}`)
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}
