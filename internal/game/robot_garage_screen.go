package game

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/ajkula/ajkurig/colorx"
	"github.com/ajkula/ajkurig/draw"
	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/input"
	"github.com/ajkula/ajkurig/mathx"
	"github.com/ajkula/ajkurig/rig"
	"github.com/ajkula/ajkurig/rng"
	"github.com/ajkula/ajkurig/shell"
	"github.com/ajkula/ajkurig/ui"
)

// ===== Types =====

type RobotThumb struct {
	Image   *ebiten.Image
	Bounds  image.Rectangle
	IsOwned bool
}

type RobotPreview struct {
	Game     *Game
	Renderer Renderer
	Camera   draw.Camera
	Image    *ebiten.Image
	Angle    float32
	Recoil   float32
}

type RobotGarageScreen struct {
	Game       *Game
	Row        int
	Columns    [robotSlotCount]int
	Thumbs     [robotSlotCount][robotSlotPartMax]RobotThumb
	ThumbColor RobotColor
	ThumbArt   *ArtCanvas
	RankingArt *ArtCanvas
	Preview    *RobotPreview
}

// ===== Constants =====

const (
	screenRobotGarage = "robot_garage"

	garageTitleX     = 84
	garageTitleY     = 30
	garageLabelX     = 48
	garageCellsX     = 136
	garageCellsY     = 98
	garageCellPitchX = 70
	garageCellPitchY = 84
	garageCellWidth  = 66
	garageCellHeight = 80
	garageCellInner  = 58
	garageConditionY = 612

	garageDim         = 0.82
	garageThumbScale  = 1.3
	garageLockedGray  = 150
	garageLockedAlpha = 0.6
	garagePieceGap    = 2

	garagePreviewX             = 720
	garagePreviewY             = 52
	garagePreviewDisplayWidth  = 512
	garagePreviewDisplayHeight = 288
	garagePreviewWidth         = 512.0 / 3
	garagePreviewHeight        = 288.0 / 3
	garageRankingY             = 356
	garageRankingZoom          = 4

	garagePreviewOrbit    = 600
	garagePreviewTurnRate = 0.25
	garagePreviewShotRate = 1.1
	garagePreviewSeed     = 77
)

var garageThumbPose = ArtPose{AnchorX: artWidth / 2, AnchorY: artHeight - 2, Scale: artModelScale * garageThumbScale}

// ===== Public API =====

func NewRobotGarageScreen(game *Game) *RobotGarageScreen {
	screen := &RobotGarageScreen{
		Game: game, ThumbArt: NewArtCanvas(), RankingArt: NewArtCanvas(),
		Preview: newRobotPreview(game),
	}
	for kind := range screen.Thumbs {
		for part := range screen.Thumbs[kind] {
			screen.Thumbs[kind][part].Image = ebiten.NewImage(artWidth, artHeight)
		}
	}
	return screen
}

func (s *RobotGarageScreen) Enter(app *rig.App) {
	s.Row = 0
	for kind := range robotSlots {
		s.Columns[kind] = robotSlots[kind].Part(&s.Game.robotLook)
	}
	s.rebuildThumbs()
}

func (s *RobotGarageScreen) Update(frame *rig.Frame) {
	game, app := s.Game, s.Game.app
	game.shell.UpdateBackdrop(frame)
	s.Preview.step(frame.Delta)
	isActivating := app.IsConfirming() || (app.MenuLockSeconds == 0 && frame.Controls.JustPressed&ActionFire != 0)
	if s.handleInput(frame.Controls.JustPressed, app.IsGoingBack(), isActivating) {
		game.saveRobotGarage()
		app.Switch(shell.ScreenMenu)
	}
}

