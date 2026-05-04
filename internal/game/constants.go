package game

// Server-side gameplay constants that don't live in env config because they
// either define the data shape or describe non-tunable invariants.
const (
	OverrunCloseRange   = 4.0
	OverrunCloseCount   = 4
	MinTroopFloor       = 2
	StartingTroopAmmo   = 30
	StartingTroopHP     = 60
	TroopFireRateMs     = 600
	SnapshotEventLimit  = 8
	PlayerStartY        = 1.6
	WriteDeadlineMs     = 2000
)
