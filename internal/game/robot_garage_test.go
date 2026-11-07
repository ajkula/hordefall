package game

import (
	"testing"

	"github.com/ajkula/ajkurig/input"
	"github.com/ajkula/ajkurig/store"
)

// ===== Tests =====

func TestEveryPartExceptTheStarterRobotHasOneUnlock(t *testing.T) {
	for kind := range robotSlotCount {
		slot := &robotSlots[kind]
		for part := range slot.Count {
			matches := 0
			for index := range robotUnlocks {
				matches += boolToIndex(robotUnlocks[index].Slot == kind && robotUnlocks[index].Part == part)
			}
			isStarter := kind == RobotSlotColor || part == slot.Part(&defaultRobotLook)
			if matches != 1-boolToIndex(isStarter) {
				t.Fatalf("slot %d part %d has %d unlocks", kind, part, matches)
			}
		}
	}
}

func TestUnlockGoalsRiseWithinEachCounter(t *testing.T) {
	var previous [robotCounterCount]int
	for index := range robotUnlocks {
		unlock := &robotUnlocks[index]
		if unlock.Goal <= previous[unlock.Counter] {
			t.Fatalf("unlock %d does not rise", index)
		}
		previous[unlock.Counter] = unlock.Goal
	}
}

func TestClassicRobotPartsAreLateAndScattered(t *testing.T) {
	classic := RobotLook{}
	previous := -2
	found := 0
	for index := range robotUnlocks {
		unlock := &robotUnlocks[index]
		if unlock.Part != robotSlots[unlock.Slot].Part(&classic) {
			continue
		}
		if index < len(robotUnlocks)/2 || index == previous+1 {
			t.Fatalf("classic part at unlock %d is too early or next to another", index)
		}
		previous = index
		found++
	}
	if found != 5 {
		t.Fatalf("%d classic parts, want 5", found)
	}
}

func TestNewPlayerOwnsOnlyTheStarterRobotAndEveryColor(t *testing.T) {
	garage := LoadRobotGarage(store.InDirectory(t.TempDir()))
	if garage.Look != lanternRobotLook {
		t.Fatalf("default look %+v", garage.Look)
	}
	owned := 0
	for kind := range RobotSlotColor {
		for part := range robotSlots[kind].Count {
			owned += boolToIndex(garage.Owns(kind, part))
		}
	}
	if owned != int(RobotSlotColor) {
		t.Fatalf("%d parts owned, want one per slot", owned)
	}
	for color := range robotSlots[RobotSlotColor].Count {
		if !garage.Owns(RobotSlotColor, color) {
			t.Fatalf("color %d is locked", color)
		}
	}
}

func TestSavedGarageReloadsAndRefusesLockedOrInvalidParts(t *testing.T) {
	saves := store.InDirectory(t.TempDir())
	saved := RobotGarage{Look: parasolRobotLook, Counts: [robotCounterCount]int{5000, 2}}
	saved.Look.Legs = RobotLegsRib
	saved.Look.Eyes = RobotEyesCyclops
	saved.Look.Head = RobotHeadCrook
	saved.Look.Abdomen = robotAbdomenCount + 3
	if err := saves.Save(robotGarageFileName, &saved); err != nil {
		t.Fatal(err)
	}
	loaded := LoadRobotGarage(saves)
	if loaded.Counts != saved.Counts || loaded.Look.Legs != RobotLegsRib {
		t.Fatalf("lost progress: %+v", loaded)
	}
	if loaded.Look.Head != defaultRobotLook.Head || loaded.Look.Abdomen != defaultRobotLook.Abdomen {
		t.Fatalf("kept a forbidden part: %+v", loaded.Look)
	}
}

func TestPlayerKillsCountAndDemoOrBossHitsDoNot(t *testing.T) {
	game := newHeadlessGame(t)
	definition := &enemyTable[EnemySwarmer]
	game.grantKillRewardsIf(0, 0, definition, false, false)
	if game.garage.Counts[RobotCounterKills] != 0 {
		t.Fatalf("a kill not made by the player counted")
	}
	game.grantKillRewardsIf(0, 0, definition, true, false)
	game.grantKillRewardsIf(0, 0, &enemyTable[EnemySpiderTank], true, false)
	if game.garage.Counts != [robotCounterCount]int{2, 1} {
		t.Fatalf("counts %v", game.garage.Counts)
	}
	game.isDemo = true
	game.countWormDefeat()
	game.isDemo = false
	game.countWormDefeat()
	if game.garage.Counts[RobotCounterBosses] != 2 {
		t.Fatalf("bosses %d", game.garage.Counts[RobotCounterBosses])
	}
}

func TestProgressIsSavedOnDeathAndKeptAcrossRuns(t *testing.T) {
	directory := t.TempDir()
	_, game := newTestAppIn(t, directory)
	game.grantKillRewardsIf(0, 0, &enemyTable[EnemySwarmer], true, false)
	game.player.Health = 0
	game.saveRobotGarageOnDeath()
	_, reloaded := newTestAppIn(t, directory)
	if reloaded.garage.Counts[RobotCounterKills] != 1 {
		t.Fatalf("counts %v", reloaded.garage.Counts)
	}
}

