package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ===== Public API =====

func TestThemeSongParses(t *testing.T) {
	song, err := ParseSong(themeSource)
	if err != nil {
		t.Fatalf("theme does not parse: %v", err)
	}
	t.Logf("%q: tempo %d, speed %d, %d patterns, order length %d", song.Title, song.Tempo, song.Speed, len(song.Patterns), len(song.Order))
}

func TestSongParserReportsLineOfError(t *testing.T) {
	row := "C-4 01 ... | X-9 01 ..." + strings.Repeat(" | --- .. ...", trackerChannels-2)
	_, err := ParseSong("tempo 120\norder 0\npattern 0\n" + row + "\n")
	if err == nil || !strings.Contains(err.Error(), "X-9") {
		t.Fatalf("invalid note was not reported: %v", err)
	}
	t.Logf("error message: %v", err)
}

func TestMusicIsCleanAtFullIntensity(t *testing.T) {
	tracker := newThemeTracker(t)
	tracker.SetSignals([musicSignalCount]float32{1, 1, maximumSpidersAlive, 1, 1})
	peak, rms := renderStatistics(t, tracker, 16*sampleRate)
	if peak > 1 || rms < 0.02 {
		t.Fatalf("full mix peak %.3f rms %.3f, want peak <= 1 and audible rms", peak, rms)
	}
	t.Logf("full intensity: peak %.3f rms %.3f", peak, rms)
}

func TestLayersFollowSignals(t *testing.T) {
	calm := newThemeTracker(t)
	_, calmRMS := renderStatistics(t, calm, 8*sampleRate)
	intense := newThemeTracker(t)
	intense.SetSignals([musicSignalCount]float32{1, 1, maximumSpidersAlive, 1, 1})
	_, intenseRMS := renderStatistics(t, intense, 8*sampleRate)
	if intenseRMS <= calmRMS*1.2 {
		t.Fatalf("intense mix rms %.3f is not louder than calm mix rms %.3f", intenseRMS, calmRMS)
	}
	t.Logf("calm rms %.3f, intense rms %.3f", calmRMS, intenseRMS)
}

func TestEachSpiderTankAddsADeeperVoice(t *testing.T) {
	song, err := ParseSong(themeSource)
	if err != nil {
		t.Fatalf("theme does not parse: %v", err)
	}
	bossChannels := []int{}
	for channel, layer := range song.Layers {
		bossChannels = appendIf(bossChannels, channel, layer.Signal == SignalBoss)
	}
	if len(bossChannels) != maximumSpidersAlive {
		t.Fatalf("%d boss layers, want one per spider tank (%d)", len(bossChannels), maximumSpidersAlive)
	}
	previousNote := 1 << 30
	for spiders, channel := range bossChannels {
		if song.Layers[channel].Threshold != float32(spiders+1) {
			t.Fatalf("boss layer on channel %d starts at %.0f tanks, want %d", channel+1, song.Layers[channel].Threshold, spiders+1)
		}
		firstNote := song.Patterns[song.Order[0]].Rows[0][channel].Note
		if firstNote >= previousNote {
			t.Fatalf("boss voice %d (note %d) is not deeper than the previous one (note %d)", spiders+1, firstNote, previousNote)
		}
		previousNote = firstNote
	}
}

func TestEverySoundEffectProducesSound(t *testing.T) {
	for kind := SoundPop; kind < soundKindCount; kind++ {
		engine := &AudioEngine{random: NewRandom(1), EffectsVolume: 1}
		engine.startSound(&soundTable[kind])
		peak := float32(0)
		for range sampleRate / 2 {
			left, right := engine.renderEffects()
			peak = max(peak, abs(left), abs(right))
		}
		if peak < 0.01 || peak > 2 {
			t.Fatalf("sound %d peak %.3f out of range", kind, peak)
		}
	}
}

// ===== Internal =====

func newThemeTracker(t *testing.T) *Tracker {
	song, err := ParseSong(themeSource)
	if err != nil {
		t.Fatalf("theme does not parse: %v", err)
	}
	return NewTracker(song)
}

