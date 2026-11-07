package game

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ajkula/ajkurig/ui"
)

// ===== Types =====

type ChatterEvent uint8

type Chatter struct {
	*ui.Chatter
	RecentReactions float32
	WasHealthLow    bool
	HasGreeted      bool
}

// ===== Constants =====

const (
	ChatterRunStart ChatterEvent = iota
	ChatterLevelUp
	ChatterHeal
	ChatterSpiderInbound
	ChatterSpiderDown
	ChatterHordeIncoming
	ChatterHordeCleared
	ChatterBigReaction
	ChatterLowHealth
	ChatterStorm
	ChatterHeatwave
	ChatterBlizzard
	ChatterEvolution
	ChatterIdle
	ChatterWormInbound
	ChatterWormDown
	ChatterWormBlast
	ChatterWormFrost
	ChatterWormHead
	ChatterRivalTooClose
	ChatterRivalStoleKill
	ChatterRivalOvertaken
	ChatterRivalLeftBehind
)

const (
	chatterScrollSpeed     = 70
	chatterLift            = 60
	chatterTextRise        = 3
	chatterGlobalGap       = 12
	chatterUrgentGap       = 3
	chatterGreetingDelay   = 2.5
	chatterIdleSeconds     = 50
	chatterBigReactionRate = 12
	chatterReactionDecay   = 1
	chatterLowHealthRatio  = 0.3
	chatterBoxPadding      = 2
	chatterBoxAlpha        = 0.5
	chatterBoxCharacters   = 15
	chatterTextSize        = 18
)

var chatterTopics = []ui.Topic{
	ChatterRunStart:        {Key: "run_start", LineCount: 3, CooldownSeconds: 0},
	ChatterLevelUp:         {Key: "level_up", LineCount: 3, CooldownSeconds: 60},
	ChatterHeal:            {Key: "heal", LineCount: 3, CooldownSeconds: 60},
	ChatterSpiderInbound:   {Key: "spider_inbound", LineCount: 3, CooldownSeconds: 20, IsUrgent: true},
	ChatterSpiderDown:      {Key: "spider_down", LineCount: 2, CooldownSeconds: 20, IsUrgent: true},
	ChatterHordeIncoming:   {Key: "horde_incoming", LineCount: 2, CooldownSeconds: 20, IsUrgent: true},
	ChatterHordeCleared:    {Key: "horde_cleared", LineCount: 2, CooldownSeconds: 20, IsUrgent: true},
	ChatterBigReaction:     {Key: "big_reaction", LineCount: 3, CooldownSeconds: 40},
	ChatterLowHealth:       {Key: "low_health", LineCount: 3, CooldownSeconds: 45},
	ChatterStorm:           {Key: "storm", LineCount: 2, CooldownSeconds: 30},
	ChatterHeatwave:        {Key: "heatwave", LineCount: 2, CooldownSeconds: 30},
	ChatterBlizzard:        {Key: "blizzard", LineCount: 2, CooldownSeconds: 30},
	ChatterEvolution:       {Key: "evolution", LineCount: 2, CooldownSeconds: 20},
	ChatterIdle:            {Key: "idle", LineCount: 6, CooldownSeconds: 0},
	ChatterWormInbound:     {Key: "worm_inbound", LineCount: 2, CooldownSeconds: 20, IsUrgent: true},
	ChatterWormDown:        {Key: "worm_down", LineCount: 2, CooldownSeconds: 20, IsUrgent: true},
	ChatterWormBlast:       {Key: "worm_blast", LineCount: 2, CooldownSeconds: 20, IsUrgent: true},
	ChatterWormFrost:       {Key: "worm_frost", LineCount: 2, CooldownSeconds: 30, IsUrgent: true},
	ChatterWormHead:        {Key: "worm_head", LineCount: 2, CooldownSeconds: 20, IsUrgent: true},
	ChatterRivalTooClose:   {Key: "rival_too_close", LineCount: 2, CooldownSeconds: 30},
	ChatterRivalStoleKill:  {Key: "rival_stole_kill", LineCount: 2, CooldownSeconds: 30},
	ChatterRivalOvertaken:  {Key: "rival_overtaken", LineCount: 2, CooldownSeconds: 45},
	ChatterRivalLeftBehind: {Key: "rival_left_behind", LineCount: 2, CooldownSeconds: 45},
}

var chatterStyle = ui.BubbleStyle{
	KeyPrefix: "tsukumo", GlobalGap: chatterGlobalGap, UrgentGap: chatterUrgentGap, ScrollSpeed: chatterScrollSpeed,
	BoxCharacters: chatterBoxCharacters, Padding: chatterBoxPadding, Alpha: chatterBoxAlpha, Lift: chatterLift, TextRise: chatterTextRise,
	Face: ui.FaceMono, TextColor: textColor, BoxColor: panelColor,
}

var weatherChatter = [weatherKindCount]ChatterEvent{WeatherStorm: ChatterStorm, WeatherHeatwave: ChatterHeatwave, WeatherBlizzard: ChatterBlizzard}

// ===== Internal =====

func newChatter() Chatter {
	return Chatter{Chatter: ui.NewChatter(chatterStyle, chatterTopics)}
}

func (g *Game) resetChatter() {
	g.chatter.Reset()
	g.chatter.RecentReactions, g.chatter.WasHealthLow, g.chatter.HasGreeted = 0, false, false
}

func (g *Game) updateChatter(deltaSeconds float32) {
	chatter := &g.chatter
	chatter.Update(deltaSeconds)
	chatter.RecentReactions = max(0, chatter.RecentReactions-chatterBigReactionRate*chatterReactionDecay*deltaSeconds)
	g.sayIf(ChatterRunStart, !chatter.HasGreeted && g.elapsedSeconds >= chatterGreetingDelay)
	isHealthLow := g.player.Health < g.player.MaximumHealth*chatterLowHealthRatio
	g.sayIf(ChatterLowHealth, isHealthLow && !chatter.WasHealthLow)
	chatter.WasHealthLow = isHealthLow
	g.sayIf(ChatterBigReaction, chatter.RecentReactions >= chatterBigReactionRate)
	g.sayIf(ChatterIdle, chatter.QuietSeconds >= chatterIdleSeconds)
}

func (g *Game) noteChatterReaction() {
	g.chatter.RecentReactions++
}

func (g *Game) sayIf(event ChatterEvent, shouldSay bool) {
	if !shouldSay {
		return
	}
	g.say(event)
}

func (g *Game) say(event ChatterEvent) {
	chatter := &g.chatter
	chatter.IsSilenced = g.isDemo
	hasSpoken := chatter.Say(int(event), &g.random)
	chatter.HasGreeted = chatter.HasGreeted || hasSpoken
	g.playSoundIf(SoundChatter, hasSpoken)
}

func (u *UI) drawChatter(g *Game, screen *ebiten.Image) {
	playerX, playerY := g.renderer.camera.ToScreen(g.player.X, g.player.Y)
	u.kit.DrawBubble(screen, g.chatter.Chatter, playerX, playerY)
}
