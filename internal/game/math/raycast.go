package gmath

type Ray struct {
	Origin    Vector3
	Direction Vector3
}

type RayHit struct {
	TargetID string
	Distance float64
	Dot      float64
}

// CheckRayAgainstPoint returns whether a ray's direction roughly points at a
// target point, the distance from the ray origin to the point, and the cosine
// (dot product) between the ray direction and the unit vector to the target.
//
// The ray direction is normalized internally, so callers may pass an
// unnormalized direction. The target offset is also normalized for the dot.
func CheckRayAgainstPoint(ray Ray, target Vector3, maxDistance, angleThreshold float64) (RayHit, bool) {
	dir := Normalize(ray.Direction)
	if IsZero(dir) {
		return RayHit{}, false
	}
	offset := Sub(target, ray.Origin)
	dist := Length(offset)
	if dist > maxDistance || dist == 0 {
		return RayHit{}, false
	}
	cos := Dot(dir, Normalize(offset))
	if cos < angleThreshold {
		return RayHit{}, false
	}
	return RayHit{Distance: dist, Dot: cos}, true
}

// CheckRayAgainstSphere returns whether a ray intersects a sphere-like target.
// It is useful for human-sized targets where a single center point is too
// strict for a visible body mesh.
func CheckRayAgainstSphere(ray Ray, center Vector3, radius, maxDistance float64) (RayHit, bool) {
	dir := Normalize(ray.Direction)
	if IsZero(dir) || radius <= 0 {
		return RayHit{}, false
	}

	toCenter := Sub(center, ray.Origin)
	along := Dot(toCenter, dir)
	if along < 0 || along > maxDistance {
		return RayHit{}, false
	}

	closest := Add(ray.Origin, Scale(dir, along))
	missDist := Distance(closest, center)
	if missDist > radius {
		return RayHit{}, false
	}

	distToEntry := along - radius
	if distToEntry < 0 {
		distToEntry = along
	}
	return RayHit{Distance: distToEntry, Dot: 1}, true
}
