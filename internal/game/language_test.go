package game

import (
	"testing"

	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/input"
	"github.com/ajkula/ajkurig/mathx"
	"github.com/ajkula/ajkurig/rig"
	"github.com/ajkula/ajkurig/shell"
	"github.com/ajkula/ajkurig/ui"
)

// ===== Constants =====

const (
	gameplayLanguageOption = 0
	introMaximumSeconds    = 30
)

// ===== Public API =====

func TestLanguageDrumWrapsRepeatsAndConfirms(t *testing.T) {
	app, _ := newHomeApp(t)
	drum := focusLanguageDrum(app, i18n.FallbackCode)
	count := len(i18n.Languages())
	app.Step(input.Controls{Held: input.ActionUp, JustPressed: input.ActionUp, RawJustPressed: input.NoInput})
	if mathx.WrapIndex(drum.Target, count) != count-1 || i18n.Current().Code != i18n.Languages()[count-1].Code {
		t.Fatalf("up from the first language did not wrap to the last: %d", drum.Target)
	}
	app.Step(input.Controls{Held: input.ActionDown, JustPressed: input.ActionDown, RawJustPressed: input.NoInput})
	stepFor(app, ui.DrumRepeatDelay-2*deltaSeconds, input.ActionDown)
	if mathx.WrapIndex(drum.Target, count) != 0 {
		t.Fatalf("the drum moved before the hold delay: %d", drum.Target)
	}
	stepFor(app, 1, input.ActionDown)
	moved := mathx.WrapIndex(drum.Target, count)
	if moved < 3 || moved > 5 {
		t.Fatalf("one second of auto scroll moved %d languages, want a calm 3 to 5", moved)
	}
	press(app, input.ActionConfirm)
	if app.CurrentScreen() != shell.ScreenGameplay || drum.IsFocused || app.Settings.Language != i18n.Languages()[moved].Code {
		t.Fatalf("confirm chose %q, screen %q, focused %v", app.Settings.Language, app.CurrentScreen(), drum.IsFocused)
	}
}

func TestLanguageDrumCancelRestoresTheLanguage(t *testing.T) {
	app, _ := newHomeApp(t)
	drum := focusLanguageDrum(app, "fr")
	app.Step(input.Controls{Held: input.ActionDown, JustPressed: input.ActionDown, RawJustPressed: input.NoInput})
	press(app, input.ActionPause|input.ActionBack)
	if drum.IsFocused || i18n.Current().Code != "fr" || app.Settings.Language != "fr" {
		t.Fatalf("cancel left %q active, setting %q", i18n.Current().Code, app.Settings.Language)
	}
}

func TestFirstRunAsksForTheLanguageThenOpensTheMenu(t *testing.T) {
	app, _ := newTestApp(t)
	app.Start()
	for tick := 0; app.CurrentScreen() == shell.ScreenIntro && tick < introMaximumSeconds*ticksPerSecond; tick++ {
		app.Step(input.Controls{RawJustPressed: input.NoInput})
	}
	drum := menuOption(app, shell.ScreenLanguage, gameplayLanguageOption).Drum
	if app.CurrentScreen() != shell.ScreenLanguage || !drum.IsFocused {
		t.Fatalf("first run screen %q, focused %v", app.CurrentScreen(), drum.IsFocused)
	}
	waitForMenuLock(app)
	press(app, input.ActionPause|input.ActionBack)
	if app.CurrentScreen() != shell.ScreenLanguage {
		t.Fatal("the first run language page can be skipped without choosing")
	}
	press(app, input.ActionConfirm)
	if app.CurrentScreen() != shell.ScreenMenu || !app.Settings.IsLanguageChosen() {
		t.Fatalf("confirm left screen %q, language %q", app.CurrentScreen(), app.Settings.Language)
	}
}

// ===== Internal =====

func focusLanguageDrum(app *rig.App, code string) *ui.Drum {
	app.Settings.Language = code
	app.ApplyLanguage()
	app.Switch(shell.ScreenGameplay)
	waitForMenuLock(app)
	drum := menuOption(app, shell.ScreenGameplay, gameplayLanguageOption).Drum
	drum.Focus()
	return drum
}
