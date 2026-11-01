package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/ajkula/ajkurig/colorx"
	"github.com/ajkula/ajkurig/draw"
	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/scores"
	"github.com/ajkula/ajkurig/shell"
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
	panelColor       = [3]float32{0.04, 0.05, 0.08}
	textColor        = [3]float32{1, 1, 1}
	healthColor      = [3]float32{0.9, 0.22, 0.25}
	systemClockColor = [3]float32{0.992, 0.961, 0}
	systemClockRim   = [3]float32{0.443, 0, 0}
	experienceBlue   = [3]float32{0.35, 0.65, 1}
	accentColor      = [3]float32{1, 0.75, 0.3}
	outlineColor     = [3]float32{0, 0, 0}
	highScoreColor   = [3]float32{0.8, 0.42, 0.08}
)

const (
	systemClockSetting = "systemClock"
	systemClockOn      = "on"
	systemClockOff     = "off"
	systemClockLayout  = "15:04"
	systemClockCenter  = 146
	systemClockTop     = 46
)

// ===== Public API =====

func NewUI(kit *ui.Kit) *UI {
	return &UI{kit: kit, art: NewArtCanvas()}
}

func (u *UI) drawStatus(g *Game, screen *ebiten.Image) {
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
	u.drawSystemClock(g, screen)
	u.drawWeaponList(g, screen)
	u.drawSpiderBars(g, screen)
	u.drawHordeEventBar(g, screen)
	u.drawWeatherLabel(g, screen)
}

// ===== Internal =====

func (u *UI) drawLevelUpBanner(g *Game, screen *ebiten.Image) {
	if g.levelUpBannerSeconds <= 0 {
		return
	}
	fade := clamp(g.levelUpBannerSeconds/levelUpBannerFade, 0, 1)
	u.kit.Text.Draw(screen, g.levelUpBanner, ui.FaceBold, screenWidth/2, levelUpBannerTop, accentColor, fade, text.AlignCenter)
}

func (u *UI) drawSystemClock(g *Game, screen *ebiten.Image) {
	if !g.isSystemClockShown() {
		return
	}
	u.kit.Text.DrawOutlined(screen, time.Now().Format(systemClockLayout), ui.FaceBold, systemClockCenter, systemClockTop, systemClockColor, systemClockRim, text.AlignCenter)
}

func (g *Game) isSystemClockShown() bool {
	return g.app.Settings.Custom[systemClockSetting] == systemClockOn
}

func (g *Game) toggleSystemClock() {
	g.app.Settings.Custom[systemClockSetting] = [2]string{systemClockOn, systemClockOff}[boolToIndex(g.isSystemClockShown())]
	g.app.SaveSettings()
}

func (g *Game) gameplayOptions() []ui.Option {
	return []ui.Option{{Label: shell.OnOffLabel("gameplay.system_clock", g.isSystemClockShown), Activate: g.toggleSystemClock}}
}

func (u *UI) drawWeatherLabel(g *Game, screen *ebiten.Image) {
	intensity := g.weather.Intensity()
	if intensity <= 0 {
		return
	}
	definition := g.weather.Definition()
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
		draw.FillRect(screen, 16, y+4, 12, 12, colorx.NRGBA(definition.Color, 1))
		u.kit.Text.Draw(screen, fmt.Sprintf("%s  %d", definition.DisplayName(), weapon.Level), ui.FaceSmall, 36, y, textColor, 1, text.AlignStart)
	}
}

func (u *UI) drawSpiderBars(g *Game, screen *ebiten.Image) {
	for slot, rig := range g.spiders {
		u.drawSpiderBar(g, screen, &rig, float32(spiderBarTop+slot*(spiderBarHeight+spiderBarGap)))
	}
	for slot := range g.worms {
		u.drawWormBar(screen, &g.worms[slot], float32(spiderBarTop+(len(g.spiders)+slot)*(spiderBarHeight+spiderBarGap)))
	}
}

func (u *UI) drawWormBar(screen *ebiten.Image, worm *DrillWorm, y float32) {
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

func drawBar(screen *ebiten.Image, x, y, width, height, fraction float32, tint [3]float32) {
	draw.FillRect(screen, x, y, width, height, colorx.NRGBA(panelColor, 0.75))
	draw.FillRect(screen, x, y, width*clamp(fraction, 0, 1), height, colorx.NRGBA(tint, 1))
}

func drawBossBar(screen *ebiten.Image, y, fraction float32, tint [3]float32) {
	x := float32(screenWidth/2 - spiderBarWidth/2)
	draw.FillRoundedRect(screen, x, y, spiderBarWidth, spiderBarHeight, bossBarCornerRadius, colorx.NRGBA(panelColor, 0.75))
	draw.FillRoundedRect(screen, x, y, spiderBarWidth*clamp(fraction, 0, 1), spiderBarHeight, bossBarCornerRadius, colorx.NRGBA(tint, 1))
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
