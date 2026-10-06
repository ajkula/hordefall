package game

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"hordefall/internal/i18n"
)

// ===== Internal =====

func loadLanguages() {
	directories := make([]string, 0, 2)
	if executable, err := os.Executable(); err == nil {
		directories = append(directories, filepath.Dir(executable))
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		directories = append(directories, workingDirectory)
	}
	_ = i18n.Load(directories...)
}

func (g *Game) applyLanguage() {
	code := [2]string{g.settings.Language, i18n.FallbackCode}[boolToIndex(g.settings.Language == i18n.DefaultCode)]
	i18n.Use(code)
	g.ui.ResetFaces()
}

func (g *Game) isLanguageChosen() bool {
	return g.settings.Language != i18n.DefaultCode
}

func (g *Game) openLanguageMenu(returnState GameState) {
	g.languageReturnState = returnState
	g.languageMenu = g.languageMenu[:0]
	languages := i18n.Languages()
	if len(languages) == 0 {
		return
	}
	for _, language := range languages {
		g.languageMenu = append(g.languageMenu, MenuOption{fixedText(language.Name), chooseLanguage(language.Code), nil})
	}
	preferred := [2]string{i18n.Current().Code, i18n.Detect()}[boolToIndex(!g.isLanguageChosen())]
	g.switchState(StateLanguage)
	g.menuSelection = max(0, slices.IndexFunc(languages, func(language i18n.Language) bool { return language.Code == preferred }))
	g.previewLanguage(languages[g.menuSelection].Code)
}

func (g *Game) openLanguageMenuIfFirstRun() {
	if g.isLanguageChosen() {
		return
	}
	g.openLanguageMenu(StateMainMenu)
}

func (g *Game) openLanguageFromGameplay() {
	g.openLanguageMenu(StateGameplay)
}

func chooseLanguage(code string) func(game *Game) {
	return func(g *Game) {
		g.settings.Language = code
		g.saveSettings()
		g.applyLanguage()
		g.switchState(g.languageReturnState)
	}
}

func (g *Game) previewLanguage(code string) {
	i18n.Use(code)
	g.ui.ResetFaces()
}

func (g *Game) updateLanguageMenu() {
	g.updateBackdropDemo()
	if g.isLanguageChosen() && g.isGoingBack() {
		g.applyLanguage()
		g.switchState(g.languageReturnState)
		return
	}
	previous := g.menuSelection
	g.navigateMenu(g.languageMenu)
	isStillChoosing := g.state == StateLanguage && g.menuSelection != previous
	if !isStillChoosing {
		return
	}
	g.previewLanguage(i18n.Languages()[g.menuSelection].Code)
}

func (g *Game) drawLanguageMenu(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawSubmenu(g, screen, i18n.T("title.language"), g.languageMenu, i18n.T("hint.language"))
}

func (g *Game) languageLabel() string {
	return i18n.F("gameplay.language", i18n.Current().Name)
}

func fixedText(label string) func(game *Game) string {
	return func(*Game) string { return label }
}
