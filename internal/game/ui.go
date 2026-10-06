package game

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"hordefall/internal/i18n"
)

// ===== Types =====

type WrapToken struct {
	Text     string
	IsSpaced bool
}

type UI struct {
	small   *text.GoTextFace
	regular *text.GoTextFace
	bold    *text.GoTextFace
	title   *text.GoTextFace
	pixels  PixelBatch
	art     *ArtCanvas

	scaledFaces map[*text.GoTextFace]text.Face
	fontChains  *FontChains
	faceScale   float32

	titleHeight   float32
	menuPositions []float32

	benchmarkLeft    string
	benchmarkRight   string
	fpsText          string
	nextStatsRefresh float32
}

// ===== Constants =====

//go:embed fonts/Rajdhani-Medium.ttf
var rajdhaniMedium []byte

//go:embed fonts/Rajdhani-Bold.ttf
var rajdhaniBold []byte

const (
	cardWidth                = 300
	cardHeight               = 180
	cardSpacing              = 28
	descriptionWrapWidth     = cardWidth - 40
	spiderBarTop             = 50
	mainMenuOptionsTop       = 505
	menuTitleTop             = 70
	menuHintTop              = 660
	menuDim                  = 0.42
	levelUpBannerTop         = 150
	levelUpBannerFade        = 0.6
	spiderBarWidth           = 520
	spiderBarHeight          = 18
	spiderBarGap             = 3
	textOutlineWidth         = 1
	selectionMarkerGap       = 14
	selectionMarkerThickness = 3
	selectionMarkerShade     = 0.5
	selectionMarkerNudge     = 3
	selectionMarkerSpeed     = 5
	selectionMarkerCenter    = 0.5
	statsRefreshSeconds      = 0.5
	benchmarkPanelHeight     = 6*24 + 16
	benchmarkRightColumn     = 196
)

var textOutlineOffsets = [8][2]float32{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}

var (
	panelColor     = [3]float32{0.04, 0.05, 0.08}
	textColor      = [3]float32{1, 1, 1}
	healthColor    = [3]float32{0.9, 0.22, 0.25}
	experienceBlue = [3]float32{0.35, 0.65, 1}
	accentColor    = [3]float32{1, 0.75, 0.3}
	outlineColor   = [3]float32{0, 0, 0}
	highScoreColor = [3]float32{0.8, 0.42, 0.08}
)

// ===== Public API =====

func NewUI() *UI {
	regularSource := mustLoadFace(rajdhaniMedium)
	boldSource := mustLoadFace(rajdhaniBold)
	ui := &UI{
		small:   &text.GoTextFace{Source: regularSource, Size: 16},
		regular: &text.GoTextFace{Source: regularSource, Size: 20},
		bold:    &text.GoTextFace{Source: boldSource, Size: 24},
		title:   &text.GoTextFace{Source: boldSource, Size: 80},

		scaledFaces: map[*text.GoTextFace]text.Face{},
		fontChains:  NewFontChains(FontFamily{regularSource, boldSource}),
		faceScale:   1,
		art:         NewArtCanvas(),
	}
	_, titleHeight := text.Measure("HORDEFALL", ui.title, 0)
	ui.titleHeight = float32(titleHeight)
	return ui
}

