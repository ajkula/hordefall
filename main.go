package main

import (
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ===== Types =====

type GameState uint8

type StateHandler struct {
	Update func(game *Game)
	Draw   func(game *Game, screen *ebiten.Image)
}

type Game struct {
	state                 GameState
	input                 InputReader
	controls              Controls
	random                Random
	player                *Player
	enemies               *EnemyStore
	projectiles           *ProjectileStore
	gems                  *GemStore
	effects               *Effects
	ground                *Ground
	grid                  *SpatialGrid
	renderer              *Renderer
	post                  *PostProcessor
	audio                 *AudioEngine
	musicSignals          [musicSignalCount]float32
	previousReactions     int
	ui                    *UI
	director              SpawnDirector
	bursts                []Burst
	candidates            []int32
	separationCandidates  []int32
	chainedEnemies        []int32
	spiders               []SpiderRig
	demoReinforceCooldown float32
	mines                 []StaticMine
	bossKillsRequired     int
	bossProgressKills     int
	bossEventCount        int
	hordeEvent            HordeEvent
	offers                []UpgradeOffer
	offerPool             []UpgradeOffer
	selectedOffer         int
	lastRotatedPassive    PassiveKind
	levelUpBanner         string
	musicBanner           string
	musicBannerSeconds    float32
	levelUpBannerSeconds  float32
	menuSelection         int
	menuLockSeconds       float32
	reactionCounts        [reactionKindCount]int
	elapsedSeconds        float32
	clockSeconds          float32
	kills                 int
	killScore             int
	highScore             HighScore
	isNewHighScore        bool
	scoreMessage          string
	frame                 uint32
	runCount              uint32
	isDemo                bool
	demoSeconds           float32
	demoSequence          int
	demoSlot              int
	currentDemo           DemoSequence
	simulationMillis      float32
	isQuitRequested       bool
	isInputDebugVisible   bool
	remapStep             int
	remapReturnState      GameState
	remapButtons          [3]int
	bindingsMessage       string
	settings              Settings
	settingsMessage       string
	playlistMenu          []MenuOption
	enabledSongs          []int
	musicSlot             int
	musicRandom           Random
}

// ===== Constants =====

const (
	StateMainMenu GameState = iota
	StatePlaying
	StateLevelUp
	StatePaused
	StateGameOver
	StateRemap
	StateOptions
	StatePlaylist
	StateGraphics
	StateAudio
	stateCount
)

const (
	screenWidth             = 1280
	screenHeight            = 720
	arenaSize               = 4096
	ticksPerSecond          = 60
	deltaSeconds            = 1.0 / ticksPerSecond
	spatialCellSize         = 32
	menuLockDuration        = 0.45
	levelUpBannerDuration   = 2.2
	groundSeedBase          = 0x5EED
	selectionDirections     = ActionLeft | ActionUp
	simulationTimeSmoothing = 0.05
)

var stateHandlers = [stateCount]StateHandler{
	StateMainMenu: {(*Game).updateMainMenu, (*Game).drawMainMenu},
	StatePlaying:  {(*Game).updatePlaying, (*Game).drawPlaying},
	StateLevelUp:  {(*Game).updateLevelUp, (*Game).drawLevelUp},
	StatePaused:   {(*Game).updatePaused, (*Game).drawPaused},
	StateGameOver: {(*Game).updateGameOver, (*Game).drawGameOver},
	StateRemap:    {(*Game).updateRemap, (*Game).drawRemap},
	StateOptions:  {(*Game).updateOptions, (*Game).drawOptions},
	StatePlaylist: {(*Game).updatePlaylist, (*Game).drawPlaylist},
	StateGraphics: {(*Game).updateGraphics, (*Game).drawGraphics},
	StateAudio:    {(*Game).updateAudioMenu, (*Game).drawAudioMenu},
}

// ===== Public API =====

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Hordefall")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}

func NewGame() *Game {
	game := &Game{
		random:      NewRandom(0xA11CE),
		ui:          NewUI(),
		enemies:     NewEnemyStore(maximumEnemies),
		projectiles: NewProjectileStore(maximumProjectiles),
		gems:        NewGemStore(maximumGems),
		effects:     NewEffects(),
		grid:        NewSpatialGrid(arenaSize, spatialCellSize, maximumEnemies),
		musicSlot:   noMusicSlot,
		musicRandom: NewRandom(uint32(time.Now().UnixNano())),
	}
	game.resetRun()
	game.renderer = NewRenderer(game.ground.Columns, game.ground.Rows)
	game.post = NewPostProcessor()
	game.loadBindings()
	game.loadHighScore()
	game.audio = NewAudioEngine()
	game.loadSettings()
	game.startDemo()
	return game
}

func (g *Game) Update() error {
	if ebiten.IsWindowBeingClosed() || g.isQuitRequested {
		g.recordHighScore()
		return ebiten.Termination
	}
	g.controls = g.input.Read()
	isSelectTogglingDebug := g.controls.JustPressed&ActionSelect != 0 && (g.state == StatePlaying || g.state == StateLevelUp)
	g.isInputDebugVisible = g.isInputDebugVisible != (g.controls.JustPressed&ActionDebug != 0 || isSelectTogglingDebug)
	g.clockSeconds += deltaSeconds
	g.menuLockSeconds = max(0, g.menuLockSeconds-deltaSeconds)
	stateHandlers[g.state].Update(g)
	g.toggleFullscreenIfRequested()
	g.updateAudio()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	postSettings := g.postSettings()
	stateHandlers[g.state].Draw(g, g.post.Begin(screen, postSettings))
	g.post.Finish(screen, postSettings)
	if !g.isInputDebugVisible {
		return
	}
	g.ui.DrawDebugOverlay(append(g.input.DescribeDevices(g.controls), g.describeMusic()), screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	renderScale = resolutionChoices[g.settings.Resolution].Scale(outsideWidth, outsideHeight)
	return int(screenWidth * renderScale), int(screenHeight * renderScale)
}

// ===== Internal =====

func (g *Game) resetRun() {
	g.runCount++
	g.player = NewPlayer()
	g.enemies.Count, g.projectiles.Count, g.gems.Count = 0, 0, 0
	g.effects.Clear()
	g.ground = NewGround(arenaSize, groundSeedBase+g.runCount*7919)
	g.director = NewSpawnDirector()
	g.bursts = g.bursts[:0]
	g.reactionCounts = [reactionKindCount]int{}
	g.elapsedSeconds, g.kills, g.killScore = 0, 0, 0
	g.spiders = g.spiders[:0]
	g.mines = g.mines[:0]
	g.bossKillsRequired = bossKillsRequiredAt(0)
	g.bossProgressKills, g.bossEventCount = 0, 0
	g.hordeEvent = HordeEvent{}
	g.isDemo, g.isNewHighScore = false, false
	g.levelUpBannerSeconds = 0
	g.lastRotatedPassive = passiveKindCount - 1
}

func (g *Game) isConfirming() bool {
	return g.menuLockSeconds == 0 && g.controls.JustPressed&ActionConfirm != 0
}

func (g *Game) switchState(state GameState) {
	g.state = state
	g.menuLockSeconds = menuLockDuration
	g.menuSelection = 0
}

func (g *Game) loadBindings() {
	bindings, err := LoadButtonBindings()
	if err != nil {
		g.bindingsMessage = "Default buttons. Select / F2 to configure."
		return
	}
	g.input.UseCustomBindings(bindings)
	g.bindingsMessage = "Custom buttons loaded. Select / F2 to reconfigure."
}

func (g *Game) updatePlaying() {
	if g.controls.JustPressed&ActionPause != 0 {
		g.switchState(StatePaused)
		return
	}
	g.simulate()
	if g.player.Health <= 0 {
		g.recordHighScore()
		g.switchState(StateGameOver)
		return
	}
	g.openLevelUpIfPending()
}

func (g *Game) simulate() {
	defer g.measureSimulation(time.Now())
	g.frame++
	g.elapsedSeconds += deltaSeconds
	g.grid.Rebuild(g.enemies.PositionX, g.enemies.PositionY, g.enemies.Count)
	g.updatePlayer(deltaSeconds)
	g.updateWeapons(deltaSeconds)
	g.updateMines(deltaSeconds)
	g.updateProjectiles(deltaSeconds)
	g.updateEnemies(deltaSeconds)
	g.processBursts()
	g.removeDeadEnemies()
	g.updateSpiders(deltaSeconds)
	g.updateSpawning(deltaSeconds)
	g.spawnSpiderIfDue()
	g.tickGroundIfDue()
	g.effects.Update(deltaSeconds)
	g.levelUpBannerSeconds = max(0, g.levelUpBannerSeconds-deltaSeconds)
	g.musicBannerSeconds = max(0, g.musicBannerSeconds-deltaSeconds)
}

func (g *Game) measureSimulation(started time.Time) {
	elapsedMillis := float32(time.Since(started).Microseconds()) / 1000
	g.simulationMillis += (elapsedMillis - g.simulationMillis) * simulationTimeSmoothing
}

func (g *Game) tickGroundIfDue() {
	if g.frame%groundTickFrames != 0 {
		return
	}
	g.ground.Tick()
}

func (g *Game) openLevelUpIfPending() {
	if g.player.PendingLevelUps == 0 {
		return
	}
	g.buildUpgradeOffers()
	g.playSound(SoundLevelUp)
	if len(g.offers) == 1 {
		g.applySoleOffer()
		return
	}
	g.switchState(StateLevelUp)
}

func (g *Game) applySoleOffer() {
	offer := g.offers[0]
	g.applyOffer(offer)
	g.levelUpBanner = "LEVEL UP!   " + offer.Title() + "  " + offer.Subtitle()
	g.levelUpBannerSeconds = levelUpBannerDuration
}

func (g *Game) updateLevelUp() {
	isPrevious := g.controls.JustPressed&selectionDirections != 0
	isNext := g.controls.JustPressed&(ActionRight|ActionDown) != 0
	offerCount := len(g.offers)
	g.selectedOffer = (g.selectedOffer + offerCount + boolToIndex(isNext) - boolToIndex(isPrevious)) % offerCount
	if !g.isConfirming() {
		return
	}
	g.applyOffer(g.offers[g.selectedOffer])
	g.switchState(StatePlaying)
	g.openLevelUpIfPending()
}

func (g *Game) drawPlaying(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawHud(g, screen)
}

func (g *Game) drawLevelUp(screen *ebiten.Image) {
	g.drawPlaying(screen)
	g.ui.DrawLevelUp(g, screen)
}
