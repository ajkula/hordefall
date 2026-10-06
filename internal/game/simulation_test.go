package game

import (
	"testing"
	"time"

	"hordefall/internal/spatial"

	"hordefall/internal/rng"

	"hordefall/internal/audio"
)

// ===== Constants =====

const simulatedMinutes = 8

// ===== Public API =====

func TestLongRunWithEveryWeaponStaysStable(t *testing.T) {
	game := newHeadlessGame()
	equipEveryWeapon(game)
	frameCount := simulatedMinutes * 60 * ticksPerSecond
	slowestFrame := time.Duration(0)
	for frame := range frameCount {
		started := time.Now()
		stepHeadless(game)
		slowestFrame = max(slowestFrame, time.Since(started)*time.Duration(boolToIndex(frame > frameCount-600)))
	}
	t.Logf("enemies=%d kills=%d slowest late frame=%v reactions=%v", game.enemies.Count, game.kills, slowestFrame, game.reactionCounts)
}

func BenchmarkSimulateFullHorde(b *testing.B) {
	game := newHeadlessGame()
	equipEveryWeapon(game)
	for range 5 * 60 * ticksPerSecond {
		stepHeadless(game)
	}
	b.ResetTimer()
	for b.Loop() {
		stepHeadless(game)
	}
	b.ReportMetric(float64(game.enemies.Count), "enemies")
}

// ===== Internal =====

func newHeadlessGame() *Game {
	game := &Game{
		random:      rng.New(0xA11CE),
		enemies:     NewEnemyStore(maximumEnemies),
		projectiles: NewProjectileStore(maximumProjectiles),
		gems:        NewGemStore(maximumGems),
		effects:     NewEffects(),
		grid:        spatial.NewGrid(arenaSize, spatialCellSize, maximumEnemies),
		musicSlot:   noMusicSlot,
	}
	game.resetRun()
	game.state = StatePlaying
	return game
}

func equipEveryWeapon(game *Game) {
	game.player.Weapons = game.player.Weapons[:0]
	for kind := range weaponKindCount {
		game.player.Weapons = append(game.player.Weapons, WeaponState{Kind: kind, Level: maximumWeaponLevel})
	}
}

func stepHeadless(game *Game) {
	game.frame++
	game.controls = Controls{MoveX: cosine(float32(game.frame) * 0.002), MoveY: sine(float32(game.frame) * 0.002), Held: ActionFire}
	game.player.Health = game.player.MaximumHealth
	game.player.PendingLevelUps = 0
	game.simulate()
}

func BenchmarkSimulateTenThousandEnemies(b *testing.B) {
	game := newHeadlessGame()
	for spawned := range 10000 {
		angle := float32(spawned) * 0.6180339 * 6.2831853
		distance := 300 + float32(spawned%400)*2.5
		game.spawnEnemyAt(EnemyKind(spawned%int(enemyKindCount)), game.player.X+cosine(angle)*distance, game.player.Y+sine(angle)*distance)
	}
	game.director.accumulator = -1e9
	game.player.Weapons = game.player.Weapons[:0]
	b.ResetTimer()
	for b.Loop() {
		stepHeadless(game)
	}
	b.ReportMetric(float64(game.enemies.Count), "enemies")
}

func BenchmarkGroundRefreshAndTick(b *testing.B) {
	ground := NewGround(arenaSize, 1)
	for b.Loop() {
		ground.RefreshPixels(1)
		ground.Tick()
	}
}

func TestLifestealIsOneTimeAndCapped(t *testing.T) {
	game := newHeadlessGame()
	player := game.player
	game.applyOffer(UpgradeOffer{Kind: UpgradePassive, Passive: PassiveLifesteal, NextLevel: 1})
	if abs(player.KillHealFraction-0.01) > 1e-6 {
		t.Fatalf("lifesteal fraction %.3f, want 0.01", player.KillHealFraction)
	}
	if offers := game.appendPassiveOffer(nil, PassiveLifesteal); len(offers) != 0 {
		t.Fatalf("lifesteal offered again after being taken")
	}
	player.Health = 10
	player.KillHealBudget = maximumKillHealPerSecond * player.MaximumHealth
	for range 100 {
		game.HealFromKill()
	}
	if healed := player.Health - 10; healed > maximumKillHealPerSecond*player.MaximumHealth+1e-3 {
		t.Fatalf("healed %.1f in one instant, cap is %.1f", healed, maximumKillHealPerSecond*player.MaximumHealth)
	}
}

func TestExperienceCostAlwaysGrows(t *testing.T) {
	for level := 1; level < 200; level++ {
		if experienceForLevel(level+1) <= experienceForLevel(level) {
			t.Fatalf("level %d costs %d, level %d costs %d", level, experienceForLevel(level), level+1, experienceForLevel(level+1))
		}
	}
	t.Logf("cost at level 1=%d 10=%d 30=%d", experienceForLevel(1), experienceForLevel(10), experienceForLevel(30))
}

