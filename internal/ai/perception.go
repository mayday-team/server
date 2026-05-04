package ai

import gmath "github.com/example/mayday-server/internal/game/math"

// PerceptionInput is the minimal subset of the world the AI needs to make a
// decision for one troop. It deliberately does not reference game-package
// state types so the ai package stays a leaf dependency.
type PerceptionInput struct {
	PlayerAlive    bool
	PlayerPosition gmath.Vector3

	DetectionRange float64
	AttackRange    float64
}

// PerceptionResult captures what a troop "sees" at a given moment.
type PerceptionResult struct {
	PlayerVisible bool
	InAttackRange bool
	Distance      float64
	ToPlayer      gmath.Vector3
}

func Perceive(troopPos gmath.Vector3, in PerceptionInput) PerceptionResult {
	res := PerceptionResult{ToPlayer: gmath.Sub(in.PlayerPosition, troopPos)}
	if !in.PlayerAlive {
		return res
	}
	dist := gmath.Length(res.ToPlayer)
	res.Distance = dist
	if dist <= in.DetectionRange {
		res.PlayerVisible = true
	}
	if dist <= in.AttackRange {
		res.InAttackRange = true
	}
	return res
}
