package systems

import "github.com/mayday-team/server/internal/game/state"

// SessionStats summarizes the things we record per session for both the
// final session_ended event and the persisted game_sessions row.
type SessionStats struct {
	ShotsFired        int
	ShotsHit          int
	DamageTaken       int
	TroopsNeutralized int
	EventsRecorded    int
	SurvivedMs        int64
}

// AccumulateSurvival increments the player's survived time on each tick
// while the player is alive.
func AccumulateSurvival(p *state.CivilianPlayerState, deltaMs int64) {
	if p == nil || !p.IsAlive {
		return
	}
	if deltaMs <= 0 {
		return
	}
	p.SurvivalTimeMs += deltaMs
}
