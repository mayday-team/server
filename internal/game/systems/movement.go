package systems

import (
	gmath "github.com/example/mayday-server/internal/game/math"
	"github.com/example/mayday-server/internal/game/state"
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
		dir.Z += 1
	}
	if in.Backward {
		dir.Z -= 1
	}
	if in.Right {
		dir.X += 1
	}
	if in.Left {
		dir.X -= 1
	}
	if gmath.IsZero(dir) {
		p.Velocity = gmath.Vector3{}
		return
	}
	dir = gmath.Normalize(dir)
	dt := float64(deltaMs) / 1000.0
	step := gmath.Scale(dir, speed*dt)
	p.Position = gmath.Add(p.Position, step)
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

// ApplyPlayerLook updates yaw and pitch on the player. No clamping is done
// here beyond rejecting NaN; the client UI is expected to deliver sane
// values, but the server still accepts whatever it gets without crashing.
func ApplyPlayerLook(p *state.CivilianPlayerState, yaw, pitch float64) {
	if p == nil {
		return
	}
	if yaw != yaw || pitch != pitch { // NaN check
		return
	}
	p.Yaw = yaw
	p.Pitch = pitch
}
