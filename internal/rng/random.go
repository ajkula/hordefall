package rng

import "math"

// ===== Types =====

type Random struct {
	state uint32
}

// ===== Public API =====

func New(seed uint32) Random {
	return Random{state: seed | 1}
}

func (r *Random) Next() uint32 {
	state := r.state
	state ^= state << 13
	state ^= state >> 17
	state ^= state << 5
	r.state = state
	return state
}

func (r *Random) Float() float32 {
	return float32(r.Next()>>8) / (1 << 24)
}

func (r *Random) Between(minimum, maximum float32) float32 {
	return minimum + (maximum-minimum)*r.Float()
}

func (r *Random) Below(limit int) int {
	return int(r.Next() % uint32(limit))
}

func (r *Random) Chance(probability float32) bool {
	return r.Float() < probability
}

func (r *Random) Angle() float32 {
	return r.Float() * 2 * math.Pi
}
