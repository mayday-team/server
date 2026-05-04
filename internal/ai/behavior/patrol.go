package behavior

import (
	"github.com/example/mayday-server/internal/ai"
	gmath "github.com/example/mayday-server/internal/game/math"
)

// Patrol returns the canonical action sequence for a troop on patrol. The
// MVP implementation is intentionally minimal: stand still and look forward.
// Tests and future patrol-route logic build on top of this primitive.
func Patrol(_ gmath.Vector3) []ai.Action {
	return []ai.Action{ai.Idle()}
}
