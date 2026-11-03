package game

import (
	"testing"
)

// ===== Tests =====

func TestBossesFollowDiscoveryListThenLoopingStack(t *testing.T) {
	game := newHeadlessGame(t)
	expected := []BossEventKind{
		BossSpider, BossHorde, BossWorm,
		BossSpider, BossSpider, BossSpider, BossHorde, BossWorm,
		BossSpider, BossSpider, BossSpider, BossHorde, BossWorm,
	}
	for event, kind := range expected {
		if next := game.nextBossEvent(); next != kind {
			t.Fatalf("event %d: next boss %d, want %d", event+1, next, kind)
		}
		game.bossEventCount++
	}
}

func TestDiscoveryFreezesCounterUntilBossIsDefeated(t *testing.T) {
	game := newHeadlessGame(t)
	for _, kind := range bossDiscoveryList {
		triggerNextBoss(game)
		if game.livingBossCount() != 1 || !game.isBossCounterFrozen() {
			t.Fatalf("boss %d: %d alive, frozen %v", kind, game.livingBossCount(), game.isBossCounterFrozen())
		}
		spawnedBefore := game.bossEventCount
		triggerNextBoss(game)
		if game.bossEventCount != spawnedBefore {
			t.Fatalf("boss %d: a second boss came while the first lived", kind)
		}
		defeatEveryBoss(game)
		if game.isBossCounterFrozen() {
			t.Fatalf("boss %d: the counter stayed frozen after its defeat", kind)
		}
	}
}

func TestFrozenCounterIgnoresKillsButNotRewards(t *testing.T) {
	game := newHeadlessGame(t)
	triggerNextBoss(game)
	game.bossProgressKills = 0
	killsBefore := game.kills
	grantOrdinaryKill(game)
	if game.bossProgressKills != 0 || game.kills != killsBefore+1 {
		t.Fatalf("frozen: boss progress %d (want 0), kills %d (want %d)", game.bossProgressKills, game.kills, killsBefore+1)
	}
	defeatEveryBoss(game)
	grantOrdinaryKill(game)
	if game.bossProgressKills != 1 {
		t.Fatalf("after the defeat: boss progress %d, want 1", game.bossProgressKills)
	}
}

func TestStackSpawnsOneBossPerCounterUpToThreeAlive(t *testing.T) {
	game := newHeadlessGame(t)
	finishDiscovery(game)
	for alive := 1; alive <= maximumBossesAlive; alive++ {
		triggerNextBoss(game)
		if game.livingBossCount() != alive || game.bossProgressKills != 0 {
			t.Fatalf("stack spawn %d: %d alive, progress %d", alive, game.livingBossCount(), game.bossProgressKills)
		}
	}
	if !game.isBossCounterFrozen() {
		t.Fatalf("the counter still runs with %d bosses alive", maximumBossesAlive)
	}
	triggerNextBoss(game)
	if game.livingBossCount() != maximumBossesAlive {
		t.Fatalf("%d bosses alive, the cap is %d", game.livingBossCount(), maximumBossesAlive)
	}
}

func TestLastDiscoveryBossStillFreezesTheCounter(t *testing.T) {
	game := newHeadlessGame(t)
	game.bossEventCount = len(bossDiscoveryList) - 1
	triggerNextBoss(game)
	if len(game.worms) != 1 || !game.isBossCounterFrozen() {
		t.Fatalf("the last discovery boss did not freeze the counter: worms %d", len(game.worms))
	}
}

func TestHordeCountsTowardTheCap(t *testing.T) {
	game := newHeadlessGame(t)
	finishDiscovery(game)
	game.startHordeEvent()
	game.spawnSpiderEvent()
	game.spawnSpiderEvent()
	if game.livingBossCount() != maximumBossesAlive || !game.isBossCounterFrozen() {
		t.Fatalf("a horde and two spiders: %d alive, frozen %v", game.livingBossCount(), game.isBossCounterFrozen())
	}
}

func TestSeveralWormsLiveAndDieIndependently(t *testing.T) {
	game := newHeadlessGame(t)
	finishDiscovery(game)
	game.spawnWormEvent()
	game.spawnWormEvent()
	if len(game.worms) != 2 {
		t.Fatalf("%d drill-worms alive, want 2", len(game.worms))
	}
	game.worms[0].IsActive = false
	stepHeadlessIdle(game)
	if len(game.worms) != 1 || !game.worms[0].IsActive {
		t.Fatalf("killing one worm left %d", len(game.worms))
	}
}

func TestWormsSurfaceApartEvenOnAStillPlayer(t *testing.T) {
	game := newHeadlessGame(t)
	for range maximumBossesAlive {
		game.spawnWormEvent()
	}
	for range int(2 * wormMaximumBurrowSeconds * ticksPerSecond) {
		stepHeadlessIdle(game)
	}
	for first := range game.worms {
		for second := first + 1; second < len(game.worms); second++ {
			assertWormsApart(t, &game.worms[first], &game.worms[second])
		}
	}
}

