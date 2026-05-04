package behavior

import (
	"github.com/mayday-team/server/internal/ai"
	gmath "github.com/mayday-team/server/internal/game/math"
)

func Flank(target, lookAt gmath.Vector3) []ai.Action {
	return []ai.Action{ai.FlankTo(target), ai.LookAt(lookAt)}
}
