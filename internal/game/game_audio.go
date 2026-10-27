package game

import (
	"github.com/ajkula/ajkurig/audio"

	"hordefall/music"
)

// ===== Constants =====

const (
	musicSignalFrames       = 6
	hordeSignalRadius       = 420
	hordeSignalFullCount    = 110
	reactionSignalFullRate  = 18
	reactionSignalSmoothing = 0.15
	dangerHealthThreshold   = 0.45
)

var laserDangerPhases = [laserPhaseCount]float32{LaserCharging: 1, LaserLocked: 1, LaserFiring: 1}

var fullMusicSignals = [music.SignalCount]float32{
	music.SignalAlways: 1, music.SignalHorde: 1, music.SignalBoss: 3, music.SignalReactions: 1, music.SignalDanger: 1,
}

// ===== Internal =====

func (g *Game) attachAudio(engine *audio.Engine, ids SoundIDs) {
	g.audio, g.soundIDs = engine, ids
	for signal := range music.SignalCount {
		g.signalIDs[signal] = engine.Signals().Register(signal.Name())
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

func (g *Game) computeMusicSignals() [music.SignalCount]float32 {
	if g.player.Health <= 0 {
		return fullMusicSignals
	}
	var signals [music.SignalCount]float32
	signals[music.SignalAlways] = 1
	hordeSignal := clamp(float32(g.countEnemiesNear(hordeSignalRadius))/hordeSignalFullCount, 0, 1)
	signals[music.SignalHorde] = hordeSignal
	signals[music.SignalBoss] = float32(len(g.spiders) + boolToIndex(g.hordeEvent.IsActive) + boolToIndex(g.worm.IsActive))
	signals[music.SignalReactions] = g.reactionSignal()
	signals[music.SignalDanger] = max(g.lowHealthSignal(), g.laserDangerSignal())
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
	return g.musicSignals[music.SignalReactions] + (target-g.musicSignals[music.SignalReactions])*reactionSignalSmoothing
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
