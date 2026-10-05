package game

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ===== Types =====

type HighScoreColumn struct {
	Header string
	X      float32
	Align  text.Align
	Value  func(rank int, entry HighScoreEntry) string
}

// ===== Constants =====

const (
	rankingPauseSeconds   = 2.5
	rankingScrollSeconds  = 5
	highScoreBoardSeconds = 2*rankingPauseSeconds + rankingScrollSeconds
	demoClipsPerBoard     = 4
	rankingGameTitleTop   = 24
	rankingTitleTop       = 122
	rankingTitlePixel     = 8
	rankingHeaderTop      = 196
	rankingHeaderPixel    = 3
	rankingViewportTop    = 232
	rankingViewportBottom = 646
	rankingRowPixel       = 7
	rankingRowHeight      = 68
	rankingHeaderGap      = 36
	gameOverRankedTop     = 400
	highScoreBoardDim     = 0.82
	emptyHighScoreCell    = "-"
)

var gameplayMenuOptions = []MenuOption{
	{fixedLabel("High scores"), (*Game).openHighScores, nil},
	{fixedLabel("Reset high scores"), (*Game).openResetScores, nil},
	{fixedLabel("Back"), (*Game).closeGameplay, nil},
}

var resetScoresMenuOptions = []MenuOption{
	{fixedLabel("No, keep them"), (*Game).closeResetScores, nil},
	{fixedLabel("Yes, erase all high scores"), (*Game).resetHighScores, nil},
}

var rankOrdinals = [highScoreTableSize]string{"1ST", "2ND", "3RD", "4TH", "5TH", "6TH", "7TH", "8TH", "9TH", "10TH"}

var rankingColor = [3]float32{1, 0.44, 0.1}

var highScoreColumns = []HighScoreColumn{
	{"RANK", 171, text.AlignStart, func(rank int, entry HighScoreEntry) string { return rankOrdinals[rank] }},
	{"SCORE", 675, text.AlignEnd, func(rank int, entry HighScoreEntry) string { return formatThousands(entry.Score) }},
	{"TIME", 934, text.AlignEnd, func(rank int, entry HighScoreEntry) string { return formatClock(entry.SurvivedSeconds) }},
	{"NAME", 990, text.AlignStart, func(rank int, entry HighScoreEntry) string { return entry.Initials }},
}

// ===== Internal =====

func (g *Game) openGameplay() {
	g.settingsMessage = ""
	g.switchState(StateGameplay)
}

func (g *Game) closeGameplay() {
	g.switchState(StateOptions)
}

func (g *Game) openHighScores() {
	g.newEntryRank = noRank
	g.boardOpenedSeconds = g.clockSeconds
	g.switchState(StateHighScores)
}

func (g *Game) openResetScores() {
	g.switchState(StateResetScores)
}

func (g *Game) closeResetScores() {
	g.switchState(StateGameplay)
	g.menuSelection = 1
}

func (g *Game) resetHighScores() {
	g.highScores = HighScoreTable{}
	g.saveHighScores()
	g.settingsMessage = "High scores erased."
	g.closeResetScores()
}

func (g *Game) updateGameplay() {
	g.updateBackdropDemo()
	if g.isGoingBack() {
		g.closeGameplay()
		return
	}
	g.navigateMenu(gameplayMenuOptions)
}

func (g *Game) updateHighScores() {
	g.updateBackdropDemo()
	isLeaving := g.isGoingBack() || g.isConfirming()
	if !isLeaving {
		return
	}
	g.switchState(StateGameplay)
}

func (g *Game) updateResetScores() {
	g.updateBackdropDemo()
	if g.isGoingBack() {
		g.closeResetScores()
		return
	}
	g.navigateMenu(resetScoresMenuOptions)
}

func (g *Game) drawGameplay(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawSubmenu(g, screen, "GAMEPLAY", gameplayMenuOptions, "Up / Down to choose, Fire to confirm, Start / Esc / Aim lock to go back")
}

func (g *Game) drawResetScores(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawSubmenu(g, screen, "ERASE HIGH SCORES?", resetScoresMenuOptions, "This cannot be undone. Start / Esc / Aim lock to cancel")
}

func (g *Game) drawHighScores(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	elapsed := g.clockSeconds - g.boardOpenedSeconds
	g.ui.DrawHighScoreBoard(g, screen, elapsed-highScoreBoardSeconds*float32(int(elapsed/highScoreBoardSeconds)), "Fire / Start / Esc / Aim lock to go back")
}

