package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// ===== Types =====

type UI struct {
	small   *text.GoTextFace
	regular *text.GoTextFace
	bold    *text.GoTextFace
	title   *text.GoTextFace
}

// ===== Constants =====

const (
	cardWidth          = 300
	cardHeight         = 180
	cardSpacing        = 28
	descriptionWrapLen = 34
	spiderBarTop       = 50
	spiderBarWidth     = 520
	spiderBarHeight    = 18
	spiderBarGap       = 3
)

var (
	panelColor     = [3]float32{0.04, 0.05, 0.08}
	textColor      = [3]float32{0.92, 0.94, 0.98}
	mutedTextColor = [3]float32{0.6, 0.65, 0.75}
	healthColor    = [3]float32{0.9, 0.22, 0.25}
	experienceBlue = [3]float32{0.35, 0.65, 1}
	accentColor    = [3]float32{1, 0.75, 0.3}
)

var reactionHints = []string{
	"Fire + Oiled = INFERNO (chains, ignites grass)",
	"Shock + Wet = ELECTROCUTE (arcs through puddles)",
	"Frost + Wet or Chilled = FREEZE",
	"Blades or Shock + Frozen = SHATTER (x3 damage)",
	"Fire + Chilled, Frost + Burning = STEAM",
	"Burning enemies set grass ablaze. Fire hurts you too.",
}

// ===== Public API =====

func NewUI() *UI {
	regularSource := mustLoadFace(goregular.TTF)
	boldSource := mustLoadFace(gobold.TTF)
	return &UI{
		small:   &text.GoTextFace{Source: regularSource, Size: 14},
		regular: &text.GoTextFace{Source: regularSource, Size: 18},
		bold:    &text.GoTextFace{Source: boldSource, Size: 22},
		title:   &text.GoTextFace{Source: boldSource, Size: 72},
	}
}

func (u *UI) DrawHud(g *Game, screen *ebiten.Image) {
	player := g.player
	drawBar(screen, 0, 0, screenWidth, 8, float32(player.Experience)/float32(player.ExperienceToNext), experienceBlue)
	drawBar(screen, 16, 20, 260, 16, player.Health/player.MaximumHealth, healthColor)
	drawBar(screen, 16, 40, 260, 4, 1-player.DashCooldown/dashCooldownSeconds, [3]float32{0.5, 0.85, 1})
	u.drawText(screen, fmt.Sprintf("%.0f / %.0f", max(0, player.Health), player.MaximumHealth), u.small, 22, 20, textColor, 1, text.AlignStart)
	u.drawText(screen, fmt.Sprintf("Lv %d", player.Level), u.bold, 290, 16, accentColor, 1, text.AlignStart)
	u.drawText(screen, formatClock(g.elapsedSeconds), u.bold, screenWidth/2, 16, textColor, 1, text.AlignCenter)
	status := fmt.Sprintf("Kills %d   Horde %d   FPS %.0f", g.kills, g.enemies.Count, ebiten.ActualFPS())
	u.drawText(screen, status, u.regular, screenWidth-16, 18, mutedTextColor, 1, text.AlignEnd)
	u.drawWeaponList(g, screen)
	u.drawSpiderBars(g, screen)
	u.drawPopups(g, screen)
}

func (u *UI) DrawTitle(screen *ebiten.Image, clockSeconds float32, bindingsMessage string) {
	dimScreen(screen, 0.55)
	u.drawText(screen, "HORDEFALL", u.title, screenWidth/2, 120, accentColor, 1, text.AlignCenter)
	u.drawText(screen, "A systemic horde survivor. Burn, freeze, soak, shock. Combine.", u.regular, screenWidth/2, 215, textColor, 1, text.AlignCenter)
	u.drawText(screen, strings.Join(reactionHints, "\n"), u.regular, screenWidth/2, 280, mutedTextColor, 1, text.AlignCenter)
	controls := "Move: stick / WASD    Fire: button 1 / J    Aim lock: button 2 / K    Dash: button 3 / Space    Pause: Start / Esc"
	u.drawText(screen, controls, u.small, screenWidth/2, 510, textColor, 0.9, text.AlignCenter)
	u.drawText(screen, bindingsMessage, u.small, screenWidth/2, 536, mutedTextColor, 1, text.AlignCenter)
	u.drawText(screen, "Press Fire, Start or Enter to begin", u.bold, screenWidth/2, 590, accentColor, pulse(clockSeconds), text.AlignCenter)
}

func (u *UI) DrawLevelUp(g *Game, screen *ebiten.Image) {
	dimScreen(screen, 0.6)
	u.drawText(screen, "LEVEL UP", u.title, screenWidth/2, 110, accentColor, 1, text.AlignCenter)
	totalWidth := float32(len(g.offers))*cardWidth + float32(len(g.offers)-1)*cardSpacing
	startX := (screenWidth - totalWidth) / 2
	for index, offer := range g.offers {
		u.drawOfferCard(screen, offer, startX+float32(index)*(cardWidth+cardSpacing), 260, index == g.selectedOffer)
	}
	u.drawText(screen, "Left / Right to choose, Fire to confirm", u.regular, screenWidth/2, 500, mutedTextColor, 1, text.AlignCenter)
}

