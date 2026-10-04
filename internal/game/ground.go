package game

import "hordefall/internal/rng"

// ===== Types =====

type GroundKind uint8

type GroundStimulus uint8

type GroundDefinition struct {
	Name             string
	BaseColor        [3]uint8
	ColorVariation   uint8
	LifetimeMinTicks uint16
	LifetimeMaxTicks uint16
	ContactElement   Element
	EmitsHeat        bool
	Glows            bool
}

type GroundRule struct {
	Stimulus      GroundStimulus
	From          GroundKind
	Into          GroundKind
	ChancePer1024 uint32
}

type GroundTransition struct {
	Into          GroundKind
	ChancePer1024 uint32
}

type Ground struct {
	Columns    int
	Rows       int
	cells      []GroundKind
	lifetimes  []uint16
	shades     []uint8
	bornOnTick []uint32
	tick       uint32
	random     rng.Random
	palette    [groundKindCount][groundShadeLevels][4]byte
	Pixels     []byte
}

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
	StimulusNone GroundStimulus = iota
	StimulusHeat
	StimulusChill
	StimulusOil
	StimulusDouse
	StimulusScorch
	StimulusExpire
	stimulusCount
)

const (
	groundCellSize     = 16
	groundShadeLevels  = 16
	groundTickFrames   = 6
	spawnClearingCells = 8
)

var groundTable = [groundKindCount]GroundDefinition{
	GroundGrass: {Name: "Grass", BaseColor: [3]uint8{46, 86, 44}, ColorVariation: 16},
	GroundDirt:  {Name: "Dirt", BaseColor: [3]uint8{88, 70, 50}, ColorVariation: 14},
	GroundAsh: {
		Name: "Ash", BaseColor: [3]uint8{54, 52, 52}, ColorVariation: 12,
		LifetimeMinTicks: 200, LifetimeMaxTicks: 400,
	},
	GroundOil: {
		Name: "Oil", BaseColor: [3]uint8{70, 48, 88}, ColorVariation: 22,
		LifetimeMinTicks: 600, LifetimeMaxTicks: 900, ContactElement: ElementOil,
	},
	GroundFire: {
		Name: "Fire", BaseColor: [3]uint8{250, 110, 25}, ColorVariation: 70,
		LifetimeMinTicks: 25, LifetimeMaxTicks: 50, ContactElement: ElementFire, EmitsHeat: true, Glows: true,
	},
	GroundIce: {
		Name: "Ice", BaseColor: [3]uint8{165, 210, 240}, ColorVariation: 18,
		LifetimeMinTicks: 150, LifetimeMaxTicks: 250, ContactElement: ElementFrost,
	},
	GroundWater: {Name: "Water", BaseColor: [3]uint8{36, 84, 165}, ColorVariation: 16, ContactElement: ElementWater},
}

var groundRules = []GroundRule{
	{StimulusHeat, GroundGrass, GroundFire, 36},
	{StimulusHeat, GroundOil, GroundFire, 1024},
	{StimulusHeat, GroundIce, GroundWater, 500},
	{StimulusChill, GroundWater, GroundIce, 1024},
	{StimulusChill, GroundFire, GroundAsh, 1024},
	{StimulusOil, GroundGrass, GroundOil, 1024},
	{StimulusOil, GroundDirt, GroundOil, 1024},
	{StimulusOil, GroundAsh, GroundOil, 1024},
	{StimulusDouse, GroundFire, GroundAsh, 1024},
	{StimulusScorch, GroundGrass, GroundFire, 700},
	{StimulusScorch, GroundOil, GroundFire, 1024},
	{StimulusScorch, GroundIce, GroundWater, 1024},
	{StimulusScorch, GroundDirt, GroundAsh, 500},
	{StimulusExpire, GroundFire, GroundAsh, 1024},
	{StimulusExpire, GroundAsh, GroundGrass, 1024},
	{StimulusExpire, GroundIce, GroundWater, 1024},
	{StimulusExpire, GroundOil, GroundDirt, 1024},
}

