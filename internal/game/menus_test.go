package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/input"
	"github.com/ajkula/ajkurig/rig"
	"github.com/ajkula/ajkurig/rng"
	"github.com/ajkula/ajkurig/scores"
	"github.com/ajkula/ajkurig/shell"
	"github.com/ajkula/ajkurig/ui"
)

// ===== Constants =====

const (
	pauseOptionsOption = 1
	pauseMusicOption   = 2
	audioMusicVolume   = 1
	audioEffectsVolume = 2
)

// ===== Public API =====

func TestMusicSettingIsSavedAndRestored(t *testing.T) {
	directory := t.TempDir()
	app, _ := newTestAppIn(t, directory)
	music := menuOption(app, shell.ScreenPause, pauseMusicOption)
	if !app.Settings.IsMusicOn || music.Label() != "Music: ON" {
		t.Fatalf("music should default to on, label %q", music.Label())
	}
	music.Activate()
	restarted, _ := newTestAppIn(t, directory)
	label := menuOption(restarted, shell.ScreenPause, pauseMusicOption).Label()
	if restarted.Settings.IsMusicOn || restarted.Audio.IsMusicOn() || label != "Music: OFF" {
		t.Fatalf("music off was not restored after restart, label %q", label)
	}
}

func TestSongRotatesEveryFiveMinutes(t *testing.T) {
	app, _ := newTestApp(t)
	app.Jukebox.Random = rng.Random{}
	songCount := app.Audio.SongCount()
	for slot := range songCount + 1 {
		minutes := float32(slot*5) + 4.9*boolToFloat(slot == 0)
		app.Jukebox.Update(app.Jukebox.RunSlot(minutes*60), deltaSeconds)
		expected := slot % songCount
		if app.Audio.CurrentSong() != expected {
			t.Fatalf("at %.1f min song %d, want %d", minutes, app.Audio.CurrentSong(), expected)
		}
	}
	if app.Jukebox.Banner.Message == "" {
		t.Fatalf("no music banner shown on song change")
	}
}

func TestDemoChangesSongEveryFullCycle(t *testing.T) {
	app, game := newHomeApp(t)
	app.Jukebox = rig.NewJukebox(app)
	app.Jukebox.Random = rng.Random{}
	cycleSlots := 2 * len(demoSequences)
	steps := []struct{ slot, song int }{{0, 0}, {cycleSlots - 1, 0}, {cycleSlots, 1}, {2 * cycleSlots, 2}}
	for _, step := range steps {
		game.shell.DemoSlot = step.slot
		game.shell.UpdateBackdrop(app.Frame())
		if app.Audio.CurrentSong() != step.song {
			t.Fatalf("demo slot %d plays song %d, want %d", step.slot, app.Audio.CurrentSong(), step.song)
		}
	}
}

func TestPlaylistTogglesMatchWhatIsHeard(t *testing.T) {
	app, _ := newTestApp(t)
	jukebox := app.Jukebox
	jukebox.Update(0, deltaSeconds)
	playing := app.Audio.CurrentSong()
	other := (playing + 2) % app.Audio.SongCount()
	jukebox.ToggleSong(other)
	jukebox.Update(0, deltaSeconds)
	if app.Audio.CurrentSong() != playing {
		t.Fatalf("disabling %q (not playing) switched the music to %q", app.Audio.SongTitle(other), app.Audio.SongTitle(app.Audio.CurrentSong()))
	}
	jukebox.Banner.Message = ""
	jukebox.ToggleSong(other)
	if app.Audio.CurrentSong() != other || jukebox.Banner.Message == "" {
		t.Fatalf("enabling %q should play it right away, playing %q", app.Audio.SongTitle(other), app.Audio.SongTitle(app.Audio.CurrentSong()))
	}
	jukebox.ToggleSong(other)
	jukebox.Update(0, deltaSeconds)
	if app.Audio.CurrentSong() == other || !jukebox.IsSongEnabled(app.Audio.CurrentSong()) {
		t.Fatalf("disabling the playing track should move to an enabled one, playing %q", app.Audio.SongTitle(app.Audio.CurrentSong()))
	}
}

