package behavior

import (
	"github.com/mayday-team/server/internal/ai"
	gmath "github.com/mayday-team/server/internal/game/math"
)

func BlockExit(point, lookAt gmath.Vector3) []ai.Action {
	return []ai.Action{ai.BlockExit(point), ai.LookAt(lookAt)}
}