func (s *RobotGarageScreen) Draw(frame *rig.Frame, screen *ebiten.Image) {
	game := s.Game
	kit := game.app.Kit
	game.shell.DrawBackdrop(frame, screen)
	kit.Dim(screen, garageDim)
	kit.Text.Draw(screen, i18n.T("garage.title"), ui.FaceHeading, garageTitleX, garageTitleY, kit.Theme.Accent, 1, text.AlignStart)
	for kind := range robotSlots {
		s.drawRow(screen, kit, kind)
	}
	s.drawCondition(screen, kit)
	look := s.previewLook()
	s.Preview.draw(screen, &look)
	s.drawRanking(screen, frame.Clock)
	game.shell.DrawMusicBanner(screen)
	kit.Hints(screen, game.app.Hint("hint.garage"))
}

// ===== Internal: cursor =====

func (s *RobotGarageScreen) handleInput(pressed input.Action, isGoingBack, isActivating bool) bool {
	equipped := robotSlots[s.Row].Part(&s.Game.robotLook)
	isOnEquipped := s.Columns[s.Row] == equipped
	if isGoingBack && isOnEquipped {
		return true
	}
	if isGoingBack {
		s.Columns[s.Row] = equipped
		s.Game.app.PlayMenuEvent(ui.EventMoved)
		return false
	}
	if isActivating {
		s.equipHoveredIfOwned()
		return false
	}
	s.moveCursor(pressed)
	return false
}

func (s *RobotGarageScreen) previewLook() RobotLook {
	look := s.Game.robotLook
	part := s.Columns[s.Row]
	if s.Game.garage.Owns(RobotSlotKind(s.Row), part) {
		robotSlots[s.Row].SetPart(&look, part)
	}
	return look
}

func (s *RobotGarageScreen) moveCursor(pressed input.Action) {
	rowStep := boolToIndex(pressed&input.ActionDown != 0) - boolToIndex(pressed&input.ActionUp != 0)
	columnStep := boolToIndex(pressed&input.ActionRight != 0) - boolToIndex(pressed&input.ActionLeft != 0)
	if rowStep == 0 && columnStep == 0 {
		return
	}
	s.Game.app.PlayMenuEvent(ui.EventMoved)
	if rowStep != 0 {
		s.Columns[s.Row] = robotSlots[s.Row].Part(&s.Game.robotLook)
		s.Row = (s.Row + rowStep + len(robotSlots)) % len(robotSlots)
	}
	count := robotSlots[s.Row].Count
	s.Columns[s.Row] = (s.Columns[s.Row] + columnStep + count) % count
}

func (s *RobotGarageScreen) equipHoveredIfOwned() {
	game, kind, part := s.Game, RobotSlotKind(s.Row), s.Columns[s.Row]
	if !game.garage.Owns(kind, part) {
		return
	}
	game.app.PlayMenuEvent(ui.EventActivated)
	look := game.robotLook
	robotSlots[kind].SetPart(&look, part)
	game.isGarageDirty = game.isGarageDirty || look != game.robotLook
	game.applyRobotLook(look)
	if look.Color != s.ThumbColor {
		s.rebuildThumbs()
	}
}

// ===== Internal: thumbnails =====

func (s *RobotGarageScreen) rebuildThumbs() {
	game := s.Game
	s.ThumbColor = game.robotLook.Color
	for kind := range robotSlotCount {
		for part := range robotSlots[kind].Count {
			thumb := &s.Thumbs[kind][part]
			thumb.IsOwned = game.garage.Owns(kind, part)
			thumb.Bounds = s.ThumbArt.paintRobotThumb(kind, part, s.ThumbColor, thumb.IsOwned)
			thumb.Image.WritePixels(s.ThumbArt.Final)
		}
	}
}

func (c *ArtCanvas) paintRobotThumb(kind RobotSlotKind, part int, color RobotColor, isOwned bool) image.Rectangle {
	clear(c.Final)
	c.beginLayer()
	robotSlots[kind].Paint(c, garageThumbPose, part, color)
	c.commitLayer()
	bounds := image.Rectangle{Min: image.Pt(c.Width, c.Height)}
	for index := range c.Width * c.Height {
		pixel := c.Final[index*4 : index*4+4]
		if pixel[3] == 0 {
			continue
		}
		flattenPixelIf(pixel, !isOwned)
		x, y := index%c.Width, index/c.Width
		bounds.Min = image.Pt(min(bounds.Min.X, x), min(bounds.Min.Y, y))
		bounds.Max = image.Pt(max(bounds.Max.X, x+1), max(bounds.Max.Y, y+1))
	}
	return bounds
}

