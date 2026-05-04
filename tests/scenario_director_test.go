package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mayday-team/server/internal/game/scenario"
)

func TestDirectorPhaseProgressionOverTime(t *testing.T) {
	now := time.Unix(0, 0)
	cfg := scenario.Config{
		FinalStandAfter:  30 * time.Second,
		ForceDefeatAfter: 60 * time.Second,
		MaxTroops:        10,
	}
	d := scenario.NewDirector(now, cfg)
	assert.Equal(t, scenario.PhaseInitialContact, d.CurrentPhase())

	step := func(seconds int) scenario.Update {
		now = now.Add(time.Duration(seconds) * time.Second)
		return d.Tick(scenario.Input{
			Now:                 now,
			PlayerHP:            100,
			PlayerMaxHP:         100,
			PlayerAmmo:          24,
			PlayerMaxAmmo:       24,
			PlayerAlive:         true,
			SurvivingTroopCount: 4,
		})
	}

	step(5)
	assert.Equal(t, scenario.PhaseEscalation, d.CurrentPhase())

	step(8)
	assert.Equal(t, scenario.PhaseReinforcement, d.CurrentPhase())

	step(12)
	assert.Equal(t, scenario.PhaseEncirclement, d.CurrentPhase())
	assert.True(t, d.EscapeBlocked())

	upd := step(10)
	assert.True(t,
		d.CurrentPhase() == scenario.PhaseFinalStand || d.CurrentPhase() == scenario.PhaseDefeat,
		"phase should be FINAL_STAND or DEFEAT after long elapsed time, got %s", d.CurrentPhase())
	_ = upd
}

func TestDirectorTriggersDefeatOnPlayerDeath(t *testing.T) {
	now := time.Unix(0, 0)
	d := scenario.NewDirector(now, scenario.Config{
		FinalStandAfter:  30 * time.Second,
		ForceDefeatAfter: 60 * time.Second,
		MaxTroops:        10,
	})
	upd := d.Tick(scenario.Input{
		Now:                 now.Add(1 * time.Second),
		PlayerHP:            0,
		PlayerMaxHP:         100,
		PlayerAmmo:          0,
		PlayerMaxAmmo:       24,
		PlayerAlive:         false,
		SurvivingTroopCount: 4,
	})
	assert.True(t, upd.TriggeredDefeat)
	assert.Equal(t, scenario.DefeatPlayerKilled, upd.DefeatReason)
	assert.Equal(t, scenario.PhaseDefeat, d.CurrentPhase())
}

func TestDirectorEventuallyForcesDefeat(t *testing.T) {
	now := time.Unix(0, 0)
	d := scenario.NewDirector(now, scenario.Config{
		FinalStandAfter:  30 * time.Second,
		ForceDefeatAfter: 60 * time.Second,
		MaxTroops:        10,
	})
	for elapsed := 1; elapsed <= 90; elapsed++ {
		now = now.Add(1 * time.Second)
		upd := d.Tick(scenario.Input{
			Now:                 now,
			PlayerHP:            100,
			PlayerMaxHP:         100,
			PlayerAmmo:          24,
			PlayerMaxAmmo:       24,
			PlayerAlive:         true,
			SurvivingTroopCount: 4,
		})
		if upd.TriggeredDefeat {
			assert.Contains(t,
				[]scenario.DefeatReason{scenario.DefeatScriptedFinalStand, scenario.DefeatEncircled, scenario.DefeatAmmoExhausted},
				upd.DefeatReason)
			assert.Equal(t, scenario.PhaseDefeat, d.CurrentPhase())
			return
		}
	}
	t.Fatal("director never triggered defeat after force_defeat_after")
}

func TestDirectorMarkDisconnected(t *testing.T) {
	now := time.Unix(0, 0)
	d := scenario.NewDirector(now, scenario.Config{
		FinalStandAfter:  30 * time.Second,
		ForceDefeatAfter: 60 * time.Second,
		MaxTroops:        10,
	})
	upd := d.MarkDisconnected(now.Add(time.Second))
	assert.True(t, upd.TriggeredDefeat)
	assert.Equal(t, scenario.DefeatDisconnected, upd.DefeatReason)
	assert.Equal(t, scenario.PhaseDefeat, d.CurrentPhase())

	// Idempotent: a second call should not flip back state.
	upd2 := d.MarkDisconnected(now.Add(2 * time.Second))
	assert.False(t, upd2.TriggeredDefeat)
}
