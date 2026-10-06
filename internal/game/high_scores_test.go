package game

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const boardColumnMinimumGap = 48

func TestHighScoreTableKeepsTheTopTenInOrder(t *testing.T) {
	var table HighScoreTable
	for score := 1; score <= 15; score++ {
		table.Insert(HighScoreEntry{Initials: "AAA", Score: score * 100})
	}
	if len(table.Entries) != highScoreTableSize || table.Best().Score != 1500 || table.Entries[highScoreTableSize-1].Score != 600 {
		t.Fatalf("unexpected table %+v", table.Entries)
	}
	if table.RankOf(550) != noRank || table.RankOf(1200) != 3 || table.RankOf(0) != noRank {
		t.Fatalf("ranks %d %d %d", table.RankOf(550), table.RankOf(1200), table.RankOf(0))
	}
}

func TestDeathWithRankedScoreAsksForInitials(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.killScore = 5000
	game.finishRun()
	if game.state != StateNameEntry {
		t.Fatalf("state %d, want name entry", game.state)
	}
	game.menuLockSeconds = 0
	game.controls = Controls{JustPressed: ActionDown}
	game.updateNameEntry()
	for range initialsLength {
		game.controls = Controls{JustPressed: ActionConfirm}
		game.updateNameEntry()
	}
	best := game.highScores.Best()
	if game.state != StateGameOver || game.newEntryRank != 0 || best.Initials != ".AA" || best.Score != 5000 || game.settings.Initials != ".AA" {
		t.Fatalf("state %d, best %+v, initials %q", game.state, best, game.settings.Initials)
	}
	game.recordHighScore()
	if len(game.highScores.Entries) != 1 {
		t.Fatalf("score recorded twice: %+v", game.highScores.Entries)
	}
}

func TestHoldingUpScrollsLettersAfterTheDelay(t *testing.T) {
	game := newHeadlessGame()
	game.killScore = 5000
	game.finishRun()
	game.menuLockSeconds = 0
	game.controls = Controls{Held: ActionUp, JustPressed: ActionUp}
	game.updateNameEntry()
	game.controls = Controls{Held: ActionUp}
	for range letterRepeatDelay*ticksPerSecond - 2 {
		game.updateNameEntry()
	}
	if game.nameEntryLetters[0] != 'B' {
		t.Fatalf("letter %c before the delay, want B", game.nameEntryLetters[0])
	}
	for range ticksPerSecond {
		game.updateNameEntry()
	}
	if game.nameEntryLetters[0] < 'L' {
		t.Fatalf("letter %c one second after the delay, repeat too slow", game.nameEntryLetters[0])
	}
}

func TestResetHighScoresNeedsConfirmation(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.highScores.Insert(HighScoreEntry{Initials: "ABC", Score: 900})
	game.openResetScores()
	game.menuLockSeconds = 0
	game.controls = Controls{JustPressed: ActionConfirm}
	game.updateResetScores()
	if len(game.highScores.Entries) != 1 || game.state != StateGameplay {
		t.Fatalf("default choice erased the scores")
	}
	game.openResetScores()
	game.menuLockSeconds = 0
	game.controls = Controls{JustPressed: ActionDown}
	game.updateResetScores()
	game.controls = Controls{JustPressed: ActionConfirm}
	game.updateResetScores()
	if len(game.highScores.Entries) != 0 {
		t.Fatalf("confirmed reset kept %d scores", len(game.highScores.Entries))
	}
}

func TestRankingBoardPausesTheDemo(t *testing.T) {
	game := newHeadlessGame()
	game.startDemo()
	game.highScoreBoardSeconds = attractBoardSeconds
	frame, demoSeconds := game.frame, game.demoSeconds
	for range 60 {
		game.updateDemo()
	}
	if game.frame != frame || game.demoSeconds != demoSeconds {
		t.Fatal("the demo kept running behind the ranking")
	}
	game.highScoreBoardSeconds = deltaSeconds
	game.updateDemo()
	game.updateDemo()
	if game.frame == frame {
		t.Fatal("the demo did not resume after the ranking")
	}
}

func TestDemoShowsHighScoresEveryFewClips(t *testing.T) {
	game := newHeadlessGame()
	game.startDemo()
	for clip := 1; clip < demoClipsPerBoard; clip++ {
		game.demoSeconds = demoSequenceSeconds + 1
		game.startNextDemoSequenceIfDone()
		if game.highScoreBoardSeconds > 0 {
			t.Fatalf("board shown after clip %d", clip)
		}
	}
	game.demoSeconds = demoSequenceSeconds + 1
	game.startNextDemoSequenceIfDone()
	if game.highScoreBoardSeconds != attractBoardSeconds {
		t.Fatalf("board %v", game.highScoreBoardSeconds)
	}
}

func TestButtonLabelsFollowTheLastDevice(t *testing.T) {
	var reader InputReader
	reader.UseBindings(DefaultControlBindings())
	reader.lastDevice = pickDevice(DeviceGamepad, true, false)
	if reader.ButtonLabel(ActionStart) != "ENTER" || reader.ButtonLabel(ActionSelect) != "BACKSPACE" {
		t.Fatalf("keyboard labels %q %q", reader.ButtonLabel(ActionStart), reader.ButtonLabel(ActionSelect))
	}
	bindings := DefaultControlBindings()
	bindings.Buttons[RemapFire], bindings.Buttons[RemapAimLock], bindings.Buttons[RemapDash] = 2, 3, 0
	reader.UseBindings(bindings)
	reader.lastDevice = pickDevice(DeviceKeyboard, false, true)
	if reader.ButtonLabel(ActionStart) != "START" || reader.ButtonLabel(ActionFire) != "BUTTON 2" {
		t.Fatalf("gamepad labels %q %q", reader.ButtonLabel(ActionStart), reader.ButtonLabel(ActionFire))
	}
	if pickDevice(DeviceGamepad, false, false) != DeviceGamepad {
		t.Fatal("idle input changed the device")
	}
}

func TestBoardColumnsFitTheLongestRow(t *testing.T) {
	longest := [4]string{rankOrdinals[highScoreTableSize-1], formatThousands(maximumScore), formatClock(99*60 + 59), "WWW"}
	previousRight := float32(0)
	for index, column := range highScoreColumns {
		width := PixelTextWidth(longest[index], rankingRowPixel)
		left := column.X - width*boolToFloat(column.Align == text.AlignEnd)
		if left-previousRight < boardColumnMinimumGap {
			t.Fatalf("column %s starts %.0f px after the previous one", column.Header, left-previousRight)
		}
		previousRight = left + width
	}
	if previousRight > screenWidth {
		t.Fatalf("the board overflows the screen at %.0f px", previousRight)
	}
}

func TestScoresStopAtTheCapAndTheNewestTieRanksFirst(t *testing.T) {
	game := newHeadlessGame()
	game.killScore = math.MaxInt32 * 4
	if game.CurrentScore() != maximumScore {
		t.Fatalf("score %d is not capped", game.CurrentScore())
	}
	var table HighScoreTable
	table.Insert(HighScoreEntry{Initials: "OLD", Score: maximumScore})
	if rank := table.Insert(HighScoreEntry{Initials: "NEW", Score: maximumScore}); rank != 0 || table.Best().Initials != "NEW" {
		t.Fatalf("the newest capped score ranks %d, best is %s", rank, table.Best().Initials)
	}
}
