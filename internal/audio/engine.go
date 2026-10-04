package audio

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"hordefall/internal/rng"
)

// ===== Types =====

type Engine struct {
	context         *audio.Context
	player          *audio.Player
	mutex           sync.Mutex
	effectVoices    [effectVoiceCount]Voice
	nextVoice       int
	pending         [maximumPendingSounds]SoundKind
	pendingCount    int
	lastPlayed      [soundKindCount]float32
	tracker         *Tracker
	fadingTracker   *Tracker
	crossfade       float32
	songs           []*Song
	songIndex       int
	room            StereoRoom
	random          rng.Random
	MusicVolume     float32
	EffectsVolume   float32
	IsMusicMuted    bool
	IsPlaylistEmpty bool
	SongError       error
}

// ===== Constants =====

const (
	effectVoiceCount     = 40
	maximumPendingSounds = 64
	bytesPerFrame        = 8
	audioBufferDuration  = 60 * time.Millisecond
	effectPanSpread      = 0.15
	songCrossfadeSeconds = 3
)

//go:embed music/theme.trk
var themeSource string

// ===== Public API =====

func NewEngine() *Engine {
	engine := NewSilentEngine(uint32(time.Now().UnixNano()))
	engine.MusicVolume, engine.EffectsVolume = 0.5, 0.8
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

func ParseTheme() (*Song, error) {
	return ParseSong(themeSource)
}

func NewSilentEngine(seed uint32) *Engine {
	engine := &Engine{random: rng.New(0x50D)}
	song, err := ParseSong(themeSource)
	engine.SongError = err
	engine.attachSongs(song, seed)
	return engine
}

func (e *Engine) Play(kind SoundKind, nowSeconds float32) {
	if e == nil || nowSeconds-e.lastPlayed[kind] < soundTable[kind].MinimumInterval {
		return
	}
	e.lastPlayed[kind] = nowSeconds
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.pending[e.pendingCount] = kind
	e.pendingCount = min(e.pendingCount+1, maximumPendingSounds-1)
}

func (e *Engine) SetMusicSignals(signals [SignalCount]float32) {
	if e == nil || e.tracker == nil {
		return
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.tracker.SetSignals(signals)
	e.signalFadingTracker(signals)
}

func (e *Engine) SongCount() int {
	if e == nil {
		return 0
	}
	return len(e.songs)
}

func (e *Engine) CurrentSong() int {
	if e == nil {
		return 0
	}
	return e.songIndex
}

func (e *Engine) PlayingDescription() string {
	if e == nil || e.tracker == nil {
		return "no audio"
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	song := e.tracker.song
	return fmt.Sprintf("%s (tempo %d, speed %d)", song.Title, song.Tempo, song.Speed)
}

func (e *Engine) SongTitle(index int) string {
	if e == nil {
		return ""
	}
	return e.songs[index].Title
}

func (e *Engine) SelectSong(index int) bool {
	if e == nil || index == e.songIndex || index >= len(e.songs) {
		return false
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	signals := e.tracker.signals
	e.fadingTracker = e.tracker
	e.tracker = NewTracker(e.songs[index])
	e.tracker.SetSignals(signals)
	e.crossfade = 0
	e.songIndex = index
	return true
}

func (e *Engine) SetEffectsVolume(volume float32) {
	if e == nil {
		return
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.EffectsVolume = volume
}

func (e *Engine) IsMusicOn() bool {
	if e == nil {
		return true
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return !e.IsMusicMuted
}

func (e *Engine) SetMusicVolume(volume float32) {
	if e == nil {
		return
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.MusicVolume = volume
}

func (e *Engine) SetPlaylistEmpty(isEmpty bool) {
	if e == nil {
		return
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.IsPlaylistEmpty = isEmpty
}

func (e *Engine) SetMusicOn(isOn bool) {
	if e == nil {
		return
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.IsMusicMuted = !isOn
}

func (e *Engine) Read(buffer []byte) (int, error) {
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

func (e *Engine) attachSongs(theme *Song, seed uint32) {
	if theme == nil {
		return
	}
	e.songs = append(e.songs, theme)
	for index := range musicStyles {
		e.songs = append(e.songs, ComposeSong(&musicStyles[index], theme, seed+uint32(index)*7919))
	}
	e.tracker = NewTracker(theme)
	e.crossfade = 1
}

func (e *Engine) startPendingSounds() {
	for index := range e.pendingCount {
		e.startSound(&soundTable[e.pending[index]])
	}
	e.pendingCount = 0
}

func (e *Engine) startSound(definition *SoundDefinition) {
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

func (e *Engine) renderFrame() (float32, float32) {
	musicLeft, musicRight := e.renderMusic()
	effectsLeft, effectsRight := e.renderEffects()
	musicGain := e.MusicVolume * boolToFloat(!e.IsMusicMuted && !e.IsPlaylistEmpty)
	return softClip(musicLeft*musicGain + effectsLeft*e.EffectsVolume), softClip(musicRight*musicGain + effectsRight*e.EffectsVolume)
}

func (e *Engine) renderMusic() (float32, float32) {
	if e.tracker == nil {
		return 0, 0
	}
	e.crossfade = min(1, e.crossfade+1/(songCrossfadeSeconds*sampleRate))
	left, right := e.tracker.Render()
	fadingLeft, fadingRight := e.renderFadingTracker()
	return e.room.Process(left*e.crossfade+fadingLeft*(1-e.crossfade), right*e.crossfade+fadingRight*(1-e.crossfade))
}

func (e *Engine) renderFadingTracker() (float32, float32) {
	if e.fadingTracker == nil || e.crossfade >= 1 {
		e.fadingTracker = nil
		return 0, 0
	}
	return e.fadingTracker.Render()
}

func (e *Engine) signalFadingTracker(signals [SignalCount]float32) {
	if e.fadingTracker == nil {
		return
	}
	e.fadingTracker.SetSignals(signals)
}

func (e *Engine) renderEffects() (float32, float32) {
	left, right := float32(0), float32(0)
	for index := range e.effectVoices {
		voiceLeft, voiceRight := e.renderEffectVoice(&e.effectVoices[index])
		left += voiceLeft
		right += voiceRight
	}
	return left, right
}

func (e *Engine) renderEffectVoice(voice *Voice) (float32, float32) {
	if !voice.IsActive {
		return 0, 0
	}
	return voice.Render()
}

func softClip(value float32) float32 {
	limited := clamp(value, -1.5, 1.5)
	return limited - limited*limited*limited/6.75
}