func TestPlaylistSkipsDisabledTracksAndPersists(t *testing.T) {
	directory := t.TempDir()
	app, _ := newTestAppIn(t, directory)
	for index := range app.Audio.SongCount() {
		app.Jukebox.ToggleSong(index)
	}
	if app.Jukebox.HasEnabledSong() || !app.Audio.IsPlaylistEmpty() {
		t.Fatalf("disabling every track should leave an empty, silent playlist: %v", app.Settings.Tracks)
	}
	app.Jukebox.ToggleSong(0)
	for _, minutes := range []float32{0, 5, 10, 15} {
		app.Jukebox.Update(app.Jukebox.RunSlot(minutes*60), deltaSeconds)
		if app.Audio.CurrentSong() != 0 {
			t.Fatalf("at %.0f min a disabled track (%d) played", minutes, app.Audio.CurrentSong())
		}
	}
	restarted, _ := newTestAppIn(t, directory)
	description := strings.Join(restarted.Bench.DebugLines(), "\n")
	if !restarted.Jukebox.IsSongEnabled(0) || restarted.Jukebox.IsSongEnabled(1) || !strings.Contains(description, "enabled: Hordefall Theme") {
		t.Fatalf("playlist not restored after restart: %v, %q", restarted.Settings.Tracks, description)
	}
	restarted.Jukebox.ToggleSong(3)
	if !restarted.Jukebox.IsSongEnabled(3) || restarted.Jukebox.IsSongEnabled(2) {
		t.Fatalf("re-enabling a track failed: %v", restarted.Settings.Tracks)
	}
}

func TestDemoMusicFollowsTheActionLikeARun(t *testing.T) {
	_, game := newHomeApp(t)
	game.enemies.Count = 0
	if quiet := game.computeMusicSignals()[SignalHorde]; quiet != 0 {
		t.Fatalf("an empty demo still plays the horde layers, horde signal %.2f", quiet)
	}
	for range hordeSignalFullCount / 2 {
		game.enemies.Spawn(EnemySwarmer, game.player.X, game.player.Y, 1)
	}
	if half := game.computeMusicSignals()[SignalHorde]; half != 0.5 {
		t.Fatalf("the demo horde signal does not follow nearby enemies: %.2f", half)
	}
	game.resetRun()
	game.enemies.Count = 0
	if game.computeMusicSignals()[SignalHorde] != 0 {
		t.Fatalf("in a run with no enemies the horde layers should stay silent")
	}
}

func TestTracksMapIsTheSingleSourceOfTruth(t *testing.T) {
	app, _ := newTestApp(t)
	app.Settings.Tracks = map[string]bool{"Hordefall Theme": false, "Neon Pursuit": false, "Frozen Wastes": false, "Ember March": true, "Skyline Rush": false, "Grey Transmission": false}
	for range 5 {
		app.Jukebox = rig.NewJukebox(app)
		app.Jukebox.Update(0, deltaSeconds)
		if app.Audio.SongTitle(app.Audio.CurrentSong()) != "Ember March" {
			t.Fatalf("only Ember March is true, playing %q", app.Audio.SongTitle(app.Audio.CurrentSong()))
		}
	}
	app.Settings.Tracks["Ember March"] = false
	if app.Jukebox.HasEnabledSong() {
		t.Fatalf("all tracks false but a track is still considered enabled")
	}
	before := app.Audio.CurrentSong()
	app.Jukebox.Update(1, deltaSeconds)
	if app.Audio.CurrentSong() != before {
		t.Fatalf("rotation changed song although every track is false")
	}
}

