package main

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ===== Constants =====

const (
	musicVolumeSteps     = 5
	maximumMusicVolume   = 0.5
	sliderTrayWidth      = 220
	sliderSlotHeight     = 10
	sliderSlotGap        = 4
	sliderRimThickness   = 2
	sliderRimHeight      = 8
	sliderTrayPadding    = 3
	sliderOutlineMargin  = 1
	sliderEmptySlotAlpha = 0.55
)

var musicVolumeSlider = MenuSlider{
	Steps:  musicVolumeSteps,
	Level:  func(game *Game) int { return game.settings.MusicVolumeStep },
	Adjust: (*Game).adjustMusicVolume,
}

var musicMenuOptions = []MenuOption{
	{(*Game).musicLabel, (*Game).toggleMusic, nil},
	{(*Game).musicVolumeLabel, (*Game).cycleMusicVolume, &musicVolumeSlider},
	{fixedLabel("Playlist"), (*Game).openPlaylist, nil},
	{fixedLabel("Back"), (*Game).closeMusicMenu, nil},
}

// ===== Public API =====

func (u *UI) DrawMusicMenu(g *Game, screen *ebiten.Image, options []MenuOption) {
	dimScreen(screen, menuDim)
	u.drawText(screen, "MUSIC", u.title, screenWidth/2, menuTitleTop, accentColor, 1, text.AlignCenter)
	u.drawText(screen, g.settingsMessage, u.small, screenWidth/2, menuTitleTop+110, healthColor, 1, text.AlignCenter)
	u.drawMenuOptions(g, screen, options, menuOptionSpacing)
	u.drawBenchmarkPanel(g, screen)
	u.drawMusicBanner(g, screen)
	u.drawText(screen, "Up / Down to choose, Left / Right to set the volume, Fire to confirm, Start / Esc to go back", u.small, screenWidth/2, menuHintTop, textColor, 1, text.AlignCenter)
	u.drawText(screen, "Music only: sound effects stay on. Settings are saved.", u.small, screenWidth/2, menuHintTop+24, textColor, 1, text.AlignCenter)
}

// ===== Internal =====

func (g *Game) openMusicMenu() {
	g.settingsMessage = ""
	g.switchState(StateMusic)
}

func (g *Game) closeMusicMenu() {
	g.switchState(StateOptions)
}

func (g *Game) updateMusicMenu() {
	g.updateDemo()
	if g.menuLockSeconds == 0 && g.controls.JustPressed&ActionPause != 0 {
		g.closeMusicMenu()
		return
	}
	g.navigateMenu(musicMenuOptions)
}

func (g *Game) drawMusicMenu(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawMusicMenu(g, screen, musicMenuOptions)
}

func (g *Game) musicVolumeLabel() string {
	return "Music volume: " + strconv.Itoa(g.settings.MusicVolumeStep*100/musicVolumeSteps) + "%"
}

func (g *Game) adjustMusicVolume(delta int) {
	g.setMusicVolumeStep(clampInt(g.settings.MusicVolumeStep+delta, 0, musicVolumeSteps))
}

func (g *Game) cycleMusicVolume() {
	g.setMusicVolumeStep((g.settings.MusicVolumeStep + 1) % (musicVolumeSteps + 1))
}

func (g *Game) setMusicVolumeStep(step int) {
	g.settings.MusicVolumeStep = step
	g.applyMusicVolume()
	g.saveSettings()
}

func (g *Game) applyMusicVolume() {
	g.audio.SetMusicVolume(maximumMusicVolume * float32(g.settings.MusicVolumeStep) / musicVolumeSteps)
}

func (u *UI) drawSliderIf(g *Game, screen *ebiten.Image, slider *MenuSlider, top float32) {
	if slider == nil {
		return
	}
	left := float32(screenWidth/2 - sliderTrayWidth/2)
	trayBottom := top + sliderSlotHeight + sliderTrayPadding
	outlineLeft := left - sliderTrayPadding - sliderRimThickness - sliderOutlineMargin
	outlineWidth := float32(sliderTrayWidth + 2*(sliderTrayPadding+sliderRimThickness+sliderOutlineMargin))
	outlineTop := trayBottom + sliderRimThickness - sliderRimHeight - sliderOutlineMargin
	fillRect(screen, outlineLeft, outlineTop, outlineWidth, sliderRimHeight+2*sliderOutlineMargin, toColor(outlineColor, 0.85))
	fillRect(screen, left-sliderTrayPadding, top-1, sliderTrayWidth+2*sliderTrayPadding, sliderSlotHeight+2, toColor(outlineColor, 0.85))
	u.drawSliderTray(screen, left, trayBottom)
	slotWidth := (sliderTrayWidth - float32(slider.Steps-1)*sliderSlotGap) / float32(slider.Steps)
	level := slider.Level(g)
	for slot := range slider.Steps {
		isFilled := slot < level
		slotColors := [2][3]float32{panelColor, accentColor}
		slotAlphas := [2]float32{sliderEmptySlotAlpha, 1}
		x := left + float32(slot)*(slotWidth+sliderSlotGap)
		fillRect(screen, x, top, slotWidth, sliderSlotHeight, toColor(slotColors[boolToIndex(isFilled)], slotAlphas[boolToIndex(isFilled)]))
	}
}

func (u *UI) drawSliderTray(screen *ebiten.Image, left, trayBottom float32) {
	rimLeft := left - sliderTrayPadding - sliderRimThickness
	rimRight := left + sliderTrayWidth + sliderTrayPadding
	rimTop := trayBottom + sliderRimThickness - sliderRimHeight
	trayColor := toColor(textColor, 1)
	fillRect(screen, rimLeft, trayBottom, sliderTrayWidth+2*(sliderTrayPadding+sliderRimThickness), sliderRimThickness, trayColor)
	fillRect(screen, rimLeft, rimTop, sliderRimThickness, sliderRimHeight, trayColor)
	fillRect(screen, rimRight, rimTop, sliderRimThickness, sliderRimHeight, trayColor)
}
