package game

import (
	"testing"

	"hordefall/internal/i18n"
)

func TestLanguageDrumWrapsRepeatsAndConfirms(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	if err := i18n.Load(); err != nil {
		t.Fatal(err)
	}
	game := newHeadlessGame()
	game.settings.Language = i18n.FallbackCode
	game.applyLanguage()
	game.switchState(StateGameplay)
	game.focusLanguageDrum()
	count := len(i18n.Languages())
	game.menuLockSeconds = 0
	game.controls = Controls{Held: ActionUp, JustPressed: ActionUp}
	game.updateGameplay()
	if wrapIndex(game.languageTarget, count) != count-1 || i18n.Current().Code != i18n.Languages()[count-1].Code {
		t.Fatalf("up from the first language did not wrap to the last: %d", game.languageTarget)
	}
	game.controls = Controls{Held: ActionDown, JustPressed: ActionDown}
	game.updateGameplay()
	game.controls = Controls{Held: ActionDown}
	for range drumRepeatDelay*ticksPerSecond - 2 {
		game.updateGameplay()
	}
	if wrapIndex(game.languageTarget, count) != 0 {
		t.Fatalf("the drum moved before the hold delay: %d", game.languageTarget)
	}
	for range ticksPerSecond {
		game.updateGameplay()
	}
	moved := wrapIndex(game.languageTarget, count)
	if moved < 3 || moved > 5 {
		t.Fatalf("one second of auto scroll moved %d languages, want a calm 3 to 5", moved)
	}
	game.controls = Controls{JustPressed: ActionConfirm}
	game.updateGameplay()
	if game.state != StateGameplay || game.isLanguageDrumFocused || game.settings.Language != i18n.Languages()[moved].Code {
		t.Fatalf("confirm chose %q, state %d, focused %v", game.settings.Language, game.state, game.isLanguageDrumFocused)
	}
}

func TestLanguageDrumCancelRestoresTheLanguage(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	if err := i18n.Load(); err != nil {
		t.Fatal(err)
	}
	game := newHeadlessGame()
	game.settings.Language = "fr"
	game.applyLanguage()
	game.switchState(StateGameplay)
	game.focusLanguageDrum()
	game.menuLockSeconds = 0
	game.controls = Controls{Held: ActionDown, JustPressed: ActionDown}
	game.updateGameplay()
	game.controls = Controls{JustPressed: ActionPause}
	game.updateGameplay()
	if game.isLanguageDrumFocused || i18n.Current().Code != "fr" || game.settings.Language != "fr" {
		t.Fatalf("cancel left %q active, setting %q", i18n.Current().Code, game.settings.Language)
	}
}

func TestFirstRunAsksForTheLanguageThenOpensTheMenu(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	if err := i18n.Load(); err != nil {
		t.Fatal(err)
	}
	game := newHeadlessGame()
	game.openLanguageMenuIfFirstRun()
	if game.state != StateLanguage || !game.isLanguageDrumFocused {
		t.Fatalf("first run state %d, focused %v", game.state, game.isLanguageDrumFocused)
	}
	game.menuLockSeconds = 0
	game.controls = Controls{JustPressed: ActionPause}
	game.updateLanguageMenu()
	if game.state != StateLanguage {
		t.Fatal("the first run language page can be skipped without choosing")
	}
	game.controls = Controls{JustPressed: ActionConfirm}
	game.updateLanguageMenu()
	if game.state != StateMainMenu || game.settings.Language == i18n.DefaultCode {
		t.Fatalf("confirm left state %d, language %q", game.state, game.settings.Language)
	}
}