func TestFirstSongIsPickedAmongEnabledTracks(t *testing.T) {
	app, _ := newTestApp(t)
	picked := map[int]bool{}
	for seed := range 40 {
		app.Jukebox = rig.NewJukebox(app)
		app.Jukebox.Random = rng.New(uint32(seed*7919 + 1))
		app.Jukebox.Update(0, deltaSeconds)
		picked[app.Audio.CurrentSong()] = true
	}
	if len(picked) < 3 {
		t.Fatalf("first song is not varied across launches: %v", picked)
	}
	t.Logf("first songs picked over 40 launches: %v", picked)
}

func TestGraphicsSettingsDefaultsAndSanitizing(t *testing.T) {
	defaults := rig.DefaultSettings()
	expected := rig.Settings{
		IsMusicOn: true, Tracks: map[string]bool{}, Resolution: rig.ResolutionNative,
		ShakeLevel: 1, EffectsLevel: 2, BloomLevel: 1, IsCRTOn: true, MusicVolumeStep: 3, EffectsVolumeStep: 4,
		Initials: scores.DefaultInitials, Language: i18n.DefaultCode, Custom: map[string]string{},
	}
	if fmt.Sprintf("%+v", defaults) != fmt.Sprintf("%+v", expected) {
		t.Fatalf("unexpected defaults: got %+v, want %+v", defaults, expected)
	}
	broken := rig.Settings{Resolution: 9, ShakeLevel: -4, EffectsLevel: 12, BloomLevel: 7}
	broken.Sanitize()
	if broken.Resolution != rig.ResolutionNative || broken.ShakeLevel != 0 || broken.EffectsLevel != rig.LevelCount-1 || broken.BloomLevel != rig.LevelCount-1 {
		t.Fatalf("out of range values were not clamped: %+v", broken)
	}
}

