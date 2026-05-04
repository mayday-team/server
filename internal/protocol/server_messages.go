package protocol

import (
	gmath "github.com/mayday-team/server/internal/game/math"
	"github.com/mayday-team/server/internal/game/scenario"
)

// Server → Client message types
const (
	ServerMsgWelcome              = "welcome"
	ServerMsgSessionStarted       = "session_started"
	ServerMsgStateSnapshot        = "state_snapshot"
	ServerMsgTroopSpawned         = "troop_spawned"
	ServerMsgShotResult           = "shot_result"
	ServerMsgDamageTaken          = "damage_taken"
	ServerMsgPlayerDied           = "player_died"
	ServerMsgScenarioPhaseChanged = "scenario_phase_changed"
	ServerMsgPressureChanged      = "pressure_changed"
	ServerMsgDefeatTriggered      = "defeat_triggered"
	ServerMsgSessionEnded         = "session_ended"
	ServerMsgEventLogged          = "event_logged"
	ServerMsgPong                 = "pong"
	ServerMsgError                = "error"
)

// ServerMessage is the typed wrapper used to send a message to the client.
// The transport layer is responsible for marshaling it onto the wire.
type ServerMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type WelcomePayload struct {
	ServerVersion string `json:"server_version"`
	ServerTime    int64  `json:"server_time"`
}

type SessionStartedPayload struct {
	SessionID string `json:"session_id"`
	TickRate  int    `json:"tick_rate"`
	StartedAt int64  `json:"started_at"`
}

type PlayerSnapshot struct {
	ID                    string        `json:"id"`
	Name                  string        `json:"name"`
	Position              gmath.Vector3 `json:"position"`
	Yaw                   float64       `json:"yaw"`
	Pitch                 float64       `json:"pitch"`
	HP                    int           `json:"hp"`
	MaxHP                 int           `json:"max_hp"`
	Ammo                  int           `json:"ammo"`
	MaxAmmo               int           `json:"max_ammo"`
	IsAlive               bool          `json:"is_alive"`
	LastProcessedInputSeq int64         `json:"last_processed_input_seq"`
	SurvivalTimeMs        int64         `json:"survival_time_ms"`
	Morale                float64       `json:"morale"`
}

type TroopSnapshot struct {
	ID       string        `json:"id"`
	Position gmath.Vector3 `json:"position"`
	Yaw      float64       `json:"yaw"`
	HP       int           `json:"hp"`
	MaxHP    int           `json:"max_hp"`
	State    string        `json:"state"`
	IsAlive  bool          `json:"is_alive"`
	SquadID  string        `json:"squad_id"`
}

type EventSnapshot struct {
	Type       string `json:"type"`
	ServerTick int64  `json:"server_tick"`
}

type StateSnapshotPayload struct {
	ServerTick        int64                  `json:"server_tick"`
	SessionID         string                 `json:"session_id"`
	ScenarioPhase     scenario.Phase         `json:"scenario_phase"`
	PressureLevel     float64                `json:"pressure_level"`
	EncirclementLevel float64                `json:"encirclement_level"`
	Player            PlayerSnapshot         `json:"player"`
	Troops            []TroopSnapshot        `json:"troops"`
	Events            []EventSnapshot        `json:"events"`
}

type TroopSpawnedPayload struct {
	Troop      TroopSnapshot `json:"troop"`
	ServerTick int64         `json:"server_tick"`
}

type ShotResultPayload struct {
	Seq         int64   `json:"seq"`
	Accepted    bool    `json:"accepted"`
	Reason      string  `json:"reason"`
	HitTroopID  string  `json:"hit_troop_id"`
	HitDistance float64 `json:"hit_distance"`
	DamageDealt int     `json:"damage_dealt"`
	TroopKilled bool    `json:"troop_killed"`
	AmmoLeft    int     `json:"ammo_left"`
}

type DamageTakenPayload struct {
	Source      string `json:"source"`
	SourceID    string `json:"source_id"`
	Damage      int    `json:"damage"`
	RemainingHP int    `json:"remaining_hp"`
}

type PlayerDiedPayload struct {
	SessionID string `json:"session_id"`
	Tick      int64  `json:"tick"`
}

type ScenarioPhaseChangedPayload struct {
	PreviousPhase scenario.Phase `json:"previous_phase"`
	CurrentPhase  scenario.Phase `json:"current_phase"`
	Tick          int64          `json:"tick"`
}

type PressureChangedPayload struct {
	PressureLevel     float64 `json:"pressure_level"`
	EncirclementLevel float64 `json:"encirclement_level"`
}

type DefeatTriggeredPayload struct {
	Reason scenario.DefeatReason `json:"reason"`
	Tick   int64                 `json:"tick"`
}

type SessionEndedPayload struct {
	SessionID         string                `json:"session_id"`
	SurvivedMs        int64                 `json:"survived_ms"`
	FinalPhase        scenario.Phase        `json:"final_phase"`
	DefeatReason      scenario.DefeatReason `json:"defeat_reason"`
	ShotsFired        int                   `json:"shots_fired"`
	ShotsHit          int                   `json:"shots_hit"`
	DamageTaken       int                   `json:"damage_taken"`
	TroopsNeutralized int                   `json:"troops_neutralized"`
	EventsRecorded    int                   `json:"events_recorded"`
}

type EventLoggedPayload struct {
	Type       string `json:"type"`
	ServerTick int64  `json:"server_tick"`
}

type PongPayload struct {
	ClientTime int64 `json:"client_time"`
	ServerTime int64 `json:"server_time"`
}
