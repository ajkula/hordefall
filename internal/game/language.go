package game

import (
	"image"
	"math"
	"os"
	"path/filepath"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"hordefall/internal/audio"
	"hordefall/internal/i18n"
)

// ===== Constants =====

const (
	drumSlotHeight        = 26
	drumWindowHalf        = 21
	drumWidth             = 150
	drumLabelGap          = 28
	drumArrowIndent       = 6
	drumFadePerSlot       = 0.75
	drumEasing            = 16
	drumRepeatDelay       = 2
	drumRepeatInterval    = 0.28
	drumArrowWidth        = 24
	drumArrowHeight       = 7
	drumArrowGap          = 11
	drumArrowFlashSeconds = 0.15
	drumTextHeight        = 24
	drumTextLift          = 4
)

var languageDrumSlider = MenuSlider{IsDrum: true}

var languageMenuOption = MenuOption{translatedLabel("gameplay.language"), (*Game).focusLanguageDrum, &languageDrumSlider}

var firstRunLanguageMenu = []MenuOption{languageMenuOption}

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
	code := [2]string{i18n.FallbackCode, g.settings.Language}[boolToIndex(g.isLanguageChosen())]
	i18n.Use(code)
	g.ui.ResetFaces()
}

func (g *Game) isLanguageChosen() bool {
	return g.settings.Language != i18n.DefaultCode && g.settings.Language != ""
}

func (g *Game) openLanguageMenuIfFirstRun() {
	if g.isLanguageChosen() || len(i18n.Languages()) == 0 {
		return
	}
	g.switchState(StateLanguage)
	g.previewLanguage(i18n.Detect())
	g.focusLanguageDrum()
}

func (g *Game) focusLanguageDrum() {
	languages := i18n.Languages()
	if len(languages) == 0 {
		return
	}
	g.isLanguageDrumFocused = true
	g.languageTarget = currentLanguageIndex(languages)
	g.languageScroll, g.languageHoldSeconds = float32(g.languageTarget), 0
	g.menuLockSeconds = menuLockDuration
}

func (g *Game) updateLanguageMenu() {
	g.updateBackdropDemo()
	g.updateLanguageDrum()
}

func (g *Game) updateLanguageDrum() {
	languages := i18n.Languages()
	g.languageArrowSeconds = moveTowardZero(g.languageArrowSeconds, deltaSeconds)
	g.languageScroll += (float32(g.languageTarget) - g.languageScroll) * min(1, drumEasing*deltaSeconds)
	if g.isLanguageChosen() && g.isGoingBack() {
		g.isLanguageDrumFocused = false
		g.applyLanguage()
		g.menuLockSeconds = menuLockDuration
		return
	}
	if g.isConfirming() {
		g.playSound(audio.SoundMenuSelect)
		g.chooseLanguage(languages[wrapIndex(g.languageTarget, len(languages))].Code)
		return
	}
	step := g.drumStep()
	if step == 0 {
		return
	}
	g.languageTarget += step
	g.languageArrowSeconds = drumArrowFlashSeconds * float32(step)
	g.playSound(audio.SoundMenuMove)
	g.previewLanguage(languages[wrapIndex(g.languageTarget, len(languages))].Code)
}

func (g *Game) chooseLanguage(code string) {
	g.settings.Language = code
	g.saveSettings()
	g.applyLanguage()
	g.isLanguageDrumFocused = false
	g.menuLockSeconds = menuLockDuration
	if g.state != StateLanguage {
		return
	}
	g.switchState(StateMainMenu)
}

func (g *Game) previewLanguage(code string) {
	i18n.Use(code)
	g.ui.ResetFaces()
}

func (g *Game) drumStep() int {
	pressed, held := g.controls.JustPressed, g.controls.Held
	direction := boolToIndex(held&ActionDown != 0) - boolToIndex(held&ActionUp != 0)
	isFreshPress := pressed&(ActionUp|ActionDown) != 0
	g.languageHoldSeconds = (g.languageHoldSeconds + deltaSeconds) * boolToFloat(direction != 0 && !isFreshPress)
	repeats := repeatCount(g.languageHoldSeconds, drumRepeatDelay, drumRepeatInterval) - repeatCount(g.languageHoldSeconds-deltaSeconds, drumRepeatDelay, drumRepeatInterval)
	fresh := boolToIndex(pressed&ActionDown != 0) - boolToIndex(pressed&ActionUp != 0)
	return fresh + direction*repeats
}

