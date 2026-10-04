package main

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ===== Constants =====

const (
	volumeSteps          = 5
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
	Steps:  volumeSteps,
	Level:  func(game *Game) int { return game.settings.MusicVolumeStep },
	Adjust: (*Game).adjustMusicVolume,
}

var effectsVolumeSlider = MenuSlider{
	Steps:  volumeSteps,
	Level:  func(game *Game) int { return game.settings.EffectsVolumeStep },
	Adjust: (*Game).adjustEffectsVolume,
}

var audioMenuOptions = []MenuOption{
	{(*Game).musicLabel, (*Game).toggleMusic, nil},
	{(*Game).musicVolumeLabel, (*Game).cycleMusicVolume, &musicVolumeSlider},
	{(*Game).effectsVolumeLabel, (*Game).cycleEffectsVolume, &effectsVolumeSlider},
	{fixedLabel("Playlist"), (*Game).openPlaylist, nil},
	{fixedLabel("Back"), (*Game).closeAudioMenu, nil},
}

// ===== Public API =====

func (u *UI) DrawAudioMenu(g *Game, screen *ebiten.Image, options []MenuOption) {
	dimScreen(screen, menuDim)
	u.drawText(screen, "AUDIO", u.title, screenWidth/2, menuTitleTop, accentColor, 1, text.AlignCenter)
	u.drawText(screen, g.settingsMessage, u.small, screenWidth/2, menuTitleTop+110, healthColor, 1, text.AlignCenter)
	u.drawMenuOptions(g, screen, options, menuOptionSpacing)
	u.drawBenchmarkPanel(g, screen)
	u.drawMusicBanner(g, screen)
	u.drawText(screen, "Up / Down to choose, Left / Right to set the volume, Fire to confirm, Start / Esc / Aim lock to go back", u.small, screenWidth/2, menuHintTop, textColor, 1, text.AlignCenter)
	u.drawText(screen, "Music OFF keeps the sound effects. Settings are saved.", u.small, screenWidth/2, menuHintTop+24, textColor, 1, text.AlignCenter)
}

// ===== Internal =====

func (g *Game) openAudioMenu() {
	g.settingsMessage = ""
	g.switchState(StateAudio)
}

func (g *Game) closeAudioMenu() {
	g.switchState(StateOptions)
}

func (g *Game) updateAudioMenu() {
	g.updateDemo()
	if g.isGoingBack() {
		g.closeAudioMenu()
		return
	}
	g.navigateMenu(audioMenuOptions)
}

func (g *Game) drawAudioMenu(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawAudioMenu(g, screen, audioMenuOptions)
}

func (g *Game) musicVolumeLabel() string {
	return volumeLabel("Music volume", g.settings.MusicVolumeStep)
}

func (g *Game) effectsVolumeLabel() string {
	return volumeLabel("Effects volume", g.settings.EffectsVolumeStep)
}

func volumeLabel(name string, step int) string {
	return name + ": " + strconv.Itoa(step*100/volumeSteps) + "%"
}

func (g *Game) adjustMusicVolume(delta int) {
	g.setVolumeStep(&g.settings.MusicVolumeStep, g.settings.MusicVolumeStep+delta)
}

func (g *Game) cycleMusicVolume() {
	g.setVolumeStep(&g.settings.MusicVolumeStep, (g.settings.MusicVolumeStep+1)%(volumeSteps+1))
}

func (g *Game) adjustEffectsVolume(delta int) {
	g.setVolumeStep(&g.settings.EffectsVolumeStep, g.settings.EffectsVolumeStep+delta)
}

func (g *Game) cycleEffectsVolume() {
	g.setVolumeStep(&g.settings.EffectsVolumeStep, (g.settings.EffectsVolumeStep+1)%(volumeSteps+1))
	g.playSound(SoundMenuMove)
}

func (g *Game) setVolumeStep(step *int, value int) {
	*step = clampInt(value, 0, volumeSteps)
	g.applyMusicVolume()
	g.saveSettings()
}

func (g *Game) effectsVolumeScale() float32 {
	return float32(g.settings.EffectsVolumeStep) / volumeSteps
}

func (g *Game) applyMusicVolume() {
	g.audio.SetMusicVolume(maximumMusicVolume * float32(g.settings.MusicVolumeStep) / volumeSteps)
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