func TestEveryHordeIsLargerThanTheLast(t *testing.T) {
	game := newHeadlessGame(t)
	for wave := 1; wave <= 3; wave++ {
		game.hordeEvent, game.enemies.Count = HordeEvent{}, 0
		game.startHordeEvent()
		if game.hordeEvent.Total != hordeEventBaseSize*wave {
			t.Fatalf("horde %d has %d enemies, want %d", wave, game.hordeEvent.Total, hordeEventBaseSize*wave)
		}
	}
}

func TestBossKillsRequiredGrowWithRankAndNeverShrinkWithTime(t *testing.T) {
	expected := map[int]int{0: 300, 1: 423, 2: 552, 3: 583, 4: 105, 5: 124, 6: 958, 8: 1234, 9: 206}
	for rank, kills := range expected {
		if got := computeBossKillsRequired(rank, 0); got != kills {
			t.Fatalf("rank %d: %d kills, want %d", rank, got, kills)
		}
	}
	if got := computeBossKillsRequired(3, 16*60); got != 1167 {
		t.Fatalf("fourth boss at 16 minutes: %d kills, want 1167", got)
	}
	if got := computeBossKillsRequired(3, 60); got != 583 {
		t.Fatalf("a fast player got %d kills, want the base 583", got)
	}
}

func TestOnlyTheRepeatedSpiderTanksOfTheStackNeedFewerKills(t *testing.T) {
	expected := []bool{false, false, false, false, true, true, false, false, false, true, true, false}
	for bossesSpawned, isRepeat := range expected {
		if got := isStackRepeat(bossesSpawned); got != isRepeat {
			t.Fatalf("boss %d: repeat %v, want %v", bossesSpawned, got, isRepeat)
		}
	}
}

func TestHordeHuntsFasterFarAwayAndAsItDwindles(t *testing.T) {
	game := newHeadlessGame(t)
	game.player.MoveSpeed = 1000
	game.startHordeEvent()
	index := game.enemies.Count - 1
	near := game.hordeHuntSpeed(index, 40, hordeHuntNearDistance)
	far := game.hordeHuntSpeed(index, 40, hordeHuntNearDistance+hordeHuntDistanceRange)
	if near != 40 || far != 40*(1+hordeHuntDistanceBoost) {
		t.Fatalf("a full horde hunts at %.0f near and %.0f far", near, far)
	}
	game.hordeEvent.Remaining = 0
	if last := game.hordeHuntSpeed(index, 40, hordeHuntNearDistance); last != 40*(1+hordeHuntDwindleBoost) {
		t.Fatalf("the last member hunts at %.0f", last)
	}
}

func TestHordeHuntNeverOutrunsThePlayerNorSlowsAFastEnemy(t *testing.T) {
	game := newHeadlessGame(t)
	game.player.MoveSpeed = 200
	game.startHordeEvent()
	index := game.enemies.Count - 1
	if capped := game.hordeHuntSpeed(index, 100, 5000); capped != 180 {
		t.Fatalf("hunting speed %.0f, want the cap of 180", capped)
	}
	if fast := game.hordeHuntSpeed(index, 190, 5000); fast != 190 {
		t.Fatalf("a fast enemy went from 190 to %.0f", fast)
	}
	if frozen := game.hordeHuntSpeed(index, 0, 5000); frozen != 0 {
		t.Fatalf("a frozen enemy moves at %.0f", frozen)
	}
}

func TestOrdinaryEnemiesDoNotHunt(t *testing.T) {
	game := newHeadlessGame(t)
	game.spawnEnemyAt(EnemyBrute, game.player.X+2000, game.player.Y)
	if speed := game.hordeHuntSpeed(game.enemies.Count-1, 40, 2000); speed != 40 {
		t.Fatalf("an ordinary enemy hunts at %.0f", speed)
	}
}

func TestDemosNeverSummonBosses(t *testing.T) {
	game := newHeadlessGame(t)
	game.isDemo = true
	game.bossProgressKills = game.bossKillsRequired
	game.startBossEventIfDue()
	if game.bossEventCount != 0 {
		t.Fatalf("a demo summoned a boss")
	}
}

// ===== Helpers =====

func assertWormsApart(t *testing.T, first, second *DrillWorm) {
	t.Helper()
	distance := length(first.HoleX-second.HoleX, first.HoleY-second.HoleY)
	if distance < wormMinimumSpacing-1 {
		t.Fatalf("two drill-worms surfaced %.0f apart, want %d", distance, wormMinimumSpacing)
	}
}

func triggerNextBoss(game *Game) {
	game.bossProgressKills = game.bossKillsRequired
	game.startBossEventIfDue()
}

func grantOrdinaryKill(game *Game) {
	game.grantKillRewardsIf(0, 0, &enemyTable[EnemySwarmer], true, false)
}

func defeatEveryBoss(game *Game) {
	game.spiders, game.worms, game.hordeEvent = game.spiders[:0], game.worms[:0], HordeEvent{}
	game.enemies.Count = 0
}

func finishDiscovery(game *Game) {
	game.bossEventCount = len(bossDiscoveryList) + 1
}
