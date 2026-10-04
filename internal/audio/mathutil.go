package audio

import "math"

// ===== Internal =====

func clamp(value, minimum, maximum float32) float32 {
	return max(minimum, min(maximum, value))
}

func clampInt(value, minimum, maximum int) int {
	return max(minimum, min(maximum, value))
}

func abs(value float32) float32 {
	return max(value, -value)
}

func sine(angle float32) float32 {
	return float32(math.Sin(float64(angle)))
}

func boolToIndex(condition bool) int {
	if condition {
		return 1
	}
	return 0
}

func boolToFloat(condition bool) float32 {
	return float32(boolToIndex(condition))
}
