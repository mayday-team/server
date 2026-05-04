package behavior

import "github.com/mayday-team/server/internal/ai"

func TakeCover() []ai.Action {
	return []ai.Action{ai.TakeCover()}
}
