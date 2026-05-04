package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	gmath "github.com/mayday-team/server/internal/game/math"
	"github.com/mayday-team/server/internal/game/state"
	"github.com/mayday-team/server/internal/game/systems"
)

func basePlayer() *state.CivilianPlayerState {
	return &state.CivilianPlayerState{
		ID: "p1", IsAlive: true,
		HP: 100, MaxHP: 100,
		Ammo: 24, MaxAmmo: 24,
	}
}

func baseTroop(id string, pos gmath.Vector3) *state.MartialTroopState {
	return &state.MartialTroopState{
		ID: id, Position: pos,
		HP: 60, MaxHP: 60,
		Ammo: 30, MaxAmmo: 30,
		IsAlive: true,
	}
}

func shootCfg() systems.ShootConfig {
	return systems.ShootConfig{
		MaxDistance:    60,
		AngleThreshold: 0.96,
		Damage:         25,
		FireRateLimit:  250 * time.Millisecond,
	}
}

func TestPlayerCannotShootWhenDead(t *testing.T) {
	p := basePlayer()
	p.IsAlive = false
	out := systems.ProcessPlayerShoot(p, nil,
		gmath.Vector3{}, gmath.Vector3{Z: 1},
		shootCfg(), time.Now(),
	)
	assert.False(t, out.Accepted)
	assert.Equal(t, "dead", out.Reason)
}

func TestPlayerCannotShootWithoutAmmo(t *testing.T) {
	p := basePlayer()
	p.Ammo = 0
	out := systems.ProcessPlayerShoot(p, nil,
		gmath.Vector3{}, gmath.Vector3{Z: 1},
		shootCfg(), time.Now(),
	)
	assert.False(t, out.Accepted)
	assert.Equal(t, "no_ammo", out.Reason)
}

func TestFireRateLimit(t *testing.T) {
	p := basePlayer()
	now := time.Now()
	p.LastShotAt = now
	out := systems.ProcessPlayerShoot(p, nil,
		gmath.Vector3{}, gmath.Vector3{Z: 1},
		shootCfg(), now.Add(50*time.Millisecond),
	)
	assert.False(t, out.Accepted)
	assert.Equal(t, "fire_rate", out.Reason)
}

func TestShootHit(t *testing.T) {
	p := basePlayer()
	troops := map[string]*state.MartialTroopState{
		"t1": baseTroop("t1", gmath.Vector3{Z: 8}),
	}
	out := systems.ProcessPlayerShoot(p, troops,
		gmath.Vector3{}, gmath.Vector3{Z: 1},
		shootCfg(), time.Now(),
	)
	assert.True(t, out.Accepted)
	assert.Equal(t, "hit", out.Reason)
	assert.Equal(t, "t1", out.HitTroopID)
	assert.Equal(t, 25, out.DamageDealt)
	assert.Equal(t, 23, p.Ammo, "ammo decremented to 23")
	assert.Equal(t, 35, troops["t1"].HP, "troop took 25 damage from 60")
}

func TestShootMissOffAngle(t *testing.T) {
	p := basePlayer()
	troops := map[string]*state.MartialTroopState{
		"t1": baseTroop("t1", gmath.Vector3{X: 10}),
	}
	out := systems.ProcessPlayerShoot(p, troops,
		gmath.Vector3{}, gmath.Vector3{Z: 1},
		shootCfg(), time.Now(),
	)
	assert.True(t, out.Accepted)
	assert.Equal(t, "miss", out.Reason)
	assert.Empty(t, out.HitTroopID)
	assert.Equal(t, 23, p.Ammo, "ammo still consumed on a fired shot")
}

func TestShootKillNeutralizesTroop(t *testing.T) {
	p := basePlayer()
	cfg := shootCfg()
	cfg.Damage = 100
	troops := map[string]*state.MartialTroopState{
		"t1": baseTroop("t1", gmath.Vector3{Z: 5}),
	}
	out := systems.ProcessPlayerShoot(p, troops,
		gmath.Vector3{}, gmath.Vector3{Z: 1},
		cfg, time.Now(),
	)
	assert.True(t, out.Accepted)
	assert.True(t, out.TroopKilled)
	assert.False(t, troops["t1"].IsAlive)
}
