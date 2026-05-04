package ai

import gmath "github.com/example/mayday-server/internal/game/math"

// Controller is a thin façade that bundles perception + decision for one
// troop. The game session calls Update once per troop per tick and applies
// the resulting actions through its game systems.
type Controller struct{}

func NewController() *Controller { return &Controller{} }

// Update runs perception and decision for a single troop. It is a pure
// function on its inputs and does not mutate any state.
func (c *Controller) Update(troopPos gmath.Vector3, in DecisionInput, p PerceptionInput) (PerceptionResult, Decision) {
	in.Perception = Perceive(troopPos, p)
	dec := Decide(in)
	return in.Perception, dec
}
