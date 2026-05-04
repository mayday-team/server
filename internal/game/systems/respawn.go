package systems

import "github.com/mayday-team/server/internal/game/state"

// Respawning the civilian player is intentionally not part of the gameplay
// loop: Mayday's design constraint is that defeat is permanent. This file
// exists only to make the design choice explicit and to host any limited
// resource-replenishment helpers (ammo pickups, medic interactions) that
// later phases may add.
//
// Until that future work lands, the only operation here is a no-op intended
// to be the single seam through which any "respawn-like" behavior must pass.
func DenyRespawn(_ *state.CivilianPlayerState) {}
