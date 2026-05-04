package scenario

type DefeatReason string

const (
	DefeatNone               DefeatReason = ""
	DefeatPlayerKilled       DefeatReason = "PLAYER_KILLED"
	DefeatOverrun            DefeatReason = "OVERRUN"
	DefeatAmmoExhausted      DefeatReason = "AMMO_EXHAUSTED"
	DefeatEncircled          DefeatReason = "ENCIRCLED"
	DefeatScriptedFinalStand DefeatReason = "SCRIPTED_FINAL_STAND"
	DefeatDisconnected       DefeatReason = "DISCONNECTED"
)
