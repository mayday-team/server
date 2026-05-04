package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayday-team/server/internal/ai"
	gmath "github.com/mayday-team/server/internal/game/math"
	"github.com/mayday-team/server/internal/game/scenario"
)

func newTroopAt(pos gmath.Vector3) ai.TroopSnapshot {
	return ai.TroopSnapshot{
		ID: "troop-1", Position: pos,
		HP: 60, MaxHP: 60, Ammo: 30,
		IsAlive: true,
		State:   ai.StatePatrol,
	}
}

func TestTroopAttacksWhenPlayerInRange(t *testing.T) {
	troopPos := gmath.Vector3{}
	playerPos := gmath.Vector3{Z: 5}
	perc := ai.Perceive(troopPos, ai.PerceptionInput{
		PlayerAlive:    true,
		PlayerPosition: playerPos,
		DetectionRange: 35,
		AttackRange:    22,
	})
	dec := ai.Decide(ai.DecisionInput{
		Troop:      newTroopAt(troopPos),
		Perception: perc,
		Phase:      scenario.PhaseInitialContact,
		Pressure:   0.1,
		MaxTroops:  10,
	})
	assert.Equal(t, ai.StateAttack, dec.NextState)
	var sawShoot bool
	for _, a := range dec.Actions {
		if a.Kind == ai.ActionShoot {
			sawShoot = true
		}
	}
	assert.True(t, sawShoot, "expected an SHOOT action when player is in range")
}

func TestTroopChasesWhenPlayerVisibleButOutOfRange(t *testing.T) {
	troopPos := gmath.Vector3{}
	playerPos := gmath.Vector3{Z: 30} // beyond 22 attack range, within 35 detection
	perc := ai.Perceive(troopPos, ai.PerceptionInput{
		PlayerAlive:    true,
		PlayerPosition: playerPos,
		DetectionRange: 35,
		AttackRange:    22,
	})
	dec := ai.Decide(ai.DecisionInput{
		Troop:      newTroopAt(troopPos),
		Perception: perc,
		Phase:      scenario.PhaseInitialContact,
		Pressure:   0.1,
		MaxTroops:  10,
	})
	assert.Equal(t, ai.StateChase, dec.NextState)
}

func TestTroopFlanksDuringEncirclement(t *testing.T) {
	troopPos := gmath.Vector3{}
	playerPos := gmath.Vector3{Z: 30}
	perc := ai.Perceive(troopPos, ai.PerceptionInput{
		PlayerAlive:    true,
		PlayerPosition: playerPos,
		DetectionRange: 35,
		AttackRange:    22,
	})
	dec := ai.Decide(ai.DecisionInput{
		Troop:         newTroopAt(troopPos),
		Perception:    perc,
		Phase:         scenario.PhaseEncirclement,
		Pressure:      0.7,
		Encirclement:  0.7,
		EscapeBlocked: false,
		MaxTroops:     10,
	})
	assert.Equal(t, ai.StateFlank, dec.NextState)
}

func TestTroopCallsReinforcementWhenLow(t *testing.T) {
	troopPos := gmath.Vector3{}
	playerPos := gmath.Vector3{Z: 5}
	perc := ai.Perceive(troopPos, ai.PerceptionInput{
		PlayerAlive:    true,
		PlayerPosition: playerPos,
		DetectionRange: 35,
		AttackRange:    22,
	})
	dec := ai.Decide(ai.DecisionInput{
		Troop:      newTroopAt(troopPos),
		Perception: perc,
		Phase:      scenario.PhaseReinforcement,
		Pressure:   0.5,
		TroopCount: 1,
		MinTroops:  3,
		MaxTroops:  10,
	})
	assert.Equal(t, ai.StateCallReinforcement, dec.NextState)
}

func TestDeadTroopGoesDead(t *testing.T) {
	dec := ai.Decide(ai.DecisionInput{
		Troop: ai.TroopSnapshot{
			ID: "t-dead", IsAlive: false, HP: 0, MaxHP: 60,
		},
		Perception: ai.PerceptionResult{},
	})
	assert.Equal(t, ai.StateDead, dec.NextState)
}