func TestNoMoreThanThreeSpiderTanks(t *testing.T) {
	game := newHeadlessGame()
	for range 10 {
		game.bossProgressKills = game.bossKillsRequired
		game.bossEventCount = 0
		game.startBossEventIfDue()
	}
	if len(game.spiders) != maximumBossesAlive {
		t.Fatalf("%d spider tanks alive, want %d", len(game.spiders), maximumBossesAlive)
	}
}

func TestBoltStopsOnToughTarget(t *testing.T) {
	game := newHeadlessGame()
	game.enemies.Spawn(EnemyBrute, game.player.X+40, game.player.Y, 1)
	game.grid.Rebuild(game.enemies.PositionX, game.enemies.PositionY, game.enemies.Count)
	game.projectiles.SpawnBolt(game.player.X+40, game.player.Y, 0, 0, 10, ElementPhysical, 5, [3]float32{1, 1, 1})
	game.updateBolt(0)
	if game.projectiles.Life[0] > 0 {
		t.Fatalf("bolt with pierce 5 went through a brute with %.0f health", game.enemies.Health[0]+10)
	}
}

func TestBossBeamCycleLocksThenFires(t *testing.T) {
	game := newHeadlessGame()
	game.bossProgressKills = game.bossKillsRequired
	game.startBossEventIfDue()
	phasesSeen := map[LaserPhase]bool{}
	lockedAngle := float32(0)
	for range 4 * ticksPerSecond {
		stepHeadlessIdle(game)
		laser := &game.spiders[0].Laser
		phasesSeen[laser.Phase] = true
		isEnteringLock := laser.Phase == LaserLocked && lockedAngle == 0
		lockedAngle += laser.BeamAngle * boolToFloat(isEnteringLock)
		if laser.Phase == LaserLocked && laser.BeamAngle != lockedAngle {
			t.Fatalf("beam moved while locked")
		}
	}
	for phase := range laserPhaseCount {
		if !phasesSeen[phase] {
			t.Fatalf("laser never reached phase %d", phase)
		}
	}
}

func TestLaserBurnsPlayerUnlessDashing(t *testing.T) {
	game := newHeadlessGame()
	beam := &BossBeam{Profile: &spiderBeamProfile, Phase: LaserFiring, OriginX: game.player.X - 300, OriginY: game.player.Y}
	game.burnPlayerInBeam(beam, 1, 1, 0, 0.1)
	if game.player.Health >= game.player.MaximumHealth {
		t.Fatalf("player standing in the beam took no damage")
	}
	healthAfterHit := game.player.Health
	game.player.DashSeconds = 0.1
	game.burnPlayerInBeam(beam, 1, 1, 0, 0.1)
	if game.player.Health != healthAfterHit {
		t.Fatalf("dashing player was burned by the beam")
	}
}

func TestBossKillsGiveNoReward(t *testing.T) {
	game := newHeadlessGame()
	game.enemies.Spawn(EnemySwarmer, 100, 100, 1)
	game.enemies.LastHitByBoss[0] = true
	game.enemies.Health[0] = 0
	game.removeDeadEnemies()
	if game.gems.Count != 0 || game.kills != 0 {
		t.Fatalf("boss kill dropped %d gems and counted %d kills", game.gems.Count, game.kills)
	}
	game.enemies.Spawn(EnemySwarmer, 100, 100, 1)
	game.HitEnemy(0, 1000, ElementPhysical)
	game.removeDeadEnemies()
	if game.gems.Count != 1 || game.kills != 1 {
		t.Fatalf("player kill dropped %d gems and counted %d kills", game.gems.Count, game.kills)
	}
}

func stepHeadlessIdle(game *Game) {
	game.frame++
	game.controls = Controls{}
	game.player.Health = game.player.MaximumHealth
	game.simulate()
}

func TestHighScoreIsSavedAndReloaded(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.killScore, game.elapsedSeconds, game.kills = 4200, 90, 321
	game.recordHighScoreAs("GRG")
	if game.newEntryRank != 0 {
		t.Fatalf("first score ranked %d, want 0", game.newEntryRank)
	}
	reloaded, err := LoadHighScores()
	best := reloaded.Best()
	if err != nil || best.Score != game.CurrentScore() || best.Kills != 321 || best.Initials != "GRG" {
		t.Fatalf("reloaded %+v (err %v), want score %d", best, err, game.CurrentScore())
	}
}

func TestDemoNeverRecordsHighScore(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.startDemo()
	game.killScore = 999999
	game.recordHighScore()
	if len(game.highScores.Entries) != 0 {
		t.Fatalf("demo recorded a high score of %d", game.highScores.Best().Score)
	}
}

func TestPauseFreezesTheWorld(t *testing.T) {
	game := newHeadlessGame()
	game.effects.AddShake(20)
	stepHeadless(game)
	game.state = StatePaused
	frame, shakeX, enemyCount := game.frame, game.effects.ShakeX, game.enemies.Count
	for range 120 {
		game.controls = Controls{}
		game.updatePaused()
	}
	if game.frame != frame || game.effects.ShakeX != shakeX || game.enemies.Count != enemyCount {
		t.Fatalf("world changed while paused")
	}
}

