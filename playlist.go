package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ===== Constants =====

const (
	menuOptionSpacing = 48
	menuListBottomGap = 16
)

var trackStateLabels = [2]string{"OFF", "ON"}

// ===== Public API =====

func (g *Game) IsSongEnabled(index int) bool {
	isEnabled, isKnown := g.settings.Tracks[g.audio.SongTitle(index)]
	return isEnabled || !isKnown
}

func (g *Game) HasEnabledSong() bool {
	for index := range g.audio.SongCount() {
		if g.IsSongEnabled(index) {
			return true
		}
	}
	return false
}

// ===== Internal =====

func (g *Game) buildPlaylistMenu() {
	g.playlistMenu = g.playlistMenu[:0]
	for index := range g.audio.SongCount() {
		g.playlistMenu = append(g.playlistMenu, MenuOption{
			Label:    func(game *Game) string { return game.trackLabel(index) },
			Activate: func(game *Game) { game.toggleSong(index) },
		})
	}
	g.playlistMenu = append(g.playlistMenu, MenuOption{fixedLabel("Back"), (*Game).closePlaylist})
	g.audio.SetPlaylistEmpty(!g.HasEnabledSong())
}

func (g *Game) trackLabel(index int) string {
	return g.audio.SongTitle(index) + ": " + trackStateLabels[boolToIndex(g.IsSongEnabled(index))]
}

func (g *Game) toggleSong(index int) {
	isNowEnabled := !g.IsSongEnabled(index)
	g.settings.Tracks[g.audio.SongTitle(index)] = isNowEnabled
	g.audio.SetPlaylistEmpty(!g.HasEnabledSong())
	g.saveSettings()
	g.previewSongIf(index, isNowEnabled)
}

func (g *Game) previewSongIf(index int, shouldPreview bool) {
	if !shouldPreview {
		return
	}
	g.playSong(index)
}

func (g *Game) openPlaylist() {
	g.settingsMessage = ""
	g.switchState(StatePlaylist)
}

func (g *Game) closePlaylist() {
	g.switchState(StateOptions)
}

func (g *Game) updatePlaylist() {
	g.updateDemo()
	if g.menuLockSeconds == 0 && g.controls.JustPressed&ActionPause != 0 {
		g.closePlaylist()
		return
	}
	g.navigateMenu(g.playlistMenu)
}

func (g *Game) drawPlaylist(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawPlaylistMenu(g, screen, g.playlistMenu)
}

func (u *UI) DrawPlaylistMenu(g *Game, screen *ebiten.Image, options []MenuOption) {
	dimScreen(screen, menuDim)
	u.drawText(screen, "PLAYLIST", u.title, screenWidth/2, menuTitleTop, accentColor, 1, text.AlignCenter)
	u.drawText(screen, g.settingsMessage, u.small, screenWidth/2, menuTitleTop+110, healthColor, 1, text.AlignCenter)
	u.drawMenuOptions(g, screen, options, menuListTop(len(options)))
	u.drawBenchmarkPanel(g, screen)
	u.drawMusicBanner(g, screen)
	u.drawText(screen, "Up / Down to choose, Fire to switch a track on or off, Start / Esc to go back", u.small, screenWidth/2, menuHintTop, textColor, 0.9, text.AlignCenter)
	u.drawText(screen, "Only enabled tracks play, in game and on the main menu. All off means no music.", u.small, screenWidth/2, menuHintTop+24, mutedTextColor, 1, text.AlignCenter)
}

func menuListTop(optionCount int) float32 {
	return min(mainMenuOptionsTop, menuHintTop-menuListBottomGap-float32(optionCount*menuOptionSpacing))
}
