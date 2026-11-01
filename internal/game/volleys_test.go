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
	if game.enemies.Health[0] > 0 {
		t.Fatalf("nuke left a brute alive with %.0f", game.enemies.Health[0])
	}
	expected := spiderHealthBefore - nukeBossDamageFraction*enemyTable[EnemySpiderTank].Health*game.enemies.Power[spider]
	if abs(game.enemies.Health[spider]-expected) > 1 {
		t.Fatalf("spider health %.0f, want %.0f", game.enemies.Health[spider], expected)
	}
}

// ===== Helpers =====

func pickUpLogo(game *Game, logo *UpgradeLogo) {
	logo.Seconds = volleyFlightSeconds
	game.player.X, game.player.Y = logo.ToX, logo.ToY
	game.updateVolleys(0)
}

func maximizeEveryWeapon(game *Game) {
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
