package game

import "testing"

// ===== Tests =====

func TestABurstHurtsThreePartsOfABodyWithFallingDamage(t *testing.T) {
	game, parts := newGameWithBodyParts(t, 5)
	game.ApplyBurst(Burst{X: 1000, Y: 1000, Radius: 200, Damage: 100, Element: ElementPhysical})
	wantedLosses := []float32{100, 25, 6.25, 0, 0}
	for rank, wanted := range wantedLosses {
		if lost := bodyPartHealthLost(game, parts[rank]); lost != wanted {
			t.Fatalf("part %d lost %v, want %v", rank, lost, wanted)
		}
	}
}

func TestOnlyTheNearestBodyPartCatchesAStatus(t *testing.T) {
	game, parts := newGameWithBodyParts(t, 3)
	game.ApplyBurst(Burst{X: 1000, Y: 1000, Radius: 200, Damage: 1, Element: ElementFire})
	for rank, isBurningWanted := range []bool{true, false, false} {
		index := game.enemies.IndexOfID(parts[rank])
		if isBurning := game.enemies.Status[index]&StatusBurning != 0; isBurning != isBurningWanted {
			t.Fatalf("part %d burning %v, want %v", rank, isBurning, isBurningWanted)
		}
	}
}

func TestEachBodyCountsItsOwnThreeParts(t *testing.T) {
	game, parts := newGameWithBodyParts(t, 4)
	game.enemies.BodyID[game.enemies.IndexOfID(parts[3])] = testOtherBodyID
	game.ApplyBurst(Burst{X: 1000, Y: 1000, Radius: 200, Damage: 100, Element: ElementPhysical})
	if lost := bodyPartHealthLost(game, parts[3]); lost != 100 {
		t.Fatalf("the nearest part of the other body lost %v, want 100", lost)
	}
}

func TestABurstHurtsOrdinaryEnemiesInFull(t *testing.T) {
	game, _ := newGameWithBodyParts(t, 0)
	for offset := range 5 {
		game.enemies.Spawn(EnemyBrute, 1000, 1000+float32(offset)*10, 1)
	}
	game.grid.Rebuild(game.enemies.PositionX, game.enemies.PositionY, game.enemies.Count)
	game.ApplyBurst(Burst{X: 1000, Y: 1000, Radius: 200, Damage: 10, Element: ElementPhysical})
	for index := range 5 {
		if lost := enemyTable[EnemyBrute].Health - game.enemies.Health[index]; lost != 10 {
			t.Fatalf("enemy %d lost %v, want 10", index, lost)
		}
	}
}

func TestArcLightningLeavesABodyAfterThreeParts(t *testing.T) {
	game, parts := newGameWithBodyParts(t, 5)
	game.player.X, game.player.Y, game.player.AimX, game.player.AimY = 1000, 960, 0, 1
	straggler := game.enemies.Spawn(EnemyBrute, 1000, 1045, 1)
	game.grid.Rebuild(game.enemies.PositionX, game.enemies.PositionY, game.enemies.Count)
	definition := &weaponTable[WeaponArcLightning]
	game.castArcLightning(&WeaponState{Kind: WeaponArcLightning, Level: maximumWeaponLevel}, definition)
	damage := definition.DamageAt(maximumWeaponLevel) * game.player.DamageMultiplier
	for rank, wanted := range []float32{damage, damage * 0.25, damage * 0.0625, 0, 0} {
		if lost := bodyPartHealthLost(game, parts[rank]); lost != wanted {
			t.Fatalf("part %d lost %v, want %v", rank, lost, wanted)
		}
	}
	if game.enemies.Health[game.enemies.IndexOfID(straggler)] >= enemyTable[EnemyBrute].Health {
		t.Fatalf("the arc did not go on to another enemy after leaving the body")
	}
}