func (u *UI) DrawHud(g *Game, screen *ebiten.Image) {
	player := g.player
	drawBar(screen, 0, 0, screenWidth, 8, float32(player.Experience)/float32(player.ExperienceToNext), experienceBlue)
	drawBar(screen, 16, 20, 260, 16, player.Health/player.MaximumHealth, healthColor)
	drawBar(screen, 16, 40, 260, 4, 1-player.DashCooldown/dashCooldownSeconds, [3]float32{0.5, 0.85, 1})
	u.drawText(screen, fmt.Sprintf("%.0f / %.0f", max(0, player.Health), player.MaximumHealth), u.small, 22, 20, textColor, 1, text.AlignStart)
	u.drawText(screen, i18n.F("hud.level", player.Level), u.bold, 290, 16, accentColor, 1, text.AlignStart)
	u.drawText(screen, formatClock(g.elapsedSeconds), u.bold, screenWidth/2, 16, textColor, 1, text.AlignCenter)
	u.refreshStatsIfDue(g)
	fpsTexts := [2]string{"", u.fpsText}
	status := i18n.F("hud.status", g.kills, g.enemies.Count) + fpsTexts[boolToIndex(g.settings.IsFPSShown)]
	u.drawText(screen, status, u.regular, screenWidth-16, 18, textColor, 1, text.AlignEnd)
	highScoreText := "HI " + formatThousands(max(g.highScores.Best().Score, g.CurrentScore()))
	highScoreWidth, _ := text.Measure(highScoreText, u.bold, 0)
	u.drawText(screen, highScoreText, u.bold, screenWidth-16, 44, highScoreColor, 1, text.AlignEnd)
	u.drawText(screen, "SCORE "+formatThousands(g.CurrentScore()), u.bold, screenWidth-16-float32(highScoreWidth)-24, 44, accentColor, 1, text.AlignEnd)
	u.drawWeaponList(g, screen)
	u.drawSpiderBars(g, screen)
	u.drawHordeEventBar(g, screen)
	u.drawWeatherLabel(g, screen)
	u.drawLevelUpBanner(g, screen)
	u.drawMusicBanner(g, screen)
	u.drawPopups(g, screen)
}

func (u *UI) DrawMainMenu(g *Game, screen *ebiten.Image, options []MenuOption) {
	dimScreen(screen, menuDim)
	u.drawText(screen, "HORDEFALL", u.title, screenWidth/2, menuTitleTop, accentColor, 1, text.AlignCenter)
	u.drawText(screen, describeHighScore(g.highScores.Best()), u.bold, screenWidth/2, menuTitleTop+110, highScoreColor, 1, text.AlignCenter)
	u.drawMenuOptions(g, screen, options, mainMenuOptionSpacing)
	u.drawBenchmarkPanel(g, screen)
	u.drawDemoSequence(g, screen)
	u.drawMusicBanner(g, screen)
	controls := i18n.T("main.controls")
	u.drawText(screen, controls, u.small, screenWidth/2, menuHintTop, textColor, 1, text.AlignCenter)
	u.drawText(screen, i18n.T(g.bindingsMessage), u.small, screenWidth/2, menuHintTop+24, textColor, 1, text.AlignCenter)
}

func (u *UI) DrawPauseMenu(g *Game, screen *ebiten.Image, options []MenuOption) {
	dimScreen(screen, menuDim)
	u.drawText(screen, i18n.T("title.paused"), u.title, screenWidth/2, menuTitleTop, textColor, 1, text.AlignCenter)
	u.drawMenuOptions(g, screen, options, menuOptionSpacing)
	u.drawText(screen, i18n.T("hint.menu_resume"), u.small, screenWidth/2, menuHintTop, textColor, 1, text.AlignCenter)
}

func (u *UI) DrawOptionsMenu(g *Game, screen *ebiten.Image, options []MenuOption) {
	dimScreen(screen, menuDim)
	u.drawText(screen, i18n.T("title.options"), u.title, screenWidth/2, menuTitleTop, accentColor, 1, text.AlignCenter)
	u.drawText(screen, g.settingsMessage, u.small, screenWidth/2, menuTitleTop+110, healthColor, 1, text.AlignCenter)
	u.drawMenuOptions(g, screen, options, menuOptionSpacing)
	u.drawBenchmarkPanel(g, screen)
	u.drawDemoSequence(g, screen)
	u.drawMusicBanner(g, screen)
	u.drawText(screen, i18n.T("hint.menu_back"), u.small, screenWidth/2, menuHintTop, textColor, 1, text.AlignCenter)
	u.drawText(screen, i18n.T("hint.settings_saved"), u.small, screenWidth/2, menuHintTop+24, textColor, 1, text.AlignCenter)
}