func renderStatistics(t *testing.T, tracker *Tracker, samples int) (float32, float32) {
	peak, sumSquares := float32(0), float64(0)
	for range samples {
		left, right := tracker.Render()
		mixedLeft, mixedRight := softClip(left*0.5), softClip(right*0.5)
		if math.IsNaN(float64(mixedLeft)) || math.IsNaN(float64(mixedRight)) {
			t.Fatalf("NaN in music output")
		}
		peak = max(peak, abs(mixedLeft), abs(mixedRight))
		sumSquares += float64(mixedLeft*mixedLeft + mixedRight*mixedRight)
	}
	return peak, float32(math.Sqrt(sumSquares / float64(2*samples)))
}

func TestMusicToggleKeepsSoundEffects(t *testing.T) {
	engine := &AudioEngine{random: NewRandom(1), EffectsVolume: 1, MusicVolume: 1, tracker: newThemeTracker(t)}
	engine.SetMusicOn(false)
	if engine.IsMusicOn() {
		t.Fatalf("music still on after toggle")
	}
	engine.startSound(&soundTable[SoundPop])
	peak := float32(0)
	for range sampleRate / 10 {
		left, right := engine.renderFrame()
		peak = max(peak, abs(left), abs(right))
	}
	if peak < 0.01 {
		t.Fatalf("sound effects silenced by the music toggle (peak %.3f)", peak)
	}
}

func TestMusicSettingIsSavedAndRestored(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.audio = &AudioEngine{random: NewRandom(1)}
	game.loadSettings()
	if !game.settings.IsMusicOn || game.musicLabel() != "Music: ON" {
		t.Fatalf("music should default to on, label %q", game.musicLabel())
	}
	game.toggleMusic()
	restarted := newHeadlessGame()
	restarted.audio = &AudioEngine{random: NewRandom(1)}
	restarted.loadSettings()
	if restarted.settings.IsMusicOn || restarted.audio.IsMusicOn() || restarted.musicLabel() != "Music: OFF" {
		t.Fatalf("music off was not restored after restart, label %q", restarted.musicLabel())
	}
}

func TestComposedStylesAreValidCleanAndDistinct(t *testing.T) {
	theme, err := ParseSong(themeSource)
	if err != nil {
		t.Fatalf("theme does not parse: %v", err)
	}
	signatures := map[string]bool{}
	for index := range musicStyles {
		song := ComposeSong(&musicStyles[index], theme, 1234)
		if err := validateSong(song); err != nil {
			t.Fatalf("%s is invalid: %v", song.Title, err)
		}
		tracker := NewTracker(song)
		tracker.SetSignals([musicSignalCount]float32{1, 1, maximumSpidersAlive, 1, 1})
		peak, rms := renderStatistics(t, tracker, 6*sampleRate)
		if peak > 1 || rms < 0.02 {
			t.Fatalf("%s: peak %.3f rms %.3f", song.Title, peak, rms)
		}
		firstLeadNote := song.Patterns[0].Rows[0][channelChords].Note
		signature := fmt.Sprintf("%d/%d/%d", song.Tempo, firstLeadNote, song.Instruments[instrumentLead].Waveform)
		signatures[signature] = true
		t.Logf("%-14s tempo %d  peak %.2f  rms %.3f", song.Title, song.Tempo, peak, rms)
	}
	if len(signatures) != len(musicStyles) {
		t.Fatalf("styles are not distinct: %v", signatures)
	}
}

func TestSongRotatesEveryFiveMinutesWithCleanCrossfade(t *testing.T) {
	theme := newThemeTracker(t).song
	engine := &AudioEngine{random: NewRandom(1), MusicVolume: 1}
	engine.attachSongs(theme, 99)
	game := newHeadlessGame()
	game.audio = engine
	game.buildPlaylistMenu()
	songCount := engine.SongCount()
	for slot := range songCount + 1 {
		minutes := float32(slot*5) + 4.9*boolToFloat(slot == 0)
		game.elapsedSeconds = minutes * 60
		game.rotateSong()
		expected := slot % songCount
		if engine.songIndex != expected {
			t.Fatalf("at %.1f min song %d, want %d", minutes, engine.songIndex, expected)
		}
		for range songCrossfadeSeconds * sampleRate / 4 {
			left, right := engine.renderMusic()
			if math.IsNaN(float64(left)) || math.IsNaN(float64(right)) || abs(left) > 4 {
				t.Fatalf("bad sample during crossfade at %.1f min", minutes)
			}
		}
	}
	if game.musicBanner == "" {
		t.Fatalf("no music banner shown on song change")
	}
}