func TestEffectsDensityAndShakeFollowSettings(t *testing.T) {
	game := newHeadlessGame(t)
	game.app.Settings.EffectsLevel, game.app.Settings.ShakeLevel = 0, 0
	game.applyGraphicsSettings()
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

func TestMusicVolumeSliderStepsAndClamps(t *testing.T) {
	app, _ := newTestApp(t)
	volume := menuOption(app, shell.ScreenAudio, audioMusicVolume)
	app.Settings.MusicVolumeStep = rig.VolumeSteps
	volume.Slider.Adjust(1)
	if app.Settings.MusicVolumeStep != rig.VolumeSteps {
		t.Fatalf("volume went past 100%%: step %d", app.Settings.MusicVolumeStep)
	}
	for range rig.VolumeSteps + 2 {
		volume.Slider.Adjust(-1)
	}
	if app.Settings.MusicVolumeStep != 0 || volume.Label() != "Music volume: 0%" {
		t.Fatalf("volume did not stop at 0%%: %q", volume.Label())
	}
	volume.Activate()
	if volume.Label() != "Music volume: 20%" {
		t.Fatalf("fire should raise the volume by one 20%% step: %q", volume.Label())
	}
	app.Settings.MusicVolumeStep = rig.VolumeSteps
	volume.Activate()
	if app.Settings.MusicVolumeStep != 0 {
		t.Fatalf("fire at 100%% should wrap to 0%%, got step %d", app.Settings.MusicVolumeStep)
	}
}

func TestEffectsVolumeSliderScalesSoundEffects(t *testing.T) {
	app, _ := newRunningApp(t)
	effects := menuOption(app, shell.ScreenAudio, audioEffectsVolume)
	app.Settings.EffectsVolumeStep = rig.VolumeSteps
	stepFor(app, 0.1, 0)
	fullVolume := app.Audio.EffectsVolume()
	effects.Slider.Adjust(-2)
	stepFor(app, 0.1, 0)
	if effects.Label() != "Effects volume: 60%" || abs(app.Audio.EffectsVolume()-fullVolume*0.6) > 0.001 {
		t.Fatalf("60%% step: label %q, volume %.3f (full %.3f)", effects.Label(), app.Audio.EffectsVolume(), fullVolume)
	}
	effects.Slider.Adjust(-10)
	stepFor(app, 0.1, 0)
	if app.Audio.EffectsVolume() != 0 || app.Settings.MusicVolumeStep != rig.DefaultSettings().MusicVolumeStep {
		t.Fatalf("0%% effects should be silent without touching music: effects %.3f, music step %d", app.Audio.EffectsVolume(), app.Settings.MusicVolumeStep)
	}
}

func TestAimLockGoesBackInMenus(t *testing.T) {
	aimLock := remaps[1]
	if aimLock.Primary != ActionAimLock || aimLock.KeyboardActions&input.ActionBack == 0 || aimLock.GamepadActions&input.ActionBack == 0 {
		t.Fatalf("aim lock does not also mean back: %+v", aimLock)
	}
	app, _ := newRunningApp(t)
	steps := []struct{ from, to string }{
		{shell.ScreenAudio, shell.ScreenOptions},
		{shell.ScreenGraphics, shell.ScreenOptions},
		{shell.ScreenPlaylist, shell.ScreenAudio},
		{shell.ScreenOptions, shell.ScreenMenu},
		{shell.ScreenPause, shell.ScreenPlay},
	}
	for _, step := range steps {
		app.Switch(step.from)
		waitForMenuLock(app)
		press(app, ActionAimLock|input.ActionBack)
		if app.CurrentScreen() != step.to {
			t.Fatalf("aim lock on %q led to %q, want %q", step.from, app.CurrentScreen(), step.to)
		}
	}
}

func TestOptionsFromPauseKeepTheRunFrozen(t *testing.T) {
	app, game := newRunningApp(t)
	press(app, input.ActionPause|input.ActionBack)
	menuOption(app, shell.ScreenPause, pauseOptionsOption).Activate()
	frame := game.frame
	stepFor(app, 2, 0)
	if app.CurrentScreen() != shell.ScreenOptions || game.frame != frame || game.isDemo {
		t.Fatalf("options advanced the paused run: screen %q, frame %d -> %d", app.CurrentScreen(), frame, game.frame)
	}
	press(app, input.ActionPause|input.ActionBack)
	if app.CurrentScreen() != shell.ScreenPause {
		t.Fatalf("back from options went to %q, want pause", app.CurrentScreen())
	}
}

func TestSelectTogglesTheBenchmarkPanelOnTheTitle(t *testing.T) {
	app, _ := newHomeApp(t)
	press(app, input.ActionSelect)
	if app.CurrentScreen() != shell.ScreenMenu || !app.Settings.IsBenchmarkHidden {
		t.Fatalf("select opened %q instead of hiding the benchmark", app.CurrentScreen())
	}
	press(app, input.ActionSelect)
	if app.Settings.IsBenchmarkHidden {
		t.Fatal("select did not show the benchmark again")
	}
}

func TestIdleGameOverShowsTheRankingThenTheTitle(t *testing.T) {
	app, game := newRunningApp(t)
	game.killScore = 4000
	game.shell.RecordScoreAs("GRG")
	app.Switch(shell.ScreenGameOver)
	for range gameOverIdleSeconds*ticksPerSecond - 1 {
		app.Step(input.Controls{RawJustPressed: input.NoInput})
	}
	if app.CurrentScreen() != shell.ScreenGameOver {
		t.Fatal("game over left before the idle delay")
	}
	app.Step(input.Controls{RawJustPressed: input.NoInput})
	if app.CurrentScreen() != shell.ScreenMenu || game.shell.BoardSeconds <= 0 || game.shell.NewEntryRank != 0 {
		t.Fatalf("screen %q, board %v, rank %d", app.CurrentScreen(), game.shell.BoardSeconds, game.shell.NewEntryRank)
	}
}

// ===== Internal =====

func menuOption(app *rig.App, screen string, index int) *ui.Option {
	menu := app.Screen(screen).(*shell.MenuScreen)
	return &menu.Menu.Options[index]
}
