package main

import "github.com/hajimehoshi/ebiten/v2"

// ===== Types =====

type MenuOption struct {
	Label    string
	Activate func(game *Game)
}

// ===== Constants =====

var mainMenuOptions = []MenuOption{
	{"Play", (*Game).startRun},
	{"Configure buttons", (*Game).openRemap},
	{"Quit", (*Game).requestQuit},
}

var pauseMenuOptions = []MenuOption{
	{"Resume", (*Game).resumeRun},
	{"Back to main menu", (*Game).abandonRun},
}

// ===== Internal =====

func (g *Game) navigateMenu(options []MenuOption) {
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

func (g *Game) startRun() {
	g.resetRun()
	g.switchState(StatePlaying)
}

func (g *Game) goHome() {
	g.startDemo()
	g.switchState(StateMainMenu)
}

func (g *Game) openRemap() {
	g.remapStep = 0
	g.switchState(StateRemap)
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
		g.openRemap()
		return
	}
	g.navigateMenu(mainMenuOptions)
}

func (g *Game) updatePaused() {
	if g.menuLockSeconds == 0 && g.controls.JustPressed&ActionPause != 0 {
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
		g.switchState(StateMainMenu)
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
	g.switchState(StateMainMenu)
}

func (g *Game) drawMainMenu(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawMainMenu(g, screen, mainMenuOptions)
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
