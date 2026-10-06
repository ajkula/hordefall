package game

import (
	"image"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"hordefall/internal/audio"
	"hordefall/internal/i18n"
)

// ===== Types =====

type ChatterEvent uint8

type ChatterTopic struct {
	Key             string
	LineCount       int
	CooldownSeconds float32
}

type Chatter struct {
	Runes           []rune
	Seconds         float32
	Cooldowns       [chatterEventCount]float32
	LastLines       [chatterEventCount]int
	QuietSeconds    float32
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
	ChatterRivalTooClose
	ChatterRivalStoleKill
	ChatterRivalOvertaken
	ChatterRivalLeftBehind
	chatterEventCount
)

const (
	chatterScrollSpeed     = 70
	chatterLift            = 60
	chatterTextRise        = 3
	chatterGlobalGap       = 12
	chatterGreetingDelay   = 2.5
	chatterIdleSeconds     = 50
	chatterBigReactionRate = 12
	chatterReactionDecay   = 1
	chatterLowHealthRatio  = 0.3
	chatterBoxPadding      = 2
	chatterBoxAlpha        = 0.5
	chatterSample          = "MMMMMMMMMMMMMMM"
	chatterTextSize        = 18
)

var chatterTopics = [chatterEventCount]ChatterTopic{
	ChatterRunStart:        {"run_start", 3, 0},
	ChatterLevelUp:         {"level_up", 3, 60},
	ChatterHeal:            {"heal", 3, 60},
	ChatterSpiderInbound:   {"spider_inbound", 3, 20},
	ChatterSpiderDown:      {"spider_down", 2, 20},
	ChatterHordeIncoming:   {"horde_incoming", 2, 20},
	ChatterHordeCleared:    {"horde_cleared", 2, 20},
	ChatterBigReaction:     {"big_reaction", 3, 40},
	ChatterLowHealth:       {"low_health", 3, 45},
	ChatterStorm:           {"storm", 2, 30},
	ChatterHeatwave:        {"heatwave", 2, 30},
	ChatterBlizzard:        {"blizzard", 2, 30},
	ChatterEvolution:       {"evolution", 2, 20},
	ChatterIdle:            {"idle", 6, 0},
	ChatterRivalTooClose:   {"rival_too_close", 2, 30},
	ChatterRivalStoleKill:  {"rival_stole_kill", 2, 30},
	ChatterRivalOvertaken:  {"rival_overtaken", 2, 45},
	ChatterRivalLeftBehind: {"rival_left_behind", 2, 45},
}

var weatherChatter = [weatherKindCount]ChatterEvent{WeatherStorm: ChatterStorm, WeatherHeatwave: ChatterHeatwave, WeatherBlizzard: ChatterBlizzard}

// ===== Internal =====

func (g *Game) resetChatter() {
	g.chatter = Chatter{}
	for event := range g.chatter.LastLines {
		g.chatter.LastLines[event] = -1
	}
}

func (g *Game) updateChatter(deltaSeconds float32) {
	chatter := &g.chatter
	chatter.Seconds += deltaSeconds
	chatter.QuietSeconds += deltaSeconds
	chatter.RecentReactions = max(0, chatter.RecentReactions-chatterBigReactionRate*chatterReactionDecay*deltaSeconds)
	for event := range chatter.Cooldowns {
		chatter.Cooldowns[event] = max(0, chatter.Cooldowns[event]-deltaSeconds)
	}
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
	isBusy := g.isDemo || chatter.QuietSeconds < chatterGlobalGap || chatter.Cooldowns[event] > 0
	if isBusy {
		return
	}
	topic := &chatterTopics[event]
	line := g.random.Below(topic.LineCount)
	line = (line + boolToIndex(line == chatter.LastLines[event] && topic.LineCount > 1)) % topic.LineCount
	chatter.LastLines[event] = line
	chatter.Runes = []rune(i18n.T("tachikoma." + topic.Key + "." + strconv.Itoa(line+1)))
	chatter.Seconds, chatter.QuietSeconds = 0, 0
	chatter.Cooldowns[event] = topic.CooldownSeconds
	chatter.HasGreeted = true
	g.playSound(audio.SoundChatter)
}

func (c *Chatter) IsSpeaking(textWidth, boxWidth float32) bool {
	return len(c.Runes) > 0 && c.Seconds*chatterScrollSpeed < textWidth+boxWidth
}

func marqueeOffset(seconds, boxWidth float32) float32 {
	return boxWidth - float32(int(seconds*chatterScrollSpeed))
}

func (u *UI) drawChatter(g *Game, screen *ebiten.Image) {
	message := string(g.chatter.Runes)
	textWidth := u.textWidth(message, u.mono)
	boxWidth := u.textWidth(chatterSample, u.mono)
	if !g.chatter.IsSpeaking(textWidth, boxWidth) {
		return
	}
	playerX, playerY := g.renderer.ToScreen(g.player.X, g.player.Y)
	height := float32(u.mono.Size) + 2*chatterBoxPadding
	left := float32(int(clamp(playerX-boxWidth/2-chatterBoxPadding, 0, screenWidth-boxWidth-2*chatterBoxPadding)))
	top := float32(int(clamp(playerY-chatterLift-height, 0, screenHeight-height)))
	fillRect(screen, left, top, boxWidth+2*chatterBoxPadding, height, toColor(panelColor, chatterBoxAlpha))
	window := screen.SubImage(image.Rect(int((left+chatterBoxPadding)*renderScale), int(top*renderScale), int((left+chatterBoxPadding+boxWidth)*renderScale), int((top+height)*renderScale))).(*ebiten.Image)
	textX := left + chatterBoxPadding + marqueeOffset(g.chatter.Seconds, boxWidth)
	u.drawText(window, message, u.mono, textX, top+chatterBoxPadding-chatterTextRise, textColor, 1, text.AlignStart)
}
