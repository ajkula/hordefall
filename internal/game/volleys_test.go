package game

import (
	"testing"
)

// ===== Tests =====

func TestLevelUpGrantsPassiveAndBurstsVolley(t *testing.T) {
	game := newHeadlessGame(t)
	passivesBefore := sumPassiveLevels(game.player)
	game.player.PendingLevelUps = 1
	game.resolveLevelUps()
	if sumPassiveLevels(game.player) != passivesBefore+1 {
		t.Fatalf("level up did not grant the next passive")
	}
	if game.volleyCount != 1 || game.volleys[0].Count != offersPerLevel {
		t.Fatalf("volley count %d, logos %d", game.volleyCount, game.volleys[0].Count)
	}
	assertNoDuplicateOffer(t, &game.volleys[0])
}

func TestLogosSpreadEvenlyAndStayInsideTheArena(t *testing.T) {
	game := newHeadlessGame(t)
	game.player.X, game.player.Y = volleyArenaMargin, volleyArenaMargin
	game.burstVolley()
	volley := &game.volleys[0]
	for slot := range volley.Count {
		logo := &volley.Logos[slot]
		isInside := logo.ToX >= volleyArenaMargin && logo.ToX <= arenaSize-volleyArenaMargin && logo.ToY >= volleyArenaMargin && logo.ToY <= arenaSize-volleyArenaMargin
		if !isInside {
			t.Fatalf("logo %d lands outside the arena at %.0f,%.0f", slot, logo.ToX, logo.ToY)
		}
	}
}

func TestOnlyTwoVolleysWaitOnTheGround(t *testing.T) {
	game := newHeadlessGame(t)
	game.player.PendingLevelUps = 4
	game.resolveLevelUps()
	if game.volleyCount != maximumVolleys || game.pendingVolleys != 2 {
		t.Fatalf("volleys %d on the ground, %d pending", game.volleyCount, game.pendingVolleys)
	}
}

func TestLogoCannotBePickedBeforeItSettles(t *testing.T) {
	game := newHeadlessGame(t)
	game.burstVolley()
	logo := &game.volleys[0].Logos[0]
	game.player.X, game.player.Y = logo.ToX, logo.ToY
	game.updateVolleys(volleyFlightSeconds / 2)
	if game.volleyCount != 1 {
		t.Fatalf("a flying logo was picked up")
	}
	game.updateVolleys(volleyFlightSeconds)
	if game.volleyCount != 0 {
		t.Fatalf("a settled logo under the tachikoma was not picked up")
	}
}

func TestPickupAppliesOfferAndBurstsPendingVolley(t *testing.T) {
	game := newHeadlessGame(t)
	game.player.PendingLevelUps = 3
	game.resolveLevelUps()
	game.volleys[0].Logos[0].Offer = UpgradeOffer{Kind: UpgradeWeapon, Weapon: WeaponFrostNova, NextLevel: 1}
	logo := game.volleys[0].Logos[0]
	weaponsBefore := len(game.player.Weapons)
	pickUpLogo(game, &game.volleys[0].Logos[0])
	isApplied := game.player.WeaponLevel(logo.Offer.Weapon) == logo.Offer.NextLevel || len(game.player.Weapons) != weaponsBefore
	if !isApplied {
		t.Fatalf("picked %s level %d but it was not applied", weaponTable[logo.Offer.Weapon].Key, logo.Offer.NextLevel)
	}
	if game.volleyCount != maximumVolleys || game.pendingVolleys != 0 {
		t.Fatalf("after pickup: volleys %d, pending %d", game.volleyCount, game.pendingVolleys)
	}
}

func TestOtherVolleyShowsTheLevelAfterAPickup(t *testing.T) {
	game := newHeadlessGame(t)
	game.player.Weapons = []WeaponState{{Kind: WeaponOrbitBlades, Level: 2}}
	game.volleys[0] = Volley{Count: 1, Logos: [offersPerLevel]UpgradeLogo{{Offer: UpgradeOffer{Kind: UpgradeWeapon, Weapon: WeaponOrbitBlades, NextLevel: 3}}}}
	game.volleys[1] = Volley{Count: 1, Logos: [offersPerLevel]UpgradeLogo{{Offer: UpgradeOffer{Kind: UpgradeWeapon, Weapon: WeaponOrbitBlades, NextLevel: 3}}}}
	game.volleyCount = 2
	pickUpLogo(game, &game.volleys[0].Logos[0])
	if game.volleyCount != 1 || game.volleys[0].Logos[0].Offer.NextLevel != 4 {
		t.Fatalf("remaining logo shows level %d, want 4", game.volleys[0].Logos[0].Offer.NextLevel)
	}
}

