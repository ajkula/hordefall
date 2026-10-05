package game

import (
	"fmt"
	"testing"

	"hordefall/internal/rng"

	"hordefall/internal/audio"
)

// ===== Public API =====

func TestMusicSettingIsSavedAndRestored(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.audio = audio.NewSilentEngine(1)
	game.loadSettings()
	if !game.settings.IsMusicOn || game.musicLabel() != "Music: ON" {
		t.Fatalf("music should default to on, label %q", game.musicLabel())
	}
	game.toggleMusic()
	restarted := newHeadlessGame()
	restarted.audio = audio.NewSilentEngine(1)
	restarted.loadSettings()
	if restarted.settings.IsMusicOn || restarted.audio.IsMusicOn() || restarted.musicLabel() != "Music: OFF" {
		t.Fatalf("music off was not restored after restart, label %q", restarted.musicLabel())
	}
}

func TestSongRotatesEveryFiveMinutes(t *testing.T) {
	engine := audio.NewSilentEngine(99)
	engine.MusicVolume = 1
	game := newHeadlessGame()
	game.audio = engine
	game.buildPlaylistMenu()
	songCount := engine.SongCount()
	for slot := range songCount + 1 {
		minutes := float32(slot*5) + 4.9*boolToFloat(slot == 0)
		game.elapsedSeconds = minutes * 60
		game.rotateSong()
		expected := slot % songCount
		if engine.CurrentSong() != expected {
			t.Fatalf("at %.1f min song %d, want %d", minutes, engine.CurrentSong(), expected)
		}
	}
	if game.musicBanner == "" {
		t.Fatalf("no music banner shown on song change")
	}
}

func TestDemoChangesSongEveryFullCycle(t *testing.T) {
	game := newHeadlessGame()
	engine := audio.NewSilentEngine(7)
	game.audio = engine
	game.buildPlaylistMenu()
	game.startDemo()
	game.state = StateMainMenu
	cycleSlots := 2 * len(demoSequences)
	steps := []struct{ slot, song int }{{0, 0}, {cycleSlots - 1, 0}, {cycleSlots, 1}, {2 * cycleSlots, 2}}
	for _, step := range steps {
		game.demoSlot = step.slot
		game.rotateSong()
		if engine.CurrentSong() != step.song {
			t.Fatalf("demo slot %d plays song %d, want %d", step.slot, engine.CurrentSong(), step.song)
		}
	}
}

