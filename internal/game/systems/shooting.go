package systems

import (
	"time"

	gmath "github.com/mayday-team/server/internal/game/math"
	"github.com/mayday-team/server/internal/game/state"
	"github.com/mayday-team/server/internal/protocol"
)

// ShootConfig is the validation parameter set for one fire attempt.
type ShootConfig struct {
	MaxDistance    float64
	AngleThreshold float64
	Damage         int
	FireRateLimit  time.Duration
}

// ShotOutcome describes the result of one server-validated shot.
type ShotOutcome struct {
	Accepted    bool
	Reason      protocol.ShotReason
	HitTroopID  string
	HitDistance float64
	DamageDealt int
	TroopKilled bool
}

// ProcessPlayerShoot validates a player shoot attempt and applies hit
// resolution against the supplied troop set. The function never trusts the
// client's claimed hit; it performs its own raycast.
func ProcessPlayerShoot(
	p *state.CivilianPlayerState,
	troops map[string]*state.MartialTroopState,
	origin, direction gmath.Vector3,
	cfg ShootConfig,
	now time.Time,
) ShotOutcome {
	if p == nil {
		return ShotOutcome{Reason: protocol.ShotReasonNoPlayer}
	}
	if !p.IsAlive {
		return ShotOutcome{Reason: protocol.ShotReasonDead}
	}
	if p.Ammo <= 0 {
		return ShotOutcome{Reason: protocol.ShotReasonNoAmmo}
	}
	if !p.LastShotAt.IsZero() && now.Sub(p.LastShotAt) < cfg.FireRateLimit {
		return ShotOutcome{Reason: protocol.ShotReasonFireRate}
	}
	if gmath.IsZero(direction) {
		return ShotOutcome{Reason: protocol.ShotReasonBadDirection}
	}

	p.Ammo--
	p.LastShotAt = now

	ray := gmath.Ray{Origin: origin, Direction: direction}

	var (
		hitID   string
		hitDist = cfg.MaxDistance + 1
	)
	for id, t := range troops {
		if t == nil || !t.IsAlive {
			continue
		}
		target := t.Position
		target.Y += 1.2
		hit, ok := gmath.CheckRayAgainstPoint(ray, target, cfg.MaxDistance, cfg.AngleThreshold)
		if !ok {
			continue
		}
		if hit.Distance < hitDist {
			hitID = id
			hitDist = hit.Distance
		}
	}

	if hitID == "" {
		return ShotOutcome{Accepted: true, Reason: protocol.ShotReasonMiss}
	}

	t := troops[hitID]
	dmgRes := ApplyDamageToTroop(t, cfg.Damage)

	return ShotOutcome{
		Accepted:    true,
		Reason:      protocol.ShotReasonHit,
		HitTroopID:  hitID,
		HitDistance: hitDist,
		DamageDealt: dmgRes.AppliedDamage,
		TroopKilled: dmgRes.Killed,
	}
}

// TroopShootAttempt is the server-side equivalent for AI-driven shots. It
// applies damage to the player using the same fire-rate gating used for
// players. Returns the damage actually applied.
func TroopShootAttempt(
	t *state.MartialTroopState,
	p *state.CivilianPlayerState,
	damage int,
	fireRate time.Duration,
	now time.Time,
) (DamageResult, bool) {
	if t == nil || p == nil || !t.IsAlive || !p.IsAlive {
		return DamageResult{}, false
	}
	if t.Ammo <= 0 {
		return DamageResult{}, false
	}
	if !t.LastShotAt.IsZero() && now.Sub(t.LastShotAt) < fireRate {
		return DamageResult{}, false
	}
	t.Ammo--
	t.LastShotAt = now
	res := ApplyDamageToPlayer(p, damage)
	return res, true
}
