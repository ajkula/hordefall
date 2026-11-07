package game

import "github.com/ajkula/ajkurig/mathx"

// ===== Types =====

type LakeChange struct {
	Region    int32
	AtSeconds float32
}

type LakeTide struct {
	Direction  int
	regions    [][]int32
	changes    []LakeChange
	candidates []int32
	picks      []int
}

// ===== Constants =====

const (
	lakeChosenShare     = 0.4
	lakeCellsMinimum    = 3
	lakeCellsSpread     = 2
	lakeTideGrows       = 1
	lakeTideShrinks     = -1
	lakeTideIndexOffset = 1
)

var lakeTideDirections = [weatherKindCount]int{WeatherStorm: lakeTideGrows, WeatherHeatwave: lakeTideShrinks}

var lakeShifts = [3]func(g *Game, region int){
	lakeTideShrinks + lakeTideIndexOffset: (*Game).shrinkLake,
	lakeTideGrows + lakeTideIndexOffset:   (*Game).growLake,
}

var isWetGround = [groundKindCount]bool{GroundWater: true, GroundIce: true}

var isFloodableGround = [groundKindCount]bool{
	GroundGrass: true, GroundDirt: true, GroundAsh: true, GroundScorched: true,
	GroundBurntShore: true, GroundSprout: true, GroundRegrowing: true,
}

var isShoreableGround = [groundKindCount]bool{
	GroundGrass: true, GroundAsh: true, GroundScorched: true, GroundSprout: true, GroundRegrowing: true,
}

// ===== Internal =====

func (g *Game) resetLakeTide() {
	g.lakeTide.Direction = 0
	g.lakeTide.regions, g.lakeTide.changes = nil, g.lakeTide.changes[:0]
}

func (g *Game) planLakeTideIf(hasStarted bool) {
	if !hasStarted {
		return
	}
	g.resetLakeTide()
	tide := &g.lakeTide
	tide.Direction = lakeTideDirections[g.weather.Kind]
	if tide.Direction == 0 {
		return
	}
	tide.regions = g.ground.Regions(isWetGround[:])
	chosenCount := int(float32(len(tide.regions))*lakeChosenShare + 0.5)
	tide.picks = g.random.PickDistinct(chosenCount, len(tide.regions), tide.picks)
	for _, region := range tide.picks {
		g.scheduleLakeChanges(region, lakeCellsMinimum+g.random.Below(lakeCellsSpread))
	}
}

func (g *Game) scheduleLakeChanges(region, count int) {
	for range count {
		change := LakeChange{Region: int32(region), AtSeconds: g.random.Between(0, weatherActiveSeconds)}
		g.lakeTide.changes = append(g.lakeTide.changes, change)
	}
}

func (g *Game) advanceLakeTide(deltaSeconds float32) {
	tide := &g.lakeTide
	kept := tide.changes[:0]
	for _, change := range tide.changes {
		change.AtSeconds -= deltaSeconds
		isDue := change.AtSeconds <= 0
		g.shiftLakeIf(int(change.Region), isDue)
		kept = mathx.AppendIf(kept, change, !isDue)
	}
	tide.changes = kept
}

func (g *Game) shiftLakeIf(region int, isDue bool) {
	if !isDue {
		return
	}
	lakeShifts[g.lakeTide.Direction+lakeTideIndexOffset](g, region)
}

func (g *Game) growLake(region int) {
	tide := &g.lakeTide
	tide.candidates = tide.candidates[:0]
	for _, cell := range tide.regions[region] {
		tide.candidates = g.appendNeighborsOf(tide.candidates, int(cell), isFloodableGround[:])
	}
	if len(tide.candidates) == 0 {
		return
	}
	cell := tide.candidates[g.random.Below(len(tide.candidates))]
	g.paintGroundCell(int(cell), GroundWater)
	tide.regions[region] = append(tide.regions[region], cell)
	tide.candidates = g.appendNeighborsOf(tide.candidates[:0], int(cell), isShoreableGround[:])
	for _, shore := range tide.candidates {
		g.paintGroundCell(int(shore), GroundDirt)
	}
}

func (g *Game) shrinkLake(region int) {
	tide := &g.lakeTide
	cells := tide.regions[region]
	tide.candidates = tide.candidates[:0]
	for position, cell := range cells {
		isEdge := isWetGround[g.ground.MaterialAtCell(g.groundCellPlace(int(cell)))] && g.hasDryNeighbor(int(cell))
		tide.candidates = mathx.AppendIf(tide.candidates, int32(position), isEdge)
	}
	if len(tide.candidates) == 0 {
		return
	}
	position := tide.candidates[g.random.Below(len(tide.candidates))]
	g.paintGroundCell(int(cells[position]), GroundDirt)
	cells[position] = cells[len(cells)-1]
	tide.regions[region] = cells[:len(cells)-1]
}

func (g *Game) appendNeighborsOf(cells []int32, cell int, isWanted []bool) []int32 {
	column, row := g.groundCellPlace(cell)
	for _, offset := range nukeGroundNeighbors {
		neighbor, isInside := g.groundCellIndex(column+offset[0], row+offset[1])
		cells = mathx.AppendIf(cells, int32(neighbor), isInside && isWanted[g.ground.MaterialAtCell(g.groundCellPlace(neighbor))])
	}
	return cells
}

func (g *Game) hasDryNeighbor(cell int) bool {
	column, row := g.groundCellPlace(cell)
	for _, offset := range nukeGroundNeighbors {
		neighbor, isInside := g.groundCellIndex(column+offset[0], row+offset[1])
		if isInside && !isWetGround[g.ground.MaterialAtCell(g.groundCellPlace(neighbor))] {
			return true
		}
	}
	return false
}

func (g *Game) paintGroundCell(cell int, material GroundKind) {
	column, row := g.groundCellPlace(cell)
	g.ground.PaintCell(column, row, material)
}

func (g *Game) groundCellPlace(cell int) (int, int) {
	return cell % g.ground.Columns, cell / g.ground.Columns
}

func (g *Game) groundCellIndex(column, row int) (int, bool) {
	isInside := column >= 0 && row >= 0 && column < g.ground.Columns && row < g.ground.Rows
	return row*g.ground.Columns + column, isInside
}
