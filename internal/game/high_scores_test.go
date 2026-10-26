package game

import (
	"math"
	"testing"

	"github.com/ajkula/ajkurig/input"
	"github.com/ajkula/ajkurig/rig"
	"github.com/ajkula/ajkurig/scores"
	"github.com/ajkula/ajkurig/shell"
)

// ===== Public API =====

func TestHighScoreTableKeepsTheTopTenInOrder(t *testing.T) {
	table := scores.NewTable(scores.DefaultTableSize)
	for score := 1; score <= 15; score++ {
		table.Insert(scores.Entry{Initials: "AAA", Score: score * 100})
	}
	if len(table.Entries) != scores.DefaultTableSize || table.Best().Score != 1500 || table.Entries[scores.DefaultTableSize-1].Score != 600 {
		t.Fatalf("unexpected table %+v", table.Entries)
	}
	if table.RankOf(550) != scores.NoRank || table.RankOf(1200) != 3 || table.RankOf(0) != scores.NoRank {
		t.Fatalf("ranks %d %d %d", table.RankOf(550), table.RankOf(1200), table.RankOf(0))
	}
}

func TestDeathWithRankedScoreAsksForInitials(t *testing.T) {
	app, game := newRunningApp(t)
	game.killScore = 5000
	dieAndWait(app, game)
	if app.CurrentScreen() != shell.ScreenNameEntry {
		t.Fatalf("screen %q, want name entry", app.CurrentScreen())
	}
	press(app, input.ActionDown)
	for range rig.InitialsLength {
		press(app, input.ActionConfirm)
	}
	best := app.Scores.Best()
	if app.CurrentScreen() != shell.ScreenGameOver || game.shell.NewEntryRank != 0 || best.Initials != ".AA" || best.Score != 5000 || app.Settings.Initials != ".AA" {
		t.Fatalf("screen %q, best %+v, initials %q", app.CurrentScreen(), best, app.Settings.Initials)
	}
	game.shell.RecordScore()
	if len(app.Scores.Entries) != 1 {
		t.Fatalf("score recorded twice: %+v", app.Scores.Entries)
	}
}

func TestEveryDeathAsksForInitialsStartingAtAAA(t *testing.T) {
	app, game := newRunningApp(t)
	app.Settings.Initials = "XYZ"
	for range scores.DefaultTableSize {
		app.Scores.Insert(scores.Entry{Initials: "TOP", Score: 90000})
	}
	game.killScore = 100
	dieAndWait(app, game)
	initials := nameEntry(app).Entry.Initials()
	if app.CurrentScreen() != shell.ScreenNameEntry || initials != scores.DefaultInitials || game.shell.NewEntryRank != scores.NoRank {
		t.Fatalf("screen %q, letters %q, rank %d", app.CurrentScreen(), initials, game.shell.NewEntryRank)
	}
	for range rig.InitialsLength {
		press(app, input.ActionConfirm)
	}
	if app.CurrentScreen() != shell.ScreenGameOver || app.Scores.Best().Initials != "TOP" {
		t.Fatalf("an unranked score reached the table: screen %q, best %+v", app.CurrentScreen(), app.Scores.Best())
	}
}

func TestHoldingUpScrollsLettersAfterTheDelay(t *testing.T) {
	app, game := newRunningApp(t)
	game.killScore = 5000
	dieAndWait(app, game)
	letters := nameEntry(app).Entry.Letters
	app.Step(input.Controls{Held: input.ActionUp, JustPressed: input.ActionUp, RawJustPressed: input.NoInput})
	stepFor(app, scores.LetterRepeatDelay-2*deltaSeconds, input.ActionUp)
	if letters[0] != 'B' {
		t.Fatalf("letter %c before the delay, want B", letters[0])
	}
	stepFor(app, 1, input.ActionUp)
	if letters[0] < 'L' {
		t.Fatalf("letter %c one second after the delay, repeat too slow", letters[0])
	}
}

