package main

import (
	_ "embed"
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// ===== Types =====

type AudioEngine struct {
	context       *audio.Context
	player        *audio.Player
	mutex         sync.Mutex
	effectVoices  [effectVoiceCount]Voice
	nextVoice     int
	pending       [maximumPendingSounds]SoundKind
	pendingCount  int
	lastPlayed    [soundKindCount]float32
	tracker       *Tracker
	random        Random
	MusicVolume   float32
	EffectsVolume float32
	IsMusicMuted  bool
	SongError     error
}

// ===== Constants =====

const (
	effectVoiceCount     = 40
	maximumPendingSounds = 64
	bytesPerFrame        = 8
	audioBufferDuration  = 60 * time.Millisecond
	effectPanSpread      = 0.15
)

//go:embed music/theme.trk
var themeSource string

// ===== Public API =====

func NewAudioEngine() *AudioEngine {
	engine := &AudioEngine{random: NewRandom(0x50D), MusicVolume: 0.5, EffectsVolume: 0.8}
	song, err := ParseSong(themeSource)
	engine.SongError = err
	engine.attachSong(song)
	engine.context = audio.NewContext(sampleRate)
	player, err := engine.context.NewPlayerF32(engine)
	if err != nil {
		return engine
	}
	player.SetBufferSize(audioBufferDuration)
	player.Play()
	engine.player = player
	return engine
}

func (e *AudioEngine) Play(kind SoundKind, nowSeconds float32) {
	if e == nil || nowSeconds-e.lastPlayed[kind] < soundTable[kind].MinimumInterval {
		return
	}
	e.lastPlayed[kind] = nowSeconds
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.pending[e.pendingCount] = kind
	e.pendingCount = min(e.pendingCount+1, maximumPendingSounds-1)
}

func (e *AudioEngine) SetMusicSignals(signals [musicSignalCount]float32) {
	if e == nil || e.tracker == nil {
		return
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.tracker.SetSignals(signals)
}

func (e *AudioEngine) SetEffectsVolume(volume float32) {
	if e == nil {
		return
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.EffectsVolume = volume
}

func (e *AudioEngine) ToggleMusic() {
	if e == nil {
		return
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.IsMusicMuted = !e.IsMusicMuted
}

func (e *AudioEngine) Read(buffer []byte) (int, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.startPendingSounds()
	frameCount := len(buffer) / bytesPerFrame
	for frame := range frameCount {
		left, right := e.renderFrame()
		binary.LittleEndian.PutUint32(buffer[frame*bytesPerFrame:], math.Float32bits(left))
		binary.LittleEndian.PutUint32(buffer[frame*bytesPerFrame+4:], math.Float32bits(right))
	}
	return frameCount * bytesPerFrame, nil
}

// ===== Internal =====

func (e *AudioEngine) attachSong(song *Song) {
	if song == nil {
		return
	}
	e.tracker = NewTracker(song)
}

func (e *AudioEngine) startPendingSounds() {
	for index := range e.pendingCount {
		e.startSound(&soundTable[e.pending[index]])
	}
	e.pendingCount = 0
}

func (e *AudioEngine) startSound(definition *SoundDefinition) {
	pan := 0.5 + e.random.Between(-effectPanSpread, effectPanSpread)
	pitchRoll := e.random.Between(-1, 1)
	for _, layer := range definition.Layers {
		settings := layer.Settings
		settings.Frequency *= 1 + layer.PitchJitter*pitchRoll
		settings.Pan = pan
		e.effectVoices[e.nextVoice].Start(settings, e.random.Next())
		e.nextVoice = (e.nextVoice + 1) % effectVoiceCount
	}
}

func (e *AudioEngine) renderFrame() (float32, float32) {
	musicLeft, musicRight := e.renderMusic()
	effectsLeft, effectsRight := e.renderEffects()
	musicGain := e.MusicVolume * boolToFloat(!e.IsMusicMuted)
	return softClip(musicLeft*musicGain + effectsLeft*e.EffectsVolume), softClip(musicRight*musicGain + effectsRight*e.EffectsVolume)
}

func (e *AudioEngine) renderMusic() (float32, float32) {
	if e.tracker == nil {
		return 0, 0
	}
	return e.tracker.Render()
}

func (e *AudioEngine) renderEffects() (float32, float32) {
	left, right := float32(0), float32(0)
	for index := range e.effectVoices {
		voiceLeft, voiceRight := e.renderEffectVoice(&e.effectVoices[index])
		left += voiceLeft
		right += voiceRight
	}
	return left, right
}

func (e *AudioEngine) renderEffectVoice(voice *Voice) (float32, float32) {
	if !voice.IsActive {
		return 0, 0
	}
	return voice.Render()
}

func softClip(value float32) float32 {
	limited := clamp(value, -1.5, 1.5)
	return limited - limited*limited*limited/6.75
}
