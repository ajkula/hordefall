package game

import (
	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/input"
	"github.com/ajkula/ajkurig/rig"
	"github.com/ajkula/ajkurig/scores"
	"github.com/ajkula/ajkurig/shell"
	"github.com/ajkula/ajkurig/ui"
)

// ===== Public API =====

func (g *Game) StartRun(app *rig.App) {
	g.isDemo = false
}

func (g *Game) StartDemo(app *rig.App, clip int) shell.DemoClip {
	candidates := [2]DemoSequence{demoSequences[(clip/2)%len(demoSequences)], g.randomDemo}
	g.currentDemo = candidates[clip%2]
	g.stageDemo(&g.currentDemo)
	return shell.DemoClip{Name: g.currentDemo.Name, Subtitle: g.currentDemo.Subtitle}
}

func (g *Game) Autopilot(frame *rig.Frame) input.Controls {
	return g.autopilotControls(g.shell.DemoSeconds)
}

func (g *Game) Status() shell.RunStatus {
	status := shell.RunStatus{
		IsOver: g.player.Health <= 0, Score: g.CurrentScore(), Seconds: g.elapsedSeconds,
		PendingLevelUps: g.player.PendingLevelUps, Stats: g.runStats(),
	}
	if !status.IsOver {
		return status
	}
	status.Summary = i18n.F("gameover.summary", scores.FormatClock(g.elapsedSeconds), g.player.Level, g.kills)
	status.Details = describeReactionCounts(g.reactionCounts)
	status.KilledBy = g.damageLog.Causes(g.elapsedSeconds)
	return status
}

func (g *Game) OfferCards() []ui.Card {
	g.buildUpgradeOffers()
	g.playSound(SoundLevelUp)
	g.say(ChatterLevelUp)
	cards := make([]ui.Card, len(g.offers))
	for index, offer := range g.offers {
		cards[index] = ui.Card{Title: offer.Title(), Subtitle: offer.Subtitle(), Description: offer.Description(), Color: offer.Color()}
	}
	return cards
}

func (g *Game) ChooseOffer(index int) {
	offer := g.offers[index]
	g.applyOffer(offer)
	if len(g.offers) > 1 {
		return
	}
	g.levelUpBanner = "LEVEL UP!   " + offer.Title() + "  " + offer.Subtitle()
	g.levelUpBannerSeconds = levelUpBannerDuration
}

// ===== Internal =====

func (g *Game) runStats() map[string]int {
	g.stats[statKills], g.stats[statLevel] = g.kills, g.player.Level
	return g.stats
}
