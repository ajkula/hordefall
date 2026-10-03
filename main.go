package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// ===== Types =====

type GameState uint8

type StateHandler struct {
	Update func(game *Game)
	Draw   func(game *Game, screen *ebiten.Image)
}

type Game struct {
	state                GameState
	input                InputReader
	controls             Controls
	random               Random
	player               *Player
	enemies              *EnemyStore
	projectiles          *ProjectileStore
	gems                 *GemStore
	effects              *Effects
	ground               *Ground
	grid                 *SpatialGrid
	renderer             *Renderer
	ui                   *UI
	director             SpawnDirector
	bursts               []Burst
	candidates           []int32
	separationCandidates []int32
	chainedEnemies       []int32
	spiders              []SpiderRig
	nextSpiderKills      int
	offers               []UpgradeOffer
	offerPool            []UpgradeOffer
	selectedOffer        int
	menuLockSeconds      float32
	reactionCounts       [reactionKindCount]int
	elapsedSeconds       float32
	clockSeconds         float32
	kills                int
	frame                uint32
	runCount             uint32
	isInputDebugVisible  bool
	remapStep            int
	remapButtons         [3]int
	bindingsMessage      string
}

// ===== Constants =====

const (
	StateTitle GameState = iota
	StatePlaying
	StateLevelUp
	StatePaused
	StateGameOver
	StateRemap
	stateCount
)

const (
	screenWidth         = 1280
	screenHeight        = 720
	arenaSize           = 4096
	ticksPerSecond      = 60
	deltaSeconds        = 1.0 / ticksPerSecond
	spatialCellSize     = 32
	menuLockDuration    = 0.45
	groundSeedBase      = 0x5EED
	selectionDirections = ActionLeft | ActionUp
)

var stateHandlers = [stateCount]StateHandler{
	StateTitle:    {(*Game).updateTitle, (*Game).drawTitle},
	StatePlaying:  {(*Game).updatePlaying, (*Game).drawPlaying},
	StateLevelUp:  {(*Game).updateLevelUp, (*Game).drawLevelUp},
	StatePaused:   {(*Game).updatePaused, (*Game).drawPaused},
	StateGameOver: {(*Game).updateGameOver, (*Game).drawGameOver},
	StateRemap:    {(*Game).updateRemap, (*Game).drawRemap},
}

// ===== Public API =====

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Hordefall")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
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
	}
	game.resetRun()
	game.renderer = NewRenderer(game.ground.Columns, game.ground.Rows)
	game.loadBindings()
	return game
}

func (g *Game) Update() error {
	g.controls = g.input.Read()
	isSelectTogglingDebug := g.controls.JustPressed&ActionSelect != 0 && g.state != StateTitle && g.state != StateRemap
	g.isInputDebugVisible = g.isInputDebugVisible != (g.controls.JustPressed&ActionDebug != 0 || isSelectTogglingDebug)
	g.frame++
	g.clockSeconds += deltaSeconds
	g.menuLockSeconds = max(0, g.menuLockSeconds-deltaSeconds)
	stateHandlers[g.state].Update(g)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	stateHandlers[g.state].Draw(g, screen)
	if !g.isInputDebugVisible {
		return
	}
	g.ui.DrawDebugOverlay(g.input.DescribeDevices(g.controls), screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
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
	g.elapsedSeconds, g.kills = 0, 0
	g.spiders = g.spiders[:0]
	g.nextSpiderKills = firstSpiderKills
}

func (g *Game) isConfirming() bool {
	return g.menuLockSeconds == 0 && g.controls.JustPressed&ActionConfirm != 0
}

func (g *Game) switchState(state GameState) {
	g.state = state
	g.menuLockSeconds = menuLockDuration
}

func (g *Game) loadBindings() {
	bindings, err := LoadButtonBindings()
	if err != nil {
		g.bindingsMessage = "Default buttons. Press Select / F2 to configure."
		return
	}
	g.input.UseCustomBindings(bindings)
	g.bindingsMessage = "Custom buttons loaded. Press Select / F2 to reconfigure."
}

func (g *Game) updateTitle() {
	g.tickGroundIfDue()
	if g.controls.JustPressed&ActionSelect != 0 {
		g.remapStep = 0
		g.switchState(StateRemap)
		return
	}
	if !g.isConfirming() {
		return
	}
	g.switchState(StatePlaying)
}

func (g *Game) updatePlaying() {
	if g.controls.JustPressed&ActionPause != 0 {
		g.switchState(StatePaused)
		return
	}
	g.simulate()
	if g.player.Health <= 0 {
		g.switchState(StateGameOver)
		return
	}
	g.openLevelUpIfPending()
}

func (g *Game) simulate() {
	g.elapsedSeconds += deltaSeconds
	g.grid.Rebuild(g.enemies.PositionX, g.enemies.PositionY, g.enemies.Count)
	g.updatePlayer(deltaSeconds)
	g.updateWeapons(deltaSeconds)
	g.updateProjectiles(deltaSeconds)
	g.updateEnemies(deltaSeconds)
	g.processBursts()
	g.removeDeadEnemies()
	g.updateSpiders(deltaSeconds)
	g.updateSpawning(deltaSeconds)
	g.spawnSpiderIfDue()
	g.tickGroundIfDue()
	g.effects.Update(deltaSeconds)
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
	g.switchState(StateLevelUp)
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

func (g *Game) updatePaused() {
	if g.controls.JustPressed&ActionPause == 0 {
		return
	}
	g.state = StatePlaying
}

func (g *Game) updateGameOver() {
	g.effects.Update(deltaSeconds)
	if !g.isConfirming() {
		return
	}
	g.resetRun()
	g.switchState(StatePlaying)
}

func (g *Game) updateRemap() {
	if g.controls.JustPressed&ActionPause != 0 {
		g.switchState(StateTitle)
		return
	}
	button := g.controls.RawJustPressed
	isSystemButton := g.controls.Held&(ActionPause|ActionSelect) != 0
	if button < 0 || isSystemButton || g.isAlreadyRemapped(button) {
		return
	}
	g.remapButtons[g.remapStep] = button
	g.remapStep++
	if g.remapStep < len(g.remapButtons) {
		return
	}
	g.finishRemap()
}

func (g *Game) isAlreadyRemapped(button int) bool {
	for step := range g.remapStep {
		if g.remapButtons[step] == button {
			return true
		}
	}
	return false
}

func (g *Game) finishRemap() {
	bindings := BindingsFromSteps(g.remapButtons)
	g.input.UseCustomBindings(bindings)
	g.bindingsMessage = "Buttons saved."
	if err := SaveButtonBindings(bindings); err != nil {
		g.bindingsMessage = "Buttons active for this session, save failed: " + err.Error()
	}
	g.switchState(StateTitle)
}

func (g *Game) drawTitle(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawTitle(screen, g.clockSeconds, g.bindingsMessage)
}

func (g *Game) drawRemap(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawRemap(screen, g.remapStep, g.remapButtons)
}

func (g *Game) drawPlaying(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawHud(g, screen)
}

func (g *Game) drawLevelUp(screen *ebiten.Image) {
	g.drawPlaying(screen)
	g.ui.DrawLevelUp(g, screen)
}

func (g *Game) drawPaused(screen *ebiten.Image) {
	g.drawPlaying(screen)
	g.ui.DrawPaused(screen)
}

func (g *Game) drawGameOver(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawGameOver(g, screen)
}