func TestInvalidLogoIsReplacedOrExplodes(t *testing.T) {
	game := newHeadlessGame(t)
	game.player.Weapons = []WeaponState{{Kind: WeaponOrbitBlades, Level: maximumWeaponLevel - 1}}
	game.volleys[0] = Volley{Count: 1, Logos: [offersPerLevel]UpgradeLogo{{Offer: UpgradeOffer{Kind: UpgradeWeapon, Weapon: WeaponOrbitBlades, NextLevel: maximumWeaponLevel}}}}
	game.volleys[1] = Volley{Count: 1, Logos: [offersPerLevel]UpgradeLogo{{Offer: UpgradeOffer{Kind: UpgradeWeapon, Weapon: WeaponOrbitBlades, NextLevel: maximumWeaponLevel}}}}
	game.volleyCount = 2
	pickUpLogo(game, &game.volleys[0].Logos[0])
	if game.volleyCount != 1 || game.volleys[0].Logos[0].Offer.Weapon == WeaponOrbitBlades {
		t.Fatalf("a maxed weapon is still offered on the ground")
	}
	maximizeEveryWeapon(game)
	game.volleys[0] = Volley{Count: 1, Logos: [offersPerLevel]UpgradeLogo{{IsRoll: true}}}
	game.volleys[1] = Volley{Count: 1, Logos: [offersPerLevel]UpgradeLogo{{Offer: UpgradeOffer{Kind: UpgradeWeapon, Weapon: WeaponOrbitBlades, NextLevel: maximumWeaponLevel}}}}
	game.volleyCount = 2
	pickUpLogo(game, &game.volleys[0].Logos[0])
	if game.volleyCount != 0 {
		t.Fatalf("an invalid logo with an empty pool did not explode")
	}
}

func TestEmptyPoolBurstsARoll(t *testing.T) {
	game := newHeadlessGame(t)
	maximizeEveryWeapon(game)
	game.burstVolley()
	logo := &game.volleys[0].Logos[0]
	if game.volleys[0].Count != 1 || !logo.IsRoll {
		t.Fatalf("expected a single roll logo")
	}
	first, _ := logo.RollBonus()
	logo.Seconds = rollBonusSeconds + 0.01
	if next, _ := logo.RollBonus(); next != (first+1)%rollBonusCount {
		t.Fatalf("roll did not rotate after %d s", rollBonusSeconds)
	}
}

func TestNukeKillsHordeAndHurtsBosses(t *testing.T) {
	game := newHeadlessGame(t)
	game.spawnEnemyAt(EnemyBrute, game.player.X+60, game.player.Y)
	game.spawnSpiderAt(0, 200)
	spider := game.enemies.Count - 1
	spiderHealthBefore := game.enemies.Health[spider]
	game.detonateNuke()
	finishNukeBlasts(game)
	if game.enemies.Health[0] > 0 {
		t.Fatalf("nuke left a brute alive with %.0f", game.enemies.Health[0])
	}
	expected := spiderHealthBefore - nukeBossDamageFraction*enemyTable[EnemySpiderTank].Health*game.enemies.Power[spider]
	if abs(game.enemies.Health[spider]-expected) > 1 {
		t.Fatalf("spider health %.0f, want %.0f", game.enemies.Health[spider], expected)
	}
}

func TestLifestealIsAChoiceNeverGrantedByRotation(t *testing.T) {
	game := newHeadlessGame(t)
	for range 200 {
		game.grantNextPassive()
	}
	if level := game.player.PassiveLevels[PassiveLifesteal]; level != 0 {
		t.Fatalf("the rotation granted lifesteal, level %d", level)
	}
}

func TestLifestealLogoIsTakenOnceAndThenLeavesTheGround(t *testing.T) {
	game := newHeadlessGame(t)
	maximizeEveryWeapon(game)
	game.player.PassiveLevels[PassiveLifesteal] = 0
	game.burstVolley()
	game.burstVolley()
	first, second := &game.volleys[0], &game.volleys[1]
	if first.Count != 1 || first.Logos[0].Offer.Kind != UpgradePassive || second.Logos[0].Offer.Kind != UpgradePassive {
		t.Fatalf("the two volleys should offer lifesteal: %+v", first.Logos[0].Offer)
	}
	pickUpLogo(game, &first.Logos[0])
	if game.player.KillHealFraction != lifestealKillHealFraction {
		t.Fatalf("picking the logo did not grant lifesteal")
	}
	if game.volleyCount != 0 {
		t.Fatalf("%d volleys remain, the second lifesteal logo should explode", game.volleyCount)
	}
	game.burstVolley()
	if !game.volleys[0].Logos[0].IsRoll {
		t.Fatalf("lifesteal was offered again after being taken")
	}
}