func TestLabNeverWritesTheGarage(t *testing.T) {
	directory := t.TempDir()
	_, game := newTestAppIn(t, directory)
	game.enableRobotLab()
	game.isGarageDirty = true
	game.saveRobotGarage()
	if err := game.app.Store.Load(robotGarageFileName, &RobotGarage{}); err == nil {
		t.Fatalf("the lab wrote a garage file")
	}
}

func TestMovingOnlyPreviewsAndFireEquipsOwnedParts(t *testing.T) {
	game := newHeadlessGame(t)
	screen := NewRobotGarageScreen(game)
	screen.Enter(game.app)
	before := game.robotLook
	screen.handleInput(input.ActionUp, false, false)
	screen.handleInput(input.ActionRight, false, false)
	if game.robotLook != before || game.isGarageDirty {
		t.Fatalf("moving equipped a part")
	}
	if screen.previewLook().Color == before.Color {
		t.Fatalf("the preview does not show the hovered color")
	}
	screen.handleInput(0, false, true)
	if game.robotLook.Color == before.Color || game.ui.art.Look != game.robotLook || !game.isGarageDirty {
		t.Fatalf("fire did not equip the color: %+v", game.robotLook)
	}
}

func TestFireNeverEquipsALockedPart(t *testing.T) {
	game := newHeadlessGame(t)
	screen := NewRobotGarageScreen(game)
	screen.Enter(game.app)
	before := game.robotLook
	screen.handleInput(input.ActionRight, false, false)
	screen.handleInput(0, false, true)
	if game.robotLook != before {
		t.Fatalf("a locked part was equipped")
	}
	if screen.previewLook() != before {
		t.Fatalf("a locked part is previewed")
	}
	if game.garage.DescribeUnlock(RobotSlotHead, screen.Columns[RobotSlotHead]) == "" {
		t.Fatalf("no unlock condition for the locked part")
	}
}

func TestCancelReturnsToTheActivePartThenLeaves(t *testing.T) {
	game := newHeadlessGame(t)
	screen := NewRobotGarageScreen(game)
	screen.Enter(game.app)
	screen.handleInput(input.ActionRight, false, false)
	if screen.handleInput(0, true, false) {
		t.Fatalf("cancel left the screen instead of returning the cursor")
	}
	if screen.Columns[RobotSlotHead] != int(game.robotLook.Head) || screen.previewLook() != game.robotLook {
		t.Fatalf("the cursor did not return to the active part")
	}
	if !screen.handleInput(0, true, false) {
		t.Fatalf("cancel on the active part must leave")
	}
}

func TestEveryThumbnailPaintsSomething(t *testing.T) {
	canvas := NewArtCanvas()
	for kind := range robotSlotCount {
		for part := range robotSlots[kind].Count {
			if canvas.paintRobotThumb(kind, part, RobotColorBlue, true).Empty() {
				t.Fatalf("slot %d part %d is empty", kind, part)
			}
		}
	}
}

func TestLockedThumbnailIsAFlatGraySilhouette(t *testing.T) {
	canvas := NewArtCanvas()
	bounds := canvas.paintRobotThumb(RobotSlotHead, int(RobotHeadHelmet), RobotColorRose, false)
	if bounds.Empty() {
		t.Fatalf("the silhouette is empty")
	}
	for index := 0; index < len(canvas.Final); index += 4 {
		isOpaque := canvas.Final[index+3] != 0
		isFlat := canvas.Final[index] == garageLockedGray && canvas.Final[index+1] == garageLockedGray && canvas.Final[index+2] == garageLockedGray
		if isOpaque && !isFlat {
			t.Fatalf("pixel %d keeps a detail", index/4)
		}
	}
}

func TestPreviewWalksWithoutTouchingTheGame(t *testing.T) {
	game := newHeadlessGame(t)
	preview := newRobotPreview(game)
	playerX := game.player.X
	for range 600 {
		preview.step(deltaSeconds)
	}
	if game.player.X != playerX {
		t.Fatalf("the preview moved the real player")
	}
	if preview.Game.player.X == arenaSize/2+garagePreviewOrbit {
		t.Fatalf("the preview robot did not travel")
	}
	distance := length(preview.Game.player.X-arenaSize/2, preview.Game.player.Y-arenaSize/2)
	if distance < garagePreviewOrbit-1 || distance > garagePreviewOrbit+1 {
		t.Fatalf("the preview left its orbit: %v", distance)
	}
}

func TestFreshPlayerWheelsStartUnderTheRobot(t *testing.T) {
	player := NewPlayer()
	for leg := range tsukumoLegCount {
		state := &player.Rig.Legs[leg]
		if length(state.FootX-player.X, state.FootY-player.Y) > tsukumoThighLength+tsukumoShinLength+60 {
			t.Fatalf("leg %d wheel starts at %v, %v, far from the robot", leg, state.FootX, state.FootY)
		}
	}
}