func (u *UI) DrawSubmenu(g *Game, screen *ebiten.Image, title string, options []MenuOption, hint string) {
	dimScreen(screen, menuDim)
	u.drawText(screen, title, u.title, screenWidth/2, menuTitleTop, accentColor, 1, text.AlignCenter)
	u.drawText(screen, g.settingsMessage, u.small, screenWidth/2, menuTitleTop+110, healthColor, 1, text.AlignCenter)
	u.drawMenuOptions(g, screen, options, menuOptionSpacing)
	u.drawMusicBanner(g, screen)
	u.drawText(screen, hint, u.small, screenWidth/2, menuHintTop, textColor, 1, text.AlignCenter)
}

func (u *UI) DrawHighScoreBoard(g *Game, screen *ebiten.Image, elapsedSeconds float32, hint string) {
	dimScreen(screen, highScoreBoardDim)
	u.drawText(screen, "HORDEFALL", u.title, screenWidth/2, rankingGameTitleTop, accentColor, 1, text.AlignCenter)
	u.pixels.DrawText(screen, "RANKING", screenWidth/2, rankingTitleTop, rankingTitlePixel, rankingColor, 1, text.AlignCenter)
	for _, column := range highScoreColumns {
		u.pixels.DrawText(screen, column.Header, column.X, rankingHeaderTop, rankingHeaderPixel, rankingColor, 1, column.Align)
	}
	viewport := screen.SubImage(image.Rect(0, int(rankingViewportTop*renderScale), screen.Bounds().Dx(), int(rankingViewportBottom*renderScale))).(*ebiten.Image)
	scroll := rankingScrollOffset(elapsedSeconds)
	for rank := range highScoreTableSize {
		u.drawHighScoreRow(g, viewport, rank, rankingViewportTop+float32(rank)*rankingRowHeight-scroll)
	}
	u.drawMusicBanner(g, screen)
	u.drawText(screen, hint, u.small, screenWidth/2, menuHintTop, textColor, 1, text.AlignCenter)
}

func (u *UI) drawHighScoreRow(g *Game, viewport *ebiten.Image, rank int, y float32) {
	entry := g.highScores.EntryAt(rank)
	isNewEntry := rank == g.newEntryRank
	alpha := pulse(g.clockSeconds)*boolToFloat(isNewEntry) + boolToFloat(!isNewEntry)
	isFilled := entry.Score > 0
	for index, column := range highScoreColumns {
		values := [2]string{emptyHighScoreCell, column.Value(rank, entry)}
		value := values[boolToIndex(isFilled || index == 0)]
		u.pixels.DrawText(viewport, value, column.X, y, rankingRowPixel, rankingColor, alpha, column.Align)
	}
}

func (u *UI) drawRankedLine(g *Game, screen *ebiten.Image, y float32) {
	if g.newEntryRank == noRank {
		return
	}
	for _, column := range highScoreColumns {
		u.pixels.DrawText(screen, column.Header, column.X, y-rankingHeaderGap, rankingHeaderPixel, rankingColor, 1, column.Align)
	}
	u.drawHighScoreRow(g, screen, g.newEntryRank, y)
}

func rankingScrollOffset(elapsedSeconds float32) float32 {
	maximumScroll := float32(highScoreTableSize*rankingRowHeight - (rankingViewportBottom - rankingViewportTop))
	progress := clamp((elapsedSeconds-rankingPauseSeconds)/rankingScrollSeconds, 0, 1)
	return maximumScroll * progress
}

func boardElapsedSeconds(remainingSeconds float32) float32 {
	return highScoreBoardSeconds - remainingSeconds
}

func (g *Game) advanceDemoBoard() {
	g.highScoreBoardSeconds = max(0, g.highScoreBoardSeconds-deltaSeconds)
}

func (g *Game) startDemoBoardIfDue() {
	g.demoClipCount++
	if g.demoClipCount%demoClipsPerBoard != 0 {
		return
	}
	g.highScoreBoardSeconds = highScoreBoardSeconds
	g.demoClipSeconds += highScoreBoardSeconds
}

func (g *Game) dismissDemoBoardIfPressed() bool {
	if g.highScoreBoardSeconds <= 0 || g.controls.JustPressed == 0 {
		return false
	}
	g.highScoreBoardSeconds = 0
	g.menuLockSeconds = menuLockDuration
	return true
}
