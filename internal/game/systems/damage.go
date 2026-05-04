package systems

import "github.com/mayday-team/server/internal/game/state"

// DamageResult is returned by the damage helpers so callers can emit
// appropriate events.
type DamageResult struct {
	AppliedDamage int
	RemainingHP   int
	Killed        bool
}

// ApplyDamageToPlayer reduces player HP and flags death when HP hits zero.
// Damage is clamped to non-negative; HP is clamped to zero.
func ApplyDamageToPlayer(p *state.CivilianPlayerState, dmg int) DamageResult {
	if p == nil || !p.IsAlive || dmg <= 0 {
		return DamageResult{}
	}
	p.HP -= dmg
	if p.HP <= 0 {
		p.HP = 0
		p.IsAlive = false
		return DamageResult{AppliedDamage: dmg, RemainingHP: 0, Killed: true}
	}
	return DamageResult{AppliedDamage: dmg, RemainingHP: p.HP}
}

// ApplyDamageToTroop reduces troop HP and flags death when HP hits zero.
func ApplyDamageToTroop(t *state.MartialTroopState, dmg int) DamageResult {
	if t == nil || !t.IsAlive || dmg <= 0 {
		return DamageResult{}
	}
	t.HP -= dmg
	if t.HP <= 0 {
		t.HP = 0
		t.IsAlive = false
		return DamageResult{AppliedDamage: dmg, RemainingHP: 0, Killed: true}
	}
	return DamageResult{AppliedDamage: dmg, RemainingHP: t.HP}
}