func TestDemoSequencesLoopCleanly(t *testing.T) {
	game := newHeadlessGame()
	game.startDemo()
	seen := map[int]bool{}
	randomClips := 0
	for game.demoClipCount <= 2*len(demoSequences) {
		isStarting := game.demoSeconds+deltaSeconds > demoSequenceSeconds
		game.updateDemo()
		seen[game.demoSequence] = true
		randomClips += boolToIndex(isStarting && game.currentDemo.Name == randomClipName)
	}
	if randomClips < len(demoSequences)-1 {
		t.Fatalf("only %d random clips played between scripted sequences", randomClips)
	}
	if len(seen) != len(demoSequences) {
		t.Fatalf("demo visited %d of %d sequences", len(seen), len(demoSequences))
	}
}

func TestBossCycleIsThreeSpidersHordeWormHorde(t *testing.T) {
	game := newHeadlessGame()
	expected := []BossEventKind{BossSpider, BossSpider, BossSpider, BossHorde, BossWorm, BossHorde, BossSpider, BossSpider, BossSpider, BossHorde, BossWorm, BossHorde}
	expectedHordeSizes := []int{500, 1000, 1500, 2000}
	hordesSeen := 0
	for event, kind := range expected {
		game.spiders, game.enemies.Count, game.hordeEvent, game.worm = game.spiders[:0], 0, HordeEvent{}, DrillWorm{}
		game.startBossEvent()
		seen := [bossEventKindCount]bool{BossSpider: len(game.spiders) == 1, BossHorde: game.hordeEvent.IsActive, BossWorm: game.worm.IsActive}
		if seen != [bossEventKindCount]bool{kind == BossSpider, kind == BossHorde, kind == BossWorm} {
			t.Fatalf("event %d: want %d, got spider %v horde %v worm %v", event+1, kind, seen[BossSpider], seen[BossHorde], seen[BossWorm])
		}
		if kind == BossHorde && game.hordeEvent.Total != expectedHordeSizes[hordesSeen] {
			t.Fatalf("horde %d has %d enemies, want %d", hordesSeen+1, game.hordeEvent.Total, expectedHordeSizes[hordesSeen])
		}
		hordesSeen += boolToIndex(kind == BossHorde)
	}
}

func TestHordeKillsDoNotSummonBossesAndClearingEndsEvent(t *testing.T) {
	game := newHeadlessGame()
	game.bossEventCount = spidersBeforeHorde
	game.startBossEvent()
	total := game.hordeEvent.Total
	for index := range game.enemies.Count {
		game.enemies.Health[index] = 0
	}
	game.removeDeadEnemies()
	if game.bossProgressKills != 0 || game.kills != total {
		t.Fatalf("horde kills: boss progress %d (want 0), kills %d (want %d)", game.bossProgressKills, game.kills, total)
	}
	if game.hordeEvent.IsActive {
		t.Fatalf("horde still active after every member died")
	}
	game.bossProgressKills = game.bossKillsRequired
	game.startBossEventIfDue()
	if !game.worm.IsActive {
		t.Fatalf("no drill-worm after the horde was cleared")
	}
}

func TestBossesKeepComingDuringAHorde(t *testing.T) {
	game := newHeadlessGame()
	game.bossEventCount = spidersBeforeHorde
	game.startBossEvent()
	game.bossEventCount = 0
	game.bossProgressKills = game.bossKillsRequired
	game.startBossEventIfDue()
	if len(game.spiders) != 1 || !game.hordeEvent.IsActive {
		t.Fatalf("no spider tank while the horde was alive: spiders %d, horde %v", len(game.spiders), game.hordeEvent.IsActive)
	}
}

func TestBossesKeepComingWhileTheWormLives(t *testing.T) {
	game := newHeadlessGame()
	game.bossEventCount = spidersBeforeHorde + 1
	game.startBossEvent()
	game.bossProgressKills = game.bossKillsRequired
	game.startBossEventIfDue()
	if !game.worm.IsActive || !game.hordeEvent.IsActive {
		t.Fatalf("the horde did not follow the living drill-worm")
	}
	for range len(bossEventCycle) - 1 {
		game.bossProgressKills = game.bossKillsRequired
		game.spiders = game.spiders[:0]
		game.startBossEventIfDue()
	}
	if game.nextBossEvent() != BossWorm || !game.worm.IsActive {
		t.Fatalf("the cycle did not stop in front of the next drill-worm: next %d", game.nextBossEvent())
	}
}

func TestOnlyOneWormAtATimeEvenAmongSpiders(t *testing.T) {
	game := newHeadlessGame()
	game.bossEventCount = spidersBeforeHorde + 1
	game.startBossEvent()
	game.bossEventCount = spidersBeforeHorde + 1
	game.bossProgressKills = game.bossKillsRequired
	game.startBossEventIfDue()
	if game.bossEventCount != spidersBeforeHorde+1 {
		t.Fatalf("a second drill-worm event started while one was alive")
	}
	game.worm.IsActive = false
	for range maximumBossesAlive - 1 {
		game.spawnSpiderEvent()
	}
	game.startBossEventIfDue()
	if !game.worm.IsActive {
		t.Fatalf("the drill-worm was held back by the spider tanks")
	}
}

