package behavior

import (
	"github.com/example/mayday-server/internal/ai"
	gmath "github.com/example/mayday-server/internal/game/math"
)

func Advance(target gmath.Vector3) []ai.Action {
	return []ai.Action{ai.MoveTo(target), ai.LookAt(target)}
}
