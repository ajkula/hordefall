package game

import (
	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/input"
	"github.com/ajkula/ajkurig/rig"
	"github.com/ajkula/ajkurig/scores"
	"github.com/ajkula/ajkurig/shell"
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
		Stats: g.runStats(),
	}
	if !status.IsOver {
		return status
	}
	status.Summary = i18n.F("gameover.summary", scores.FormatClock(g.elapsedSeconds), g.player.Level, g.kills)
	status.Details = describeReactionCounts(g.reactionCounts)
	status.KilledBy = g.damageLog.Causes(g.elapsedSeconds)
	return status
}

// ===== Internal =====

func (g *Game) runStats() map[string]int {
	g.stats[statKills], g.stats[statLevel] = g.kills, g.player.Level
	return g.stats
}
