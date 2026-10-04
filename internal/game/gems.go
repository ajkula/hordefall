package game

import "hordefall/internal/rng"

// ===== Types =====

type GemStore struct {
	Count       int
	X           []float32
	Y           []float32
	Value       []int32
	IsAttracted []bool
	random      rng.Random
}

type GemTier struct {
	MinimumValue int32
	Color        [3]float32
	Radius       float32
}

// ===== Constants =====

const (
	maximumGems      = 4000
	gemAttractSpeed  = 620
	gemCollectRadius = 18
)

var gemTiers = []GemTier{
	{1, [3]float32{0.35, 0.65, 1}, 4},
	{3, [3]float32{0.35, 1, 0.55}, 5},
	{8, [3]float32{0.85, 0.45, 1}, 6.5},
	{25, [3]float32{1, 0.85, 0.3}, 8},
}

// ===== Public API =====

func NewGemStore(capacity int) *GemStore {
	return &GemStore{
		X: make([]float32, capacity), Y: make([]float32, capacity),
		Value: make([]int32, capacity), IsAttracted: make([]bool, capacity),
		random: rng.New(0xC0FFEE),
	}
}

func (s *GemStore) Drop(x, y float32, value int) {
	if s.Count >= len(s.X) {
		s.Value[s.random.Below(s.Count)] += int32(value)
		return
	}
	index := s.Count
	s.Count++
	s.X[index], s.Y[index], s.Value[index], s.IsAttracted[index] = x, y, int32(value), false
}

func (s *GemStore) Collect(playerX, playerY, magnetRadius, deltaSeconds float32) int {
	collected := 0
	for index := s.Count - 1; index >= 0; index-- {
		collected += s.updateGem(index, playerX, playerY, magnetRadius, deltaSeconds)
	}
	return collected
}

func (s *GemStore) AttractAll() {
	for index := range s.Count {
		s.IsAttracted[index] = true
	}
}

func GemTierFor(value int32) *GemTier {
	tier := &gemTiers[0]
	for index := range gemTiers {
		tier = selectTier(tier, &gemTiers[index], value)
	}
	return tier
}

// ===== Internal =====

func selectTier(current, candidate *GemTier, value int32) *GemTier {
	if value >= candidate.MinimumValue {
		return candidate
	}
	return current
}

func (s *GemStore) updateGem(index int, playerX, playerY, magnetRadius, deltaSeconds float32) int {
	distance := length(playerX-s.X[index], playerY-s.Y[index])
	s.IsAttracted[index] = s.IsAttracted[index] || distance < magnetRadius
	directionX, directionY := normalize(playerX-s.X[index], playerY-s.Y[index])
	step := gemAttractSpeed * deltaSeconds * boolToFloat(s.IsAttracted[index])
	s.X[index] += directionX * step
	s.Y[index] += directionY * step
	if distance > gemCollectRadius {
		return 0
	}
	value := int(s.Value[index])
	s.remove(index)
	return value
}

func (s *GemStore) remove(index int) {
	last := s.Count - 1
	s.X[index], s.Y[index] = s.X[last], s.Y[last]
	s.Value[index], s.IsAttracted[index] = s.Value[last], s.IsAttracted[last]
	s.Count--
}