func (u *UI) DrawLevelUp(g *Game, screen *ebiten.Image) {
	dimScreen(screen, 0.6)
	u.drawText(screen, "LEVEL UP", u.title, screenWidth/2, 110, accentColor, 1, text.AlignCenter)
	totalWidth := float32(len(g.offers))*cardWidth + float32(len(g.offers)-1)*cardSpacing
	startX := (screenWidth - totalWidth) / 2
	for index, offer := range g.offers {
		u.drawOfferCard(screen, offer, startX+float32(index)*(cardWidth+cardSpacing), 260, index == g.selectedOffer)
	}
	u.drawText(screen, i18n.T("hint.level_up"), u.regular, screenWidth/2, 500, textColor, 1, text.AlignCenter)
}

func (u *UI) DrawGameOver(g *Game, screen *ebiten.Image) {
	dimScreen(screen, 0.7)
	u.drawText(screen, i18n.T("title.you_fell"), u.title, screenWidth/2, 70, healthColor, 1, text.AlignCenter)
	u.drawText(screen, "SCORE "+formatThousands(g.CurrentScore()), u.title, screenWidth/2, 160, accentColor, 1, text.AlignCenter)
	isRanked := g.newEntryRank != noRank
	rankState := boolToIndex(isRanked) + boolToIndex(g.newEntryRank == 0)
	highScoreLines := [3]string{describeHighScore(g.highScores.Best()), i18n.F("gameover.ranked", rankOrdinals[max(0, g.newEntryRank)]), "NEW HIGH SCORE!"}
	highScoreColors := [3][3]float32{highScoreColor, accentColor, accentColor}
	highScoreAlpha := pulse(g.clockSeconds)*boolToFloat(isRanked) + boolToFloat(!isRanked)
	u.drawText(screen, highScoreLines[rankState], u.bold, screenWidth/2, 250, highScoreColors[rankState], highScoreAlpha, text.AlignCenter)
	summary := i18n.F("gameover.summary", formatClock(g.elapsedSeconds), g.player.Level, g.kills)
	u.drawText(screen, summary, u.bold, screenWidth/2, 295, textColor, 1, text.AlignCenter)
	reactionTexts := [2]string{describeReactionCounts(g.reactionCounts), ""}
	u.drawText(screen, reactionTexts[boolToIndex(isRanked)], u.regular, screenWidth/2, 345, textColor, 1, text.AlignCenter)
	u.drawRankedLine(g, screen, gameOverRankedTop)
	u.drawText(screen, g.scoreMessage, u.small, screenWidth/2, 560, healthColor, 1, text.AlignCenter)
	choices := i18n.F("gameover.choices", g.input.ButtonLabel(ActionStart), g.input.ButtonLabel(ActionSelect))
	u.drawText(screen, choices, u.bold, screenWidth/2, 600, accentColor, 1, text.AlignCenter)
}

func (u *UI) DrawDebugOverlay(lines []string, screen *ebiten.Image) {
	height := float32(len(lines))*20 + 16
	fillRect(screen, 12, screenHeight-height-12, 560, height, toColor(panelColor, 0.85))
	u.drawText(screen, strings.Join(lines, "\n"), u.small, 22, screenHeight-height-4, textColor, 1, text.AlignStart)
}

// ===== Internal =====

func loadFaceSourceOrNil(fontData []byte) *text.GoTextFaceSource {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(fontData))
	if err != nil {
		return nil
	}
	return source
}

func (u *UI) ResetFaces() {
	if u == nil {
		return
	}
	clear(u.scaledFaces)
}

func mustLoadFace(fontData []byte) *text.GoTextFaceSource {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(fontData))
	if err != nil {
		panic(err)
	}
	return source
}

func (u *UI) drawText(screen *ebiten.Image, message string, face *text.GoTextFace, x, y float32, tint [3]float32, alpha float32, align text.Align) {
	scaledFace := u.scaledFace(face)
	options := &text.DrawOptions{}
	options.PrimaryAlign = align
	options.LineSpacing = face.Size * float64(renderScale) * 1.45
	options.ColorScale.ScaleWithColor(toColor(outlineColor, alpha))
	for _, offset := range textOutlineOffsets {
		options.GeoM.Reset()
		options.GeoM.Translate(float64((x+offset[0]*textOutlineWidth)*renderScale), float64((y+offset[1]*textOutlineWidth)*renderScale))
		text.Draw(screen, message, scaledFace, options)
	}
	options.GeoM.Reset()
	options.GeoM.Translate(float64(x*renderScale), float64(y*renderScale))
	options.ColorScale.Reset()
	options.ColorScale.ScaleWithColor(toColor(tint, alpha))
	text.Draw(screen, message, scaledFace, options)
}

