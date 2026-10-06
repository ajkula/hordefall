package game

import (
	"math/bits"

	"hordefall/internal/audio"
)

// ===== Types =====

type EnemyKind uint8

type EnemyDefinition struct {
	Name             string
	Color            [3]float32
	Radius           float32
	Speed            float32
	Health           float32
	ContactDamage    float32
	Experience       int
	UnlockSeconds    float32
	SpawnWeight      float32
	Immunities       StatusFlags
	DeathElement     Element
	DeathRadius      float32
	DeathDamage      float32
	DeathStimulus    GroundStimulus
	DeathGroundCells int
	IsHeavy          bool
	DeathSound       audio.SoundKind
	IsCustomDrawn    bool
}

type EnemyStore struct {
	Count         int
	nextID        uint32
	ID            []uint32
	PositionX     []float32
	PositionY     []float32
	KnockbackX    []float32
	KnockbackY    []float32
	Health        []float32
	Power         []float32
	LastHitByBoss []bool
	IsHordeEvent  []bool
	Kind          []EnemyKind
	Status        []StatusFlags
	StatusTimers  [statusCount][]float32
	StatusLevels  [statusCount][]uint8
	HitFlash      []float32
	BladeCooldown []float32
}

// ===== Constants =====

const (
	EnemySwarmer EnemyKind = iota
	EnemyRunner
	EnemyBloater
	EnemyBrute
	EnemyFrostling
	EnemyEmberling
	EnemySpiderTank
	EnemyDrillWorm
	EnemyDrillWormSegment
	enemyKindCount
)

const maximumEnemies = 12000

var enemyTable = [enemyKindCount]EnemyDefinition{
	EnemySwarmer: {
		Name: "Swarmer", Color: [3]float32{0.85, 0.25, 0.3}, Radius: 8, Speed: 72, Health: 10,
		ContactDamage: 6, Experience: 1, SpawnWeight: 10, DeathSound: audio.SoundPop,
	},
	EnemyRunner: {
		Name: "Runner", Color: [3]float32{0.95, 0.75, 0.3}, Radius: 6, Speed: 125, Health: 7,
		ContactDamage: 5, Experience: 1, UnlockSeconds: 50, SpawnWeight: 5, DeathSound: audio.SoundPop,
	},
	EnemyBloater: {
		Name: "Bloater", Color: [3]float32{0.45, 0.3, 0.55}, Radius: 13, Speed: 48, Health: 32,
		ContactDamage: 10, Experience: 3, UnlockSeconds: 35, SpawnWeight: 2.5, DeathSound: audio.SoundSplash,
		DeathElement: ElementOil, DeathRadius: 60, DeathStimulus: StimulusOil, DeathGroundCells: 2,
	},
	EnemyBrute: {
		Name: "Brute", Color: [3]float32{0.55, 0.2, 0.2}, Radius: 18, Speed: 40, Health: 110,
		ContactDamage: 18, Experience: 8, UnlockSeconds: 90, SpawnWeight: 1.2, DeathSound: audio.SoundThud,
	},
	EnemyFrostling: {
		Name: "Frostling", Color: [3]float32{0.5, 0.75, 0.95}, Radius: 9, Speed: 66, Health: 22,
		ContactDamage: 8, Experience: 2, UnlockSeconds: 120, SpawnWeight: 2.5, DeathSound: audio.SoundShatter,
		Immunities:   StatusChilled | StatusFrozen,
		DeathElement: ElementFrost, DeathRadius: 55, DeathDamage: 4, DeathStimulus: StimulusChill, DeathGroundCells: 2,
	},
	EnemyEmberling: {
		Name: "Emberling", Color: [3]float32{1, 0.45, 0.1}, Radius: 9, Speed: 80, Health: 20,
		ContactDamage: 8, Experience: 2, UnlockSeconds: 160, SpawnWeight: 2.5, DeathSound: audio.SoundThud,
		Immunities:   StatusBurning,
		DeathElement: ElementFire, DeathRadius: 45, DeathDamage: 6, DeathStimulus: StimulusHeat, DeathGroundCells: 1,
	},
	EnemySpiderTank: {
		Name: "Spider Tank", Color: [3]float32{0.46, 0.52, 0.44}, Radius: 60, Speed: 52, Health: 2600,
		ContactDamage: 28, Experience: 150,
		DeathElement: ElementFire, DeathRadius: 150, DeathDamage: 70, DeathStimulus: StimulusHeat, DeathGroundCells: 5,
		IsHeavy: true, IsCustomDrawn: true, DeathSound: audio.SoundExplosion,
	},
	EnemyDrillWorm: {
		Name: "DrillWorm", Color: [3]float32{0.55, 0.6, 0.68}, Radius: 36, Health: 1500, Experience: 250,
		IsHeavy: true, IsCustomDrawn: true, DeathSound: audio.SoundExplosion,
	},
	EnemyDrillWormSegment: {
		Name: "DrillWorm Segment", Color: [3]float32{0.55, 0.6, 0.68}, Radius: 26, Health: 600, Experience: 40,
		IsHeavy: true, IsCustomDrawn: true, DeathSound: audio.SoundExplosion,
	},
}

