package ai

type FSMState string

const (
	StatePatrol            FSMState = "PATROL"
	StateAdvance           FSMState = "ADVANCE"
	StateSuppress          FSMState = "SUPPRESS"
	StateFlank             FSMState = "FLANK"
	StateChase             FSMState = "CHASE"
	StateAttack            FSMState = "ATTACK"
	StateTakeCover         FSMState = "TAKE_COVER"
	StateCallReinforcement FSMState = "CALL_REINFORCEMENT"
	StateBlockExit         FSMState = "BLOCK_EXIT"
	StateDead              FSMState = "DEAD"
)
