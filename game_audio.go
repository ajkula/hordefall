package main

// ===== Constants =====

const (
	musicSignalFrames       = 6
	hordeSignalRadius       = 420
	hordeSignalFullCount    = 110
	reactionSignalFullRate  = 18
	reactionSignalSmoothing = 0.15
	dangerHealthThreshold   = 0.45
	menuEffectsVolume       = 0.45
	gameEffectsVolume       = 0.8
)

var laserDangerPhases = [laserPhaseCount]float32{LaserCharging: 1, LaserLocked: 1, LaserFiring: 1}

// ===== Internal =====

func (g *Game) playSound(kind SoundKind) {
	g.audio.Play(kind, g.clockSeconds)
}

func (g *Game) playSoundIf(kind SoundKind, shouldPlay bool) {
	if !shouldPlay {
		return
	}
	g.playSound(kind)
}

func (g *Game) updateAudio() {
	if g.controls.JustPressed&ActionMute != 0 {
		g.toggleMusic()
	}
	g.audio.SetEffectsVolume(menuEffectsVolume + (gameEffectsVolume-menuEffectsVolume)*boolToFloat(!g.isDemo))
	if g.frame%musicSignalFrames != 0 {
		return
	}
	g.musicSignals = g.computeMusicSignals()
	g.audio.SetMusicSignals(g.musicSignals)
}

func (g *Game) computeMusicSignals() [musicSignalCount]float32 {
	var signals [musicSignalCount]float32
	isFighting := boolToFloat(g.state != StateGameOver)
	signals[SignalAlways] = 1
	signals[SignalHorde] = clamp(float32(g.countEnemiesNear(hordeSignalRadius))/hordeSignalFullCount, 0, 1) * isFighting
	signals[SignalBoss] = float32(len(g.spiders)) * isFighting
	signals[SignalReactions] = g.reactionSignal() * isFighting
	signals[SignalDanger] = max(g.lowHealthSignal(), g.laserDangerSignal()) * isFighting
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
