package game

import (
	"image"
	"image/color"

	"hordefall/internal/i18n"

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
	graphicsLevelCount  = 3
	maximumRenderScale  = 3
	defaultShakeLevel   = 1
	defaultEffectsLevel = 2
	defaultBloomLevel   = 1
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
	{translatedLabel("menu.back"), (*Game).closeGraphics, nil},
}

var renderScale float32 = 1

// ===== Public API =====

func (u *UI) DrawGraphicsMenu(g *Game, screen *ebiten.Image, options []MenuOption) {
	dimScreen(screen, menuDim)
	u.drawText(screen, i18n.T("title.graphics"), u.title, screenWidth/2, menuTitleTop, accentColor, 1, text.AlignCenter)
	u.drawText(screen, g.settingsMessage, u.small, screenWidth/2, menuTitleTop+110, healthColor, 1, text.AlignCenter)
	u.drawMenuOptions(g, screen, options, menuOptionSpacing)
	u.drawBenchmarkPanel(g, screen)
	u.drawMusicBanner(g, screen)
	u.drawText(screen, i18n.T("hint.graphics"), u.small, screenWidth/2, menuHintTop, textColor, 1, text.AlignCenter)
	u.drawText(screen, i18n.T("hint.graphics_native"), u.small, screenWidth/2, menuHintTop+24, textColor, 1, text.AlignCenter)
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
	return i18n.F("graphics.bloom", translateValue(bloomLevels[g.settings.BloomLevel].Label))
}

func (g *Game) crtLabel() string {
	return i18n.F("graphics.crt", translateValue(onOffLabels[boolToIndex(g.settings.IsCRTOn)]))
}

func (g *Game) fullscreenLabel() string {
	return i18n.F("graphics.fullscreen", translateValue(onOffLabels[boolToIndex(g.settings.IsFullscreen)]))
}

func (g *Game) vsyncLabel() string {
	return i18n.F("graphics.vsync", translateValue(onOffLabels[boolToIndex(g.settings.IsVsyncOn)]))
}

func (g *Game) resolutionLabel() string {
	return i18n.F("graphics.resolution", translateValue(resolutionChoices[g.settings.Resolution].Label))
}

func (g *Game) fpsLabel() string {
	return i18n.F("graphics.show_fps", translateValue(onOffLabels[boolToIndex(g.settings.IsFPSShown)]))
}

func (g *Game) shakeLabel() string {
	return i18n.F("graphics.shake", translateValue(shakeLevels[g.settings.ShakeLevel].Label))
}

func (g *Game) effectsLabel() string {
	return i18n.F("graphics.effects", translateValue(effectsLevels[g.settings.EffectsLevel].Label))
}

func (g *Game) openGraphics() {
	g.settingsMessage = ""
	g.switchState(StateGraphics)
}

func (g *Game) closeGraphics() {
	g.switchState(StateOptions)
}

func (g *Game) updateGraphics() {
	g.updateBackdropDemo()
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

func (u *UI) scaledFace(face *text.GoTextFace) text.Face {
	if u.faceScale != renderScale {
		clear(u.scaledFaces)
		u.faceScale = renderScale
	}
	scaled, isCached := u.scaledFaces[face]
	if isCached {
		return scaled
	}
	size := face.Size * float64(renderScale)
	scaled = combineFaces(u.fontChains.Faces(face.Source, size))
	u.scaledFaces[face] = scaled
	return scaled
}

func combineFaces(faces []text.Face) text.Face {
	combined, err := text.NewMultiFace(faces...)
	if err != nil {
		return faces[0]
	}
	return combined
}

func fillTriangle(screen *ebiten.Image, points [3][2]float32, tint [3]float32, alpha float32) {
	vertices := make([]ebiten.Vertex, 0, 3)
	for _, point := range points {
		vertices = append(vertices, ebiten.Vertex{
			DstX: point[0] * renderScale, DstY: point[1] * renderScale, SrcX: 1.5, SrcY: 1.5,
			ColorR: tint[0] * alpha, ColorG: tint[1] * alpha, ColorB: tint[2] * alpha, ColorA: alpha,
		})
	}
	solidPixel = ensureSolidPixel(solidPixel)
	screen.DrawTriangles32(vertices, []uint32{0, 1, 2}, solidPixel, &ebiten.DrawTrianglesOptions{AntiAlias: true})
}

var solidPixel *ebiten.Image

func ensureSolidPixel(existing *ebiten.Image) *ebiten.Image {
	if existing != nil {
		return existing
	}
	canvas := ebiten.NewImage(3, 3)
	canvas.Fill(color.White)
	return canvas.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}

func fillRect(screen *ebiten.Image, x, y, width, height float32, tint color.Color) {
	vector.FillRect(screen, x*renderScale, y*renderScale, width*renderScale, height*renderScale, tint, false)
}

func fillRoundedRect(screen *ebiten.Image, x, y, width, height, radius float32, tint color.Color) {
	if width <= 0 || height <= 0 {
		return
	}
	left, top, right, bottom := x*renderScale, y*renderScale, (x+width)*renderScale, (y+height)*renderScale
	corner := min(radius, width/2, height/2) * renderScale
	var path vector.Path
	path.MoveTo(left+corner, top)
	path.ArcTo(right, top, right, bottom, corner)
	path.ArcTo(right, bottom, left, bottom, corner)
	path.ArcTo(left, bottom, left, top, corner)
	path.ArcTo(left, top, right, top, corner)
	path.Close()
	options := &vector.DrawPathOptions{AntiAlias: true}
	options.ColorScale.ScaleWithColor(tint)
	vector.FillPath(screen, &path, &vector.FillOptions{}, options)
}

func strokeRect(screen *ebiten.Image, x, y, width, height, strokeWidth float32, tint color.Color) {
	vector.StrokeRect(screen, x*renderScale, y*renderScale, width*renderScale, height*renderScale, strokeWidth*renderScale, tint, false)
}