func (g *Game) drawLanguageMenu(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawSubmenu(g, screen, i18n.T("title.language"), firstRunLanguageMenu, i18n.T("hint.language"))
}

func (u *UI) drawDrumRow(g *Game, screen *ebiten.Image, option MenuOption, y, alpha float32, isSelected bool) {
	label := option.Label(g)
	labelWidth := u.textWidth(label, u.bold)
	colors := [2][3]float32{textColor, accentColor}
	u.drawText(screen, label, u.bold, screenWidth/2-drumLabelGap/2, y, colors[boolToIndex(isSelected)], alpha, text.AlignEnd)
	u.drawSelectionMarkersAround(g, screen, 2*max(labelWidth, drumWidth)+drumLabelGap, y, alpha, isSelected)
	namesX := float32(screenWidth)/2 + drumLabelGap/2
	centerY := y + drumTextHeight/2 + drumTextLift
	u.drawDrumNames(g, screen, namesX, centerY, isSelected)
	arrowX := namesX + drumArrowWidth/2 + drumArrowIndent
	arrowAlpha := boolToFloat(g.isLanguageDrumFocused)
	u.drawDrumArrow(screen, arrowX, centerY-drumWindowHalf-drumArrowGap, -1, g.languageArrowSeconds < 0, arrowAlpha)
	u.drawDrumArrow(screen, arrowX, centerY+drumWindowHalf+drumArrowGap, 1, g.languageArrowSeconds > 0, arrowAlpha)
}

func (u *UI) drawDrumNames(g *Game, screen *ebiten.Image, namesX, centerY float32, isSelected bool) {
	languages := i18n.Languages()
	if len(languages) == 0 {
		return
	}
	window := screen.SubImage(image.Rect(0, int((centerY-drumWindowHalf)*renderScale), screen.Bounds().Dx(), int((centerY+drumWindowHalf)*renderScale))).(*ebiten.Image)
	scroll := [2]float32{float32(currentLanguageIndex(languages)), g.languageScroll}[boolToIndex(g.isLanguageDrumFocused)]
	base := int(math.Floor(float64(scroll)))
	colors := [2][3]float32{textColor, accentColor}
	reach := boolToIndex(g.isLanguageDrumFocused)
	for offset := -reach; offset <= 2*reach; offset++ {
		slot := base + offset
		distance := float32(slot) - scroll
		alpha := clamp(1-abs(distance)*drumFadePerSlot, 0, 1)
		isHighlighted := abs(distance) < 0.5 && (isSelected || g.isLanguageDrumFocused)
		y := centerY + distance*drumSlotHeight - drumTextHeight/2 - drumTextLift
		u.drawText(window, languages[wrapIndex(slot, len(languages))].Name, u.bold, namesX, y, colors[boolToIndex(isHighlighted)], alpha, text.AlignStart)
	}
}

func currentLanguageIndex(languages []i18n.Language) int {
	current := i18n.Current().Code
	return max(0, slices.IndexFunc(languages, func(language i18n.Language) bool { return language.Code == current }))
}

func (u *UI) drawDrumArrow(screen *ebiten.Image, x, y, direction float32, isLit bool, alpha float32) {
	if alpha <= 0 {
		return
	}
	tint := [2][3]float32{{0.75, 0.55, 0.25}, accentColor}[boolToIndex(isLit)]
	tipY := y + direction*drumArrowHeight/2
	baseY := y - direction*drumArrowHeight/2
	fillTriangle(screen, [3][2]float32{{x - drumArrowWidth/2, baseY}, {x + drumArrowWidth/2, baseY}, {x, tipY}}, tint, alpha)
}

func wrapIndex(index, count int) int {
	return ((index % count) + count) % count
}

func repeatCount(holdSeconds, delay, interval float32) int {
	return int(max(0, holdSeconds-delay+interval) / interval)
}

func moveTowardZero(value, step float32) float32 {
	return value - clamp(value, -step, step)
}
