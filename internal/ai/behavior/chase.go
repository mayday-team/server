package behavior

import (
	"github.com/mayday-team/server/internal/ai"
	gmath "github.com/mayday-team/server/internal/game/math"
)

func Chase(target gmath.Vector3) []ai.Action {
	return []ai.Action{ai.MoveTo(target), ai.LookAt(target)}
}