func TestNoMoreThanThreeBossesAtOnce(t *testing.T) {
	game := newHeadlessGame()
	game.bossEventCount = spidersBeforeHorde + 1
	game.startBossEvent()
	for range maximumBossesAlive {
		game.bossEventCount = 0
		game.bossProgressKills = game.bossKillsRequired
		game.startBossEventIfDue()
	}
	if game.livingBossCount() != maximumBossesAlive {
		t.Fatalf("%d bosses alive with the drill-worm, want %d", game.livingBossCount(), maximumBossesAlive)
	}
	game.spiders = game.spiders[:maximumBossesAlive-1]
	game.worm.IsActive = false
	game.bossEventCount = spidersBeforeHorde + 1
	game.bossProgressKills = game.bossKillsRequired
	game.spawnSpiderEvent()
	game.startBossEventIfDue()
	if game.worm.IsActive {
		t.Fatalf("the drill-worm joined three spider tanks")
	}
}

func TestOverlappingHordesMerge(t *testing.T) {
	game := newHeadlessGame()
	game.bossEventCount = spidersBeforeHorde
	game.startBossEvent()
	first := game.hordeEvent.Total
	game.bossEventCount = spidersBeforeHorde + 2
	game.startBossEvent()
	if game.hordeEvent.Total != first+2*first || game.hordeEvent.Remaining != game.hordeEvent.Total {
		t.Fatalf("merged horde %d / %d, first wave %d", game.hordeEvent.Remaining, game.hordeEvent.Total, first)
	}
}

func TestDrillWormEmergesFiresAFrostBeamThenDives(t *testing.T) {
	game := newHeadlessGame()
	game.spawnWormEvent()
	phasesSeen := [wormPhaseCount]bool{}
	hasFired, hasHittableSegmentUnderground := false, false
	for range 20 * ticksPerSecond {
		stepHeadlessIdle(game)
		worm := &game.worm
		phasesSeen[worm.Phase] = true
		hasFired = hasFired || worm.Beam.Phase == LaserFiring && worm.Beam.Profile == &frostBeamProfile
		for index := range worm.Segments {
			isHittable := worm.Segments[index].EnemyID != 0 && game.enemies.IndexOfID(worm.Segments[index].EnemyID) >= 0
			hasHittableSegmentUnderground = hasHittableSegmentUnderground || isHittable && !worm.IsBodyOut()
		}
	}
	for _, phase := range []WormPhase{WormBurrowing, WormWarning, WormEmerging, WormAiming, WormDiving} {
		if !phasesSeen[phase] {
			t.Fatalf("the drill-worm never reached phase %d", phase)
		}
	}
	if !hasFired || hasHittableSegmentUnderground || game.worm.AliveSegmentCount() != wormSegmentCount {
		t.Fatalf("fired %v, hittable underground %v, segments %d", hasFired, hasHittableSegmentUnderground, game.worm.AliveSegmentCount())
	}
}

func TestDrillWormLosesItsBodyThenRollsItsHead(t *testing.T) {
	game := newHeadlessGame()
	game.spawnWormEvent()
	for game.worm.Phase != WormAiming {
		stepHeadlessIdle(game)
	}
	for index := range game.worm.Segments {
		game.enemies.Health[game.enemies.IndexOfID(game.worm.Segments[index].EnemyID)] = 0
	}
	stepHeadlessIdle(game)
	stepHeadlessIdle(game)
	if !game.worm.IsHeadAlone() || game.enemies.IndexOfID(game.worm.HeadEnemyID) < 0 {
		t.Fatalf("the head did not break free once every segment was destroyed: phase %d", game.worm.Phase)
	}
	startX, startY := game.worm.HeadX, game.worm.HeadY
	rolled := false
	for range (wormHeadCycleSeconds + 1) * ticksPerSecond {
		stepHeadlessIdle(game)
		rolled = rolled || length(game.worm.HeadX-startX, game.worm.HeadY-startY) > wormRollDistance/2
	}
	if !rolled {
		t.Fatalf("the lone head never rolled")
	}
	game.enemies.Health[game.enemies.IndexOfID(game.worm.HeadEnemyID)] = 0
	stepHeadlessIdle(game)
	if game.worm.IsActive {
		t.Fatalf("the drill-worm survived losing its head")
	}
}

func TestSoleOfferIsAppliedWithoutPausing(t *testing.T) {
	game := newHeadlessGame()
	game.player.Weapons = game.player.Weapons[:0]
	for kind := range weaponKindCount {
		game.player.Weapons = append(game.player.Weapons, WeaponState{Kind: kind, Level: maximumWeaponLevel})
	}
	for kind := range passiveKindCount {
		game.player.PassiveLevels[kind] = passiveTable[kind].MaximumLevel
	}
	game.player.Health = 10
	game.player.PendingLevelUps = 1
	game.openLevelUpIfPending()
	if game.state != StatePlaying || game.player.PendingLevelUps != 0 || game.player.Health <= 10 {
		t.Fatalf("sole offer: state %d, pending %d, health %.0f", game.state, game.player.PendingLevelUps, game.player.Health)
	}
	if game.levelUpBannerSeconds <= 0 || game.levelUpBanner == "" {
		t.Fatalf("no level up banner shown")
	}
	t.Logf("banner: %q", game.levelUpBanner)
}

