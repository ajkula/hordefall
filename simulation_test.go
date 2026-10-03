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
