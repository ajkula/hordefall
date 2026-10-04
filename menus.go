package main

import "github.com/hajimehoshi/ebiten/v2"

// ===== Types =====

type MenuOption struct {
	Label    func(game *Game) string
	Activate func(game *Game)
	Slider   *MenuSlider
}

type MenuSlider struct {
	Steps  int
	Level  func(game *Game) int
	Adjust func(game *Game, delta int)
}

// ===== Constants =====

var mainMenuOptions = []MenuOption{
	{fixedLabel("Play"), (*Game).startRun, nil},
	{fixedLabel("Options"), (*Game).openOptions, nil},
	{fixedLabel("Quit"), (*Game).requestQuit, nil},
}

var optionsMenuOptions = []MenuOption{
	{fixedLabel("Configure buttons"), (*Game).openRemapFromOptions, nil},
	{fixedLabel("Graphics"), (*Game).openGraphics, nil},
	{fixedLabel("Audio"), (*Game).openAudioMenu, nil},
	{fixedLabel("Back"), (*Game).closeOptions, nil},
}

var pauseMenuOptions = []MenuOption{
	{fixedLabel("Resume"), (*Game).resumeRun, nil},
	{(*Game).musicLabel, (*Game).toggleMusic, nil},
	{fixedLabel("Back to main menu"), (*Game).abandonRun, nil},
}

var musicLabels = [2]string{"Music: OFF", "Music: ON"}

const (
	mainMenuOptionSpacing = 48
	menuOptionSpacing     = 40
	menuSliderExtraHeight = 18
	menuSliderOffset      = 32
	menuBackActions       = ActionPause | ActionAimLock
	menuListBottom        = mainMenuOptionsTop + 2*mainMenuOptionSpacing
)

// ===== Internal =====

func (g *Game) navigateMenu(options []MenuOption) {
	if g.adjustSelectedSlider(options[g.menuSelection].Slider) {
		return
	}
	isPrevious := g.controls.JustPressed&selectionDirections != 0
	isNext := g.controls.JustPressed&(ActionRight|ActionDown) != 0
	count := len(options)
	g.menuSelection = (g.menuSelection + count + boolToIndex(isNext) - boolToIndex(isPrevious)) % count
	g.playSoundIf(SoundMenuMove, isNext || isPrevious)
	if !g.isConfirming() {
		return
	}
	g.playSound(SoundMenuSelect)
	options[g.menuSelection].Activate(g)
}

func (g *Game) isGoingBack() bool {
	return g.menuLockSeconds == 0 && g.controls.JustPressed&menuBackActions != 0
}

func (g *Game) adjustSelectedSlider(slider *MenuSlider) bool {
	delta := boolToIndex(g.controls.JustPressed&ActionRight != 0) - boolToIndex(g.controls.JustPressed&ActionLeft != 0)
	if slider == nil || delta == 0 {
		return false
	}
	slider.Adjust(g, delta)
	g.playSound(SoundMenuMove)
	return true
}

func menuOptionPositions(options []MenuOption, spacing, minimumTop float32, positions []float32) []float32 {
	positions = positions[:0]
	span := float32(0)
	for index := range len(options) - 1 {
		span += spacing + menuSliderExtraHeight*boolToFloat(options[index].Slider != nil)
	}
	compression := min(1, (menuListBottom-minimumTop)/max(1, span))
	y := menuListBottom - span*compression
	for index := range options {
		positions = append(positions, y)
		y += (spacing + menuSliderExtraHeight*boolToFloat(options[index].Slider != nil)) * compression
	}
	return positions
}

func (g *Game) startRun() {
	g.resetRun()
	g.switchState(StatePlaying)
}

func (g *Game) goHome() {
	g.startDemo()
	g.switchState(StateMainMenu)
}

func fixedLabel(label string) func(game *Game) string {
	return func(*Game) string { return label }
}

func (g *Game) musicLabel() string {
	return musicLabels[boolToIndex(g.settings.IsMusicOn)]
}

func (g *Game) openOptions() {
	g.switchState(StateOptions)
}

func (g *Game) closeOptions() {
	g.switchState(StateMainMenu)
}

func (g *Game) openRemapFrom(returnState GameState) {
	g.remapStep = 0
	g.remapReturnState = returnState
	g.switchState(StateRemap)
}

func (g *Game) openRemapFromOptions() {
	g.openRemapFrom(StateOptions)
}

func (g *Game) requestQuit() {
	g.isQuitRequested = true
}

func (g *Game) resumeRun() {
	g.switchState(StatePlaying)
}

func (g *Game) abandonRun() {
	g.recordHighScore()
	g.goHome()
}

func (g *Game) updateMainMenu() {
	g.updateDemo()
	if g.controls.JustPressed&ActionSelect != 0 {
		g.openRemapFrom(StateMainMenu)
		return
	}
	g.navigateMenu(mainMenuOptions)
}

func (g *Game) updateOptions() {
	g.updateDemo()
	if g.isGoingBack() {
		g.closeOptions()
		return
	}
	g.navigateMenu(optionsMenuOptions)
}

func (g *Game) updatePaused() {
	if g.isGoingBack() {
		g.resumeRun()
		return
	}
	g.navigateMenu(pauseMenuOptions)
}

func (g *Game) updateGameOver() {
	if g.menuLockSeconds > 0 {
		return
	}
	if g.controls.JustPressed&ActionStart != 0 {
		g.startRun()
		return
	}
	if g.controls.JustPressed&ActionSelect != 0 {
		g.goHome()
	}
}

func (g *Game) updateRemap() {
	g.updateDemo()
	if g.controls.JustPressed&ActionPause != 0 {
		g.switchState(g.remapReturnState)
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
	g.switchState(g.remapReturnState)
}

func (g *Game) drawMainMenu(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawMainMenu(g, screen, mainMenuOptions)
}

func (g *Game) drawOptions(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawOptionsMenu(g, screen, optionsMenuOptions)
}

func (g *Game) drawPaused(screen *ebiten.Image) {
	g.drawPlaying(screen)
	g.ui.DrawPauseMenu(g, screen, pauseMenuOptions)
}

func (g *Game) drawGameOver(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawGameOver(g, screen)
}

func (g *Game) drawRemap(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawRemap(screen, g.remapStep, g.remapButtons)
}