func TestPlaylistTogglesMatchWhatIsHeard(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.audio = audio.NewSilentEngine(5)
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
		game.audio = audio.NewSilentEngine(5)
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
		if game.audio.CurrentSong() != 0 {
			t.Fatalf("at %.0f min a disabled track (%d) played", minutes, game.audio.CurrentSong())
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

func TestMenusPlayTheFullSong(t *testing.T) {
	game := newHeadlessGame()
	game.startDemo()
	game.enemies.Count = 0
	signals := game.computeMusicSignals()
	theme, err := audio.ParseTheme()
	if err != nil {
		t.Fatal(err)
	}
	for channel, layer := range theme.Layers {
		isMelodic := layer.Signal == audio.SignalHorde
		if isMelodic && signals[audio.SignalHorde] < layer.Threshold {
			t.Fatalf("channel %d (horde layer %.2f) is muted on the menu, horde signal %.2f", channel+1, layer.Threshold, signals[audio.SignalHorde])
		}
	}
	game.resetRun()
	game.enemies.Count = 0
	if game.computeMusicSignals()[audio.SignalHorde] != 0 {
		t.Fatalf("in a run with no enemies the horde layers should stay silent")
	}
}

func TestTracksMapIsTheSingleSourceOfTruth(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.audio = audio.NewSilentEngine(5)
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
		game.audio = audio.NewSilentEngine(5)
		game.buildPlaylistMenu()
		game.musicRandom = rng.New(uint32(seed*7919 + 1))
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
	expected := Settings{
		IsMusicOn: true, Tracks: map[string]bool{}, Resolution: ResolutionNative,
		ShakeLevel: 1, EffectsLevel: 2, BloomLevel: 1, IsCRTOn: true, MusicVolumeStep: 3, EffectsVolumeStep: 4,
		Initials: defaultInitials,
	}
	if fmt.Sprintf("%+v", defaults) != fmt.Sprintf("%+v", expected) {
		t.Fatalf("unexpected defaults: got %+v, want %+v", defaults, expected)
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
	game.settings.MusicVolumeStep = volumeSteps
	game.adjustMusicVolume(1)
	if game.settings.MusicVolumeStep != volumeSteps {
		t.Fatalf("volume went past 100%%: step %d", game.settings.MusicVolumeStep)
	}
	for range volumeSteps + 2 {
		game.adjustMusicVolume(-1)
	}
	if game.settings.MusicVolumeStep != 0 || game.musicVolumeLabel() != "Music volume: 0%" {
		t.Fatalf("volume did not stop at 0%%: %q", game.musicVolumeLabel())
	}
	game.cycleMusicVolume()
	if game.musicVolumeLabel() != "Music volume: 20%" {
		t.Fatalf("fire should raise the volume by one 20%% step: %q", game.musicVolumeLabel())
	}
	game.settings.MusicVolumeStep = volumeSteps
	game.cycleMusicVolume()
	if game.settings.MusicVolumeStep != 0 {
		t.Fatalf("fire at 100%% should wrap to 0%%, got step %d", game.settings.MusicVolumeStep)
	}
}

func TestEffectsVolumeSliderScalesSoundEffects(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.settings = DefaultSettings()
	game.settings.EffectsVolumeStep = volumeSteps
	game.audio = audio.NewSilentEngine(1)
	game.isDemo = false
	game.updateAudio()
	fullVolume := game.audio.EffectsVolume
	game.adjustEffectsVolume(-2)
	game.updateAudio()
	if game.effectsVolumeLabel() != "Effects volume: 60%" || abs(game.audio.EffectsVolume-fullVolume*0.6) > 0.001 {
		t.Fatalf("60%% step: label %q, volume %.3f (full %.3f)", game.effectsVolumeLabel(), game.audio.EffectsVolume, fullVolume)
	}
	game.adjustEffectsVolume(-10)
	game.updateAudio()
	if game.audio.EffectsVolume != 0 || game.settings.MusicVolumeStep != defaultMusicVolumeStep {
		t.Fatalf("0%% effects should be silent without touching music: effects %.3f, music step %d", game.audio.EffectsVolume, game.settings.MusicVolumeStep)
	}
}

func TestAimLockGoesBackInMenus(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.settings = DefaultSettings()
	steps := []struct {
		from GameState
		to   GameState
	}{
		{StateAudio, StateOptions},
		{StateGraphics, StateOptions},
		{StatePlaylist, StateAudio},
		{StateOptions, StateMainMenu},
		{StatePaused, StatePlaying},
	}
	for _, step := range steps {
		game.switchState(step.from)
		game.menuLockSeconds = 0
		game.controls = Controls{JustPressed: ActionAimLock}
		stateHandlers[step.from].Update(game)
		if game.state != step.to {
			t.Fatalf("aim lock in state %d led to %d, want %d", step.from, game.state, step.to)
		}
	}
}

func TestOptionsFromPauseKeepTheRunFrozen(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.state = StatePaused
	game.openOptionsFromPause()
	frame := game.frame
	for range 120 {
		game.menuLockSeconds = 0
		game.controls = Controls{}
		game.updateOptions()
	}
	if game.frame != frame || game.isDemo {
		t.Fatalf("options advanced the paused run: frame %d -> %d", frame, game.frame)
	}
	game.controls = Controls{JustPressed: ActionPause}
	game.updateOptions()
	if game.state != StatePaused {
		t.Fatalf("back from options went to state %d, want pause", game.state)
	}
}