func TestAWormDoesNotHurtItselfWithoutOutsideHelp(t *testing.T) {
	game := newHeadlessGame(t)
	game.spawnWormEvent()
	game.worms[0].Phase = WormHeadRolling
	game.adoptWormPart(&game.worms[0], game.enemies.Spawn(EnemyDrillWorm, 1000, 1000, 1), testBodyPartHealth)
	game.grid.Rebuild(game.enemies.PositionX, game.enemies.PositionY, game.enemies.Count)
	game.worms[0].HeadX, game.worms[0].HeadY = 1000, 1000
	game.crushUnderWormHead(&game.worms[0], 1)
	game.slamWorm(&game.worms[0], 1000, 1000)
	if game.enemies.Health[0] != testBodyPartHealth {
		t.Fatalf("the worm's own attacks took it to %v, want %v", game.enemies.Health[0], float32(testBodyPartHealth))
	}
}

func TestAWetBodyPropagatesToTwoPartsThenDriesAtOnce(t *testing.T) {
	game, parts := newGameWithBodyParts(t, 14)
	for _, id := range parts {
		game.enemies.ApplyStatus(game.enemies.IndexOfID(id), StatusWet)
	}
	game.HitEnemy(game.enemies.IndexOfID(parts[0]), 10, ElementShock)
	game.processBursts()
	fullDamage := reactionTable[ReactionElectrocute].AreaDamage * game.player.DamageMultiplier * reactionTable[ReactionElectrocute].DamageMultiplier
	hurtParts := 0
	for rank, id := range parts {
		lost := bodyPartHealthLost(game, id)
		hurtParts += boolToIndex(lost > 0)
		if rank > 0 && lost > 0 && lost != fullDamage {
			t.Fatalf("part %d lost %v, a conducted hit must stay at full damage %v", rank, lost, fullDamage)
		}
		if game.enemies.Status[game.enemies.IndexOfID(id)]&StatusWet != 0 {
			t.Fatalf("a part is still wet after the propagation")
		}
	}
	if hurtParts != 3 {
		t.Fatalf("%d parts were hurt, want the touched part and two others", hurtParts)
	}
}

func TestWetOrdinaryEnemiesStillChainFreely(t *testing.T) {
	game, _ := newGameWithBodyParts(t, 0)
	for offset := range 8 {
		index := game.enemies.IndexOfID(game.enemies.Spawn(EnemyBrute, 1000, 1000+float32(offset)*30, 1))
		game.enemies.ApplyStatus(index, StatusWet)
	}
	game.grid.Rebuild(game.enemies.PositionX, game.enemies.PositionY, game.enemies.Count)
	game.HitEnemy(0, 10, ElementShock)
	game.processBursts()
	for index := range 8 {
		if game.enemies.Health[index] >= enemyTable[EnemyBrute].Health {
			t.Fatalf("wet enemy %d was not reached by the chain", index)
		}
	}
}

func TestADirectHitOnADryBodyReachesTwoOtherPartsAtMost(t *testing.T) {
	game, parts := newGameWithBodyParts(t, 14)
	for _, id := range parts {
		game.enemies.ApplyStatus(game.enemies.IndexOfID(id), StatusOiled)
	}
	game.HitEnemy(game.enemies.IndexOfID(parts[5]), 10, ElementFire)
	game.processBursts()
	hurtParts := 0
	for _, id := range parts {
		hurtParts += boolToIndex(bodyPartHealthLost(game, id) > 0)
	}
	if hurtParts != 3 {
		t.Fatalf("%d parts were hurt, want the touched part and two others", hurtParts)
	}
}

// ===== Internal =====

const (
	testBodyID         = 1
	testOtherBodyID    = 2
	testBodyPartHealth = 100000
)

func newGameWithBodyParts(t *testing.T, count int) (*Game, []uint32) {
	game := newHeadlessGame(t)
	game.enemies.Count = 0
	parts := make([]uint32, count)
	for rank := range parts {
		parts[rank] = game.enemies.Spawn(EnemyDrillWormSegment, 1000, 1000+float32(rank+1)*10, 1)
		index := game.enemies.IndexOfID(parts[rank])
		game.enemies.Health[index], game.enemies.BodyID[index] = testBodyPartHealth, testBodyID
	}
	game.grid.Rebuild(game.enemies.PositionX, game.enemies.PositionY, game.enemies.Count)
	return game, parts
}

func bodyPartHealthLost(game *Game, id uint32) float32 {
	return testBodyPartHealth - game.enemies.Health[game.enemies.IndexOfID(id)]
}
