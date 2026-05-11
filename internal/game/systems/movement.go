package systems

import (
	"math"

	gmath "github.com/mayday-team/server/internal/game/math"
	"github.com/mayday-team/server/internal/game/state"
)

// MovementInput represents the boolean directional input for one tick.
type MovementInput struct {
	Forward  bool
	Backward bool
	Left     bool
	Right    bool
}

// MaxDeltaMs caps a delta supplied by the client to a sane upper bound so a
// stuck or malicious client cannot teleport the player.
const MaxDeltaMs = 100

const (
	playerMinX = -12.5
	playerMaxX = 12.5
	playerMinZ = -45.0
	playerMaxZ = -34.5
)

func clampPlayerPosition(pos gmath.Vector3) gmath.Vector3 {
	pos.X = math.Max(playerMinX, math.Min(playerMaxX, pos.X))
	pos.Z = math.Max(playerMinZ, math.Min(playerMaxZ, pos.Z))
	pos.Y = 7.0
	return pos
}

// ApplyClientPlayerPosition accepts the client-predicted camera position as
// the session position so server systems follow the smooth local movement.
func ApplyClientPlayerPosition(p *state.CivilianPlayerState, pos gmath.Vector3) {
	if p == nil || !p.IsAlive {
		return
	}
	p.Position = clampPlayerPosition(pos)
}

// ApplyPlayerMovement mutates the player position based on input booleans.
// The dead player cannot move. Diagonal movement is normalized.
func ApplyPlayerMovement(p *state.CivilianPlayerState, in MovementInput, deltaMs int64, speed float64) {
	if p == nil || !p.IsAlive {
		return
	}
	if deltaMs <= 0 {
		return
	}
	if deltaMs > MaxDeltaMs {
		deltaMs = MaxDeltaMs
	}

	dir := gmath.Vector3{}
	if in.Forward {
		dir.X += math.Sin(p.Yaw)
		dir.Z += math.Cos(p.Yaw)
	}
	if in.Backward {
		dir.X -= math.Sin(p.Yaw)
		dir.Z -= math.Cos(p.Yaw)
	}
	if in.Right {
		dir.X -= math.Cos(p.Yaw)
		dir.Z += math.Sin(p.Yaw)
	}
	if in.Left {
		dir.X += math.Cos(p.Yaw)
		dir.Z -= math.Sin(p.Yaw)
	}
	if gmath.IsZero(dir) {
		p.Velocity = gmath.Vector3{}
		return
	}
	dir = gmath.Normalize(dir)
	dt := float64(deltaMs) / 1000.0
	step := gmath.Scale(dir, speed*dt)
	p.Position = clampPlayerPosition(gmath.Add(p.Position, step))
	p.Velocity = gmath.Scale(dir, speed)
}

// MoveTroopToward moves a troop toward target by speed*dt, but never past
// the target. Returns the post-step distance to target so callers can decide
// whether the troop arrived.
func MoveTroopToward(t *state.MartialTroopState, target gmath.Vector3, deltaMs int64, speed float64) float64 {
	if t == nil || !t.IsAlive {
		return 0
	}
	if deltaMs <= 0 {
		return gmath.Distance(t.Position, target)
	}
	if deltaMs > MaxDeltaMs*4 {
		deltaMs = MaxDeltaMs * 4
	}
	to := gmath.Sub(target, t.Position)
	dist := gmath.Length(to)
	if dist == 0 {
		t.Velocity = gmath.Vector3{}
		return 0
	}
	dir := gmath.Scale(to, 1.0/dist)
	dt := float64(deltaMs) / 1000.0
	maxStep := speed * dt
	if maxStep >= dist {
		t.Position = target
		t.Velocity = gmath.Vector3{}
		return 0
	}
	t.Position = gmath.Add(t.Position, gmath.Scale(dir, maxStep))
	t.Velocity = gmath.Scale(dir, speed)
	return dist - maxStep
}

// ApplyPlayerLook updates yaw and pitch on the player, rejecting NaN.
func ApplyPlayerLook(p *state.CivilianPlayerState, yaw, pitch float64) {
	if p == nil {
		return
	}
	if math.IsNaN(yaw) || math.IsNaN(pitch) {
		return
	}
	p.Yaw = yaw
	p.Pitch = pitch
}
