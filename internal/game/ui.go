package game

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/scores"
	"github.com/ajkula/ajkurig/ui"
)

// ===== Types =====

type UI struct {
	kit *ui.Kit
	art *ArtCanvas
}

// ===== Constants =====

const (
	spiderBarTop        = 50
	levelUpBannerTop    = 150
	levelUpBannerFade   = 0.6
	spiderBarWidth      = 520
	spiderBarHeight     = 18
	spiderBarGap        = 3
	bossBarCornerRadius = 4
)

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

func NewUI(kit *ui.Kit) *UI {
	return &UI{kit: kit, art: NewArtCanvas()}
}

func (u *UI) DrawHud(g *Game, screen *ebiten.Image) {
	player := g.player
	drawBar(screen, 0, 0, screenWidth, 8, float32(player.Experience)/float32(player.ExperienceToNext), experienceBlue)
	drawBar(screen, 16, 20, 260, 16, player.Health/player.MaximumHealth, healthColor)
	drawBar(screen, 16, 40, 260, 4, 1-player.DashCooldown/dashCooldownSeconds, [3]float32{0.5, 0.85, 1})
	u.kit.Text.Draw(screen, fmt.Sprintf("%.0f / %.0f", max(0, player.Health), player.MaximumHealth), ui.FaceSmall, 22, 20, textColor, 1, text.AlignStart)
	u.kit.Text.Draw(screen, i18n.F("hud.level", player.Level), ui.FaceBold, 290, 16, accentColor, 1, text.AlignStart)
	u.kit.Text.Draw(screen, scores.FormatClock(g.elapsedSeconds), ui.FaceBold, screenWidth/2, 16, textColor, 1, text.AlignCenter)
	status := i18n.F("hud.status", g.kills, g.enemies.Count) + u.fpsSuffix(g)
	u.kit.Text.Draw(screen, status, ui.FaceRegular, screenWidth-16, 18, textColor, 1, text.AlignEnd)
	highScoreText := "HI " + scores.FormatThousands(max(g.app.Scores.Best().Score, g.CurrentScore()))
	highScoreWidth := u.kit.Text.Width(highScoreText, ui.FaceBold)
	u.kit.Text.Draw(screen, highScoreText, ui.FaceBold, screenWidth-16, 44, highScoreColor, 1, text.AlignEnd)
	u.kit.Text.Draw(screen, "SCORE "+scores.FormatThousands(g.CurrentScore()), ui.FaceBold, screenWidth-16-highScoreWidth-24, 44, accentColor, 1, text.AlignEnd)
	u.drawWeaponList(g, screen)
	u.drawSpiderBars(g, screen)
	u.drawHordeEventBar(g, screen)
	u.drawWeatherLabel(g, screen)
	u.drawChatter(g, screen)
	u.drawLevelUpBanner(g, screen)
	u.drawPopups(g, screen)
}

// ===== Internal =====

func (u *UI) drawLevelUpBanner(g *Game, screen *ebiten.Image) {
	if g.levelUpBannerSeconds <= 0 {
		return
	}
	fade := clamp(g.levelUpBannerSeconds/levelUpBannerFade, 0, 1)
	u.kit.Text.Draw(screen, g.levelUpBanner, ui.FaceBold, screenWidth/2, levelUpBannerTop, accentColor, fade, text.AlignCenter)
}

func (u *UI) drawWeatherLabel(g *Game, screen *ebiten.Image) {
	intensity := g.weather.Intensity()
	if intensity <= 0 {
		return
	}
	definition := &weatherTable[g.weather.Kind]
	u.kit.Text.Draw(screen, i18n.T("weather."+definition.Key), ui.FaceBold, screenWidth/2, weatherLabelTop, definition.LabelColor, intensity, text.AlignCenter)
}

func (u *UI) fpsSuffix(g *Game) string {
	if !g.app.Settings.IsFPSShown {
		return ""
	}
	return g.app.Bench.FPSText()
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
		u.kit.Text.Draw(screen, fmt.Sprintf("%s  %d", definition.DisplayName(), weapon.Level), ui.FaceSmall, 36, y, textColor, 1, text.AlignStart)
	}
}

func (u *UI) drawSpiderBars(g *Game, screen *ebiten.Image) {
	for slot, rig := range g.spiders {
		u.drawSpiderBar(g, screen, &rig, float32(spiderBarTop+slot*(spiderBarHeight+spiderBarGap)))
	}
	u.drawWormBar(g, screen, float32(spiderBarTop+len(g.spiders)*(spiderBarHeight+spiderBarGap)))
}

func (u *UI) drawWormBar(g *Game, screen *ebiten.Image, y float32) {
	worm := &g.worm
	if !worm.IsActive {
		return
	}
	drawBossBar(screen, y, worm.HealthFraction(), wormArmorColor)
	u.kit.Text.Draw(screen, "DRILL-WORM", ui.FaceSmall, screenWidth/2, y, textColor, 1, text.AlignCenter)
}

func (u *UI) drawSpiderBar(g *Game, screen *ebiten.Image, rig *SpiderRig, y float32) {
	enemyIndex := g.enemies.IndexOfID(rig.EnemyID)
	if enemyIndex < 0 {
		return
	}
	drawBossBar(screen, y, g.enemies.Health[enemyIndex]/rig.MaximumHealth, warningColor)
	u.kit.Text.Draw(screen, "SPIDER TANK", ui.FaceSmall, screenWidth/2, y, textColor, 1, text.AlignCenter)
}

func (u *UI) drawPopups(g *Game, screen *ebiten.Image) {
	for _, popup := range g.effects.Popups {
		screenX, screenY := g.renderer.ToScreen(popup.X, popup.Y)
		u.kit.Text.Draw(screen, popup.Text, ui.FaceBold, screenX, screenY, popup.Color, popup.Life/popupLifeSeconds, text.AlignCenter)
	}
}

func drawBar(screen *ebiten.Image, x, y, width, height, fraction float32, tint [3]float32) {
	fillRect(screen, x, y, width, height, toColor(panelColor, 0.75))
	fillRect(screen, x, y, width*clamp(fraction, 0, 1), height, toColor(tint, 1))
}

func drawBossBar(screen *ebiten.Image, y, fraction float32, tint [3]float32) {
	x := float32(screenWidth/2 - spiderBarWidth/2)
	fillRoundedRect(screen, x, y, spiderBarWidth, spiderBarHeight, bossBarCornerRadius, toColor(panelColor, 0.75))
	fillRoundedRect(screen, x, y, spiderBarWidth*clamp(fraction, 0, 1), spiderBarHeight, bossBarCornerRadius, toColor(tint, 1))
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