func flattenPixelIf(pixel []byte, shouldFlatten bool) {
	if !shouldFlatten {
		return
	}
	pixel[0], pixel[1], pixel[2] = garageLockedGray, garageLockedGray, garageLockedGray
}

// ===== Internal: drawing =====

func (s *RobotGarageScreen) drawRow(screen *ebiten.Image, kit *ui.Kit, kind int) {
	y := float32(garageCellsY + kind*garageCellPitchY)
	isCurrent := kind == s.Row
	tint := mathx.Pick(isCurrent, kit.Theme.Text, kit.Theme.Accent)
	kit.Text.Draw(screen, i18n.T(robotSlots[kind].NameKey), ui.FaceRegular, garageLabelX, y+garageCellHeight/2-12, tint, 1, text.AlignStart)
	equipped := robotSlots[kind].Part(&s.Game.robotLook)
	equipTint := colorx.NRGBA(robotPaints[s.Game.robotLook.Color].Body, 1)
	cursorTint := colorx.NRGBA(robotPaletteYellow, 1)
	for part := range robotSlots[kind].Count {
		x := float32(garageCellsX + part*garageCellPitchX)
		thumb := &s.Thumbs[kind][part]
		drawRobotThumb(screen, thumb, x, y)
		isEquipped, isHovered := part == equipped, isCurrent && part == s.Columns[kind]
		drawCellThirds(screen, thumb, x, y, equipTint, [3]bool{false, isEquipped, false})
		drawCellThirds(screen, thumb, x, y, cursorTint, [3]bool{isHovered, false, isHovered})
	}
}

func drawRobotThumb(screen *ebiten.Image, thumb *RobotThumb, x, y float32) {
	if thumb.Bounds.Empty() {
		return
	}
	left, top, zoom := placeRobotThumb(thumb, x, y)
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(float64(zoom*draw.Scale), float64(zoom*draw.Scale))
	options.GeoM.Translate(float64(left*draw.Scale), float64(top*draw.Scale))
	options.ColorScale.ScaleAlpha(mathx.Pick[float32](thumb.IsOwned, garageLockedAlpha, 1))
	screen.DrawImage(thumb.Image.SubImage(thumb.Bounds).(*ebiten.Image), options)
}

func placeRobotThumb(thumb *RobotThumb, x, y float32) (float32, float32, float32) {
	width, height := thumb.Bounds.Dx(), thumb.Bounds.Dy()
	zoom := max(1, min(garageCellInner/max(1, width), garageCellInner/max(1, height)))
	left := x + (garageCellWidth-float32(width*zoom))/2
	top := y + (garageCellHeight-float32(height*zoom))/2
	return left, top, float32(zoom)
}

func drawCellThirds(screen *ebiten.Image, thumb *RobotThumb, x, y float32, tint color.Color, thirds [3]bool) {
	_, pieceTop, zoom := placeRobotThumb(thumb, x, y)
	pieceBottom := pieceTop + float32(thumb.Bounds.Dy())*zoom
	topHeight := pieceTop - garagePieceGap - y
	bottomTop := pieceBottom + garagePieceGap
	third := float32(garageCellWidth) / 3
	for index, isDrawn := range thirds {
		if !isDrawn {
			continue
		}
		left := x + third*float32(index)
		draw.FillRect(screen, left, y, third, topHeight, tint)
		draw.FillRect(screen, left, bottomTop, third, y+garageCellHeight-bottomTop, tint)
	}
}