func TestNukeFrontReachesFarEnemiesLater(t *testing.T) {
	game := newHeadlessGame(t)
	x, y := game.player.X, game.player.Y
	game.spawnEnemyAt(EnemyBrute, x+50, y)
	game.spawnEnemyAt(EnemyBrute, x+nukeRadius-20, y)
	game.detonateNuke()
	game.updateNukeBlasts(deltaSeconds)
	if game.enemies.Health[0] > 0 || game.enemies.Health[1] <= 0 {
		t.Fatalf("after one tick: near %.0f, far %.0f", game.enemies.Health[0], game.enemies.Health[1])
	}
	finishNukeBlasts(game)
	if game.enemies.Health[1] > 0 {
		t.Fatalf("the front never reached the far enemy")
	}
}

func TestNukeHitsABossOnlyOncePerBlast(t *testing.T) {
	game := newHeadlessGame(t)
	game.spawnSpiderAt(0, 200)
	spider := game.enemies.Count - 1
	before := game.enemies.Health[spider]
	game.detonateNuke()
	finishNukeBlasts(game)
	expected := before - nukeBossDamageFraction*enemyTable[EnemySpiderTank].Health*game.enemies.Power[spider]
	if abs(game.enemies.Health[spider]-expected) > 1 {
		t.Fatalf("boss health %.0f, want one hit to %.0f", game.enemies.Health[spider], expected)
	}
}

func TestNukeIgnitesTheWholeOilPuddleItTouches(t *testing.T) {
	game := newHeadlessGame(t)
	column, row := paintDistantPuddle(game, GroundOil)
	game.detonateNuke()
	finishNukeBlasts(game)
	if farCell := game.ground.MaterialAtCell(column+puddleTestReachCells, row); farCell != GroundFire {
		t.Fatalf("the far end of the oil puddle is %d, want fire", farCell)
	}
}

func TestNukeSteamsWaterAndLeavesIt(t *testing.T) {
	game := newHeadlessGame(t)
	column, row := paintDistantPuddle(game, GroundWater)
	cloudsBefore := len(game.effects.Smokes)
	game.effects.Density = 1
	game.detonateNuke()
	game.nukeBlasts[0].Seconds = nukeGrowthSeconds / 2
	game.updateNukeBlasts(deltaSeconds)
	finishNukeBlasts(game)
	if len(game.effects.Smokes) <= cloudsBefore {
		t.Fatalf("no steam rose from the water")
	}
	if farCell := game.ground.MaterialAtCell(column+puddleTestReachCells, row); farCell != GroundWater {
		t.Fatalf("the water became %d", farCell)
	}
}

func TestSystemClockOptionStartsOffAndToggles(t *testing.T) {
	game := newHeadlessGame(t)
	if game.isSystemClockShown() {
		t.Fatalf("the system clock is shown by default")
	}
	option := game.gameplayOptions()[0]
	option.Activate()
	if !game.isSystemClockShown() {
		t.Fatalf("the option did not turn the clock on")
	}
	option.Activate()
	if game.isSystemClockShown() {
		t.Fatalf("the option did not turn the clock off")
	}
}

func TestNukeMeltsTheWholeIceSheetItTouches(t *testing.T) {
	game := newHeadlessGame(t)
	column, row := paintDistantPuddle(game, GroundIce)
	game.detonateNuke()
	finishNukeBlasts(game)
	if farCell := game.ground.MaterialAtCell(column+puddleTestReachCells, row); farCell != GroundWater {
		t.Fatalf("the far end of the ice is %d, want water", farCell)
	}
}

func TestNukeBurnsGrassAndDirtInsideItsRadiusAroundAClearing(t *testing.T) {
	game := newHeadlessGame(t)
	centerColumn, centerRow := int(game.player.X)/groundCellSize, int(game.player.Y)/groundCellSize
	game.ground.PaintCircle(centerColumn, centerRow, puddleTestReachCells+20, GroundGrass)
	game.detonateNuke()
	finishNukeBlasts(game)
	reachCells := puddleTestReachCells - 2
	clearingCells := nukeClearingRadius/groundCellSize - 1
	if count := countMaterialWithin(game, centerColumn, centerRow, clearingCells, GroundFire); count != 0 {
		t.Fatalf("%d burning cells inside the clearing", count)
	}
	burning := countMaterialWithin(game, centerColumn, centerRow, reachCells, GroundFire)
	if burning == 0 {
		t.Fatalf("the grass inside the radius did not catch fire")
	}
	outsideColumn := centerColumn + puddleTestReachCells + 5
	if game.ground.MaterialAtCell(outsideColumn, centerRow) != GroundGrass {
		t.Fatalf("the grass burned outside the radius")
	}
}