// ===== Public API =====

func NewEnemyStore(capacity int) *EnemyStore {
	store := &EnemyStore{
		ID:            make([]uint32, capacity),
		PositionX:     make([]float32, capacity),
		PositionY:     make([]float32, capacity),
		KnockbackX:    make([]float32, capacity),
		KnockbackY:    make([]float32, capacity),
		Health:        make([]float32, capacity),
		Power:         make([]float32, capacity),
		LastHitByBoss: make([]bool, capacity),
		IsHordeEvent:  make([]bool, capacity),
		Kind:          make([]EnemyKind, capacity),
		Status:        make([]StatusFlags, capacity),
		HitFlash:      make([]float32, capacity),
		BladeCooldown: make([]float32, capacity),
	}
	for statusIndex := range statusCount {
		store.StatusTimers[statusIndex] = make([]float32, capacity)
		store.StatusLevels[statusIndex] = make([]uint8, capacity)
	}
	return store
}

func (s *EnemyStore) Spawn(kind EnemyKind, x, y, healthScale float32) uint32 {
	if s.Count >= len(s.PositionX) {
		return 0
	}
	index := s.Count
	s.Count++
	s.nextID++
	s.ID[index] = s.nextID
	s.PositionX[index], s.PositionY[index] = x, y
	s.KnockbackX[index], s.KnockbackY[index] = 0, 0
	s.Health[index] = enemyTable[kind].Health * healthScale
	s.Power[index] = healthScale
	s.LastHitByBoss[index] = false
	s.IsHordeEvent[index] = false
	s.Kind[index] = kind
	s.Status[index] = 0
	s.HitFlash[index] = 0
	s.BladeCooldown[index] = 0
	for statusIndex := range statusCount {
		s.StatusTimers[statusIndex][index] = 0
		s.StatusLevels[statusIndex][index] = 0
	}
	return s.nextID
}

func (s *EnemyStore) IndexOfID(id uint32) int {
	for index := range s.Count {
		if s.ID[index] == id {
			return index
		}
	}
	return -1
}

func (s *EnemyStore) Remove(index int) {
	last := s.Count - 1
	s.ID[index] = s.ID[last]
	s.PositionX[index], s.PositionY[index] = s.PositionX[last], s.PositionY[last]
	s.KnockbackX[index], s.KnockbackY[index] = s.KnockbackX[last], s.KnockbackY[last]
	s.Health[index] = s.Health[last]
	s.Power[index] = s.Power[last]
	s.LastHitByBoss[index] = s.LastHitByBoss[last]
	s.IsHordeEvent[index] = s.IsHordeEvent[last]
	s.Kind[index] = s.Kind[last]
	s.Status[index] = s.Status[last]
	s.HitFlash[index] = s.HitFlash[last]
	s.BladeCooldown[index] = s.BladeCooldown[last]
	for statusIndex := range statusCount {
		s.StatusTimers[statusIndex][index] = s.StatusTimers[statusIndex][last]
		s.StatusLevels[statusIndex][index] = s.StatusLevels[statusIndex][last]
	}
	s.Count--
}

