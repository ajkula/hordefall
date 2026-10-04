package main

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
}

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
	Sound             SoundKind
}

type ReactionRule struct {
	Element  Element
	Status   StatusFlags
	Reaction ReactionKind
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
	{Name: "Burning", DurationSeconds: 3, Tint: [3]float32{1, 0.45, 0.05}, TintStrength: 0.55, SpeedFactor: 1, DamagePerSecond: 9},
	{Name: "Chilled", DurationSeconds: 3, Tint: [3]float32{0.55, 0.8, 1}, TintStrength: 0.5, SpeedFactor: 0.5},
	{Name: "Frozen", DurationSeconds: 1.8, Tint: [3]float32{0.85, 0.95, 1}, TintStrength: 0.85, SpeedFactor: 0},
	{Name: "Oiled", DurationSeconds: 6, Tint: [3]float32{0.12, 0.08, 0.05}, TintStrength: 0.55, SpeedFactor: 0.8},
	{Name: "Wet", DurationSeconds: 5, Tint: [3]float32{0.2, 0.45, 1}, TintStrength: 0.45, SpeedFactor: 0.9},
	{Name: "Shocked", DurationSeconds: 1.5, Tint: [3]float32{1, 1, 0.4}, TintStrength: 0.7, SpeedFactor: 0.3},
}

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
		GroundStimulus: StimulusHeat, GroundRadiusCells: 3, ShakeStrength: 7, DamagesPlayer: true, Sound: SoundExplosion,
	},
	ReactionSteam: {
		Name: "STEAM", Color: [3]float32{0.9, 0.92, 0.95}, DamageMultiplier: 1.5,
		RemovesStatus: StatusBurning | StatusChilled, SuppressedStatus: StatusBurning | StatusChilled,
		AreaRadius: 50, AreaDamage: 12, AreaElement: ElementWater, ShakeStrength: 1.5, Sound: SoundHiss,
	},
	ReactionFreeze: {
		Name: "FREEZE", Color: [3]float32{0.6, 0.9, 1}, DamageMultiplier: 1,
		RemovesStatus: StatusChilled | StatusWet | StatusBurning, SuppressedStatus: StatusChilled, AddsStatus: StatusFrozen,
		GroundStimulus: StimulusChill, GroundRadiusCells: 1, Sound: SoundFreeze,
	},
	ReactionShatter: {
		Name: "SHATTER", Color: [3]float32{0.8, 0.95, 1}, DamageMultiplier: 3,
		RemovesStatus: StatusFrozen,
		AreaRadius:    45, AreaDamage: 10, AreaElement: ElementPhysical, ShakeStrength: 3, Sound: SoundShatter,
	},
	ReactionElectrocute: {
		Name: "ELECTROCUTE", Color: [3]float32{1, 1, 0.45}, DamageMultiplier: 2,
		RemovesStatus: StatusWet,
		AreaRadius:    120, AreaDamage: 16, AreaElement: ElementShock, AreaRequires: StatusWet, ShakeStrength: 2, Sound: SoundElectrocute,
	},
	ReactionExtinguish: {
		Name: "HISS", Color: [3]float32{0.7, 0.75, 0.8}, DamageMultiplier: 0.5,
		RemovesStatus: StatusBurning | StatusWet, SuppressedStatus: StatusBurning | StatusWet, Sound: SoundHiss,
	},
	ReactionPlasma: {
		Name: "PLASMA", Color: [3]float32{1, 0.4, 0.9}, DamageMultiplier: 1.8,
		RemovesStatus: StatusShocked, AddsStatus: StatusBurning,
		AreaRadius: 70, AreaDamage: 18, AreaElement: ElementFire, AreaRequires: StatusShocked,
		GroundStimulus: StimulusHeat, GroundRadiusCells: 1, ShakeStrength: 3, Sound: SoundExplosion,
	},
	ReactionOverload: {
		Name: "OVERLOAD", Color: [3]float32{1, 0.92, 0.35}, DamageMultiplier: 2.5,
		RemovesStatus: StatusShocked,
		AreaRadius:    45, AreaDamage: 8, AreaElement: ElementPhysical, AreaRequires: StatusShocked, ShakeStrength: 2, Sound: SoundZap,
	},
	ReactionConduction: {
		Name: "CONDUCTION", Color: [3]float32{0.45, 0.85, 1}, DamageMultiplier: 1,
		RemovesStatus: StatusShocked, SuppressedStatus: StatusWet,
		AreaRadius: 130, AreaDamage: 10, AreaElement: ElementShock, AreaRequires: StatusWet, ShakeStrength: 1.5, Sound: SoundElectrocute,
	},
}

var reactionRules = []ReactionRule{
	{ElementFire, StatusOiled, ReactionInferno},
	{ElementShock, StatusOiled, ReactionInferno},
	{ElementOil, StatusBurning, ReactionInferno},
	{ElementFire, StatusFrozen, ReactionSteam},
	{ElementFire, StatusChilled, ReactionSteam},
	{ElementFrost, StatusBurning, ReactionSteam},
	{ElementFire, StatusWet, ReactionExtinguish},
	{ElementWater, StatusBurning, ReactionExtinguish},
	{ElementFrost, StatusWet, ReactionFreeze},
	{ElementFrost, StatusChilled, ReactionFreeze},
	{ElementShock, StatusWet, ReactionElectrocute},
	{ElementShock, StatusFrozen, ReactionShatter},
	{ElementPhysical, StatusFrozen, ReactionShatter},
	{ElementFire, StatusShocked, ReactionPlasma},
	{ElementPhysical, StatusShocked, ReactionOverload},
	{ElementWater, StatusShocked, ReactionConduction},
}

var statusPriority = []StatusFlags{StatusFrozen, StatusOiled, StatusWet, StatusBurning, StatusChilled, StatusShocked}

var elementReactions = buildElementReactions(reactionRules)

// ===== Public API =====

func FindReaction(element Element, status StatusFlags) ReactionKind {
	for _, candidate := range statusPriority {
		reaction := elementReactions[element][statusBitIndex(candidate)]
		if status&candidate != 0 && reaction != ReactionNone {
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

func statusBitIndex(flag StatusFlags) int {
	index := 0
	for flag > 1 {
		flag >>= 1
		index++
	}
	return index
}