func TestSeveralOffersStillOpenTheMenu(t *testing.T) {
	game := newHeadlessGame()
	game.player.PendingLevelUps = 1
	game.openLevelUpIfPending()
	if game.state != StateLevelUp {
		t.Fatalf("with several offers the level up menu should open, state %d", game.state)
	}
}

func TestRightStickAimsIndependentlyOfMovement(t *testing.T) {
	game := newHeadlessGame()
	game.controls = Controls{MoveX: 1, AimX: 0, AimY: -1, HasAimStick: true}
	game.updatePlayer(deltaSeconds)
	if game.player.AimY > -0.99 || game.player.FacingX < 0.99 {
		t.Fatalf("moving right while aiming up: aim (%.2f, %.2f), facing (%.2f, %.2f)", game.player.AimX, game.player.AimY, game.player.FacingX, game.player.FacingY)
	}
	game.controls = Controls{MoveY: 1}
	game.updatePlayer(deltaSeconds)
	if game.player.AimY < 0.99 {
		t.Fatalf("without the right stick the aim should follow movement, aim (%.2f, %.2f)", game.player.AimX, game.player.AimY)
	}
}

func TestRightStickFiresAutomatically(t *testing.T) {
	game := newHeadlessGame()
	expectations := []struct {
		controls   Controls
		isExpected bool
	}{
		{Controls{AimX: 1, HasAimStick: true}, true},
		{Controls{MoveX: 1}, false},
		{Controls{Held: ActionFire}, true},
	}
	for _, expectation := range expectations {
		game.controls = expectation.controls
		game.updatePlayer(deltaSeconds)
		if game.player.IsFiring != expectation.isExpected {
			t.Fatalf("controls %+v: firing %v, want %v", expectation.controls, game.player.IsFiring, expectation.isExpected)
		}
	}
}

func TestBossKillsDoNotCarryOverAndRequirementGrows(t *testing.T) {
	game := newHeadlessGame()
	firstRequirement := game.bossKillsRequired
	game.elapsedSeconds = 600
	game.bossProgressKills = firstRequirement * 5
	game.startBossEventIfDue()
	if len(game.spiders) != 1 || game.bossProgressKills != 0 {
		t.Fatalf("spiders %d, leftover progress %d (want 1, 0)", len(game.spiders), game.bossProgressKills)
	}
	if game.bossKillsRequired <= firstRequirement {
		t.Fatalf("requirement after ten minutes is %d, not above the opening %d", game.bossKillsRequired, firstRequirement)
	}
	game.bossProgressKills = game.bossKillsRequired - 1
	game.startBossEventIfDue()
	if len(game.spiders) != 1 {
		t.Fatalf("a second boss arrived one kill short of its own requirement")
	}
	t.Logf("kills required at 0, 5, 10, 20 minutes: %d %d %d %d", bossKillsRequiredAt(0), bossKillsRequiredAt(300), bossKillsRequiredAt(600), bossKillsRequiredAt(1200))
}

func TestHordeIsVariedWithExplosivesOnTheOutside(t *testing.T) {
	game := newHeadlessGame()
	game.bossEventCount = spidersBeforeHorde
	game.startBossEvent()
	kindCounts := [enemyKindCount]int{}
	nearestExplosive, farthestCore := float32(1e9), float32(0)
	for index := range game.enemies.Count {
		kind := game.enemies.Kind[index]
		kindCounts[kind]++
		distance := length(game.enemies.PositionX[index]-game.player.X, game.enemies.PositionY[index]-game.player.Y)
		isExplosive := enemyTable[kind].DeathElement != ElementNone && kind != EnemyFrostling
		nearestExplosive = min(nearestExplosive, distance+1e9*boolToFloat(!isExplosive))
		farthestCore = max(farthestCore, distance*boolToFloat(!isExplosive))
	}
	explosiveCount := kindCounts[EnemyBloater] + kindCounts[EnemyEmberling]
	if explosiveCount*5 > game.enemies.Count || nearestExplosive <= farthestCore {
		t.Fatalf("explosives %d of %d, nearest explosive %.0f, farthest core %.0f", explosiveCount, game.enemies.Count, nearestExplosive, farthestCore)
	}
	for _, kind := range []EnemyKind{EnemyBrute, EnemyFrostling, EnemyRunner, EnemySwarmer} {
		if kindCounts[kind] == 0 {
			t.Fatalf("horde has no %s", enemyTable[kind].Name)
		}
	}
	t.Logf("horde composition: %v", kindCounts)
}

func TestHordeDoesNotDetonateAtOnce(t *testing.T) {
	game := newHeadlessGame()
	equipEveryWeapon(game)
	game.elapsedSeconds = 900
	game.bossEventCount = spidersBeforeHorde
	game.startBossEvent()
	total := game.hordeEvent.Total
	for second := 1; second <= 12; second++ {
		for range ticksPerSecond {
			stepHeadless(game)
		}
		t.Logf("after %2ds: %d / %d horde members alive", second, game.hordeEvent.Remaining, total)
	}
	if game.hordeEvent.Remaining*4 < total {
		t.Fatalf("only %d of %d horde members survived twelve seconds", game.hordeEvent.Remaining, total)
	}
}