func (s *RobotGarageScreen) drawCondition(screen *ebiten.Image, kit *ui.Kit) {
	condition := s.Game.garage.DescribeUnlock(RobotSlotKind(s.Row), s.Columns[s.Row])
	centerX := float32(garageCellsX + (robotSlotPartMax*garageCellPitchX-garageCellPitchX+garageCellWidth)/2)
	kit.Text.Draw(screen, condition, ui.FaceRegular, centerX, garageConditionY, kit.Theme.Warning, 1, text.AlignCenter)
}

func (s *RobotGarageScreen) drawRanking(screen *ebiten.Image, clock float32) {
	s.RankingArt.Look = s.previewLook()
	art := s.RankingArt.Render(1, clock)
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(float64(garageRankingZoom*draw.Scale), float64(garageRankingZoom*draw.Scale))
	options.GeoM.Translate(float64(garagePreviewX*draw.Scale), float64(garageRankingY*draw.Scale))
	screen.DrawImage(art, options)
}

// ===== Internal: in-game preview =====

func newRobotPreview(game *Game) *RobotPreview {
	ground := NewGround(arenaSize, garagePreviewSeed)
	preview := &RobotPreview{
		Game:     &Game{player: NewPlayer(), effects: newEffects(), random: rng.New(garagePreviewSeed), ground: ground},
		Renderer: *game.renderer,
	}
	preview.Game.player.X, preview.Game.player.Y = arenaSize/2+garagePreviewOrbit, arenaSize/2
	preview.Game.player.AimX, preview.Game.player.AimY = 0, 1
	preview.Game.player.placeRig()
	preview.Renderer.batches = draw.NewBatches()
	preview.Renderer.camera = &preview.Camera
	preview.Renderer.groundImage = ebiten.NewImage(ground.Columns, ground.Rows)
	return preview
}

func (p *RobotPreview) step(deltaSeconds float32) {
	player := p.Game.player
	p.Game.frame++
	p.Angle += garagePreviewTurnRate * deltaSeconds
	nextX := arenaSize/2 + cosine(p.Angle)*garagePreviewOrbit
	nextY := arenaSize/2 + sine(p.Angle)*garagePreviewOrbit
	player.VelocityX, player.VelocityY = (nextX-player.X)/deltaSeconds, (nextY-player.Y)/deltaSeconds
	player.X, player.Y = nextX, nextY
	player.AimX, player.AimY = normalize(player.VelocityX, player.VelocityY)
	player.IsFiring = true
	p.Recoil += garagePreviewShotRate * deltaSeconds
	player.Rig.ArmRecoil = max(player.Rig.ArmRecoil, boolToFloat(p.Recoil >= 1))
	p.Recoil -= boolToFloat(p.Recoil >= 1)
	p.Game.updateTsukumo(deltaSeconds)
	p.Game.effects.Update(deltaSeconds)
}

func (p *RobotPreview) draw(screen *ebiten.Image, look *RobotLook) {
	width, height := int(garagePreviewWidth*draw.Scale), int(garagePreviewHeight*draw.Scale)
	if p.Image == nil || p.Image.Bounds().Dx() != width || p.Image.Bounds().Dy() != height {
		p.Image = ebiten.NewImage(width, height)
	}
	player := p.Game.player
	p.Camera.X, p.Camera.Y = player.X-garagePreviewWidth/2, player.Y-garagePreviewHeight/2
	p.Renderer.drawGround(p.Game, p.Image)
	p.Game.effects.Queue(p.Renderer.batches, p.Renderer.camera)
	isOnWater := p.Game.ground.MaterialAt(player.X, player.Y) == GroundWater
	p.Renderer.queueRobot(player, look, p.Game.frame, isOnWater)
	p.Renderer.batches.Flush(p.Image)
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(float64(garagePreviewDisplayWidth*draw.Scale)/float64(width), float64(garagePreviewDisplayHeight*draw.Scale)/float64(height))
	options.GeoM.Translate(float64(garagePreviewX*draw.Scale), float64(garagePreviewY*draw.Scale))
	screen.DrawImage(p.Image, options)
}
