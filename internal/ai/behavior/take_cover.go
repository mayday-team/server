package behavior

import "github.com/example/mayday-server/internal/ai"

func TakeCover() []ai.Action {
	return []ai.Action{ai.TakeCover()}
}
