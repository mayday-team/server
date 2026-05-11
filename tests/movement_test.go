package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	gmath "github.com/mayday-team/server/internal/game/math"
	"github.com/mayday-team/server/internal/game/state"
	"github.com/mayday-team/server/internal/game/systems"
)

func TestPlayerMovementMatchesClientLookDirection(t *testing.T) {
	player := &state.CivilianPlayerState{
		IsAlive:  true,
		Position: gmath.Vector3{Y: 7, Z: -36},
		Yaw:      0,
	}

	systems.ApplyPlayerMovement(player, systems.MovementInput{Forward: true}, 100, 10)
	assert.InDelta(t, -35.0, player.Position.Z, 0.001)

	player.Position = gmath.Vector3{Y: 7, Z: -36}
	systems.ApplyPlayerMovement(player, systems.MovementInput{Right: true}, 100, 10)
	assert.InDelta(t, -1.0, player.Position.X, 0.001)

	player.Position = gmath.Vector3{Y: 7, Z: -36}
	systems.ApplyPlayerMovement(player, systems.MovementInput{Left: true}, 100, 10)
	assert.InDelta(t, 1.0, player.Position.X, 0.001)
}
