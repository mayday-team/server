package scenario

type Phase string

const (
	PhaseInitialContact Phase = "INITIAL_CONTACT"
	PhaseEscalation     Phase = "ESCALATION"
	PhaseReinforcement  Phase = "REINFORCEMENT"
	PhaseEncirclement   Phase = "ENCIRCLEMENT"
	PhaseFinalStand     Phase = "FINAL_STAND"
	PhaseDefeat         Phase = "DEFEAT"
)

func (p Phase) IsTerminal() bool { return p == PhaseDefeat }

// Order returns a numeric ordering useful for monotonic phase progression
// checks. Higher values represent later phases.
func (p Phase) Order() int {
	switch p {
	case PhaseInitialContact:
		return 0
	case PhaseEscalation:
		return 1
	case PhaseReinforcement:
		return 2
	case PhaseEncirclement:
		return 3
	case PhaseFinalStand:
		return 4
	case PhaseDefeat:
		return 5
	default:
		return -1
	}
}