func TestOnlyPassivesLeftRotateThroughRemainingOnes(t *testing.T) {
	game := newHeadlessGame()
	equipEveryWeapon(game)
	remaining := []PassiveKind{PassiveSwiftness, PassiveHaste, PassiveMight, PassiveReach}
	for kind := range passiveKindCount {
		game.player.PassiveLevels[kind] = passiveTable[kind].MaximumLevel
	}
	for _, kind := range remaining {
		game.player.PassiveLevels[kind] = 0
	}
	game.player.PassiveLevels[PassiveMight] = passiveTable[PassiveMight].MaximumLevel - 2
	expected := []PassiveKind{
		PassiveSwiftness, PassiveHaste, PassiveMight, PassiveReach,
		PassiveSwiftness, PassiveHaste, PassiveMight, PassiveReach,
		PassiveSwiftness, PassiveHaste, PassiveReach,
		PassiveSwiftness, PassiveHaste, PassiveReach,
	}
	for turn, want := range expected {
		before := game.player.PassiveLevels
		game.player.PendingLevelUps = 1
		game.openLevelUpIfPending()
		if game.state != StatePlaying || game.player.PendingLevelUps != 0 {
			t.Fatalf("turn %d: the level up paused the game", turn)
		}
		if game.player.PassiveLevels[want] != before[want]+1 {
			t.Fatalf("turn %d: expected %s, banner %q", turn, passiveTable[want].Key, game.levelUpBanner)
		}
	}
}

func TestDemoClipsNeverRunEmpty(t *testing.T) {
	const tolerableEmptySeconds = 0.5
	game := newHeadlessGame()
	game.startDemo()
	for range 2 * len(demoSequences) * 2 {
		sequence := game.currentDemo
		emptySeconds := float32(0)
		for range demoSequenceSeconds*ticksPerSecond - 2 {
			game.updateDemo()
			emptySeconds += deltaSeconds * boolToFloat(game.enemies.Count*10 < sequence.EnemyCount)
		}
		t.Logf("%-22s %-50s spawned %4d, nearly empty for %.2fs", sequence.Name, sequence.Subtitle, sequence.EnemyCount, emptySeconds)
		if emptySeconds > tolerableEmptySeconds {
			t.Errorf("%s %s stayed nearly empty for %.2fs", sequence.Name, sequence.Subtitle, emptySeconds)
		}
		for game.currentDemo.Name == sequence.Name && game.currentDemo.Subtitle == sequence.Subtitle {
			game.updateDemo()
		}
	}
}

func TestSeismicHammerShattersEveryFrozenFoeInReach(t *testing.T) {
	game := newHeadlessGame()
	for slot := range 6 {
		angle := float32(slot)
		game.enemies.Spawn(EnemyBrute, game.player.X+cosine(angle)*80, game.player.Y+sine(angle)*80, 1)
		game.enemies.ApplyStatus(game.enemies.Count-1, StatusFrozen)
	}
	game.grid.Rebuild(game.enemies.PositionX, game.enemies.PositionY, game.enemies.Count)
	weapon := WeaponState{Kind: WeaponSeismicHammer, Level: 1}
	game.slamSeismicHammer(&weapon, &weaponTable[WeaponSeismicHammer])
	if game.reactionCounts[ReactionShatter] != 6 {
		t.Fatalf("hammer shattered %d of 6 frozen brutes", game.reactionCounts[ReactionShatter])
	}
}

func TestStaticMineArmsThenShocksWhoStepsNear(t *testing.T) {
	game := newHeadlessGame()
	weapon := WeaponState{Kind: WeaponStaticMines, Level: 1}
	game.layStaticMines(&weapon, &weaponTable[WeaponStaticMines])
	mine := game.mines[0]
	game.enemies.Spawn(EnemyBrute, mine.X+10, mine.Y, 1)
	game.grid.Rebuild(game.enemies.PositionX, game.enemies.PositionY, game.enemies.Count)
	game.updateMines(mineArmSeconds / 2)
	if len(game.mines) != 1 {
		t.Fatalf("mine went off before arming")
	}
	game.updateMines(mineArmSeconds)
	if len(game.mines) != 0 || game.enemies.Status[0]&StatusShocked == 0 {
		t.Fatalf("armed mine did not shock the brute: mines %d, status %b", len(game.mines), game.enemies.Status[0])
	}
}