func (s *EnemyStore) ApplyStatus(index int, flags StatusFlags) {
	flags &^= enemyTable[s.Kind[index]].Immunities
	previous := s.Status[index]
	s.Status[index] |= flags
	for statusIndex := range statusCount {
		isApplied := flags&(1<<statusIndex) != 0
		hadStatus := previous&(1<<statusIndex) != 0
		timers := s.StatusTimers[statusIndex]
		timers[index] = max(timers[index], statusTable[statusIndex].DurationSeconds*boolToFloat(isApplied))
		levels := s.StatusLevels[statusIndex]
		heldLevel := levels[index] * uint8(boolToIndex(hadStatus))
		raisedLevel := min(statusTable[statusIndex].MaximumLevel, heldLevel+1)
		levels[index] = [2]uint8{heldLevel, raisedLevel}[boolToIndex(isApplied)]
	}
}

func (s *EnemyStore) LevelsOf(index int) StatusLevels {
	var levels StatusLevels
	for statusIndex := range statusCount {
		levels[statusIndex] = s.StatusLevels[statusIndex][index] * uint8(boolToIndex(s.Status[index]&(1<<statusIndex) != 0))
	}
	return levels
}

func (s *EnemyStore) TickStatuses(index int, deltaSeconds float32) {
	damage := float32(0)
	for remaining := s.Status[index]; remaining != 0; remaining &= remaining - 1 {
		statusIndex := bits.TrailingZeros8(uint8(remaining))
		timers := s.StatusTimers[statusIndex]
		timers[index] = max(0, timers[index]-deltaSeconds)
		s.Status[index] &^= StatusFlags(boolToIndex(timers[index] == 0)) << statusIndex
		damage += statusTable[statusIndex].DamagePerSecond * float32(s.StatusLevels[statusIndex][index]) * deltaSeconds
	}
	s.Health[index] -= damage
}

func (s *EnemyStore) SpeedFactor(index int) float32 {
	factor := float32(1)
	for remaining := s.Status[index]; remaining != 0; remaining &= remaining - 1 {
		statusIndex := bits.TrailingZeros8(uint8(remaining))
		factor *= statusSpeedByLevel[statusIndex][s.StatusLevels[statusIndex][index]]
	}
	return factor
}

func (s *EnemyStore) TintedColor(index int) [3]float32 {
	tinted := enemyTable[s.Kind[index]].Color
	for remaining := s.Status[index]; remaining != 0; remaining &= remaining - 1 {
		statusIndex := bits.TrailingZeros8(uint8(remaining))
		status := &statusTable[statusIndex]
		level := s.StatusLevels[statusIndex][index]
		levelShare := float32(level) / float32(status.MaximumLevel)
		peakShare := float32(level-1) / float32(max(1, status.MaximumLevel-1))
		levelTint := mixColor(status.Tint, status.PeakTint, peakShare)
		tinted = mixColor(tinted, levelTint, status.TintStrength*(statusTintFloor+(1-statusTintFloor)*levelShare))
	}
	return mixColor(tinted, [3]float32{1, 1, 1}, clamp(s.HitFlash[index]*8, 0, 1))
}

// ===== Internal =====

func mixColor(from, to [3]float32, weight float32) [3]float32 {
	return [3]float32{
		from[0] + (to[0]-from[0])*weight,
		from[1] + (to[1]-from[1])*weight,
		from[2] + (to[2]-from[2])*weight,
	}
}
