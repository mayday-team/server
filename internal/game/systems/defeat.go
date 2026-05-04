package systems

import (
	"github.com/mayday-team/server/internal/game/scenario"
	"github.com/mayday-team/server/internal/game/state"
)

// CheckPlayerDefeat returns the appropriate DefeatReason for a player whose
// state already implies a terminal session, or DefeatNone otherwise.
//
// This complements scenario.Director.Tick which also tracks scripted
// timing-based defeats; the two are intentionally redundant for safety.
func CheckPlayerDefeat(p *state.CivilianPlayerState) scenario.DefeatReason {
	if p == nil {
		return scenario.DefeatNone
	}
	if !p.IsAlive {
		return scenario.DefeatPlayerKilled
	}
	return scenario.DefeatNone
}

// CheckOverrun returns true if too many troops are within close attack
// range simultaneously. This is the gameplay sense of being overrun and is
// independent of HP being zero.
func CheckOverrun(p *state.CivilianPlayerState, troops map[string]*state.MartialTroopState, closeRange float64, threshold int) bool {
	if p == nil || !p.IsAlive {
		return false
	}
	close := 0
	for _, t := range troops {
		if t == nil || !t.IsAlive {
			continue
		}
		dx := t.Position.X - p.Position.X
		dz := t.Position.Z - p.Position.Z
		if dx*dx+dz*dz <= closeRange*closeRange {
			close++
		}
	}
	return close >= threshold
}
