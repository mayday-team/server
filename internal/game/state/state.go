package state

import (
	"time"

	"github.com/example/mayday-server/internal/ai"
	gmath "github.com/example/mayday-server/internal/game/math"
)

// CivilianPlayerState is the authoritative server-side player record. The
// game session goroutine is the only goroutine allowed to mutate it.
type CivilianPlayerState struct {
	ID                    string
	Name                  string
	Position              gmath.Vector3
	Velocity              gmath.Vector3
	Yaw                   float64
	Pitch                 float64
	HP                    int
	MaxHP                 int
	Ammo                  int
	MaxAmmo               int
	IsAlive               bool
	LastProcessedInputSeq int64
	JoinedAt              time.Time
	LastSeenAt            time.Time
	LastShotAt            time.Time
	SurvivalTimeMs        int64
	Morale                float64
}

// MartialTroopState is the authoritative server-side record for one martial
// law troop. AI mutates intent (FSMState, action queue) but final positions
// and damage are applied by the systems package.
type MartialTroopState struct {
	ID                      string
	Position                gmath.Vector3
	Velocity                gmath.Vector3
	Yaw                     float64
	Pitch                   float64
	HP                      int
	MaxHP                   int
	Ammo                    int
	MaxAmmo                 int
	State                   ai.FSMState
	TargetPlayerID          string
	LastKnownTargetPosition *gmath.Vector3
	IsAlive                 bool
	Difficulty              string
	LastShotAt              time.Time
	SquadID                 string
}
