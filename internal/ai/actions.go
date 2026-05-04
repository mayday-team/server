package ai

import gmath "github.com/example/mayday-server/internal/game/math"

type ActionKind string

const (
	ActionIdle              ActionKind = "IDLE"
	ActionMoveTo            ActionKind = "MOVE_TO"
	ActionLookAt            ActionKind = "LOOK_AT"
	ActionShoot             ActionKind = "SHOOT"
	ActionSuppressArea      ActionKind = "SUPPRESS_AREA"
	ActionFlankTo           ActionKind = "FLANK_TO"
	ActionTakeCover         ActionKind = "TAKE_COVER"
	ActionCallReinforcement ActionKind = "CALL_REINFORCEMENT"
	ActionBlockExit         ActionKind = "BLOCK_EXIT"
)

// Action is a unit of intent the AI emits each tick. Game systems are
// responsible for converting actions into authoritative state mutations.
type Action struct {
	Kind     ActionKind
	Target   gmath.Vector3
	HasPoint bool
}

func MoveTo(p gmath.Vector3) Action      { return Action{Kind: ActionMoveTo, Target: p, HasPoint: true} }
func LookAt(p gmath.Vector3) Action      { return Action{Kind: ActionLookAt, Target: p, HasPoint: true} }
func Shoot(at gmath.Vector3) Action      { return Action{Kind: ActionShoot, Target: at, HasPoint: true} }
func FlankTo(p gmath.Vector3) Action     { return Action{Kind: ActionFlankTo, Target: p, HasPoint: true} }
func SuppressArea(p gmath.Vector3) Action {
	return Action{Kind: ActionSuppressArea, Target: p, HasPoint: true}
}
func TakeCover() Action         { return Action{Kind: ActionTakeCover} }
func CallReinforcement() Action { return Action{Kind: ActionCallReinforcement} }
func BlockExit(p gmath.Vector3) Action {
	return Action{Kind: ActionBlockExit, Target: p, HasPoint: true}
}
func Idle() Action { return Action{Kind: ActionIdle} }