func (u *UI) drawMenuOptions(g *Game, screen *ebiten.Image, options []MenuOption, spacing float32) {
	u.menuPositions = menuOptionPositions(options, spacing, menuTitleTop+2*u.titleHeight, u.menuPositions)
	for index, option := range options {
		isSelected := index == g.menuSelection
		label := option.Label(g)
		colors := [2][3]float32{textColor, accentColor}
		alpha := 1 - 0.3*boolToFloat(isSelected)*(1-pulse(g.clockSeconds))
		y := u.menuPositions[index]
		u.drawText(screen, label, u.bold, screenWidth/2, y, colors[boolToIndex(isSelected)], alpha, text.AlignCenter)
		u.drawSelectionMarkersIf(g, screen, label, y, alpha, isSelected)
		u.drawSliderIf(g, screen, option.Slider, y+menuSliderOffset)
	}
}

func (u *UI) drawSelectionMarkersIf(g *Game, screen *ebiten.Image, label string, y, alpha float32, isSelected bool) {
	if !isSelected {
		return
	}
	labelWidth, labelHeight := text.Measure(label, u.bold, 0)
	centerY := y + float32(labelHeight)*selectionMarkerCenter
	nudge := selectionMarkerNudge * sine(g.clockSeconds*selectionMarkerSpeed)
	offset := float32(labelWidth)/2 + selectionMarkerGap + nudge
	side := float32(labelHeight)
	top := centerY - side/2
	edgeColor := mixColor(accentColor, [3]float32{0, 0, 0}, selectionMarkerShade)
	for _, direction := range [2]float32{-1, 1} {
		tipX := screenWidth/2 + direction*offset
		baseX := tipX + direction*side
		for depth := selectionMarkerThickness; depth > 0; depth-- {
			fillTriangle(screen, markerTriangle(baseX, tipX, top, side, float32(depth)), edgeColor, alpha)
		}
		fillTriangle(screen, markerTriangle(baseX, tipX, top, side, 0), accentColor, alpha)
	}
}

func markerTriangle(baseX, tipX, top, side, drop float32) [3][2]float32 {
	return [3][2]float32{{baseX, top + drop}, {baseX, top + side + drop}, {tipX, top + side/2 + drop}}
}

func (u *UI) drawBenchmarkPanel(g *Game, screen *ebiten.Image) {
	if !g.isDemo || g.settings.IsBenchmarkHidden {
		return
	}
	u.refreshStatsIfDue(g)
	fillRect(screen, 16, 470, 330, benchmarkPanelHeight, toColor(panelColor, 0.75))
	u.drawText(screen, u.benchmarkLeft, u.small, 28, 478, textColor, 1, text.AlignStart)
	u.drawText(screen, u.benchmarkRight, u.small, benchmarkRightColumn, 478, textColor, 1, text.AlignStart)
}

func (u *UI) refreshStatsIfDue(g *Game) {
	if g.clockSeconds < u.nextStatsRefresh {
		return
	}
	u.nextStatsRefresh = g.clockSeconds + statsRefreshSeconds
	u.fpsText = fmt.Sprintf("   FPS %.0f", ebiten.ActualFPS())
	left := []string{
		i18n.T("bench.title"),
		fmt.Sprintf("FPS  %.0f", ebiten.ActualFPS()),
		i18n.F("bench.simulation", g.simulationMillis),
		i18n.F("bench.enemies", g.enemies.Count),
		i18n.F("bench.particles", g.effects.ParticleCount),
		i18n.F("bench.ground", g.ground.Columns*g.ground.Rows),
	}
	right := []string{
		"",
		fmt.Sprintf("TPS  %.0f", ebiten.ActualTPS()),
		"",
		i18n.F("bench.spiders", len(g.spiders)),
		i18n.F("bench.projectiles", g.projectiles.Count),
		i18n.F("bench.reactions", sumReactions(g.reactionCounts)),
	}
	u.benchmarkLeft, u.benchmarkRight = strings.Join(left, "\n"), strings.Join(right, "\n")
}