func TestExportSongPreviews(t *testing.T) {
	outputDirectory := os.Getenv("HORDEFALL_WAV_DIR")
	if outputDirectory == "" {
		t.Skip("HORDEFALL_WAV_DIR not set")
	}
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	engine := &AudioEngine{random: NewRandom(1)}
	engine.attachSongs(newThemeTracker(t).song, uint32(time.Now().UnixNano()))
	for index, song := range engine.songs {
		tracker := NewTracker(song)
		tracker.SetSignals([musicSignalCount]float32{1, 1, maximumSpidersAlive, 1, 1})
		path := filepath.Join(outputDirectory, fmt.Sprintf("%d_%s.wav", index+1, strings.ReplaceAll(song.Title, " ", "_")))
		writePreview(t, tracker, path, 30*sampleRate)
		t.Logf("wrote %s", path)
	}
}

func writePreview(t *testing.T, tracker *Tracker, path string, samples int) {
	pcm := make([]byte, 0, samples*4)
	for range samples {
		left, right := tracker.Render()
		pcm = binary.LittleEndian.AppendUint16(pcm, uint16(int16(softClip(left*0.5)*32000)))
		pcm = binary.LittleEndian.AppendUint16(pcm, uint16(int16(softClip(right*0.5)*32000)))
	}
	header := make([]byte, 0, 44)
	header = append(header, "RIFF"...)
	header = binary.LittleEndian.AppendUint32(header, uint32(36+len(pcm)))
	header = append(header, "WAVEfmt "...)
	header = binary.LittleEndian.AppendUint32(header, 16)
	header = binary.LittleEndian.AppendUint16(header, 1)
	header = binary.LittleEndian.AppendUint16(header, 2)
	header = binary.LittleEndian.AppendUint32(header, sampleRate)
	header = binary.LittleEndian.AppendUint32(header, sampleRate*4)
	header = binary.LittleEndian.AppendUint16(header, 4)
	header = binary.LittleEndian.AppendUint16(header, 16)
	header = append(header, "data"...)
	header = binary.LittleEndian.AppendUint32(header, uint32(len(pcm)))
	if err := os.WriteFile(path, append(header, pcm...), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDemoChangesSongEveryFullCycle(t *testing.T) {
	game := newHeadlessGame()
	engine := &AudioEngine{random: NewRandom(1)}
	engine.attachSongs(newThemeTracker(t).song, 7)
	game.audio = engine
	game.buildPlaylistMenu()
	game.startDemo()
	game.state = StateMainMenu
	cycleSlots := 2 * len(demoSequences)
	steps := []struct{ slot, song int }{{0, 0}, {cycleSlots - 1, 0}, {cycleSlots, 1}, {2 * cycleSlots, 2}}
	for _, step := range steps {
		game.demoSlot = step.slot
		game.rotateSong()
		if engine.songIndex != step.song {
			t.Fatalf("demo slot %d plays song %d, want %d", step.slot, engine.songIndex, step.song)
		}
	}
}

func TestPlaylistTogglesMatchWhatIsHeard(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.audio = &AudioEngine{random: NewRandom(1)}
	game.audio.attachSongs(newThemeTracker(t).song, 5)
	game.loadSettings()
	game.state = StatePlaylist
	game.rotateSong()
	playing := game.audio.CurrentSong()
	other := (playing + 2) % game.audio.SongCount()
	game.toggleSong(other)
	game.rotateSong()
	if game.audio.CurrentSong() != playing {
		t.Fatalf("disabling %q (not playing) switched the music to %q", game.audio.SongTitle(other), game.audio.SongTitle(game.audio.CurrentSong()))
	}
	game.toggleSong(other)
	if game.audio.CurrentSong() != other || game.musicBanner == "" {
		t.Fatalf("enabling %q should play it right away, playing %q", game.audio.SongTitle(other), game.audio.SongTitle(game.audio.CurrentSong()))
	}
	game.toggleSong(other)
	game.rotateSong()
	if game.audio.CurrentSong() == other || !game.IsSongEnabled(game.audio.CurrentSong()) {
		t.Fatalf("disabling the playing track should move to an enabled one, playing %q", game.audio.SongTitle(game.audio.CurrentSong()))
	}
}

func TestPlaylistSkipsDisabledTracksAndPersists(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	newPlaylistGame := func() *Game {
		game := newHeadlessGame()
		game.audio = &AudioEngine{random: NewRandom(1)}
		game.audio.attachSongs(newThemeTracker(t).song, 5)
		game.loadSettings()
		return game
	}
	game := newPlaylistGame()
	songCount := game.audio.SongCount()
	for index := range songCount {
		game.toggleSong(index)
	}
	if game.HasEnabledSong() || !game.audio.IsPlaylistEmpty {
		t.Fatalf("disabling every track should leave an empty, silent playlist: %v", game.settings.Tracks)
	}
	game.toggleSong(0)
	for _, minutes := range []float32{0, 5, 10, 15} {
		game.elapsedSeconds = minutes * 60
		game.rotateSong()
		if game.audio.songIndex != 0 {
			t.Fatalf("at %.0f min a disabled track (%d) played", minutes, game.audio.songIndex)
		}
	}
	restarted := newPlaylistGame()
	if !restarted.IsSongEnabled(0) || restarted.IsSongEnabled(1) || restarted.describeMusic() == "" {
		t.Fatalf("playlist not restored after restart: %v", restarted.settings.Tracks)
	}
	restarted.toggleSong(3)
	if !restarted.IsSongEnabled(3) || restarted.IsSongEnabled(2) {
		t.Fatalf("re-enabling a track failed: %v", restarted.settings.Tracks)
	}
}

func TestEmptyPlaylistSilencesMusicButNotEffects(t *testing.T) {
	engine := &AudioEngine{random: NewRandom(1), EffectsVolume: 1, MusicVolume: 1}
	engine.attachSongs(newThemeTracker(t).song, 3)
	engine.SetPlaylistEmpty(true)
	musicPeak := float32(0)
	for range sampleRate {
		left, right := engine.renderFrame()
		musicPeak = max(musicPeak, abs(left), abs(right))
	}
	engine.startSound(&soundTable[SoundPop])
	effectPeak := float32(0)
	for range sampleRate / 10 {
		left, right := engine.renderFrame()
		effectPeak = max(effectPeak, abs(left), abs(right))
	}
	if musicPeak > 0.0001 || effectPeak < 0.01 {
		t.Fatalf("empty playlist: music peak %.4f (want silence), effects peak %.3f (want audible)", musicPeak, effectPeak)
	}
}

func TestMenusPlayTheFullSong(t *testing.T) {
	game := newHeadlessGame()
	game.startDemo()
	game.enemies.Count = 0
	signals := game.computeMusicSignals()
	theme := newThemeTracker(t).song
	for channel, layer := range theme.Layers {
		isMelodic := layer.Signal == SignalHorde
		if isMelodic && signals[SignalHorde] < layer.Threshold {
			t.Fatalf("channel %d (horde layer %.2f) is muted on the menu, horde signal %.2f", channel+1, layer.Threshold, signals[SignalHorde])
		}
	}
	game.resetRun()
	game.enemies.Count = 0
	if game.computeMusicSignals()[SignalHorde] != 0 {
		t.Fatalf("in a run with no enemies the horde layers should stay silent")
	}
}

func TestTracksMapIsTheSingleSourceOfTruth(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.audio = &AudioEngine{random: NewRandom(1)}
	game.audio.attachSongs(newThemeTracker(t).song, 5)
	game.loadSettings()
	game.settings.Tracks = map[string]bool{"Hordefall Theme": false, "Neon Pursuit": false, "Frozen Wastes": false, "Ember March": true, "Skyline Rush": false, "Grey Transmission": false}
	for range 5 {
		game.musicSlot = noMusicSlot
		game.rotateSong()
		if game.audio.SongTitle(game.audio.CurrentSong()) != "Ember March" {
			t.Fatalf("only Ember March is true, playing %q", game.audio.SongTitle(game.audio.CurrentSong()))
		}
	}
	game.settings.Tracks["Ember March"] = false
	if game.HasEnabledSong() {
		t.Fatalf("all tracks false but a track is still considered enabled")
	}
	before := game.audio.CurrentSong()
	game.rotateSong()
	if game.audio.CurrentSong() != before {
		t.Fatalf("rotation changed song although every track is false")
	}
}

func TestFirstSongIsPickedAmongEnabledTracks(t *testing.T) {
	picked := map[int]bool{}
	for seed := range 40 {
		game := newHeadlessGame()
		game.audio = &AudioEngine{random: NewRandom(1)}
		game.audio.attachSongs(newThemeTracker(t).song, 5)
		game.buildPlaylistMenu()
		game.musicRandom = NewRandom(uint32(seed*7919 + 1))
		game.rotateSong()
		picked[game.audio.CurrentSong()] = true
	}
	if len(picked) < 3 {
		t.Fatalf("first song is not varied across launches: %v", picked)
	}
	t.Logf("first songs picked over 40 launches: %v", picked)
}

func TestGraphicsSettingsDefaultsAndSanitizing(t *testing.T) {
	defaults := DefaultSettings()
	if !defaults.IsVsyncOn || !defaults.IsFPSShown || defaults.IsFullscreen || defaults.ShakeLevel != graphicsLevelCount-1 || defaults.EffectsLevel != graphicsLevelCount-1 || defaults.BloomLevel != defaultBloomLevel || defaults.IsCRTOn {
		t.Fatalf("unexpected graphics defaults: %+v", defaults)
	}
	broken := Settings{Resolution: 9, ShakeLevel: -4, EffectsLevel: 12, BloomLevel: 7}
	sanitizeGraphics(&broken)
	if broken.Resolution != ResolutionNative || broken.ShakeLevel != 0 || broken.EffectsLevel != graphicsLevelCount-1 || broken.BloomLevel != graphicsLevelCount-1 {
		t.Fatalf("out of range values were not clamped: %+v", broken)
	}
}

func TestEffectsDensityAndShakeFollowSettings(t *testing.T) {
	game := newHeadlessGame()
	game.settings = DefaultSettings()
	game.settings.EffectsLevel, game.settings.ShakeLevel = 0, 0
	game.effects.ShakeFactor = shakeLevels[game.settings.ShakeLevel].Factor
	game.effects.Density = effectsLevels[game.settings.EffectsLevel].Factor
	game.effects.SpawnSparks(0, 0, 1000, [3]float32{1, 1, 1}, 100)
	game.effects.AddShake(20)
	game.effects.Update(deltaSeconds)
	if game.effects.ParticleCount < 200 || game.effects.ParticleCount > 400 {
		t.Fatalf("low effects kept %d of 1000 sparks, want about 300", game.effects.ParticleCount)
	}
	if game.effects.ShakeX != 0 || game.effects.ShakeY != 0 {
		t.Fatalf("screen shake off still moved the camera: %.2f %.2f", game.effects.ShakeX, game.effects.ShakeY)
	}
}

func TestMenuListsEndOnQuitAndStayBelowTitle(t *testing.T) {
	const minimumTop = 240
	for _, optionCount := range []int{3, 4, 9, 14} {
		options := make([]MenuOption, optionCount)
		positions := menuOptionPositions(options, menuOptionSpacing, minimumTop, nil)
		last := positions[len(positions)-1]
		if abs(last-menuListBottom) > 0.01 || positions[0] < minimumTop-0.01 {
			t.Fatalf("%d options: first %.1f, last %.1f (want last at %d, first >= %d)", optionCount, positions[0], last, menuListBottom, minimumTop)
		}
	}
	mainPositions := menuOptionPositions(mainMenuOptions, mainMenuOptionSpacing, minimumTop, nil)
	if abs(mainPositions[0]-mainMenuOptionsTop) > 0.01 {
		t.Fatalf("main menu moved: first option at %.1f, want %d", mainPositions[0], mainMenuOptionsTop)
	}
}

func TestMusicVolumeSliderStepsAndClamps(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.settings = DefaultSettings()
	game.adjustMusicVolume(1)
	if game.settings.MusicVolumeStep != musicVolumeSteps {
		t.Fatalf("volume went past 100%%: step %d", game.settings.MusicVolumeStep)
	}
	for range musicVolumeSteps + 2 {
		game.adjustMusicVolume(-1)
	}
	if game.settings.MusicVolumeStep != 0 || game.musicVolumeLabel() != "Music volume: 0%" {
		t.Fatalf("volume did not stop at 0%%: %q", game.musicVolumeLabel())
	}
	game.cycleMusicVolume()
	if game.musicVolumeLabel() != "Music volume: 20%" {
		t.Fatalf("fire should raise the volume by one 20%% step: %q", game.musicVolumeLabel())
	}
	game.settings.MusicVolumeStep = musicVolumeSteps
	game.cycleMusicVolume()
	if game.settings.MusicVolumeStep != 0 {
		t.Fatalf("fire at 100%% should wrap to 0%%, got step %d", game.settings.MusicVolumeStep)
	}
}
