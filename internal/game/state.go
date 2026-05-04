package game

import "github.com/mayday-team/server/internal/game/state"

// Re-exports so the game package can refer to the authoritative state
// types without duplicating definitions. The actual structs live in
// internal/game/state to break a would-be import cycle between this
// package and internal/game/systems.
type (
	CivilianPlayerState = state.CivilianPlayerState
	MartialTroopState   = state.MartialTroopState
)
