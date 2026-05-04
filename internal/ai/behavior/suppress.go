package behavior

import (
	"github.com/example/mayday-server/internal/ai"
	gmath "github.com/example/mayday-server/internal/game/math"
)

func Suppress(target gmath.Vector3) []ai.Action {
	return []ai.Action{ai.LookAt(target), ai.Shoot(target), ai.SuppressArea(target)}
}