func TestResetHighScoresNeedsConfirmation(t *testing.T) {
	app, _ := newHomeApp(t)
	app.Scores.Insert(scores.Entry{Initials: "ABC", Score: 900})
	app.Switch(shell.ScreenResetScores)
	waitForMenuLock(app)
	press(app, input.ActionConfirm)
	if len(app.Scores.Entries) != 1 || app.CurrentScreen() != shell.ScreenGameplay {
		t.Fatalf("default choice erased the scores")
	}
	app.Switch(shell.ScreenResetScores)
	waitForMenuLock(app)
	press(app, input.ActionDown)
	press(app, input.ActionConfirm)
	if len(app.Scores.Entries) != 0 {
		t.Fatalf("confirmed reset kept %d scores", len(app.Scores.Entries))
	}
}

func TestRankingBoardPausesTheDemo(t *testing.T) {
	app, game := newHomeApp(t)
	game.shell.BoardSeconds = shell.AttractBoardSeconds
	frame, demoSeconds := game.frame, game.shell.DemoSeconds
	for range 60 {
		game.shell.UpdateBackdrop(app.Frame())
	}
	if game.frame != frame || game.shell.DemoSeconds != demoSeconds {
		t.Fatal("the demo kept running behind the ranking")
	}
	game.shell.BoardSeconds = deltaSeconds
	game.shell.UpdateBackdrop(app.Frame())
	game.shell.UpdateBackdrop(app.Frame())
	if game.frame == frame {
		t.Fatal("the demo did not resume after the ranking")
	}
}

func TestDemoShowsHighScoresEveryFewClips(t *testing.T) {
	app, game := newHomeApp(t)
	for clip := 1; clip < game.shell.Options.ClipsPerBoard; clip++ {
		game.shell.DemoSeconds = demoSequenceSeconds + 1
		game.shell.UpdateBackdrop(app.Frame())
		if game.shell.BoardSeconds > 0 {
			t.Fatalf("board shown after clip %d", clip)
		}
	}
	game.shell.DemoSeconds = demoSequenceSeconds + 1
	game.shell.UpdateBackdrop(app.Frame())
	if game.shell.BoardSeconds != shell.AttractBoardSeconds {
		t.Fatalf("board %v", game.shell.BoardSeconds)
	}
}

func TestButtonLabelsFollowTheRemappedButtons(t *testing.T) {
	layout := input.NewLayout(remaps, len(remaps))
	bindings := input.DefaultBindings(layout)
	if bindings.Label(layout, input.DeviceKeyboard, 0) != "J" || bindings.Label(layout, input.DeviceGamepad, 0) != "A" {
		t.Fatalf("default fire labels %q %q", bindings.Label(layout, input.DeviceKeyboard, 0), bindings.Label(layout, input.DeviceGamepad, 0))
	}
	*bindings.Binding(input.DeviceGamepad, 0), *bindings.Binding(input.DeviceGamepad, 1), *bindings.Binding(input.DeviceGamepad, 2) = 2, 3, 0
	if bindings.Label(layout, input.DeviceGamepad, 0) != "BUTTON 2" {
		t.Fatalf("gamepad fire label %q", bindings.Label(layout, input.DeviceGamepad, 0))
	}
}

func TestScoresStopAtTheCapAndTheNewestTieRanksFirst(t *testing.T) {
	game := newHeadlessGame(t)
	game.killScore = math.MaxInt32 * 4
	if game.CurrentScore() != scores.MaximumScore {
		t.Fatalf("score %d is not capped", game.CurrentScore())
	}
	table := scores.NewTable(scores.DefaultTableSize)
	table.Insert(scores.Entry{Initials: "OLD", Score: scores.MaximumScore})
	if rank := table.Insert(scores.Entry{Initials: "NEW", Score: scores.MaximumScore}); rank != 0 || table.Best().Initials != "NEW" {
		t.Fatalf("the newest capped score ranks %d, best is %s", rank, table.Best().Initials)
	}
}

// ===== Internal =====

func dieAndWait(app *rig.App, game *Game) {
	game.player.Health = 0
	app.Step(input.Controls{RawJustPressed: input.NoInput})
	waitForMenuLock(app)
}

func nameEntry(app *rig.App) *shell.NameEntryScreen {
	return app.Screen(shell.ScreenNameEntry).(*shell.NameEntryScreen)
}
