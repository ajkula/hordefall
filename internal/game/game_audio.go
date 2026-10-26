package game

import (
	"github.com/ajkula/ajkurig/audio"

	"hordefall/music"
)

// ===== Types =====

type MusicSignal uint8

// ===== Constants =====

const (
	musicSignalFrames       = 6
	hordeSignalRadius       = 420
	hordeSignalFullCount    = 110
	reactionSignalFullRate  = 18
	reactionSignalSmoothing = 0.15
	dangerHealthThreshold   = 0.45
)

const (
	SignalAlways MusicSignal = iota
	SignalHorde
	SignalBoss
	SignalReactions
	SignalDanger
	SignalCount
)

var signalNames = [SignalCount]string{
	SignalAlways: audio.AlwaysName, SignalHorde: music.SignalHorde, SignalBoss: music.SignalBoss, SignalReactions: music.SignalReactions, SignalDanger: music.SignalDanger,
}

var laserDangerPhases = [laserPhaseCount]float32{LaserCharging: 1, LaserLocked: 1, LaserFiring: 1}

var fullMusicSignals = [SignalCount]float32{
	SignalAlways: 1, SignalHorde: 1, SignalBoss: 3, SignalReactions: 1, SignalDanger: 1,
}

// ===== Internal =====

func (g *Game) attachAudio(engine *audio.Engine, ids SoundIDs) {
	g.audio, g.soundIDs = engine, ids
	for signal, name := range signalNames {
		g.signalIDs[signal] = engine.Signals().Register(name)
	}
}

func (g *Game) playSound(kind SoundKind) {
	g.app.PlaySound(g.soundIDs[kind])
}

func (g *Game) playSoundIf(kind SoundKind, shouldPlay bool) {
	if !shouldPlay {
		return
	}
	g.playSound(kind)
}

func (g *Game) updateMusicSignals() {
	isDue := g.frame%musicSignalFrames == 0 || g.player.Health <= 0
	if !isDue {
		return
	}
	g.musicSignals = g.computeMusicSignals()
	board := g.audio.Signals()
	for signal, value := range g.musicSignals {
		board.Set(g.signalIDs[signal], value)
	}
}

func (g *Game) computeMusicSignals() [SignalCount]float32 {
	if g.player.Health <= 0 {
		return fullMusicSignals
	}
	var signals [SignalCount]float32
	signals[SignalAlways] = 1
	hordeSignal := clamp(float32(g.countEnemiesNear(hordeSignalRadius))/hordeSignalFullCount, 0, 1)
	signals[SignalHorde] = hordeSignal
	signals[SignalBoss] = float32(len(g.spiders) + boolToIndex(g.hordeEvent.IsActive) + boolToIndex(g.worm.IsActive))
	signals[SignalReactions] = g.reactionSignal()
	signals[SignalDanger] = max(g.lowHealthSignal(), g.laserDangerSignal())
	return signals
}

func (g *Game) countEnemiesNear(radius float32) int {
	count := 0
	radiusSquared := radius * radius
	for index := range g.enemies.Count {
		count += boolToIndex(distanceSquared(g.enemies.PositionX[index], g.enemies.PositionY[index], g.player.X, g.player.Y) < radiusSquared)
	}
	return count
}

func (g *Game) reactionSignal() float32 {
	total := sumReactions(g.reactionCounts)
	newReactions := max(0, total-g.previousReactions)
	g.previousReactions = total
	rate := float32(newReactions) * ticksPerSecond / musicSignalFrames
	target := clamp(rate/reactionSignalFullRate, 0, 1)
	return g.musicSignals[SignalReactions] + (target-g.musicSignals[SignalReactions])*reactionSignalSmoothing
}

func (g *Game) lowHealthSignal() float32 {
	healthFraction := g.player.Health / g.player.MaximumHealth
	isLow := healthFraction < dangerHealthThreshold
	return clamp(0.5+(dangerHealthThreshold-healthFraction)/0.25, 0, 1) * boolToFloat(isLow)
}

func (g *Game) laserDangerSignal() float32 {
	danger := float32(0)
	for index := range g.spiders {
		danger = max(danger, laserDangerPhases[g.spiders[index].Laser.Phase])
	}
	return danger
}