func TestShockedFoesReactToFireBladesAndRain(t *testing.T) {
	expectations := []struct {
		element  Element
		reaction ReactionKind
	}{
		{ElementFire, ReactionPlasma},
		{ElementPhysical, ReactionOverload},
		{ElementWater, ReactionConduction},
	}
	for _, expectation := range expectations {
		game := newHeadlessGame()
		game.enemies.Spawn(EnemyBrute, game.player.X+200, game.player.Y, 1)
		game.enemies.Spawn(EnemyBrute, game.player.X+240, game.player.Y, 1)
		game.enemies.ApplyStatus(0, StatusShocked)
		game.enemies.ApplyStatus(1, StatusShocked|StatusWet*StatusFlags(boolToIndex(expectation.element == ElementWater)))
		game.grid.Rebuild(game.enemies.PositionX, game.enemies.PositionY, game.enemies.Count)
		game.HitEnemy(0, 5, expectation.element)
		game.processBursts()
		if game.reactionCounts[expectation.reaction] == 0 {
			t.Fatalf("element %d on a shocked foe did not trigger %s", expectation.element, reactionTable[expectation.reaction].Name)
		}
		t.Logf("%s chained: %v", reactionTable[expectation.reaction].Name, game.reactionCounts)
	}
}

func TestBeamChargeSoundsLastUntilTheShot(t *testing.T) {
	cases := map[*BeamProfile]float32{&spiderBeamProfile: audio.LaserChargeSoundSeconds, &frostBeamProfile: audio.FrostChargeSoundSeconds}
	for profile, soundSeconds := range cases {
		untilShot := profile.Durations[LaserCharging] + profile.Durations[LaserLocked]
		if abs(soundSeconds-untilShot) > 0.001 {
			t.Fatalf("charge sound lasts %.2fs, the beam fires after %.2fs", soundSeconds, untilShot)
		}
	}
}

func TestHealOrbsDropRarelyAndHealWithoutExperience(t *testing.T) {
	game := newHeadlessGame()
	const kills = 20000
	for range kills {
		game.dropHealOrbIf(game.player.X+300, game.player.Y)
		dropped := len(game.healOrbs)
		game.healOrbs = game.healOrbs[:0]
		game.kills += dropped
	}
	rate := float32(game.kills) / kills
	if rate < 0.015 || rate > 0.025 {
		t.Fatalf("heal orb drop rate %.3f, want about %.2f", rate, healOrbDropChance)
	}
	game.player.Health = 50
	experienceBefore, levelBefore := game.player.Experience, game.player.Level
	game.healOrbs = append(game.healOrbs, HealOrb{X: game.player.X + 5, Y: game.player.Y})
	game.updateHealOrbs(deltaSeconds)
	if game.player.Health != 70 || len(game.healOrbs) != 0 {
		t.Fatalf("heal orb gave %.0f health (want 20) and left %d orbs", game.player.Health-50, len(game.healOrbs))
	}
	if game.player.Experience != experienceBefore || game.player.Level != levelBefore {
		t.Fatalf("heal orb gave experience")
	}
	game.player.Health = game.player.MaximumHealth - 5
	game.healOrbs = append(game.healOrbs, HealOrb{X: game.player.X, Y: game.player.Y})
	game.updateHealOrbs(deltaSeconds)
	if game.player.Health != game.player.MaximumHealth {
		t.Fatalf("heal orb overhealed to %.0f of %.0f", game.player.Health, game.player.MaximumHealth)
	}
}

func TestBossKillsDropNoHealOrbs(t *testing.T) {
	game := newHeadlessGame()
	for range 500 {
		game.enemies.Spawn(EnemySwarmer, game.player.X+300, game.player.Y, 1)
		game.enemies.LastHitByBoss[game.enemies.Count-1] = true
		game.enemies.Health[game.enemies.Count-1] = 0
		game.removeDeadEnemies()
	}
	if len(game.healOrbs) != 0 {
		t.Fatalf("enemies killed by a boss dropped %d heal orbs", len(game.healOrbs))
	}
}

func TestRepeatedFireHitsHeatEnemiesUpToLevelThree(t *testing.T) {
	game := newHeadlessGame()
	game.enemies.Spawn(EnemyBrute, game.player.X+300, game.player.Y, 1)
	burning := statusBitIndex(StatusBurning)
	for hit := 1; hit <= 4; hit++ {
		game.HitEnemy(0, 0, ElementFire)
		want := uint8(min(hit, maximumStatusLevel))
		if game.enemies.LevelsOf(0)[burning] != want {
			t.Fatalf("after %d fire hits burn level %d, want %d", hit, game.enemies.LevelsOf(0)[burning], want)
		}
	}
	before := game.enemies.Health[0]
	game.enemies.TickStatuses(0, 1)
	if burned := before - game.enemies.Health[0]; burned != statusTable[burning].DamagePerSecond*maximumStatusLevel {
		t.Fatalf("level 3 burn dealt %.1f per second, want %.1f", burned, statusTable[burning].DamagePerSecond*maximumStatusLevel)
	}
	game.enemies.TickStatuses(0, statusTable[burning].DurationSeconds)
	game.HitEnemy(0, 0, ElementFire)
	if game.enemies.LevelsOf(0)[burning] != 1 {
		t.Fatalf("a burn that went out should restart at level 1")
	}
}

func TestFrostFreezesOnTheThirdHit(t *testing.T) {
	game := newHeadlessGame()
	game.enemies.Spawn(EnemyBrute, game.player.X+300, game.player.Y, 1)
	for hit := 1; hit <= 3; hit++ {
		isFrozen := game.enemies.Status[0]&StatusFrozen != 0
		if isFrozen {
			t.Fatalf("frozen after only %d frost hits", hit-1)
		}
		game.HitEnemy(0, 0, ElementFrost)
	}
	if game.enemies.Status[0]&StatusFrozen == 0 || game.reactionCounts[ReactionFreeze] != 1 {
		t.Fatalf("three frost hits did not freeze: status %b", game.enemies.Status[0])
	}
}