func TestNukeFireTriggersEnemyReactionsAndStillKills(t *testing.T) {
	game := newHeadlessGame(t)
	x, y := game.player.X, game.player.Y
	game.spawnEnemyAt(EnemyBrute, x+200, y)
	game.enemies.ApplyStatus(0, StatusWet)
	game.spawnEnemyAt(EnemyBrute, x-200, y)
	game.enemies.ApplyStatus(1, StatusOiled)
	game.detonateNuke()
	finishNukeBlasts(game)
	if game.enemies.Health[0] > 0 || game.enemies.Health[1] > 0 {
		t.Fatalf("health left: wet %.0f, oiled %.0f", game.enemies.Health[0], game.enemies.Health[1])
	}
	if game.reactionCounts[ReactionExtinguish] != 1 || game.reactionCounts[ReactionInferno] != 1 {
		t.Fatalf("reactions: hiss %d, inferno %d", game.reactionCounts[ReactionExtinguish], game.reactionCounts[ReactionInferno])
	}
}

func TestNukeOnAWetBossIsHalvedLikeAnyFire(t *testing.T) {
	game := newHeadlessGame(t)
	game.spawnSpiderAt(0, 200)
	spider := game.enemies.Count - 1
	game.enemies.ApplyStatus(spider, StatusWet)
	before := game.enemies.Health[spider]
	game.detonateNuke()
	finishNukeBlasts(game)
	expected := before - nukeBossDamageFraction*enemyTable[EnemySpiderTank].Health*game.enemies.Power[spider]/2
	if abs(game.enemies.Health[spider]-expected) > 1 {
		t.Fatalf("wet boss health %.0f, want %.0f", game.enemies.Health[spider], expected)
	}
}

// ===== Helpers =====

func countMaterialWithin(game *Game, centerColumn, centerRow, radiusCells int, material GroundKind) int {
	count := 0
	for row := centerRow - radiusCells; row <= centerRow+radiusCells; row++ {
		for column := centerColumn - radiusCells; column <= centerColumn+radiusCells; column++ {
			isInside := (column-centerColumn)*(column-centerColumn)+(row-centerRow)*(row-centerRow) <= radiusCells*radiusCells
			count += boolToIndex(isInside && game.ground.MaterialAtCell(column, row) == material)
		}
	}
	return count
}

const puddleTestReachCells = nukeRadius / groundCellSize

func paintDistantPuddle(game *Game, material GroundKind) (column, row int) {
	column, row = int(game.player.X)/groundCellSize+puddleTestReachCells/2, int(game.player.Y)/groundCellSize
	for offset := range puddleTestReachCells + 1 {
		game.ground.PaintCircle(column+offset, row, 1, material)
	}
	return column, row
}

func finishNukeBlasts(game *Game) {
	for len(game.nukeBlasts) > 0 {
		game.updateNukeBlasts(deltaSeconds)
	}
}

func pickUpLogo(game *Game, logo *UpgradeLogo) {
	logo.Seconds = volleyFlightSeconds
	game.player.X, game.player.Y = logo.ToX, logo.ToY
	game.updateVolleys(0)
}

func maximizeEveryWeapon(game *Game) {
	game.player.PassiveLevels[PassiveLifesteal] = passiveTable[PassiveLifesteal].MaximumLevel
	game.player.Weapons = game.player.Weapons[:0]
	for _, kind := range []WeaponKind{WeaponVoltBolt, WeaponOrbitBlades, WeaponFrostNova, WeaponArcLightning, WeaponSeismicHammer} {
		game.player.Weapons = append(game.player.Weapons, WeaponState{Kind: kind, Level: maximumWeaponLevel})
	}
}

func sumPassiveLevels(player *Player) int {
	total := 0
	for _, level := range player.PassiveLevels {
		total += level
	}
	return total
}

func assertNoDuplicateOffer(t *testing.T, volley *Volley) {
	t.Helper()
	seen := map[WeaponKind]bool{}
	for slot := range volley.Count {
		kind := volley.Logos[slot].Offer.Weapon
		if seen[kind] {
			t.Fatalf("volley offers %s twice", weaponTable[kind].Key)
		}
		seen[kind] = true
	}
}
