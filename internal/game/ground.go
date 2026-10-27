package game

import "github.com/ajkula/ajkurig/world"

// ===== Types =====

type GroundKind = world.MaterialID

type GroundStimulus = world.StimulusID

// ===== Constants =====

const (
	GroundGrass GroundKind = iota
	GroundDirt
	GroundAsh
	GroundOil
	GroundFire
	GroundIce
	GroundWater
	groundKindCount
)

const (
	StimulusNone   = world.StimulusNone
	StimulusExpire = world.StimulusExpire
)

const (
	StimulusHeat GroundStimulus = world.FirstGameStimulus + iota
	StimulusChill
	StimulusOil
	StimulusDouse
	StimulusScorch
	stimulusCount
)

const (
	groundCellSize     = 16
	groundTickFrames   = 6
	spawnClearingCells = 8
)

var groundMaterials = []world.Material{
	GroundGrass: {Name: "Grass", BaseColor: [3]uint8{46, 86, 44}, ColorVariation: 16},
	GroundDirt:  {Name: "Dirt", BaseColor: [3]uint8{88, 70, 50}, ColorVariation: 14},
	GroundAsh: {
		Name: "Ash", BaseColor: [3]uint8{54, 52, 52}, ColorVariation: 12,
		LifetimeMinTicks: 200, LifetimeMaxTicks: 400,
	},
	GroundOil: {
		Name: "Oil", BaseColor: [3]uint8{70, 48, 88}, ColorVariation: 22,
		LifetimeMinTicks: 600, LifetimeMaxTicks: 900, Tag: uint32(ElementOil),
	},
	GroundFire: {
		Name: "Fire", BaseColor: [3]uint8{250, 110, 25}, ColorVariation: 70,
		LifetimeMinTicks: 25, LifetimeMaxTicks: 50, Tag: uint32(ElementFire), Emits: StimulusHeat, Glows: true,
	},
	GroundIce: {
		Name: "Ice", BaseColor: [3]uint8{165, 210, 240}, ColorVariation: 18,
		LifetimeMinTicks: 150, LifetimeMaxTicks: 250, Tag: uint32(ElementFrost),
	},
	GroundWater: {Name: "Water", BaseColor: [3]uint8{36, 84, 165}, ColorVariation: 16, Tag: uint32(ElementWater)},
}

var groundRules = []world.MaterialRule{
	world.Rule(StimulusHeat, GroundGrass, GroundFire, 36),
	world.Rule(StimulusHeat, GroundOil, GroundFire, 1024),
	world.Rule(StimulusHeat, GroundIce, GroundWater, 500),
	world.Rule(StimulusChill, GroundWater, GroundIce, 1024),
	world.Rule(StimulusChill, GroundFire, GroundAsh, 1024),
	world.Rule(StimulusOil, GroundGrass, GroundOil, 1024),
	world.Rule(StimulusOil, GroundDirt, GroundOil, 1024),
	world.Rule(StimulusOil, GroundAsh, GroundOil, 1024),
	world.Rule(StimulusDouse, GroundFire, GroundAsh, 1024),
	world.Rule(StimulusScorch, GroundGrass, GroundFire, 700),
	world.Rule(StimulusScorch, GroundOil, GroundFire, 1024),
	world.Rule(StimulusScorch, GroundIce, GroundWater, 1024),
	world.Rule(StimulusScorch, GroundDirt, GroundAsh, 500),
	world.Rule(StimulusExpire, GroundFire, GroundAsh, 1024),
	world.Rule(StimulusExpire, GroundAsh, GroundGrass, 1024),
	world.Rule(StimulusExpire, GroundIce, GroundWater, 1024),
	world.Rule(StimulusExpire, GroundOil, GroundDirt, 1024),
}

var terrainKinds = [4]GroundKind{GroundGrass, GroundDirt, GroundWater, GroundOil}

// ===== Public API =====

func NewGround(worldSize float32, seed uint32) *world.Ground {
	columns := int(worldSize) / groundCellSize
	ground := world.NewGround(world.GroundSettings{
		Materials: groundMaterials, Rules: groundRules, StimulusCount: int(stimulusCount),
		CellSize: groundCellSize, Columns: columns, Rows: columns, Seed: seed, Generate: generateTerrain,
	})
	ground.PaintCircle(columns/2, columns/2, spawnClearingCells, GroundGrass)
	return ground
}

func contactElement(ground *world.Ground, x, y float32) Element {
	return Element(ground.Material(ground.MaterialAt(x, y)).Tag)
}

// ===== Internal =====

func generateTerrain(column, row int, seed uint32) GroundKind {
	x, y := float32(column), float32(row)
	moisture := world.ValueNoise(seed, x/22, y/22)*0.7 + world.ValueNoise(seed+7, x/7, y/7)*0.3
	oiliness := world.ValueNoise(seed+13, x/9, y/9)
	return terrainFor(moisture, oiliness)
}

func terrainFor(moisture, oiliness float32) GroundKind {
	isShore := moisture < 0.32
	isWater := moisture < 0.24
	isOil := oiliness > 0.87 && !isShore
	return terrainKinds[max(boolToIndex(isShore), boolToIndex(isWater)*2, boolToIndex(isOil)*3)]
}
