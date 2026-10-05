package game

import "hordefall/internal/audio"

// ===== Types =====

type StatusFlags uint8

type Element uint8

type ReactionKind uint8

type StatusDefinition struct {
	Name            string
	DurationSeconds float32
	Tint            [3]float32
	TintStrength    float32
	SpeedFactor     float32
	DamagePerSecond float32
	MaximumLevel    uint8
	AuraColor       [3]float32
	PeakTint        [3]float32
}

type StatusLevels [statusCount]uint8

type ReactionDefinition struct {
	Name              string
	Color             [3]float32
	DamageMultiplier  float32
	SuppressedStatus  StatusFlags
	RemovesStatus     StatusFlags
	AddsStatus        StatusFlags
	AreaRadius        float32
	AreaDamage        float32
	AreaElement       Element
	AreaRequires      StatusFlags
	GroundStimulus    GroundStimulus
	GroundRadiusCells int
	ShakeStrength     float32
	DamagesPlayer     bool
	Sound             audio.SoundKind
}

type ReactionRule struct {
	Element      Element
	Status       StatusFlags
	Reaction     ReactionKind
	MinimumLevel uint8
}

// ===== Constants =====

const (
	StatusBurning StatusFlags = 1 << iota
	StatusChilled
	StatusFrozen
	StatusOiled
	StatusWet
	StatusShocked
)

const statusCount = 6

const (
	ElementNone Element = iota
	ElementFire
	ElementFrost
	ElementShock
	ElementPhysical
	ElementOil
	ElementWater
	elementCount
)

const (
	ReactionNone ReactionKind = iota
	ReactionInferno
	ReactionSteam
	ReactionFreeze
	ReactionShatter
	ReactionElectrocute
	ReactionExtinguish
	ReactionPlasma
	ReactionOverload
	ReactionConduction
	reactionKindCount
)

var statusTable = [statusCount]StatusDefinition{
	{Name: "Burning", DurationSeconds: 3, Tint: [3]float32{0.85, 0.2, 0.05}, PeakTint: [3]float32{1, 0.82, 0.3}, TintStrength: 0.85, SpeedFactor: 1, DamagePerSecond: 6, MaximumLevel: 3, AuraColor: [3]float32{1, 0.6, 0.15}},
	{Name: "Chilled", DurationSeconds: 3, Tint: [3]float32{0.55, 0.8, 1}, PeakTint: [3]float32{0.9, 0.97, 1}, TintStrength: 0.75, SpeedFactor: 0.7, MaximumLevel: 3, AuraColor: [3]float32{0.65, 0.9, 1}},
	{Name: "Frozen", DurationSeconds: 1.8, Tint: [3]float32{0.85, 0.95, 1}, TintStrength: 0.85, SpeedFactor: 0, MaximumLevel: 1},
	{Name: "Oiled", DurationSeconds: 6, Tint: [3]float32{0.12, 0.08, 0.05}, TintStrength: 0.55, SpeedFactor: 0.8, MaximumLevel: 1},
	{Name: "Wet", DurationSeconds: 5, Tint: [3]float32{0.2, 0.45, 1}, TintStrength: 0.45, SpeedFactor: 0.9, MaximumLevel: 1},
	{Name: "Shocked", DurationSeconds: 1.5, Tint: [3]float32{1, 1, 0.4}, PeakTint: [3]float32{1, 1, 0.8}, TintStrength: 0.85, SpeedFactor: 0.6, MaximumLevel: 3, AuraColor: [3]float32{1, 1, 0.5}},
}

const (
	maximumStatusLevel = 3
	statusTintFloor    = 0.45
)

var statusSpeedByLevel = buildStatusSpeedByLevel()

var leveledStatuses = []int{statusBitIndex(StatusBurning), statusBitIndex(StatusChilled), statusBitIndex(StatusShocked)}

var elementStatus = [elementCount]StatusFlags{
	ElementFire:  StatusBurning,
	ElementFrost: StatusChilled,
	ElementShock: StatusShocked,
	ElementOil:   StatusOiled,
	ElementWater: StatusWet,
}

