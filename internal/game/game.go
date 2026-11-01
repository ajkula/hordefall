package game

import (
	"github.com/ajkula/ajkurig/audio"
	"github.com/ajkula/ajkurig/fx"
	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/input"
	"github.com/ajkula/ajkurig/rig"
	"github.com/ajkula/ajkurig/rng"
	"github.com/ajkula/ajkurig/shell"
	"github.com/ajkula/ajkurig/spatial"
	"github.com/ajkula/ajkurig/world"

	"hordefall/music"
)

// ===== Types =====

type Game struct {
	app                   *rig.App
	stats                 map[string]int
	shell                 *shell.Shell
	controls              input.Controls
	random                rng.Random
	player                *Player
	enemies               *EnemyStore
	projectiles           *ProjectileStore
	gems                  *GemStore
	effects               *fx.Effects
	ground                *world.Ground
	grid                  *spatial.Grid
	renderer              *Renderer
	audio                 *audio.Engine
	soundIDs              SoundIDs
	signalIDs             [music.SignalCount]audio.SignalID
	musicSignals          [music.SignalCount]float32
	previousReactions     int
	ui                    *UI
	director              SpawnDirector
	damageLog             shell.DamageLog
	bursts                []Burst
	candidates            []int32
	separationCandidates  []int32
	chainedEnemies        []int32
	spiders               []SpiderRig
	demoReinforceCooldown float32
	mines                 []StaticMine
	healOrbs              []HealOrb
	bossKillsRequired     int
	bossProgressKills     int
	bossEventCount        int
	hordeEvent            HordeEvent
	weather               world.Weather
	volleys               [maximumVolleys]Volley
	volleyCount           int
	pendingVolleys        int
	offerPool             []UpgradeOffer
	lastRotatedPassive    PassiveKind
	levelUpBanner         string
	levelUpBannerSeconds  float32
	reactionCounts        [reactionKindCount]int
	elapsedSeconds        float32
	kills                 int
	killScore             int
	frame                 uint32
	runCount              uint32
	isDemo                bool
	currentDemo           DemoSequence
	randomDemo            DemoSequence
	chatter               Chatter
	worms                 []DrillWorm
	wormCraters           []WormCrater
	hordeWaveCount        int
}

// ===== Constants =====

const (
	screenWidth           = 1280
	screenHeight          = 720
	arenaSize             = 4096
	ticksPerSecond        = 60
	deltaSeconds          = 1.0 / ticksPerSecond
	spatialCellSize       = 32
	levelUpBannerDuration = 3.2
	groundSeedBase        = 0x5EED
	gameRandomSeed        = 0xA11CE
	effectsRandomSeed     = 0xBEEF
	worldModuleName       = "hordefall"
)

// ===== Public API =====

func NewGame(app *rig.App, soundIDs SoundIDs) *Game {
	game := &Game{
		app:         app,
		stats:       map[string]int{},
		random:      rng.New(gameRandomSeed),
		ui:          NewUI(app.Kit),
		enemies:     NewEnemyStore(maximumEnemies),
		projectiles: NewProjectileStore(maximumProjectiles),
		gems:        NewGemStore(maximumGems),
		effects:     newEffects(),
		grid:        spatial.NewGrid(arenaSize, arenaSize, spatialCellSize, maximumEnemies),
		chatter:     newChatter(),
		ground:      NewGround(arenaSize, groundSeedBase),
		weather:     newWeather(soundIDs),
	}
	game.resetRun()
	game.renderer = NewRenderer(app.Batches, app.Camera, game.ground.Columns, game.ground.Rows)
	game.attachAudio(app.Audio, soundIDs)
	return game
}

func (g *Game) Name() string {
	return worldModuleName
}

func (g *Game) Reset(frame *rig.Frame) {
	demoDraws[boolToIndex(frame.IsDemo)](g)
	g.resetRun()
	g.isDemo = frame.IsDemo
}

func (g *Game) Update(frame *rig.Frame) {
	g.beginTick(frame)
}

func (g *Game) Report(report *rig.Report) {
	report.Count(i18n.T("bench.enemies"), g.enemies.Count)
	report.Count(i18n.T("bench.spiders"), len(g.spiders))
	report.Count(i18n.T("bench.particles"), g.effects.ParticleCount)
	report.Count(i18n.T("bench.projectiles"), g.projectiles.Count)
	report.Count(i18n.T("bench.ground"), g.ground.Columns*g.ground.Rows)
	report.Count(i18n.T("bench.reactions"), sumReactions(g.reactionCounts))
}

// ===== Internal =====

func (g *Game) resetRun() {
	g.runCount++
	g.player = NewPlayer()
	g.enemies.Count, g.projectiles.Count, g.gems.Count = 0, 0, 0
	g.damageLog.Clear()
	g.effects.Clear()
	regenerateGround(g.ground, groundSeedBase+g.runCount*7919)
	g.director = NewSpawnDirector()
	g.bursts = g.bursts[:0]
	g.reactionCounts = [reactionKindCount]int{}
	g.elapsedSeconds, g.kills, g.killScore = 0, 0, 0
	g.spiders = g.spiders[:0]
	g.mines = g.mines[:0]
	g.healOrbs = g.healOrbs[:0]
	g.bossKillsRequired = computeBossKillsRequired(0, 0)
	g.bossProgressKills, g.bossEventCount = 0, 0
	g.hordeEvent = HordeEvent{}
	g.worms, g.hordeWaveCount, g.wormCraters = g.worms[:0], 0, g.wormCraters[:0]
	g.resetWeather()
	g.resetChatter()
	g.isDemo = false
	g.levelUpBannerSeconds = 0
	g.volleyCount, g.pendingVolleys = 0, 0
	g.lastRotatedPassive = passiveKindCount - 1
}

func (g *Game) applyGraphicsSettings() {
	g.effects.ShakeFactor = g.app.Settings.ShakeFactor()
	g.effects.Density = g.app.Settings.EffectsFactor()
}

func (g *Game) tickGroundIfDue() {
	if g.frame%groundTickFrames != 0 {
		return
	}
	g.ground.Tick()
}