func (u *UI) drawDemoSequence(g *Game, screen *ebiten.Image) {
	if !g.isDemo {
		return
	}
	right := float32(screenWidth - 16)
	u.drawText(screen, g.DemoSequenceName(), u.bold, right, 480, accentColor, 1, text.AlignEnd)
	fillRect(screen, right-260, 514, 260, 4, toColor(panelColor, 0.75))
	fillRect(screen, right-260, 514, 260*g.DemoSequenceProgress(), 4, toColor(accentColor, 1))
	u.drawText(screen, g.DemoSequenceSubtitle(), u.small, right, 524, textColor, 1, text.AlignEnd)
}

func (u *UI) drawLevelUpBanner(g *Game, screen *ebiten.Image) {
	if g.levelUpBannerSeconds <= 0 {
		return
	}
	fade := clamp(g.levelUpBannerSeconds/levelUpBannerFade, 0, 1)
	u.drawText(screen, g.levelUpBanner, u.bold, screenWidth/2, levelUpBannerTop, accentColor, fade, text.AlignCenter)
}

func (u *UI) drawWeatherLabel(g *Game, screen *ebiten.Image) {
	intensity := g.weather.Intensity()
	if intensity <= 0 {
		return
	}
	definition := &weatherTable[g.weather.Kind]
	u.drawText(screen, i18n.T("weather."+definition.Key), u.bold, screenWidth/2, weatherLabelTop, definition.LabelColor, intensity, text.AlignCenter)
}

func (u *UI) drawMusicBanner(g *Game, screen *ebiten.Image) {
	if g.musicBannerSeconds <= 0 {
		return
	}
	fade := clamp(g.musicBannerSeconds/levelUpBannerFade, 0, 1)
	u.drawText(screen, g.musicBanner, u.regular, screenWidth-16, screenHeight-34, textColor, fade, text.AlignEnd)
}

func describeHighScore(highScore HighScoreEntry) string {
	if highScore.Score == 0 {
		return i18n.T("main.no_high_score")
	}
	return i18n.F("main.high_score", formatThousands(highScore.Score), highScore.Initials, formatClock(highScore.SurvivedSeconds), highScore.Kills, highScore.Level)
}

func sumReactions(counts [reactionKindCount]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}

func (u *UI) drawWeaponList(g *Game, screen *ebiten.Image) {
	for slot, weapon := range g.player.Weapons {
		definition := &weaponTable[weapon.Kind]
		y := float32(screenHeight - 30 - (len(g.player.Weapons)-1-slot)*24)
		fillRect(screen, 16, y+4, 12, 12, toColor(definition.Color, 1))
		u.drawText(screen, fmt.Sprintf("%s  %d", definition.DisplayName(), weapon.Level), u.small, 36, y, textColor, 1, text.AlignStart)
	}
}

func (u *UI) drawSpiderBars(g *Game, screen *ebiten.Image) {
	for slot, rig := range g.spiders {
		u.drawSpiderBar(g, screen, &rig, float32(spiderBarTop+slot*(spiderBarHeight+spiderBarGap)))
	}
}

func (u *UI) drawSpiderBar(g *Game, screen *ebiten.Image, rig *SpiderRig, y float32) {
	enemyIndex := g.enemies.IndexOfID(rig.EnemyID)
	if enemyIndex < 0 {
		return
	}
	drawBar(screen, screenWidth/2-spiderBarWidth/2, y, spiderBarWidth, spiderBarHeight, g.enemies.Health[enemyIndex]/rig.MaximumHealth, warningColor)
	u.drawText(screen, "SPIDER TANK", u.small, screenWidth/2, y, textColor, 1, text.AlignCenter)
}