var reactionTable = [reactionKindCount]ReactionDefinition{
	ReactionNone: {DamageMultiplier: 1},
	ReactionInferno: {
		Name: "INFERNO", Color: [3]float32{1, 0.5, 0.1}, DamageMultiplier: 1,
		RemovesStatus: StatusOiled, AddsStatus: StatusBurning,
		AreaRadius: 75, AreaDamage: 28, AreaElement: ElementFire,
		GroundStimulus: StimulusHeat, GroundRadiusCells: 3, ShakeStrength: 7, DamagesPlayer: true, Sound: audio.SoundExplosion,
	},
	ReactionSteam: {
		Name: "STEAM", Color: [3]float32{0.9, 0.92, 0.95}, DamageMultiplier: 1.5,
		RemovesStatus: StatusBurning | StatusChilled, SuppressedStatus: StatusBurning | StatusChilled,
		AreaRadius: 50, AreaDamage: 12, AreaElement: ElementWater, ShakeStrength: 1.5, Sound: audio.SoundHiss,
	},
	ReactionFreeze: {
		Name: "FREEZE", Color: [3]float32{0.6, 0.9, 1}, DamageMultiplier: 1,
		RemovesStatus: StatusChilled | StatusWet | StatusBurning, SuppressedStatus: StatusChilled, AddsStatus: StatusFrozen,
		GroundStimulus: StimulusChill, GroundRadiusCells: 1, Sound: audio.SoundFreeze,
	},
	ReactionShatter: {
		Name: "SHATTER", Color: [3]float32{0.8, 0.95, 1}, DamageMultiplier: 3,
		RemovesStatus: StatusFrozen,
		AreaRadius:    45, AreaDamage: 10, AreaElement: ElementPhysical, ShakeStrength: 3, Sound: audio.SoundShatter,
	},
	ReactionElectrocute: {
		Name: "ELECTROCUTE", Color: [3]float32{1, 1, 0.45}, DamageMultiplier: 2,
		RemovesStatus: StatusWet,
		AreaRadius:    120, AreaDamage: 16, AreaElement: ElementShock, AreaRequires: StatusWet, ShakeStrength: 2, Sound: audio.SoundElectrocute,
	},
	ReactionExtinguish: {
		Name: "HISS", Color: [3]float32{0.7, 0.75, 0.8}, DamageMultiplier: 0.5,
		RemovesStatus: StatusBurning | StatusWet, SuppressedStatus: StatusBurning | StatusWet, Sound: audio.SoundHiss,
	},
	ReactionPlasma: {
		Name: "PLASMA", Color: [3]float32{1, 0.4, 0.9}, DamageMultiplier: 1.8,
		RemovesStatus: StatusShocked, AddsStatus: StatusBurning,
		AreaRadius: 70, AreaDamage: 18, AreaElement: ElementFire, AreaRequires: StatusShocked,
		GroundStimulus: StimulusHeat, GroundRadiusCells: 1, ShakeStrength: 3, Sound: audio.SoundExplosion,
	},
	ReactionOverload: {
		Name: "OVERLOAD", Color: [3]float32{1, 0.92, 0.35}, DamageMultiplier: 2.5,
		RemovesStatus: StatusShocked,
		AreaRadius:    45, AreaDamage: 8, AreaElement: ElementPhysical, AreaRequires: StatusShocked, ShakeStrength: 2, Sound: audio.SoundZap,
	},
	ReactionConduction: {
		Name: "CONDUCTION", Color: [3]float32{0.45, 0.85, 1}, DamageMultiplier: 1,
		RemovesStatus: StatusShocked, SuppressedStatus: StatusWet,
		AreaRadius: 130, AreaDamage: 10, AreaElement: ElementShock, AreaRequires: StatusWet, ShakeStrength: 1.5, Sound: audio.SoundElectrocute,
	},
}

var reactionRules = []ReactionRule{
	{ElementFire, StatusOiled, ReactionInferno, 0},
	{ElementShock, StatusOiled, ReactionInferno, 0},
	{ElementOil, StatusBurning, ReactionInferno, 0},
	{ElementFire, StatusFrozen, ReactionSteam, 0},
	{ElementFire, StatusChilled, ReactionSteam, 0},
	{ElementFrost, StatusBurning, ReactionSteam, 0},
	{ElementFire, StatusWet, ReactionExtinguish, 0},
	{ElementWater, StatusBurning, ReactionExtinguish, 0},
	{ElementFrost, StatusWet, ReactionFreeze, 0},
	{ElementFrost, StatusChilled, ReactionFreeze, chillLevelToFreeze},
	{ElementShock, StatusWet, ReactionElectrocute, 0},
	{ElementShock, StatusFrozen, ReactionShatter, 0},
	{ElementPhysical, StatusFrozen, ReactionShatter, 0},
	{ElementFire, StatusShocked, ReactionPlasma, 0},
	{ElementPhysical, StatusShocked, ReactionOverload, 0},
	{ElementWater, StatusShocked, ReactionConduction, 0},
}

var statusPriority = []StatusFlags{StatusFrozen, StatusOiled, StatusWet, StatusBurning, StatusChilled, StatusShocked}

var elementReactions = buildElementReactions(reactionRules)

var elementReactionLevels = buildElementReactionLevels(reactionRules)

const chillLevelToFreeze = 2

// ===== Public API =====

func FindReaction(element Element, status StatusFlags, levels StatusLevels) ReactionKind {
	for _, candidate := range statusPriority {
		statusIndex := statusBitIndex(candidate)
		reaction := elementReactions[element][statusIndex]
		isReady := levels[statusIndex] >= elementReactionLevels[element][statusIndex]
		if status&candidate != 0 && reaction != ReactionNone && isReady {
			return reaction
		}
	}
	return ReactionNone
}

// ===== Internal =====

func buildElementReactions(rules []ReactionRule) [elementCount][statusCount]ReactionKind {
	var table [elementCount][statusCount]ReactionKind
	for _, rule := range rules {
		table[rule.Element][statusBitIndex(rule.Status)] = rule.Reaction
	}
	return table
}

func buildStatusSpeedByLevel() [statusCount][maximumStatusLevel + 1]float32 {
	var table [statusCount][maximumStatusLevel + 1]float32
	for statusIndex, status := range statusTable {
		factor := float32(1)
		for level := range maximumStatusLevel + 1 {
			table[statusIndex][level] = factor
			factor *= status.SpeedFactor
		}
	}
	return table
}

func buildElementReactionLevels(rules []ReactionRule) [elementCount][statusCount]uint8 {
	var table [elementCount][statusCount]uint8
	for _, rule := range rules {
		table[rule.Element][statusBitIndex(rule.Status)] = rule.MinimumLevel
	}
	return table
}

func statusBitIndex(flag StatusFlags) int {
	index := 0
	for flag > 1 {
		flag >>= 1
		index++
	}
	return index
}