func TestShotEvolvesIntoOneElementOnly(t *testing.T) {
	game := newHeadlessGame()
	game.player.Weapons = []WeaponState{{Kind: WeaponPulseShot, Level: 3}}
	game.offerPool = game.offerPool[:0]
	for kind := range weaponKindCount {
		game.offerPool = game.appendWeaponOffer(game.offerPool, kind)
	}
	evolutions := map[WeaponKind]bool{}
	for _, offer := range game.offerPool {
		evolutions[offer.Weapon] = offer.IsEvolution
	}
	if !evolutions[WeaponEmberBolt] || !evolutions[WeaponFrostShard] || !evolutions[WeaponVoltBolt] {
		t.Fatalf("pulse shot should offer the three elemental evolutions: %v", evolutions)
	}
	game.applyWeaponUpgrade(UpgradeOffer{Kind: UpgradeWeapon, Weapon: WeaponFrostShard, NextLevel: 3, IsEvolution: true})
	if len(game.player.Weapons) != 1 || game.player.Weapons[0] != (WeaponState{Kind: WeaponFrostShard, Level: 3}) {
		t.Fatalf("evolution should replace the pulse shot and keep its level: %+v", game.player.Weapons)
	}
	game.offerPool = game.offerPool[:0]
	for kind := range weaponKindCount {
		game.offerPool = game.appendWeaponOffer(game.offerPool, kind)
	}
	for _, offer := range game.offerPool {
		isOtherShot := weaponTable[offer.Weapon].Family == FamilyMainShot && offer.Weapon != WeaponFrostShard
		if isOtherShot {
			t.Fatalf("after choosing Frost Shard, %s is still offered", weaponTable[offer.Weapon].Key)
		}
	}
}

func TestSpiderTanksDoNotOverlap(t *testing.T) {
	game := newHeadlessGame()
	game.spawnSpiderAt(0, 400)
	game.spawnSpiderAt(0.01, 400)
	game.spawnSpiderAt(0.02, 410)
	spacing := enemyTable[EnemySpiderTank].Radius * 2 * spiderSpacingFactor
	for range 3 * ticksPerSecond {
		stepHeadlessIdle(game)
	}
	for first := range game.spiders {
		for second := first + 1; second < len(game.spiders); second++ {
			a := game.enemies.IndexOfID(game.spiders[first].EnemyID)
			b := game.enemies.IndexOfID(game.spiders[second].EnemyID)
			gap := length(game.enemies.PositionX[a]-game.enemies.PositionX[b], game.enemies.PositionY[a]-game.enemies.PositionY[b])
			if gap < spacing*0.9 {
				t.Fatalf("spider tanks %d and %d overlap: %.0f apart, want %.0f", first, second, gap, spacing)
			}
		}
	}
}

func TestMouseAimsTheCannonAndClicksFireAndDash(t *testing.T) {
	game := newHeadlessGame()
	game.renderer = &Renderer{CameraX: game.player.X - screenWidth/2, CameraY: game.player.Y - screenHeight/2}
	cursorX, cursorY := float32(screenWidth/2)*renderScale, float32(screenHeight/2+200)*renderScale
	game.controls = Controls{HasMouseAim: true, CursorX: cursorX, CursorY: cursorY, Held: ActionFire, JustPressed: ActionDash}
	game.updatePlayer(deltaSeconds)
	if game.player.AimY < 0.99 || !game.player.IsFiring || game.player.DashSeconds <= 0 {
		t.Fatalf("aim %.2f,%.2f firing %v dash %.2f", game.player.AimX, game.player.AimY, game.player.IsFiring, game.player.DashSeconds)
	}
}

func TestEveryDrillWormPhaseIsUpdatedAndDrawn(t *testing.T) {
	for phase := range wormPhaseCount {
		if wormPhaseUpdaters[phase] == nil || wormGroundDrawers[phase] == nil || wormBodyDrawers[phase] == nil {
			t.Fatalf("drill-worm phase %d has no updater or drawer", phase)
		}
	}
}

func TestWormEruptionShovesThePlayerWithoutKilling(t *testing.T) {
	game := newHeadlessGame()
	game.spawnWormEvent()
	game.worm.HoleX, game.worm.HoleY = game.player.X+10, game.player.Y
	startX := game.player.X
	game.player.Health = 6
	game.eruptWorm(&game.worm)
	for range ticksPerSecond {
		game.frame++
		game.controls = Controls{}
		game.simulate()
	}
	if game.player.Health < 1 || game.worm.BlastHits != 0 {
		t.Fatalf("the eruption blast killed or never finished: health %.1f, hits left %d", game.player.Health, game.worm.BlastHits)
	}
	if startX-game.player.X < 60 || len(game.wormCraters) == 0 {
		t.Fatalf("the player was not shoved away (moved %.0f px) or no crater was left", startX-game.player.X)
	}
}
