package game

// Server-side gameplay constants that don't live in env config because they
// either define the data shape or describe non-tunable invariants.
const (
	MinTroopFloor     = 2
	StartingTroopAmmo = 30
	StartingTroopHP   = 60
	TroopFireRateMs   = 600
	PlayerStartY      = 1.6
)
