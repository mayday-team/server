package gmath

import "math"

type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func Add(a, b Vector3) Vector3 { return Vector3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }

func Sub(a, b Vector3) Vector3 { return Vector3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }

func Scale(v Vector3, s float64) Vector3 { return Vector3{v.X * s, v.Y * s, v.Z * s} }

func Dot(a, b Vector3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func Length(v Vector3) float64 { return math.Sqrt(Dot(v, v)) }

func Distance(a, b Vector3) float64 { return Length(Sub(a, b)) }

func Normalize(v Vector3) Vector3 {
	l := Length(v)
	if l == 0 {
		return Vector3{}
	}
	return Scale(v, 1.0/l)
}

func IsZero(v Vector3) bool {
	return v.X == 0 && v.Y == 0 && v.Z == 0
}
