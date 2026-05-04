package scenario

import "time"

// Config captures the timing parameters that govern phase progression and
// forced defeat. All values are absolute durations from session start.
type Config struct {
	FinalStandAfter  time.Duration
	ForceDefeatAfter time.Duration
	MaxTroops        int
}

// Input is the per-tick context the director needs to make its decisions.
type Input struct {
	Now                 time.Time
	PlayerHP            int
	PlayerMaxHP         int
	PlayerAmmo          int
	PlayerMaxAmmo       int
	PlayerAlive         bool
	SurvivingTroopCount int
}

// Update is the per-tick result returned by Director.Tick. It tells the
// session what changed, so it can emit the appropriate events to the client.
type Update struct {
	PreviousPhase     Phase
	CurrentPhase      Phase
	PhaseChanged      bool
	PressureChanged   bool
	PressureLevel     float64
	EncirclementLevel float64
	TriggeredDefeat   bool
	DefeatReason      DefeatReason
}

// Director owns scenario-level state. It is not safe for concurrent use; it
// is expected to live inside a single session goroutine.
type Director struct {
	sessionStartedAt      time.Time
	phaseStartedAt        time.Time
	currentPhase          Phase
	pressureLevel         float64
	encirclementLevel     float64
	reinforcementLevel    int
	escapeBlocked         bool
	forcedDefeatTriggered bool
	defeatReason          DefeatReason
	cfg                   Config
}

func NewDirector(now time.Time, cfg Config) *Director {
	return &Director{
		sessionStartedAt: now,
		phaseStartedAt:   now,
		currentPhase:     PhaseInitialContact,
		cfg:              cfg,
	}
}

func (d *Director) CurrentPhase() Phase            { return d.currentPhase }
func (d *Director) PressureLevel() float64         { return d.pressureLevel }
func (d *Director) EncirclementLevel() float64     { return d.encirclementLevel }
func (d *Director) ReinforcementLevel() int        { return d.reinforcementLevel }
func (d *Director) EscapeBlocked() bool            { return d.escapeBlocked }
func (d *Director) ForcedDefeatTriggered() bool    { return d.forcedDefeatTriggered }
func (d *Director) DefeatReason() DefeatReason     { return d.defeatReason }
func (d *Director) SessionStartedAt() time.Time    { return d.sessionStartedAt }
func (d *Director) PhaseStartedAt() time.Time      { return d.phaseStartedAt }

// Tick advances the director state for a single simulation tick.
func (d *Director) Tick(in Input) Update {
	prevPhase := d.currentPhase
	prevPressure := d.pressureLevel

	elapsed := in.Now.Sub(d.sessionStartedAt)
	elapsedMs := elapsed.Milliseconds()
	finalStandMs := d.cfg.FinalStandAfter.Milliseconds()
	forceDefeatMs := d.cfg.ForceDefeatAfter.Milliseconds()

	d.pressureLevel = ComputePressure(PressureInput{
		ElapsedMs:           elapsedMs,
		FullPressureAfterMs: finalStandMs,
		PlayerHP:            in.PlayerHP,
		PlayerMaxHP:         in.PlayerMaxHP,
		PlayerAmmo:          in.PlayerAmmo,
		PlayerMaxAmmo:       in.PlayerMaxAmmo,
		SurvivingTroops:     in.SurvivingTroopCount,
		MaxTroops:           d.cfg.MaxTroops,
	})
	d.encirclementLevel = ComputeEncirclement(elapsedMs, forceDefeatMs)

	d.advancePhase(in, elapsedMs, finalStandMs, forceDefeatMs)

	upd := Update{
		PreviousPhase:     prevPhase,
		CurrentPhase:      d.currentPhase,
		PhaseChanged:      prevPhase != d.currentPhase,
		PressureChanged:   significantPressureChange(prevPressure, d.pressureLevel),
		PressureLevel:     d.pressureLevel,
		EncirclementLevel: d.encirclementLevel,
	}

	if !d.forcedDefeatTriggered {
		if reason, ok := d.checkDefeatTriggers(in, elapsedMs, forceDefeatMs); ok {
			d.forcedDefeatTriggered = true
			d.defeatReason = reason
			d.transitionTo(PhaseDefeat, in.Now)
			upd.CurrentPhase = d.currentPhase
			upd.PhaseChanged = upd.PreviousPhase != d.currentPhase
			upd.TriggeredDefeat = true
			upd.DefeatReason = reason
		}
	}

	return upd
}

// MarkDisconnected forces the session into DEFEAT due to a transport-level
// disconnect. Idempotent.
func (d *Director) MarkDisconnected(now time.Time) Update {
	if d.forcedDefeatTriggered {
		return Update{PreviousPhase: d.currentPhase, CurrentPhase: d.currentPhase}
	}
	prev := d.currentPhase
	d.forcedDefeatTriggered = true
	d.defeatReason = DefeatDisconnected
	d.transitionTo(PhaseDefeat, now)
	return Update{
		PreviousPhase:   prev,
		CurrentPhase:    d.currentPhase,
		PhaseChanged:    prev != d.currentPhase,
		TriggeredDefeat: true,
		DefeatReason:    DefeatDisconnected,
	}
}

func (d *Director) advancePhase(in Input, elapsedMs, finalStandMs, _ int64) {
	if d.currentPhase == PhaseDefeat {
		return
	}

	switch d.currentPhase {
	case PhaseInitialContact:
		if elapsedMs > finalStandMs/8 || d.pressureLevel >= 0.20 {
			d.transitionTo(PhaseEscalation, in.Now)
		}
	case PhaseEscalation:
		if elapsedMs > finalStandMs/3 || d.pressureLevel >= 0.40 {
			d.reinforcementLevel = 1
			d.transitionTo(PhaseReinforcement, in.Now)
		}
	case PhaseReinforcement:
		if elapsedMs > (2*finalStandMs)/3 || d.pressureLevel >= 0.60 {
			d.escapeBlocked = true
			d.reinforcementLevel = 2
			d.transitionTo(PhaseEncirclement, in.Now)
		}
	case PhaseEncirclement:
		if elapsedMs >= finalStandMs || d.pressureLevel >= 0.80 {
			d.reinforcementLevel = 3
			d.transitionTo(PhaseFinalStand, in.Now)
		}
	}
}

func (d *Director) transitionTo(p Phase, now time.Time) {
	if d.currentPhase == p {
		return
	}
	d.currentPhase = p
	d.phaseStartedAt = now
}

func (d *Director) checkDefeatTriggers(in Input, elapsedMs, forceDefeatMs int64) (DefeatReason, bool) {
	if !in.PlayerAlive {
		return DefeatPlayerKilled, true
	}
	if forceDefeatMs > 0 && elapsedMs >= forceDefeatMs {
		return DefeatScriptedFinalStand, true
	}
	if d.currentPhase == PhaseFinalStand && in.PlayerAmmo == 0 && in.SurvivingTroopCount > 0 {
		return DefeatAmmoExhausted, true
	}
	if d.currentPhase == PhaseFinalStand && d.encirclementLevel >= 0.95 {
		return DefeatEncircled, true
	}
	return DefeatNone, false
}

func significantPressureChange(prev, next float64) bool {
	diff := next - prev
	if diff < 0 {
		diff = -diff
	}
	return diff >= 0.05
}