var groundTransitions = buildGroundTransitions(groundRules)

var cardinalOffsets = [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

// ===== Public API =====

func NewGround(worldSize float32, seed uint32) *Ground {
	columns := int(worldSize) / groundCellSize
	cellCount := columns * columns
	ground := &Ground{
		Columns:    columns,
		Rows:       columns,
		cells:      make([]GroundKind, cellCount),
		lifetimes:  make([]uint16, cellCount),
		shades:     make([]uint8, cellCount),
		bornOnTick: make([]uint32, cellCount),
		random:     rng.New(seed),
		Pixels:     make([]byte, cellCount*4),
	}
	ground.buildPalette()
	ground.generate(seed)
	return ground
}

func (g *Ground) Tick() {
	g.tick++
	for index := range g.cells {
		g.updateCell(index)
	}
}

func (g *Ground) KindAt(x, y float32) GroundKind {
	return g.cells[g.indexAt(x, y)]
}

func (g *Ground) StimulateAt(x, y float32, stimulus GroundStimulus) {
	g.stimulate(g.indexAt(x, y), stimulus)
}

func (g *Ground) StimulateArea(x, y float32, radiusCells int, stimulus GroundStimulus) {
	centerColumn, centerRow := int(x)/groundCellSize, int(y)/groundCellSize
	for row := centerRow - radiusCells; row <= centerRow+radiusCells; row++ {
		for column := centerColumn - radiusCells; column <= centerColumn+radiusCells; column++ {
			isInsideCircle := (row-centerRow)*(row-centerRow)+(column-centerColumn)*(column-centerColumn) <= radiusCells*radiusCells
			g.stimulateCellIf(column, row, stimulus, isInsideCircle)
		}
	}
}

func (g *Ground) RefreshPixels(clock uint32) {
	for index, kind := range g.cells {
		flicker := ((uint32(index)*2654435761 + clock*40503) >> 28) * uint32(boolToIndex(groundTable[kind].Glows))
		shade := (uint32(g.shades[index]) + flicker) % groundShadeLevels
		copy(g.Pixels[index*4:], g.palette[kind][shade][:])
	}
}

// ===== Internal =====

func buildGroundTransitions(rules []GroundRule) [stimulusCount][groundKindCount]GroundTransition {
	var table [stimulusCount][groundKindCount]GroundTransition
	for _, rule := range rules {
		table[rule.Stimulus][rule.From] = GroundTransition{Into: rule.Into, ChancePer1024: rule.ChancePer1024}
	}
	return table
}

func (g *Ground) indexAt(x, y float32) int {
	column := clampInt(int(x)/groundCellSize, 0, g.Columns-1)
	row := clampInt(int(y)/groundCellSize, 0, g.Rows-1)
	return row*g.Columns + column
}

func (g *Ground) stimulateCellIf(column, row int, stimulus GroundStimulus, shouldStimulate bool) {
	isInside := column >= 0 && row >= 0 && column < g.Columns && row < g.Rows
	if !shouldStimulate || !isInside {
		return
	}
	g.stimulate(row*g.Columns+column, stimulus)
}

func (g *Ground) stimulate(index int, stimulus GroundStimulus) {
	transition := groundTransitions[stimulus][g.cells[index]]
	if g.random.Next()&1023 >= transition.ChancePer1024 {
		return
	}
	g.setCell(index, transition.Into)
}

func (g *Ground) setCell(index int, kind GroundKind) {
	definition := &groundTable[kind]
	lifetimeRange := uint32(definition.LifetimeMaxTicks-definition.LifetimeMinTicks) + 1
	g.cells[index] = kind
	g.lifetimes[index] = definition.LifetimeMinTicks + uint16(g.random.Next()%lifetimeRange)
	g.shades[index] = uint8(g.random.Next() % groundShadeLevels)
	g.bornOnTick[index] = g.tick
}

func (g *Ground) updateCell(index int) {
	kind := g.cells[index]
	g.spreadHeat(index, kind)
	g.ageCell(index, kind)
}

func (g *Ground) spreadHeat(index int, kind GroundKind) {
	if !groundTable[kind].EmitsHeat || g.bornOnTick[index] == g.tick {
		return
	}
	offset := cardinalOffsets[g.random.Next()&3]
	column, row := index%g.Columns+offset[0], index/g.Columns+offset[1]
	g.stimulateCellIf(column, row, StimulusHeat, true)
}

func (g *Ground) ageCell(index int, kind GroundKind) {
	if groundTable[kind].LifetimeMaxTicks == 0 || g.bornOnTick[index] == g.tick {
		return
	}
	if g.lifetimes[index] == 0 {
		g.stimulate(index, StimulusExpire)
		return
	}
	g.lifetimes[index]--
}

func (g *Ground) generate(seed uint32) {
	for index := range g.cells {
		x, y := float32(index%g.Columns), float32(index/g.Columns)
		moisture := valueNoise(seed, x/22, y/22)*0.7 + valueNoise(seed+7, x/7, y/7)*0.3
		oiliness := valueNoise(seed+13, x/9, y/9)
		g.setCell(index, terrainFor(moisture, oiliness))
	}
	g.clearSpawnArea()
}

func (g *Ground) clearSpawnArea() {
	centerColumn, centerRow := g.Columns/2, g.Rows/2
	for row := centerRow - spawnClearingCells; row <= centerRow+spawnClearingCells; row++ {
		for column := centerColumn - spawnClearingCells; column <= centerColumn+spawnClearingCells; column++ {
			isInside := (row-centerRow)*(row-centerRow)+(column-centerColumn)*(column-centerColumn) <= spawnClearingCells*spawnClearingCells
			g.setGrassIf(row*g.Columns+column, isInside)
		}
	}
}

func (g *Ground) setGrassIf(index int, shouldSet bool) {
	if !shouldSet {
		return
	}
	g.setCell(index, GroundGrass)
}

func terrainFor(moisture, oiliness float32) GroundKind {
	kinds := [4]GroundKind{GroundGrass, GroundDirt, GroundWater, GroundOil}
	isShore := moisture < 0.32
	isWater := moisture < 0.24
	isOil := oiliness > 0.87 && !isShore
	return kinds[max(boolToIndex(isShore), boolToIndex(isWater)*2, boolToIndex(isOil)*3)]
}

func valueNoise(seed uint32, x, y float32) float32 {
	column, row := int(x), int(y)
	fractionX, fractionY := smoothstep(x-float32(column)), smoothstep(y-float32(row))
	top := lerp(latticeValue(seed, column, row), latticeValue(seed, column+1, row), fractionX)
	bottom := lerp(latticeValue(seed, column, row+1), latticeValue(seed, column+1, row+1), fractionX)
	return lerp(top, bottom, fractionY)
}

func latticeValue(seed uint32, column, row int) float32 {
	hash := uint32(column)*374761393 + uint32(row)*668265263 + seed*2246822519
	hash = (hash ^ (hash >> 13)) * 1274126177
	return float32((hash^(hash>>16))&0xFFFF) / 0xFFFF
}

func smoothstep(value float32) float32 {
	return value * value * (3 - 2*value)
}

func lerp(from, to, progress float32) float32 {
	return from + (to-from)*progress
}

func (g *Ground) buildPalette() {
	for kind := range groundKindCount {
		definition := &groundTable[kind]
		for shade := range groundShadeLevels {
			offset := (shade - groundShadeLevels/2) * int(definition.ColorVariation) / (groundShadeLevels / 2)
			g.palette[kind][shade] = [4]byte{
				byte(clampInt(int(definition.BaseColor[0])+offset, 0, 255)),
				byte(clampInt(int(definition.BaseColor[1])+offset, 0, 255)),
				byte(clampInt(int(definition.BaseColor[2])+offset, 0, 255)),
				255,
			}
		}
	}
}
