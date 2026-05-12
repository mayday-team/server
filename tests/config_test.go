package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayday-team/server/internal/config"
	"github.com/mayday-team/server/internal/game"
)

func TestDefaultPlayerShotKillsStartingTroop(t *testing.T) {
	cfg := config.Load()

	assert.GreaterOrEqual(t, cfg.PlayerShootDamage, game.StartingTroopHP)
}
