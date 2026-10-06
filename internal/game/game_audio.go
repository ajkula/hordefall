package game

import (
	"fmt"
	"strings"

	"hordefall/internal/audio"
)

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
	songRotationSeconds     = 300
	noMusicSlot             = -1
	musicBannerDuration     = 45
)

var laserDangerPhases = [laserPhaseCount]float32{LaserCharging: 1, LaserLocked: 1, LaserFiring: 1}

var fullMusicSignals = [audio.SignalCount]float32{
	audio.SignalAlways: 1, audio.SignalHorde: 1, audio.SignalBoss: 3, audio.SignalReactions: 1, audio.SignalDanger: 1,
}

// ===== Internal =====

func (g *Game) playSound(kind audio.SoundKind) {
	g.audio.Play(kind, g.clockSeconds)
}

func (g *Game) playSoundIf(kind audio.SoundKind, shouldPlay bool) {
	if !shouldPlay {
		return
	}
	g.playSound(kind)
}

func (g *Game) updateAudio() {
	if g.controls.JustPressed&ActionMute != 0 {
		g.toggleMusic()
	}
	contextVolume := menuEffectsVolume + (gameEffectsVolume-menuEffectsVolume)*boolToFloat(!g.isDemo || g.state == StateIntro)
	g.audio.SetEffectsVolume(contextVolume * g.effectsVolumeScale())
	g.rotateSong()
	if g.frame%musicSignalFrames != 0 {
		return
	}
	g.musicSignals = g.computeMusicSignals()
	g.audio.SetMusicSignals(g.musicSignals)
}

func (g *Game) rotateSong() {
	if !g.HasEnabledSong() {
		return
	}
	slot := g.currentMusicSlot()
	isFirstObservation := g.musicSlot == noMusicSlot
	isSlotChanged := slot != g.musicSlot && !isFirstObservation
	g.musicSlot = slot
	if isFirstObservation {
		songCount := g.audio.SongCount()
		g.playSong(g.nextEnabledSong((g.musicRandom.Below(songCount) + songCount - 1) % songCount))
		return
	}
	isCurrentDisabled := !g.IsSongEnabled(g.audio.CurrentSong())
	if !isSlotChanged && !isCurrentDisabled {
		return
	}
	g.playSong(g.nextEnabledSong(g.audio.CurrentSong()))
}

func (g *Game) currentMusicSlot() int {
	isInRun := !g.isDemo && g.state != StateMainMenu && g.state != StateOptions && g.state != StateRemap && g.state != StatePlaylist
	runSlot := int(g.elapsedSeconds) / songRotationSeconds
	demoSlot := max(0, g.demoSlot) / (2 * len(demoSequences))
	slots := [2]int{demoSlot, runSlot}
	return slots[boolToIndex(isInRun)]*2 + boolToIndex(isInRun)
}

func (g *Game) describeMusic() string {
	titles := make([]string, 0, g.audio.SongCount())
	for index := range g.audio.SongCount() {
		titles = appendIf(titles, g.audio.SongTitle(index), g.IsSongEnabled(index))
	}
	signals := g.musicSignals
	levels := fmt.Sprintf("   layers %d/16  horde %.2f  boss %.0f  reactions %.2f  danger %.2f", g.audio.AudibleLayerCount(), signals[audio.SignalHorde], signals[audio.SignalBoss], signals[audio.SignalReactions], signals[audio.SignalDanger])
	return "Music playing: " + g.audio.PlayingDescription() + levels + "   enabled: " + strings.Join(titles, ", ")
}

func (g *Game) nextEnabledSong(current int) int {
	songCount := g.audio.SongCount()
	for step := 1; step <= songCount; step++ {
		candidate := (current + step) % songCount
		if g.IsSongEnabled(candidate) {
			return candidate
		}
	}
	return current
}

func (g *Game) playSong(index int) {
	if !g.audio.SelectSong(index) {
		return
	}
	g.musicBanner = "\u266A  " + g.audio.SongTitle(index)
	g.musicBannerSeconds = musicBannerDuration
}

func (g *Game) computeMusicSignals() [audio.SignalCount]float32 {
	if g.state == StateGameOver {
		return fullMusicSignals
	}
	var signals [audio.SignalCount]float32
	signals[audio.SignalAlways] = 1
	hordeSignal := clamp(float32(g.countEnemiesNear(hordeSignalRadius))/hordeSignalFullCount, 0, 1)
	signals[audio.SignalHorde] = hordeSignal
	signals[audio.SignalBoss] = float32(len(g.spiders) + boolToIndex(g.hordeEvent.IsActive) + boolToIndex(g.worm.IsActive))
	signals[audio.SignalReactions] = g.reactionSignal()
	signals[audio.SignalDanger] = max(g.lowHealthSignal(), g.laserDangerSignal())
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
	return g.musicSignals[audio.SignalReactions] + (target-g.musicSignals[audio.SignalReactions])*reactionSignalSmoothing
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
