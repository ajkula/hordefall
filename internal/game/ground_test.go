package game

import "testing"

// ===== Tests =====

func TestBurntGroundStartsRegrowingAfterAMinuteAndIsMostlyGreenAtThreeMinutes(t *testing.T) {
	for _, seed := range []uint32{1, 2, 3} {
		ground := NewGround(arenaSize, seed)
		column, row := ground.Columns/2, ground.Rows/2
		ground.PaintCircle(column, row, regrowthDiscCells+4, GroundGrass)
		ground.PaintCircle(column, row, regrowthDiscCells, GroundAsh)
		greenAt := map[int]int{}
		for tick := 1; tick <= 180*groundTicksPerSecond; tick++ {
			ground.Tick()
			greenAt[tick] = greenPercentInDisc(ground.MaterialAtCell, column, row)
		}
		if early := greenAt[50*groundTicksPerSecond]; early != 0 {
			t.Fatalf("seed %d: %d%% green after 50 s, want none", seed, early)
		}
		if late := greenAt[180*groundTicksPerSecond]; late < 65 || late > 80 {
			t.Fatalf("seed %d: %d%% green after 3 min, want about 70%%", seed, late)
		}
	}
}

func TestFireOnBurntGroundDelaysItsRegrowth(t *testing.T) {
	ground := NewGround(arenaSize, 1)
	ground.PaintCircle(ground.Columns/2, ground.Rows/2, 2, GroundScorched)
	for range 250 * groundTicksPerSecond {
		ground.StimulateArea(arenaSize/2, arenaSize/2, 2, StimulusHeat)
		ground.Tick()
	}
	if material := ground.MaterialAtCell(ground.Columns/2, ground.Rows/2); material != GroundScorched {
		t.Fatalf("heated burnt ground became %d", material)
	}
}

func TestStormGrowsFortyPercentOfTheLakesByThreeOrFourCells(t *testing.T) {
	changes := lakeSizeChanges(t, WeatherStorm)
	assertLakeChanges(t, changes, 3, 4)
}

func TestHeatwaveShrinksFortyPercentOfTheLakesByThreeOrFourCells(t *testing.T) {
	changes := lakeSizeChanges(t, WeatherHeatwave)
	assertLakeChanges(t, changes, -4, -3)
}

func TestGrowingLakeKeepsAShoreAgainstTheGrass(t *testing.T) {
	game := newLakeGame(t)
	game.weather.Kind = WeatherStorm
	game.planLakeTideIf(true)
	game.advanceLakeTide(weatherActiveSeconds + 1)
	for index := range game.ground.Columns * game.ground.Rows {
		column, row := game.groundCellPlace(index)
		isWater := game.ground.MaterialAtCell(column, row) == GroundWater
		isTouchingGrass := len(game.appendNeighborsOf(nil, index, isGrassOnly[:])) > 0
		if isWater && isTouchingGrass {
			t.Fatalf("water at %d,%d touches grass", column, row)
		}
	}
}

// ===== Internal =====

const (
	regrowthDiscCells    = 30
	groundTicksPerSecond = ticksPerSecond / groundTickFrames
	testLakeCount        = 10
	testLakeRadius       = 3
	testLakeSpacing      = 20
)

var isGrassOnly = [groundKindCount]bool{GroundGrass: true}

func greenPercentInDisc(materialAt func(column, row int) GroundKind, centerColumn, centerRow int) int {
	green, total := 0, 0
	for row := centerRow - regrowthDiscCells; row <= centerRow+regrowthDiscCells; row++ {
		for column := centerColumn - regrowthDiscCells; column <= centerColumn+regrowthDiscCells; column++ {
			isInside := (row-centerRow)*(row-centerRow)+(column-centerColumn)*(column-centerColumn) <= regrowthDiscCells*regrowthDiscCells
			material := materialAt(column, row)
			isGreen := material == GroundGrass || material == GroundRegrowing || material == GroundSprout
			total += boolToIndex(isInside)
			green += boolToIndex(isInside && isGreen)
		}
	}
	return green * 100 / total
}

func newLakeGame(t *testing.T) *Game {
	game := newHeadlessGame(t)
	ground := game.ground
	ground.PaintCircle(ground.Columns/2, ground.Rows/2, ground.Columns, GroundGrass)
	for lake := range testLakeCount {
		column := testLakeSpacing * (lake + 1)
		ground.PaintCircle(column, testLakeSpacing, testLakeRadius+1, GroundDirt)
		ground.PaintCircle(column, testLakeSpacing, testLakeRadius, GroundWater)
	}
	return game
}

func lakeSizeChanges(t *testing.T, weather WeatherKind) []int {
	game := newLakeGame(t)
	before := wetCellsPerLake(game)
	game.weather.Kind = weather
	game.planLakeTideIf(true)
	game.advanceLakeTide(weatherActiveSeconds + 1)
	after := wetCellsPerLake(game)
	changes := make([]int, 0, testLakeCount)
	for lake := range testLakeCount {
		changes = append(changes, after[lake]-before[lake])
	}
	return changes
}

func wetCellsPerLake(game *Game) []int {
	counts := make([]int, 0, testLakeCount)
	for lake := range testLakeCount {
		counts = append(counts, wetCellsAround(game, testLakeSpacing*(lake+1), testLakeSpacing))
	}
	return counts
}

func wetCellsAround(game *Game, centerColumn, centerRow int) int {
	count, reach := 0, testLakeSpacing/2
	for row := centerRow - reach; row < centerRow+reach; row++ {
		for column := centerColumn - reach; column < centerColumn+reach; column++ {
			count += boolToIndex(isWetGround[game.ground.MaterialAtCell(column, row)])
		}
	}
	return count
}

func assertLakeChanges(t *testing.T, changes []int, smallest, largest int) {
	changed := 0
	for _, change := range changes {
		isExpected := change == 0 || (change >= smallest && change <= largest)
		if !isExpected {
			t.Fatalf("lake changes %v, want 0 or %d to %d", changes, smallest, largest)
		}
		changed += boolToIndex(change != 0)
	}
	if changed != testLakeCount*4/10 {
		t.Fatalf("%d of %d lakes changed, want 4: %v", changed, testLakeCount, changes)
	}
}
