package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"hordefall/internal/audio"
)

// ===== Constants =====

const (
	initialsAlphabet      = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789."
	initialsLength        = 3
	defaultInitials       = "AAA"
	nameEntryLetterTop    = 300
	nameEntryLetterGap    = 96
	nameEntryArrowSize    = 18
	nameEntryArrowGap     = 14
	nameEntryUnderlineGap = 96
)

// ===== Internal =====

func (g *Game) openNameEntry() {
	copy(g.nameEntryLetters[:], sanitizeInitials(g.settings.Initials))
	g.nameEntryCursor = 0
	g.newEntryRank = g.highScores.RankOf(g.CurrentScore())
	g.switchState(StateNameEntry)
}

func (g *Game) updateNameEntry() {
	pressed := g.controls.JustPressed
	letterDelta := boolToIndex(pressed&ActionUp != 0) - boolToIndex(pressed&ActionDown != 0)
	cursorDelta := boolToIndex(pressed&ActionRight != 0) - boolToIndex(pressed&(ActionLeft|ActionAimLock) != 0)
	g.shiftNameEntryLetter(letterDelta)
	g.nameEntryCursor = clampInt(g.nameEntryCursor+cursorDelta, 0, initialsLength-1)
	g.playSoundIf(audio.SoundMenuMove, letterDelta != 0 || cursorDelta != 0)
	if !g.isConfirming() {
		return
	}
	g.playSound(audio.SoundMenuSelect)
	if g.nameEntryCursor < initialsLength-1 {
		g.nameEntryCursor++
		return
	}
	g.commitNameEntry()
}

func (g *Game) shiftNameEntryLetter(delta int) {
	letter := &g.nameEntryLetters[g.nameEntryCursor]
	alphabetSize := len(initialsAlphabet)
	position := (indexOfLetter(*letter) + delta + alphabetSize) % alphabetSize
	*letter = initialsAlphabet[position]
}

func (g *Game) commitNameEntry() {
	initials := string(g.nameEntryLetters[:])
	g.settings.Initials = initials
	g.saveSettings()
	g.recordHighScoreAs(initials)
	g.switchState(StateGameOver)
}

func (g *Game) drawNameEntry(screen *ebiten.Image) {
	g.drawPlaying(screen)
	g.ui.DrawNameEntry(g, screen)
}

func (u *UI) DrawNameEntry(g *Game, screen *ebiten.Image) {
	dimScreen(screen, 0.75)
	titles := [2]string{"GREAT SCORE!", "NEW HIGH SCORE!"}
	u.drawText(screen, titles[boolToIndex(g.newEntryRank == 0)], u.title, screenWidth/2, menuTitleTop, accentColor, pulse(g.clockSeconds), text.AlignCenter)
	summary := "SCORE " + formatThousands(g.CurrentScore()) + "      RANK " + rankOrdinals[max(0, g.newEntryRank)]
	u.drawText(screen, summary, u.bold, screenWidth/2, 180, highScoreColor, 1, text.AlignCenter)
	u.drawText(screen, "ENTER YOUR INITIALS", u.bold, screenWidth/2, 230, textColor, 1, text.AlignCenter)
	for index, letter := range g.nameEntryLetters {
		u.drawInitial(g, screen, index, string(letter))
	}
	u.drawText(screen, "Up / Down: letter    Left / Right: move    Fire: confirm    Aim lock: back", u.small, screenWidth/2, menuHintTop, textColor, 1, text.AlignCenter)
}

func (u *UI) drawInitial(g *Game, screen *ebiten.Image, index int, letter string) {
	isCurrent := index == g.nameEntryCursor
	x := float32(screenWidth/2) + float32(index-1)*nameEntryLetterGap
	colors := [2][3]float32{textColor, accentColor}
	u.drawText(screen, letter, u.title, x, nameEntryLetterTop, colors[boolToIndex(isCurrent)], 1, text.AlignCenter)
	fillRect(screen, x-28, nameEntryLetterTop+nameEntryUnderlineGap, 56, 4, toColor(colors[boolToIndex(isCurrent)], 1))
	if !isCurrent {
		return
	}
	half := float32(nameEntryArrowSize) / 2
	top := float32(nameEntryLetterTop) - nameEntryArrowGap
	bottom := float32(nameEntryLetterTop) + nameEntryUnderlineGap + nameEntryArrowGap
	fillTriangle(screen, [3][2]float32{{x - half, top}, {x + half, top}, {x, top - half}}, accentColor, 1)
	fillTriangle(screen, [3][2]float32{{x - half, bottom}, {x + half, bottom}, {x, bottom + half}}, accentColor, 1)
}

func sanitizeInitials(initials string) string {
	isValid := len(initials) == initialsLength
	for index := range min(len(initials), initialsLength) {
		isValid = isValid && indexOfLetter(initials[index]) >= 0
	}
	return [2]string{defaultInitials, initials}[boolToIndex(isValid)]
}

func indexOfLetter(letter byte) int {
	for index := range len(initialsAlphabet) {
		if initialsAlphabet[index] == letter {
			return index
		}
	}
	return -1
}
