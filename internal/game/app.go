package game

import (
	"embed"
	"io/fs"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ajkula/ajkurig/audio"
	"github.com/ajkula/ajkurig/colorx"
	"github.com/ajkula/ajkurig/draw"
	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/input"
	"github.com/ajkula/ajkurig/intro"
	"github.com/ajkula/ajkurig/rig"
	"github.com/ajkula/ajkurig/scores"
	"github.com/ajkula/ajkurig/shell"
	"github.com/ajkula/ajkurig/store"
	"github.com/ajkula/ajkurig/ui"

	"hordefall/music"
)

// ===== Constants =====

//go:embed lang/*.json
var languageFiles embed.FS

const (
	gameTitle           = "HORDEFALL"
	windowTitle         = "Hordefall"
	saveFolder          = "hordefall"
	gameOverTitle       = "YOU FELL"
	controlsHintKey     = "main.controls"
	gameOverIdleSeconds = 20
	crackGlowAlpha      = 60.0 / 255
)

var (
	rankingColor   = [3]float32{1, 0.44, 0.1}
	selectedColor  = [3]float32{1, 0.08, 0.06}
	logoCapColor   = [3]float32{1, 0.34, 0.07}
	logoSideColor  = [3]float32{0.62, 0.15, 0.03}
	crackCoreColor = [3]float32{235.0 / 255, 245.0 / 255, 1}
	crackGlowColor = [3]float32{160.0 / 255, 200.0 / 255, 1}
	shardColor     = [3]float32{0.8, 0.9, 1}
)

// ===== Public API =====

func Run() error {
	app, _, err := NewApp(false)
	if err != nil {
		return err
	}
	return app.Run()
}

func NewApp(isSilent bool) (*rig.App, *Game, error) {
	languages, err := fs.Sub(languageFiles, "lang")
	if err != nil {
		return nil, nil, err
	}
	songs, err := music.Songs(uint32(time.Now().UnixNano()))
	if err != nil {
		return nil, nil, err
	}
	migrationError := MigrateLegacySaves(store.ForApplication(saveFolder), input.NewLayout(remaps, len(remaps)))
	sounds := audio.NewSoundBank()
	soundIDs := RegisterSounds(sounds)
	theme := hordefallTheme()
	app := rig.NewApp(rig.Config{
		Title: gameTitle, WindowTitle: windowTitle, SaveFolder: saveFolder, Languages: []fs.FS{languages}, Theme: &theme,
		Remaps: remaps, EssentialRemaps: len(remaps), Mouse: mouseBindings,
		Sounds: sounds, Signals: music.SignalNames(), Songs: songs, IsSilent: isSilent,
	})
	reportMigrationError(app, migrationError)
	game := NewGame(app, soundIDs)
	app.World.Mount(game.Modules()...)
	game.shell = shell.Mount(app, game, shell.Options{
		Title: gameTitle, GameOverTitle: gameOverTitle, DemoClipSeconds: demoSequenceSeconds, ClipsPerBoard: 2 * len(demoSequences),
		GameOverIdle: gameOverIdleSeconds, Intro: introSettings(), AttractArt: game.ui.drawAttractArt,
		HighScoreLine: describeHighScore, ControlsHint: controlsHintKey,
	})
	return app, game, nil
}

// ===== Internal =====

func hordefallTheme() ui.Theme {
	theme := ui.DefaultTheme
	theme.Panel, theme.Text, theme.Accent, theme.Outline = panelColor, textColor, accentColor, outlineColor
	theme.Warning, theme.Record, theme.Experience = warningColor, highScoreColor, experienceBlue
	theme.Ranking, theme.SelectedCard, theme.MonoSize = rankingColor, selectedColor, chatterTextSize
	return theme
}

func introSettings() intro.Settings {
	settings := intro.DefaultSettings
	settings.CapColor, settings.SideColor = logoCapColor, logoSideColor
	settings.CrackColor, settings.CrackGlow, settings.FlashColor = crackCoreColor, crackGlowColor, colorx.RGB(textColor)
	settings.GlowAlpha, settings.ShardColor = crackGlowAlpha, shardColor
	return settings
}

func reportMigrationError(app *rig.App, err error) {
	if err == nil {
		return
	}
	app.Message = i18n.F("settings.save_failed", err.Error())
}

func describeHighScore(entry scores.Entry) string {
	return i18n.F("main.high_score", scores.FormatThousands(entry.Score), entry.Initials, scores.FormatClock(entry.Seconds), entry.Stats[statKills], entry.Stats[statLevel])
}

func (u *UI) drawAttractArt(screen *ebiten.Image, progress, clock float32) {
	art := u.art.Render(progress, clock)
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(float64(artPixelScale*draw.Scale), float64(artPixelScale*draw.Scale))
	screen.DrawImage(art, options)
}
