package game

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/ajkula/ajkurig/colorx"
	"github.com/ajkula/ajkurig/draw"
	"github.com/ajkula/ajkurig/ui"
)

// ===== Types =====

type RobotLab struct {
	SetIndex  int
	EnemyTurn int
	Art       *ArtCanvas
}

type RobotLabSet struct {
	Name string
	Look RobotLook
}

type RobotLabKey struct {
	Key     ebiten.Key
	Advance func(look *RobotLook)
}

// ===== Constants =====

const (
	robotLabEnemyCount     = 2
	robotLabEnemyKindCount = int(EnemySpiderTank)
	robotLabSaveFolder     = "hordefall-robotlab"
	robotLabPanelX         = 16
	robotLabPanelY         = 62
	robotLabPanelWidth     = 300
	robotLabPanelHeight    = 204
	robotLabLineHeight     = 24
	robotLabArtScale       = 3
	robotLabArtTop         = 96
	robotLabCustomName     = "Custom mix"
)

var robotLabSets = [...]RobotLabSet{
	{"Classic", RobotLook{}},
	{"Worksite", worksiteRobotLook},
	{"Lantern", lanternRobotLook},
	{"Parasol", parasolRobotLook},
	{"Kettle", kettleRobotLook},
	{"Television", televisionRobotLook},
	{"Cart", cartRobotLook},
	{"Diver", diverRobotLook},
}

var robotLabKeys = [...]RobotLabKey{
	{ebiten.KeyDigit1, func(look *RobotLook) { look.Head = (look.Head + 1) % robotHeadCount }},
	{ebiten.KeyDigit2, func(look *RobotLook) { look.Eyes = (look.Eyes + 1) % robotEyesCount }},
	{ebiten.KeyDigit3, func(look *RobotLook) { look.Abdomen = (look.Abdomen + 1) % robotAbdomenCount }},
	{ebiten.KeyDigit4, func(look *RobotLook) { look.Legs = (look.Legs + 1) % robotLegsCount }},
	{ebiten.KeyDigit5, func(look *RobotLook) { look.Arms = (look.Arms + 1) % robotArmsCount }},
	{ebiten.KeyDigit6, func(look *RobotLook) { look.Color = (look.Color + 1) % robotColorCount }},
}

var (
	robotLabHeadNames    = [robotHeadCount]string{"Round", "Turret", "Cap", "Crook", "Lid", "Screen", "Seat", "Helmet"}
	robotLabEyeNames     = [robotEyesCount]string{"Trio", "Visor", "Cyclops", "Duo", "Pixel", "Headlight", "Porthole"}
	robotLabAbdomenNames = [robotAbdomenCount]string{"Bulb", "Crate", "Paper", "Canopy", "Iron", "Tape", "Caged core", "Tank"}
	robotLabLegNames     = [robotLegsCount]string{"Droplet", "Strut", "Bamboo", "Rib", "Stub", "Cable", "Tube", "Hose"}
	robotLabArmNames     = [robotArmsCount]string{"Manipulator", "Lamps", "Hook", "Rib", "Loop", "Antenna", "Handle", "Pincer"}
	robotLabColorNames   = [robotColorCount]string{"Blue", "Yellow", "Blood red", "Violet", "Rose", "Green"}
)

// ===== Public API =====

func RunRobotLab() error {
	if err := redirectSavesToRobotLabFolder(); err != nil {
		return err
	}
	app, game, err := NewApp(false)
	if err != nil {
		return err
	}
	game.enableRobotLab()
	ebiten.SetWindowSize(int(app.Config.Width), int(app.Config.Height))
	ebiten.SetWindowTitle(app.Config.WindowTitle + " - Robot Lab")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	app.Start()
	game.shell.StartRun()
	return ebiten.RunGame(app)
}

func NewRobotLab() *RobotLab {
	return &RobotLab{Art: NewArtCanvas()}
}

// ===== Internal =====

func redirectSavesToRobotLabFolder() error {
	folder := filepath.Join(os.Getenv("APPDATA"), robotLabSaveFolder)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return err
	}
	return os.Setenv("APPDATA", folder)
}

func (g *Game) enableRobotLab() {
	g.robotLab = NewRobotLab()
	g.garage.Counts = [robotCounterCount]int{math.MaxInt32, math.MaxInt32}
	g.applyRobotLook(RobotLook{})
}

func (g *Game) updateDirector(deltaSeconds float32) {
	if g.robotLab != nil {
		g.updateRobotLab()
		return
	}
	g.updateSpawning(deltaSeconds)
	g.startBossEventIfDue()
}

func (g *Game) updateRobotLab() {
	lab := g.robotLab
	g.player.Health = g.player.MaximumHealth
	for g.enemies.Count < robotLabEnemyCount {
		lab.spawnNextEnemy(g)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		lab.selectNextSet(g)
	}
	for index := range robotLabKeys {
		binding := &robotLabKeys[index]
		if inpututil.IsKeyJustPressed(binding.Key) {
			binding.Advance(&g.robotLook)
		}
	}
	g.ui.art.Look = g.robotLook
}

func (lab *RobotLab) spawnNextEnemy(g *Game) {
	angle := g.random.Angle()
	kind := EnemyKind(lab.EnemyTurn % robotLabEnemyKindCount)
	lab.EnemyTurn++
	g.spawnEnemyAt(kind, g.player.X+cosine(angle)*spawnDistance, g.player.Y+sine(angle)*spawnDistance)
}

func (lab *RobotLab) selectNextSet(g *Game) {
	lab.SetIndex = (lab.SetIndex + 1) % len(robotLabSets)
	g.robotLook = robotLabSets[lab.SetIndex].Look
}

func (lab *RobotLab) currentSetName(look RobotLook) string {
	set := &robotLabSets[lab.SetIndex]
	if set.Look != look {
		return robotLabCustomName
	}
	return set.Name
}

func (u *UI) drawRobotLab(g *Game, screen *ebiten.Image) {
	lab := g.robotLab
	if lab == nil {
		return
	}
	look := g.robotLook
	draw.FillRect(screen, robotLabPanelX, robotLabPanelY, robotLabPanelWidth, robotLabPanelHeight, colorx.NRGBA(panelColor, 0.8))
	lines := [...]string{
		"[Tab] Set: " + lab.currentSetName(look),
		"[1] Head: " + robotLabHeadNames[look.Head],
		"[2] Eyes: " + robotLabEyeNames[look.Eyes],
		"[3] Abdomen: " + robotLabAbdomenNames[look.Abdomen],
		"[4] Legs: " + robotLabLegNames[look.Legs],
		"[5] Arms: " + robotLabArmNames[look.Arms],
		"[6] Color: " + robotLabColorNames[look.Color],
		fmt.Sprintf("Enemies: %d   Health: infinite", g.enemies.Count),
	}
	for index, line := range lines {
		y := float32(robotLabPanelY + 6 + index*robotLabLineHeight - index/2)
		u.kit.Text.Draw(screen, line, ui.FaceRegular, robotLabPanelX+10, y, textColor, 1, text.AlignStart)
	}
	lab.drawArtPreview(g, screen)
}

func (lab *RobotLab) drawArtPreview(g *Game, screen *ebiten.Image) {
	lab.Art.Look = g.robotLook
	art := lab.Art.Render(1, g.elapsedSeconds)
	left := float64(screenWidth - artWidth*robotLabArtScale - 16)
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(robotLabArtScale*float64(draw.Scale), robotLabArtScale*float64(draw.Scale))
	options.GeoM.Translate(left*float64(draw.Scale), robotLabArtTop*float64(draw.Scale))
	screen.DrawImage(art, options)
}