func (u *UI) DrawRemap(screen *ebiten.Image, step int, buttons [3]int) {
	dimScreen(screen, 0.75)
	u.drawText(screen, "CONFIGURE BUTTONS", u.title, screenWidth/2, 110, accentColor, 1, text.AlignCenter)
	for index, name := range remapStepNames {
		labels := [3]string{fmt.Sprintf("%s:  button %d", name, buttons[index]), fmt.Sprintf("Press the button for %s", name), name}
		state := boolToIndex(index == step) + 2*boolToIndex(index > step)
		colors := [3][3]float32{textColor, accentColor, mutedTextColor}
		u.drawText(screen, labels[state], u.bold, screenWidth/2, 270+float32(index)*56, colors[state], 1, text.AlignCenter)
	}
	u.drawText(screen, "Start / Esc to cancel", u.regular, screenWidth/2, 520, mutedTextColor, 1, text.AlignCenter)
}

func (u *UI) DrawPaused(screen *ebiten.Image) {
	dimScreen(screen, 0.55)
	u.drawText(screen, "PAUSED", u.title, screenWidth/2, 260, textColor, 1, text.AlignCenter)
	u.drawText(screen, "Start / Esc to resume", u.regular, screenWidth/2, 360, mutedTextColor, 1, text.AlignCenter)
}

func (u *UI) DrawGameOver(g *Game, screen *ebiten.Image) {
	dimScreen(screen, 0.7)
	u.drawText(screen, "YOU FELL", u.title, screenWidth/2, 100, healthColor, 1, text.AlignCenter)
	summary := fmt.Sprintf("Survived %s    Level %d    Kills %d", formatClock(g.elapsedSeconds), g.player.Level, g.kills)
	u.drawText(screen, summary, u.bold, screenWidth/2, 210, textColor, 1, text.AlignCenter)
	u.drawText(screen, describeReactionCounts(g.reactionCounts), u.regular, screenWidth/2, 270, mutedTextColor, 1, text.AlignCenter)
	u.drawText(screen, "Press any button or Enter to try again", u.bold, screenWidth/2, 560, accentColor, 1, text.AlignCenter)
}

func (u *UI) DrawDebugOverlay(lines []string, screen *ebiten.Image) {
	height := float32(len(lines))*20 + 16
	vector.FillRect(screen, 12, screenHeight-height-12, 560, height, toColor(panelColor, 0.85), false)
	u.drawText(screen, strings.Join(lines, "\n"), u.small, 22, screenHeight-height-4, textColor, 1, text.AlignStart)
}

// ===== Internal =====

func mustLoadFace(fontData []byte) *text.GoTextFaceSource {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(fontData))
	if err != nil {
		panic(err)
	}
	return source
}

func (u *UI) drawText(screen *ebiten.Image, message string, face *text.GoTextFace, x, y float32, tint [3]float32, alpha float32, align text.Align) {
	options := &text.DrawOptions{}
	options.GeoM.Translate(float64(x), float64(y))
	options.ColorScale.ScaleWithColor(toColor(tint, alpha))
	options.PrimaryAlign = align
	options.LineSpacing = face.Size * 1.45
	text.Draw(screen, message, face, options)
}

func (u *UI) drawWeaponList(g *Game, screen *ebiten.Image) {
	for slot, weapon := range g.player.Weapons {
		definition := &weaponTable[weapon.Kind]
		y := float32(screenHeight - 30 - (len(g.player.Weapons)-1-slot)*24)
		vector.FillRect(screen, 16, y+4, 12, 12, toColor(definition.Color, 1), false)
		u.drawText(screen, fmt.Sprintf("%s  %d", definition.Name, weapon.Level), u.small, 36, y, textColor, 0.9, text.AlignStart)
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
	vector.FillRect(screen, x, y, cardWidth, cardHeight, toColor(panelColor, 0.92), false)
	vector.StrokeRect(screen, x, y, cardWidth, cardHeight, borderWidth, toColor(offer.Color(), 0.4+0.6*boolToFloat(isSelected)), false)
	u.drawText(screen, offer.Title(), u.bold, x+cardWidth/2, y+18, offer.Color(), 1, text.AlignCenter)
	u.drawText(screen, offer.Subtitle(), u.small, x+cardWidth/2, y+52, accentColor, 1, text.AlignCenter)
	u.drawText(screen, wrapText(offer.Description(), descriptionWrapLen), u.regular, x+cardWidth/2, y+86, textColor, 0.9, text.AlignCenter)
}

func drawBar(screen *ebiten.Image, x, y, width, height, fraction float32, tint [3]float32) {
	vector.FillRect(screen, x, y, width, height, toColor(panelColor, 0.75), false)
	vector.FillRect(screen, x, y, width*clamp(fraction, 0, 1), height, toColor(tint, 1), false)
}

func dimScreen(screen *ebiten.Image, alpha float32) {
	vector.FillRect(screen, 0, 0, screenWidth, screenHeight, toColor(panelColor, alpha), false)
}

func formatClock(seconds float32) string {
	return fmt.Sprintf("%02d:%02d", int(seconds)/60, int(seconds)%60)
}

func wrapText(message string, maximumLineLength int) string {
	lines := make([]string, 0, 4)
	current := ""
	for _, word := range strings.Fields(message) {
		candidate := strings.TrimSpace(current + " " + word)
		isOverflowing := len(candidate) > maximumLineLength && current != ""
		lines = appendIf(lines, current, isOverflowing)
		current = [2]string{candidate, word}[boolToIndex(isOverflowing)]
	}
	return strings.Join(append(lines, current), "\n")
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
