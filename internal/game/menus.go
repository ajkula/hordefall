package game

import (
	"github.com/hajimehoshi/ebiten/v2"

	"hordefall/internal/audio"
	"hordefall/internal/i18n"
)

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
	{translatedLabel("menu.play"), (*Game).startRun, nil},
	{translatedLabel("menu.options"), (*Game).openOptions, nil},
	{translatedLabel("menu.quit"), (*Game).requestQuit, nil},
}

var optionsMenuOptions = []MenuOption{
	{translatedLabel("menu.controls"), (*Game).openRemapFromOptions, nil},
	{translatedLabel("menu.gameplay"), (*Game).openGameplay, nil},
	{translatedLabel("menu.graphics"), (*Game).openGraphics, nil},
	{translatedLabel("menu.audio"), (*Game).openAudioMenu, nil},
	{translatedLabel("menu.back"), (*Game).closeOptions, nil},
}

var pauseMenuOptions = []MenuOption{
	{translatedLabel("menu.resume"), (*Game).resumeRun, nil},
	{translatedLabel("menu.options"), (*Game).openOptionsFromPause, nil},
	{(*Game).musicLabel, (*Game).toggleMusic, nil},
	{translatedLabel("menu.back_to_main"), (*Game).abandonRun, nil},
}

var musicLabelKeys = [2]string{"menu.music_off", "menu.music_on"}

const (
	mainMenuOptionSpacing = 48
	menuOptionSpacing     = 40
	menuSliderExtraHeight = 18
	menuSliderOffset      = 32
	menuBackActions       = ActionPause | ActionAimLock
	menuListBottom        = mainMenuOptionsTop + 2*mainMenuOptionSpacing
	gameOverIdleSeconds   = 20
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
	g.playSoundIf(audio.SoundMenuMove, isNext || isPrevious)
	if !g.isConfirming() {
		return
	}
	g.playSound(audio.SoundMenuSelect)
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
	g.playSound(audio.SoundMenuMove)
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

func (g *Game) musicLabel() string {
	return i18n.T(musicLabelKeys[boolToIndex(g.settings.IsMusicOn)])
}

func (g *Game) openOptions() {
	g.optionsReturnState = StateMainMenu
	g.switchState(StateOptions)
}

func (g *Game) openOptionsFromPause() {
	g.optionsReturnState = StatePaused
	g.switchState(StateOptions)
}

func (g *Game) closeOptions() {
	g.switchState(g.optionsReturnState)
}

func (g *Game) updateBackdropDemo() {
	if !g.isDemo {
		return
	}
	g.updateDemo()
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
	if g.dismissDemoBoardIfPressed() {
		return
	}
	if g.controls.JustPressed&ActionSelect != 0 {
		g.toggleBenchmarkPanel()
		return
	}
	g.navigateMenu(mainMenuOptions)
}

func (g *Game) toggleBenchmarkPanel() {
	g.settings.IsBenchmarkHidden = !g.settings.IsBenchmarkHidden
	g.saveSettings()
}

func (g *Game) updateOptions() {
	g.updateBackdropDemo()
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
	g.stateTicks++
	if g.stateTicks >= gameOverIdleSeconds*ticksPerSecond {
		g.goHomeWithRanking()
		return
	}
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

func (g *Game) drawMainMenu(screen *ebiten.Image) {
	if g.highScoreBoardSeconds > 0 {
		g.ui.DrawRankingAttract(g, screen, boardElapsedSeconds(g.highScoreBoardSeconds))
		return
	}
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
