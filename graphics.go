package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ===== Types =====

type RenderResolution uint8

type ResolutionChoice struct {
	Label string
	Scale func(outsideWidth, outsideHeight int) float32
}

type GraphicsLevel struct {
	Label  string
	Factor float32
}

// ===== Constants =====

const (
	Resolution720p RenderResolution = iota
	Resolution1080p
	ResolutionNative
	resolutionCount
)

const (
	graphicsLevelCount   = 3
	maximumRenderScale   = 3
	defaultGraphicsLevel = graphicsLevelCount - 1
	defaultBloomLevel    = 1
)

var resolutionChoices = [resolutionCount]ResolutionChoice{
	Resolution720p:   {"720p", fixedRenderScale(1)},
	Resolution1080p:  {"1080p", fixedRenderScale(1.5)},
	ResolutionNative: {"Native", nativeRenderScale},
}

var shakeLevels = [graphicsLevelCount]GraphicsLevel{{"Off", 0}, {"50%", 0.5}, {"100%", 1}}

var effectsLevels = [graphicsLevelCount]GraphicsLevel{{"Low", 0.3}, {"Medium", 0.6}, {"High", 1}}

var onOffLabels = [2]string{"OFF", "ON"}

var graphicsMenuOptions = []MenuOption{
	{(*Game).fullscreenLabel, (*Game).toggleFullscreen, nil},
	{(*Game).vsyncLabel, (*Game).toggleVsync, nil},
	{(*Game).resolutionLabel, (*Game).cycleResolution, nil},
	{(*Game).fpsLabel, (*Game).toggleFPS, nil},
	{(*Game).shakeLabel, (*Game).cycleShake, nil},
	{(*Game).effectsLabel, (*Game).cycleEffects, nil},
	{(*Game).bloomLabel, (*Game).cycleBloom, nil},
	{(*Game).crtLabel, (*Game).toggleCRT, nil},
	{fixedLabel("Back"), (*Game).closeGraphics, nil},
}

var renderScale float32 = 1

// ===== Public API =====

func (u *UI) DrawGraphicsMenu(g *Game, screen *ebiten.Image, options []MenuOption) {
	dimScreen(screen, menuDim)
	u.drawText(screen, "GRAPHICS", u.title, screenWidth/2, menuTitleTop, accentColor, 1, text.AlignCenter)
	u.drawText(screen, g.settingsMessage, u.small, screenWidth/2, menuTitleTop+110, healthColor, 1, text.AlignCenter)
	u.drawMenuOptions(g, screen, options, menuOptionSpacing)
	u.drawBenchmarkPanel(g, screen)
	u.drawMusicBanner(g, screen)
	u.drawText(screen, "Up / Down to choose, Fire to change, Start / Esc / Aim lock to go back. F11 toggles fullscreen anywhere.", u.small, screenWidth/2, menuHintTop, textColor, 1, text.AlignCenter)
	u.drawText(screen, "Native renders at your screen resolution: sharper, heavier on the GPU.", u.small, screenWidth/2, menuHintTop+24, textColor, 1, text.AlignCenter)
}

// ===== Internal =====

func fixedRenderScale(scale float32) func(outsideWidth, outsideHeight int) float32 {
	return func(int, int) float32 { return scale }
}

func nativeRenderScale(outsideWidth, outsideHeight int) float32 {
	deviceScale := float32(ebiten.Monitor().DeviceScaleFactor())
	fittedScale := min(float32(outsideWidth)/screenWidth, float32(outsideHeight)/screenHeight)
	return clamp(fittedScale*deviceScale, 1, maximumRenderScale)
}

func sanitizeGraphics(settings *Settings) {
	settings.Resolution = min(settings.Resolution, resolutionCount-1)
	settings.ShakeLevel = clampInt(settings.ShakeLevel, 0, graphicsLevelCount-1)
	settings.EffectsLevel = clampInt(settings.EffectsLevel, 0, graphicsLevelCount-1)
	settings.BloomLevel = clampInt(settings.BloomLevel, 0, graphicsLevelCount-1)
}

func (g *Game) applyGraphicsSettings() {
	ebiten.SetFullscreen(g.settings.IsFullscreen)
	ebiten.SetVsyncEnabled(g.settings.IsVsyncOn)
	g.effects.ShakeFactor = shakeLevels[g.settings.ShakeLevel].Factor
	g.effects.Density = effectsLevels[g.settings.EffectsLevel].Factor
}