func (u *UI) drawPopups(g *Game, screen *ebiten.Image) {
	for _, popup := range g.effects.Popups {
		screenX, screenY := g.renderer.ToScreen(popup.X, popup.Y)
		u.drawText(screen, popup.Text, u.bold, screenX, screenY, popup.Color, popup.Life/popupLifeSeconds, text.AlignCenter)
	}
}

func (u *UI) drawOfferCard(screen *ebiten.Image, offer UpgradeOffer, x, y float32, isSelected bool) {
	borderWidth := 1 + 2*boolToFloat(isSelected)
	fillRect(screen, x, y, cardWidth, cardHeight, toColor(panelColor, 0.92))
	strokeRect(screen, x, y, cardWidth, cardHeight, borderWidth, toColor(offer.Color(), 0.4+0.6*boolToFloat(isSelected)))
	u.drawText(screen, offer.Title(), u.bold, x+cardWidth/2, y+18, offer.Color(), 1, text.AlignCenter)
	u.drawText(screen, offer.Subtitle(), u.small, x+cardWidth/2, y+52, accentColor, 1, text.AlignCenter)
	u.drawText(screen, u.wrapText(offer.Description(), u.regular, descriptionWrapWidth), u.regular, x+cardWidth/2, y+86, textColor, 1, text.AlignCenter)
}

func drawBar(screen *ebiten.Image, x, y, width, height, fraction float32, tint [3]float32) {
	fillRect(screen, x, y, width, height, toColor(panelColor, 0.75))
	fillRect(screen, x, y, width*clamp(fraction, 0, 1), height, toColor(tint, 1))
}

func dimScreen(screen *ebiten.Image, alpha float32) {
	fillRect(screen, 0, 0, screenWidth, screenHeight, toColor(panelColor, alpha))
}

func formatClock(seconds float32) string {
	return fmt.Sprintf("%02d:%02d", int(seconds)/60, int(seconds)%60)
}

func (u *UI) wrapText(message string, face *text.GoTextFace, maximumWidth float32) string {
	lines := make([]string, 0, 4)
	current := ""
	for _, token := range wrapTokens(message) {
		separator := [2]string{"", " "}[boolToIndex(token.IsSpaced && current != "")]
		isOverflowing := current != "" && u.textWidth(current+separator+token.Text, face) > maximumWidth
		lines = appendIf(lines, current, isOverflowing)
		separator = [2]string{separator, ""}[boolToIndex(isOverflowing)]
		current = [2]string{current, ""}[boolToIndex(isOverflowing)] + separator + token.Text
	}
	return strings.Join(append(lines, current), "\n")
}

func (u *UI) textWidth(message string, face *text.GoTextFace) float32 {
	return float32(text.Advance(message, u.scaledFace(face))) / renderScale
}

func wrapTokens(message string) []WrapToken {
	tokens := make([]WrapToken, 0, len(message)/4)
	for _, word := range strings.Fields(message) {
		tokens = appendWordTokens(tokens, word)
	}
	return tokens
}

func appendWordTokens(tokens []WrapToken, word string) []WrapToken {
	if !strings.ContainsFunc(word, isWideRune) {
		return append(tokens, WrapToken{Text: word, IsSpaced: true})
	}
	for index, character := range []rune(word) {
		tokens = append(tokens, WrapToken{Text: string(character), IsSpaced: index == 0})
	}
	return tokens
}

func isWideRune(character rune) bool {
	return character >= 0x1100 && character <= 0x115F || character >= 0x2E80 && character <= 0xA4CF || character >= 0xAC00 && character <= 0xD7A3 || character >= 0xF900 && character <= 0xFAFF || character >= 0xFF00 && character <= 0xFF60
}

func describeReactionCounts(counts [reactionKindCount]int) string {
	lines := make([]string, 0, reactionKindCount)
	for reaction := ReactionInferno; reaction < reactionKindCount; reaction++ {
		lines = append(lines, fmt.Sprintf("%-12s %d", reactionTable[reaction].Name, counts[reaction]))
	}
	return strings.Join(lines, "\n")
}

func pulse(seconds float32) float32 {
	return 0.6 + 0.4*sine(seconds*4)
}
