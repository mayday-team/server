package scenario

import "math"

// PressureInput captures the variables that drive how much "pressure" the
// session should apply on the player at a given moment. Pressure rises as
// time passes, the player loses HP, ammo runs low, or troops accumulate.
type PressureInput struct {
	ElapsedMs           int64
	FullPressureAfterMs int64
	PlayerHP            int
	PlayerMaxHP         int
	PlayerAmmo          int
	PlayerMaxAmmo       int
	SurvivingTroops     int
	MaxTroops           int
}

// ComputePressure returns a value clamped to [0, 1].
func ComputePressure(in PressureInput) float64 {
	timePart := 0.0
	if in.FullPressureAfterMs > 0 {
		timePart = float64(in.ElapsedMs) / float64(in.FullPressureAfterMs)
	}

	hpPart := 0.0
	if in.PlayerMaxHP > 0 {
		hpPart = 1.0 - float64(in.PlayerHP)/float64(in.PlayerMaxHP)
	}

	ammoPart := 0.0
	if in.PlayerMaxAmmo > 0 {
		ammoPart = 1.0 - float64(in.PlayerAmmo)/float64(in.PlayerMaxAmmo)
	}

	troopPart := 0.0
	if in.MaxTroops > 0 {
		troopPart = float64(in.SurvivingTroops) / float64(in.MaxTroops)
	}

	pressure := 0.5*timePart + 0.2*hpPart + 0.1*ammoPart + 0.2*troopPart
	return clamp01(pressure)
}

// ComputeEncirclement grows mostly with elapsed time, biased late in the
// session. Used to drive flank/block-exit AI behavior.
func ComputeEncirclement(elapsedMs, fullAfterMs int64) float64 {
	if fullAfterMs <= 0 {
		return 0
	}
	t := float64(elapsedMs) / float64(fullAfterMs)
	// quadratic ease-in: encirclement accelerates over time
	return clamp01(t * t)
}

func clamp01(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
