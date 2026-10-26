package game

import (
	"fmt"
	"strings"

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
	menuEffectsVolume       = 0.45
	gameEffectsVolume       = 0.8
	songRotationSeconds     = 300
	noMusicSlot             = -1
	musicBannerDuration     = 45
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

func newAudio(seed uint32, isSilent bool) (*audio.Engine, SoundIDs) {
	bank := audio.NewSoundBank()
	ids := RegisterSounds(bank)
	board := audio.NewSignalBoard(music.SignalNames()...)
	engines := [2]func(*audio.SoundBank, *audio.SignalBoard) *audio.Engine{audio.NewEngine, audio.NewSilentEngine}
	engine := engines[boolToIndex(isSilent)](bank, board)
	songs, _ := music.Songs(seed)
	engine.SetSongs(songs)
	return engine, ids
}

func newSilentAudio(seed uint32) (*audio.Engine, SoundIDs) {
	return newAudio(seed, true)
}

func (g *Game) attachAudio(engine *audio.Engine, ids SoundIDs) {
	g.audio, g.soundIDs = engine, ids
	for signal, name := range signalNames {
		g.signalIDs[signal] = engine.Signals().Register(name)
	}
}

func (g *Game) playSound(kind SoundKind) {
	g.audio.Play(g.soundIDs[kind], g.clockSeconds)
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
	contextVolume := menuEffectsVolume + (gameEffectsVolume-menuEffectsVolume)*boolToFloat(!g.isDemo || g.state == StateIntro)
	g.audio.SetEffectsVolume(contextVolume * g.effectsVolumeScale())
	g.rotateSong()
	if g.frame%musicSignalFrames != 0 {
		return
	}
	g.musicSignals = g.computeMusicSignals()
	g.pushMusicSignals()
}

func (g *Game) pushMusicSignals() {
	if g.audio == nil {
		return
	}
	board := g.audio.Signals()
	for signal, value := range g.musicSignals {
		board.Set(g.signalIDs[signal], value)
	}
	g.audio.PushSignals()
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
	levels := fmt.Sprintf("   layers %d/16  horde %.2f  boss %.0f  reactions %.2f  danger %.2f", g.audio.AudibleLayerCount(), signals[SignalHorde], signals[SignalBoss], signals[SignalReactions], signals[SignalDanger])
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

func (g *Game) computeMusicSignals() [SignalCount]float32 {
	if g.state == StateGameOver {
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