func (g *Game) changeGraphics(change func(settings *Settings)) {
	change(&g.settings)
	g.applyGraphicsSettings()
	g.saveSettings()
}

func (g *Game) toggleFullscreenIfRequested() {
	if g.controls.JustPressed&ActionFullscreen == 0 {
		return
	}
	g.toggleFullscreen()
}

func (g *Game) toggleFullscreen() {
	g.changeGraphics(func(settings *Settings) { settings.IsFullscreen = !settings.IsFullscreen })
}

func (g *Game) toggleVsync() {
	g.changeGraphics(func(settings *Settings) { settings.IsVsyncOn = !settings.IsVsyncOn })
}

func (g *Game) toggleFPS() {
	g.changeGraphics(func(settings *Settings) { settings.IsFPSShown = !settings.IsFPSShown })
}

func (g *Game) cycleResolution() {
	g.changeGraphics(func(settings *Settings) { settings.Resolution = (settings.Resolution + 1) % resolutionCount })
}

func (g *Game) cycleShake() {
	g.changeGraphics(func(settings *Settings) { settings.ShakeLevel = (settings.ShakeLevel + 1) % graphicsLevelCount })
}

func (g *Game) cycleEffects() {
	g.changeGraphics(func(settings *Settings) { settings.EffectsLevel = (settings.EffectsLevel + 1) % graphicsLevelCount })
}

func (g *Game) cycleBloom() {
	g.changeGraphics(func(settings *Settings) { settings.BloomLevel = (settings.BloomLevel + 1) % graphicsLevelCount })
}

func (g *Game) toggleCRT() {
	g.changeGraphics(func(settings *Settings) { settings.IsCRTOn = !settings.IsCRTOn })
}

func (g *Game) postSettings() PostSettings {
	return PostSettings{BloomIntensity: bloomLevels[g.settings.BloomLevel].Factor, IsCRTOn: g.settings.IsCRTOn}
}

func (g *Game) bloomLabel() string {
	return "Bloom: " + bloomLevels[g.settings.BloomLevel].Label
}

func (g *Game) crtLabel() string {
	return "CRT filter: " + onOffLabels[boolToIndex(g.settings.IsCRTOn)]
}

func (g *Game) fullscreenLabel() string {
	return "Fullscreen: " + onOffLabels[boolToIndex(g.settings.IsFullscreen)]
}

func (g *Game) vsyncLabel() string {
	return "VSync: " + onOffLabels[boolToIndex(g.settings.IsVsyncOn)]
}

func (g *Game) resolutionLabel() string {
	return "Resolution: " + resolutionChoices[g.settings.Resolution].Label
}

func (g *Game) fpsLabel() string {
	return "Show FPS: " + onOffLabels[boolToIndex(g.settings.IsFPSShown)]
}

func (g *Game) shakeLabel() string {
	return "Screen shake: " + shakeLevels[g.settings.ShakeLevel].Label
}

func (g *Game) effectsLabel() string {
	return "Effects: " + effectsLevels[g.settings.EffectsLevel].Label
}

func (g *Game) openGraphics() {
	g.settingsMessage = ""
	g.switchState(StateGraphics)
}

func (g *Game) closeGraphics() {
	g.switchState(StateOptions)
}

func (g *Game) updateGraphics() {
	g.updateDemo()
	if g.isGoingBack() {
		g.closeGraphics()
		return
	}
	g.navigateMenu(graphicsMenuOptions)
}

func (g *Game) drawGraphics(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawGraphicsMenu(g, screen, graphicsMenuOptions)
}

func (u *UI) scaledFace(face *text.GoTextFace) *text.GoTextFace {
	if u.faceScale != renderScale {
		clear(u.scaledFaces)
		u.faceScale = renderScale
	}
	scaled, isCached := u.scaledFaces[face]
	if isCached {
		return scaled
	}
	scaled = &text.GoTextFace{Source: face.Source, Size: face.Size * float64(renderScale)}
	u.scaledFaces[face] = scaled
	return scaled
}

func fillRect(screen *ebiten.Image, x, y, width, height float32, tint color.Color) {
	vector.FillRect(screen, x*renderScale, y*renderScale, width*renderScale, height*renderScale, tint, false)
}

func strokeRect(screen *ebiten.Image, x, y, width, height, strokeWidth float32, tint color.Color) {
	vector.StrokeRect(screen, x*renderScale, y*renderScale, width*renderScale, height*renderScale, strokeWidth*renderScale, tint, false)
}
