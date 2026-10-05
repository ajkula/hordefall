package game

import "testing"

func TestHighScoreTableKeepsTheTopTenInOrder(t *testing.T) {
	var table HighScoreTable
	for score := 1; score <= 15; score++ {
		table.Insert(HighScoreEntry{Initials: "AAA", Score: score * 100})
	}
	if len(table.Entries) != highScoreTableSize || table.Best().Score != 1500 || table.Entries[highScoreTableSize-1].Score != 600 {
		t.Fatalf("unexpected table %+v", table.Entries)
	}
	if table.RankOf(550) != noRank || table.RankOf(1200) != 4 || table.RankOf(0) != noRank {
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

func TestDemoShowsHighScoresEveryFewClips(t *testing.T) {
	game := newHeadlessGame()
	game.startDemo()
	for clip := 1; clip < demoClipsPerBoard; clip++ {
		game.demoSeconds = game.demoClipSeconds + 1
		game.startNextDemoSequenceIfDone()
		if game.highScoreBoardSeconds > 0 {
			t.Fatalf("board shown after clip %d", clip)
		}
	}
	game.demoSeconds = game.demoClipSeconds + 1
	game.startNextDemoSequenceIfDone()
	if game.highScoreBoardSeconds != highScoreBoardSeconds || game.demoClipSeconds != demoSequenceSeconds+highScoreBoardSeconds {
		t.Fatalf("board %v, clip %v", game.highScoreBoardSeconds, game.demoClipSeconds)
	}
}

func TestButtonLabelsFollowTheLastDevice(t *testing.T) {
	var reader InputReader
	reader.lastDevice = pickDevice(DeviceGamepad, true, false)
	if reader.ButtonLabel(ActionStart) != "ENTER" || reader.ButtonLabel(ActionSelect) != "BACKSPACE" {
		t.Fatalf("keyboard labels %q %q", reader.ButtonLabel(ActionStart), reader.ButtonLabel(ActionSelect))
	}
	reader.UseCustomBindings(ButtonBindings{Fire: 2, AimLock: 3, Dash: 0})
	reader.lastDevice = pickDevice(DeviceKeyboard, false, true)
	if reader.ButtonLabel(ActionStart) != "START" || reader.ButtonLabel(ActionFire) != "BUTTON 2" {
		t.Fatalf("gamepad labels %q %q", reader.ButtonLabel(ActionStart), reader.ButtonLabel(ActionFire))
	}
	if pickDevice(DeviceGamepad, false, false) != DeviceGamepad {
		t.Fatal("idle input changed the device")
	}
}
