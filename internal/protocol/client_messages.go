package protocol

import (
	"encoding/json"

	gmath "github.com/mayday-team/server/internal/game/math"
)

// Client → Server message types
const (
	ClientMsgStartSession = "start_session"
	ClientMsgPlayerInput  = "player_input"
	ClientMsgPlayerLook   = "player_look"
	ClientMsgShoot        = "shoot"
	ClientMsgReload       = "reload"
	ClientMsgInteract     = "interact"
	ClientMsgPing         = "ping"
)

// Envelope is the shared client/server message wrapper. The payload is
// decoded into a typed struct only after the type is known.
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type StartSessionPayload struct {
	PlayerName string `json:"player_name"`
}

type MoveInput struct {
	Forward  bool `json:"forward"`
	Backward bool `json:"backward"`
	Left     bool `json:"left"`
	Right    bool `json:"right"`
}

type PlayerInputPayload struct {
	Seq     int64     `json:"seq"`
	Move    MoveInput `json:"move"`
	DeltaMs int64     `json:"delta_ms"`
}

type PlayerLookPayload struct {
	Yaw   float64 `json:"yaw"`
	Pitch float64 `json:"pitch"`
}

type ShootPayload struct {
	Seq        int64         `json:"seq"`
	Origin     gmath.Vector3 `json:"origin"`
	Direction  gmath.Vector3 `json:"direction"`
	ClientTime int64         `json:"client_time"`
}

type ReloadPayload struct{}

type InteractPayload struct {
	TargetID string `json:"target_id"`
}

type PingPayload struct {
	ClientTime int64 `json:"client_time"`
}
