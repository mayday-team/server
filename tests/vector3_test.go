package tests

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	gmath "github.com/mayday-team/server/internal/game/math"
)

func TestVector3Operations(t *testing.T) {
	a := gmath.Vector3{X: 1, Y: 2, Z: 3}
	b := gmath.Vector3{X: 4, Y: 5, Z: 6}

	assert.Equal(t, gmath.Vector3{X: 5, Y: 7, Z: 9}, gmath.Add(a, b))
	assert.Equal(t, gmath.Vector3{X: -3, Y: -3, Z: -3}, gmath.Sub(a, b))
	assert.InDelta(t, 32.0, gmath.Dot(a, b), 1e-9)
	assert.InDelta(t, math.Sqrt(27), gmath.Distance(a, b), 1e-9)

	unit := gmath.Normalize(gmath.Vector3{X: 0, Y: 0, Z: 5})
	assert.InDelta(t, 1.0, gmath.Length(unit), 1e-9)
	assert.InDelta(t, 0.0, unit.X, 1e-9)
	assert.InDelta(t, 1.0, unit.Z, 1e-9)

	assert.True(t, gmath.IsZero(gmath.Vector3{}))
	assert.False(t, gmath.IsZero(a))

	// Normalize of zero vector returns zero.
	assert.Equal(t, gmath.Vector3{}, gmath.Normalize(gmath.Vector3{}))
}

func TestRaycastHit(t *testing.T) {
	ray := gmath.Ray{
		Origin:    gmath.Vector3{X: 0, Y: 0, Z: 0},
		Direction: gmath.Vector3{X: 0, Y: 0, Z: 1},
	}
	// Target dead ahead at z=10 should hit easily.
	hit, ok := gmath.CheckRayAgainstPoint(ray, gmath.Vector3{Z: 10}, 60, 0.96)
	assert.True(t, ok)
	assert.InDelta(t, 10.0, hit.Distance, 1e-9)
	assert.GreaterOrEqual(t, hit.Dot, 0.96)
}

func TestRaycastMissOutOfRange(t *testing.T) {
	ray := gmath.Ray{
		Origin:    gmath.Vector3{X: 0, Y: 0, Z: 0},
		Direction: gmath.Vector3{X: 0, Y: 0, Z: 1},
	}
	_, ok := gmath.CheckRayAgainstPoint(ray, gmath.Vector3{Z: 100}, 60, 0.96)
	assert.False(t, ok, "target beyond max distance should miss")
}

func TestRaycastMissOffAngle(t *testing.T) {
	ray := gmath.Ray{
		Origin:    gmath.Vector3{X: 0, Y: 0, Z: 0},
		Direction: gmath.Vector3{X: 0, Y: 0, Z: 1},
	}
	// Target perpendicular to ray direction should fail the angle test.
	_, ok := gmath.CheckRayAgainstPoint(ray, gmath.Vector3{X: 10}, 60, 0.96)
	assert.False(t, ok)
}

func TestRaycastZeroDirection(t *testing.T) {
	ray := gmath.Ray{Direction: gmath.Vector3{}}
	_, ok := gmath.CheckRayAgainstPoint(ray, gmath.Vector3{Z: 5}, 60, 0.96)
	assert.False(t, ok)
}
