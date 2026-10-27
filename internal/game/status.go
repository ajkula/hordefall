package game

import "github.com/ajkula/ajkurig/world"

// ===== Types =====

type StatusFlags = world.StatusMask

type Element = world.ElementID

type ReactionKind = world.ReactionID

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
	ReactionNone ReactionKind = world.ReactionNone + iota
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

var statusTable = []world.StatusDefinition{
	{Name: "Burning", DurationSeconds: 3, Tint: [3]float32{0.85, 0.2, 0.05}, PeakTint: [3]float32{1, 0.82, 0.3}, TintStrength: 0.85, SpeedFactor: 1, DamagePerSecond: 6, MaximumLevel: 3, AuraColor: [3]float32{1, 0.6, 0.15}},
	{Name: "Chilled", DurationSeconds: 3, Tint: [3]float32{0.55, 0.8, 1}, PeakTint: [3]float32{0.9, 0.97, 1}, TintStrength: 0.75, SpeedFactor: 0.7, MaximumLevel: 3, AuraColor: [3]float32{0.65, 0.9, 1}},
	{Name: "Frozen", DurationSeconds: 1.8, Tint: [3]float32{0.85, 0.95, 1}, TintStrength: 0.85, SpeedFactor: 0, MaximumLevel: 1},
	{Name: "Oiled", DurationSeconds: 6, Tint: [3]float32{0.12, 0.08, 0.05}, TintStrength: 0.55, SpeedFactor: 0.8, MaximumLevel: 1},
	{Name: "Wet", DurationSeconds: 5, Tint: [3]float32{0.2, 0.45, 1}, TintStrength: 0.45, SpeedFactor: 0.9, MaximumLevel: 1},
	{Name: "Shocked", DurationSeconds: 1.5, Tint: [3]float32{1, 1, 0.4}, PeakTint: [3]float32{1, 1, 0.8}, TintStrength: 0.85, SpeedFactor: 0.6, MaximumLevel: 3, AuraColor: [3]float32{1, 1, 0.5}},
}

const maximumStatusLevel = 3

var leveledStatuses = []int{world.StatusIndex(StatusBurning), world.StatusIndex(StatusChilled), world.StatusIndex(StatusShocked)}

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

var reactionRules = []world.ReactionRule{
	{Element: ElementFire, Status: StatusOiled, Reaction: ReactionInferno},
	{Element: ElementShock, Status: StatusOiled, Reaction: ReactionInferno},
	{Element: ElementOil, Status: StatusBurning, Reaction: ReactionInferno},
	{Element: ElementFire, Status: StatusFrozen, Reaction: ReactionSteam},
	{Element: ElementFire, Status: StatusChilled, Reaction: ReactionSteam},
	{Element: ElementFrost, Status: StatusBurning, Reaction: ReactionSteam},
	{Element: ElementFire, Status: StatusWet, Reaction: ReactionExtinguish},
	{Element: ElementWater, Status: StatusBurning, Reaction: ReactionExtinguish},
	{Element: ElementFrost, Status: StatusWet, Reaction: ReactionFreeze},
	{Element: ElementFrost, Status: StatusChilled, Reaction: ReactionFreeze, MinimumLevel: chillLevelToFreeze},
	{Element: ElementShock, Status: StatusWet, Reaction: ReactionElectrocute},
	{Element: ElementShock, Status: StatusFrozen, Reaction: ReactionShatter},
	{Element: ElementPhysical, Status: StatusFrozen, Reaction: ReactionShatter},
	{Element: ElementFire, Status: StatusShocked, Reaction: ReactionPlasma},
	{Element: ElementPhysical, Status: StatusShocked, Reaction: ReactionOverload},
	{Element: ElementWater, Status: StatusShocked, Reaction: ReactionConduction},
}

var statusPriority = []StatusFlags{StatusFrozen, StatusOiled, StatusWet, StatusBurning, StatusChilled, StatusShocked}

const chillLevelToFreeze = 2

var chemistry = world.NewChemistry(world.ChemistrySettings{
	Statuses: statusTable, ElementStatus: elementStatus[:], Rules: reactionRules, Priority: statusPriority,
})
