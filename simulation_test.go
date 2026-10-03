package main

import (
	"testing"
	"time"
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
		random:      NewRandom(0xA11CE),
		enemies:     NewEnemyStore(maximumEnemies),
		projectiles: NewProjectileStore(maximumProjectiles),
		gems:        NewGemStore(maximumGems),
		effects:     NewEffects(),
		grid:        NewSpatialGrid(arenaSize, spatialCellSize, maximumEnemies),
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
		game.kills = game.nextSpiderKills
		game.spawnSpiderIfDue()
	}
	if len(game.spiders) != maximumSpidersAlive {
		t.Fatalf("%d spider tanks alive, want %d", len(game.spiders), maximumSpidersAlive)
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

func TestSpiderLaserCycleLocksThenFires(t *testing.T) {
	game := newHeadlessGame()
	game.kills = game.nextSpiderKills
	game.spawnSpiderIfDue()
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
	rig := &SpiderRig{Power: 1, Laser: SpiderLaser{Phase: LaserFiring, OriginX: game.player.X - 300, OriginY: game.player.Y}}
	game.burnPlayerInBeam(rig, 1, 0, 0.1)
	if game.player.Health >= game.player.MaximumHealth {
		t.Fatalf("player standing in the beam took no damage")
	}
	healthAfterHit := game.player.Health
	game.player.DashSeconds = 0.1
	game.burnPlayerInBeam(rig, 1, 0, 0.1)
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
	game.recordHighScore()
	if !game.isNewHighScore {
		t.Fatalf("first score was not flagged as a new high score")
	}
	reloaded, err := LoadHighScore()
	if err != nil || reloaded.Score != game.CurrentScore() || reloaded.Kills != 321 {
		t.Fatalf("reloaded %+v (err %v), want score %d", reloaded, err, game.CurrentScore())
	}
}

func TestDemoNeverRecordsHighScore(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.startDemo()
	game.killScore = 999999
	game.recordHighScore()
	if game.highScore.Score != 0 {
		t.Fatalf("demo recorded a high score of %d", game.highScore.Score)
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
	for range (2*len(demoSequences) + 1) * demoSequenceSeconds * ticksPerSecond {
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
