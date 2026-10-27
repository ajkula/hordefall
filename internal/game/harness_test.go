package game

import (
	"io/fs"
	"slices"
	"testing"

	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/input"
	"github.com/ajkula/ajkurig/rig"
	"github.com/ajkula/ajkurig/rng"
)

// ===== Constants =====

var firstHordeSlot = slices.Index(bossEventCycle, BossHorde)

// ===== Internal =====

func loadGameLanguages() error {
	languages, err := fs.Sub(languageFiles, "lang")
	if err != nil {
		return err
	}
	return i18n.Load([]fs.FS{languages})
}

func newTestApp(tb testing.TB) (*rig.App, *Game) {
	return newTestAppIn(tb, tb.TempDir())
}

func newTestAppIn(tb testing.TB, directory string) (*rig.App, *Game) {
	tb.Setenv("APPDATA", directory)
	tb.Setenv("XDG_CONFIG_HOME", directory)
	tb.Setenv("HOME", directory)
	app, game, err := NewApp(true)
	if err != nil {
		tb.Fatal(err)
	}
	return app, game
}

func newHeadlessGame(tb testing.TB) *Game {
	_, game := newTestApp(tb)
	game.runCount, game.random = 0, rng.New(gameRandomSeed)
	game.resetRun()
	return game
}

func simulateWorld(game *Game) {
	game.app.SimulateWorld(&rig.Frame{App: game.app, Delta: deltaSeconds, Controls: game.controls})
}

func newRunningApp(tb testing.TB) (*rig.App, *Game) {
	app, game := newTestApp(tb)
	app.Settings.Language = "en"
	game.shell.StartRun()
	return app, game
}

func newHomeApp(tb testing.TB) (*rig.App, *Game) {
	app, game := newTestApp(tb)
	app.Settings.Language = "en"
	game.shell.GoHome()
	return app, game
}

func pressButton(app *rig.App, held input.Action, button int) {
	app.Step(input.Controls{Held: held, RawJustPressed: button})
	waitForMenuLock(app)
}

func stepFor(app *rig.App, seconds float32, held input.Action) {
	for range int(seconds * ticksPerSecond) {
		app.Step(input.Controls{Held: held, RawJustPressed: input.NoInput})
	}
}

func press(app *rig.App, action input.Action) {
	app.Step(input.Controls{Held: action, JustPressed: action, RawJustPressed: input.NoInput})
	waitForMenuLock(app)
}

func waitForMenuLock(app *rig.App) {
	stepFor(app, rig.MenuLockSeconds+0.05, 0)
}
