package ai

import (
	gmath "github.com/mayday-team/server/internal/game/math"
	"github.com/mayday-team/server/internal/game/scenario"
)

// TroopSnapshot is the per-troop view passed to Decide. It avoids importing
// the full game state into the ai package.
type TroopSnapshot struct {
	ID       string
	Position gmath.Vector3
	HP       int
	MaxHP    int
	Ammo     int
	IsAlive  bool
	State    FSMState
}

// DecisionInput bundles everything needed to choose a next FSM state and a
// list of actions for a single troop on a single tick.
type DecisionInput struct {
	Now           int64
	Troop         TroopSnapshot
	Perception    PerceptionResult
	Phase         scenario.Phase
	Pressure      float64
	Encirclement  float64
	EscapeBlocked bool
	TroopCount    int
	MaxTroops     int
	MinTroops     int
}

// Decision is the result of one decision call.
type Decision struct {
	NextState FSMState
	Actions   []Action
}

// Decide computes the next FSM state and the actions a troop will perform
// this tick. It is a pure function; the caller mutates state.
func Decide(in DecisionInput) Decision {
	if !in.Troop.IsAlive || in.Troop.HP <= 0 {
		return Decision{NextState: StateDead, Actions: []Action{Idle()}}
	}

	low := in.Troop.MaxHP > 0 && in.Troop.HP*100/in.Troop.MaxHP <= 25
	if low && in.Pressure < 0.7 {
		return Decision{NextState: StateTakeCover, Actions: []Action{TakeCover()}}
	}

	if in.EscapeBlocked && (in.Phase == scenario.PhaseEncirclement || in.Phase == scenario.PhaseFinalStand) {
		return Decision{
			NextState: StateBlockExit,
			Actions: []Action{
				BlockExit(in.Perception.ToPlayer),
				LookAt(in.Perception.PlayerPositionFrom(in.Troop.Position)),
			},
		}
	}

	if in.MaxTroops > 0 && in.TroopCount < in.MinTroops &&
		(in.Phase == scenario.PhaseReinforcement ||
			in.Phase == scenario.PhaseEncirclement ||
			in.Phase == scenario.PhaseFinalStand) {
		return Decision{
			NextState: StateCallReinforcement,
			Actions:   []Action{CallReinforcement()},
		}
	}

	if !in.Perception.PlayerVisible {
		return Decision{
			NextState: StatePatrol,
			Actions:   []Action{Idle()},
		}
	}

	playerPos := in.Perception.PlayerPositionFrom(in.Troop.Position)

	if in.Perception.InAttackRange {
		if in.Troop.Ammo <= 0 {
			return Decision{
				NextState: StateAdvance,
				Actions:   []Action{MoveTo(playerPos), LookAt(playerPos)},
			}
		}
		actions := []Action{LookAt(playerPos), Shoot(playerPos)}
		if in.Pressure >= 0.6 {
			actions = append(actions, SuppressArea(playerPos))
			return Decision{NextState: StateSuppress, Actions: actions}
		}
		return Decision{NextState: StateAttack, Actions: actions}
	}

	if in.Phase == scenario.PhaseEncirclement || in.Phase == scenario.PhaseFinalStand ||
		in.Encirclement >= 0.5 {
		flankPoint := flankOffset(in.Troop.Position, playerPos)
		return Decision{
			NextState: StateFlank,
			Actions:   []Action{FlankTo(flankPoint), LookAt(playerPos)},
		}
	}

	return Decision{
		NextState: StateChase,
		Actions:   []Action{MoveTo(playerPos), LookAt(playerPos)},
	}
}

// PlayerPositionFrom reconstructs the player's absolute position from a
// troop position plus the perception delta. The ai package only stores the
// delta to keep PerceptionResult self-contained.
func (p PerceptionResult) PlayerPositionFrom(troopPos gmath.Vector3) gmath.Vector3 {
	return gmath.Add(troopPos, p.ToPlayer)
}

// flankOffset returns a point perpendicular to the troop->player line so the
// troop has a place to flank toward. Cheap, deterministic-ish heuristic.
func flankOffset(troop, player gmath.Vector3) gmath.Vector3 {
	dir := gmath.Sub(player, troop)
	dir = gmath.Normalize(dir)
	// rotate 90 degrees on the XZ plane: (x,z) -> (-z, x)
	perp := gmath.Vector3{X: -dir.Z, Y: 0, Z: dir.X}
	offset := gmath.Scale(perp, 6)
	return gmath.Add(player, offset)
}
