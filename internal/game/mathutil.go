package game

import "math"

// ===== Types =====

type AngularServo struct {
	Angle    float32
	Velocity float32
}

// ===== Public API =====

func length(x, y float32) float32 {
	return float32(math.Sqrt(float64(x*x + y*y)))
}

func normalize(x, y float32) (float32, float32) {
	magnitude := length(x, y)
	if magnitude < 1e-6 {
		return 0, 0
	}
	return x / magnitude, y / magnitude
}

func distanceSquared(ax, ay, bx, by float32) float32 {
	deltaX, deltaY := ax-bx, ay-by
	return deltaX*deltaX + deltaY*deltaY
}

func clamp(value, minimum, maximum float32) float32 {
	return max(minimum, min(maximum, value))
}

func abs(value float32) float32 {
	return max(value, -value)
}

func clampInt(value, minimum, maximum int) int {
	return max(minimum, min(maximum, value))
}

func cosine(angle float32) float32 {
	return float32(math.Cos(float64(angle)))
}

func sine(angle float32) float32 {
	return float32(math.Sin(float64(angle)))
}

func sqrt(value float32) float32 {
	return float32(math.Sqrt(float64(value)))
}

func angleDifference(from, to float32) float32 {
	return float32(math.Remainder(float64(to-from), 2*math.Pi))
}

func (s *AngularServo) Drive(target, stiffness, damping, deltaSeconds float32) {
	s.Velocity += (angleDifference(s.Angle, target)*stiffness - s.Velocity*damping) * deltaSeconds
	s.Angle += s.Velocity * deltaSeconds
}

func rotateOffset(forward, side, angle float32) (float32, float32) {
	cosineAngle, sineAngle := cosine(angle), sine(angle)
	return forward*cosineAngle - side*sineAngle, forward*sineAngle + side*cosineAngle
}

func atan2(y, x float32) float32 {
	return float32(math.Atan2(float64(y), float64(x)))
}

func boolToFloat(value bool) float32 {
	if value {
		return 1
	}
	return 0
}

func boolToIndex(value bool) int {
	if value {
		return 1
	}
	return 0
}
