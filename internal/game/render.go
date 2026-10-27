package game

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ajkula/ajkurig/draw"
)

// ===== Types =====

type Renderer struct {
	batches     *draw.Batches
	camera      *draw.Camera
	groundImage *ebiten.Image
	wormSprites *WormSprites
}

// ===== Constants =====

const (
	auraRimWidth     = 1.2
	auraFlickerBase  = 0.75
	auraFlickerSpeed = 0.55
	crosshairRadius  = 11
	mouseAimDeadzone = 6
)

var (
	auraGlowScales = [maximumStatusLevel + 1]float32{0, 1.6, 2.1, 2.6}
	auraGlowAlphas = [maximumStatusLevel + 1]float32{0, 0.16, 0.28, 0.42}
	shadowColor    = [3]float32{0, 0, 0}
	hurtColor      = [3]float32{1, 0.25, 0.25}
	fireCellGlow   = [3]float32{1, 0.5, 0.15}
	crosshairTicks = [4][2]float32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
)

// ===== Public API =====

func NewRenderer(batches *draw.Batches, camera *draw.Camera, groundColumns, groundRows int) *Renderer {
	return &Renderer{
		batches:     batches,
		camera:      camera,
		groundImage: ebiten.NewImage(groundColumns, groundRows),
		wormSprites: NewWormSprites(),
	}
}

// ===== Internal =====

func (r *Renderer) updateCamera(g *Game) {
	r.camera.Center(g.player.X, g.player.Y, arenaSize, arenaSize)
	r.camera.X += g.effects.ShakeX
	r.camera.Y += g.effects.ShakeY
}

func (r *Renderer) drawGround(g *Game, screen *ebiten.Image) {
	g.ground.RefreshPixels(g.frame)
	r.groundImage.WritePixels(g.ground.Pixels)
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(groundCellSize, groundCellSize)
	options.GeoM.Translate(float64(-r.camera.X), float64(-r.camera.Y))
	options.GeoM.Scale(float64(draw.Scale), float64(draw.Scale))
	screen.DrawImage(r.groundImage, options)
}

func (r *Renderer) queueGroundGlow(g *Game) {
	firstColumn := clampInt(int(r.camera.X)/groundCellSize-1, 0, g.ground.Columns-1)
	lastColumn := clampInt(int(r.camera.X+screenWidth)/groundCellSize+1, 0, g.ground.Columns-1)
	firstRow := clampInt(int(r.camera.Y)/groundCellSize-1, 0, g.ground.Rows-1)
	lastRow := clampInt(int(r.camera.Y+screenHeight)/groundCellSize+1, 0, g.ground.Rows-1)
	for row := firstRow; row <= lastRow; row++ {
		r.queueGroundRowGlow(g, row, firstColumn, lastColumn)
	}
}

func (r *Renderer) queueGroundRowGlow(g *Game, row, firstColumn, lastColumn int) {
	for column := firstColumn; column <= lastColumn; column++ {
		isFire := g.ground.MaterialAtCell(column, row) == GroundFire
		screenX, screenY := r.camera.ToScreen(float32(column*groundCellSize+groundCellSize/2), float32(row*groundCellSize+groundCellSize/2))
		r.batches.Glow.AddCircleIf(screenX, screenY, groundCellSize*1.6, fireCellGlow, 0.22, isFire)
	}
}

func (r *Renderer) queueGems(g *Game) {
	gems := g.gems
	for index := range gems.Count {
		tier := GemTierFor(gems.Value[index])
		screenX, screenY := r.camera.ToScreen(gems.X[index], gems.Y[index])
		isVisible := r.camera.IsVisible(gems.X[index], gems.Y[index], tier.Radius)
		r.batches.Solid.AddCircleIf(screenX, screenY, tier.Radius, tier.Color, 1, isVisible)
		r.batches.Glow.AddCircleIf(screenX, screenY, tier.Radius*3, tier.Color, 0.25, isVisible)
	}
}

func (r *Renderer) queueEnemies(g *Game) {
	for index := range g.enemies.Count {
		r.queueEnemyShadow(g, index)
	}
	for index := range g.enemies.Count {
		r.queueEnemyBody(g, index)
	}
}

func (r *Renderer) queueEnemyShadow(g *Game, index int) {
	enemies := g.enemies
	definition := &enemyTable[enemies.Kind[index]]
	x, y := enemies.PositionX[index], enemies.PositionY[index]
	screenX, screenY := r.camera.ToScreen(x, y)
	isDrawn := r.camera.IsVisible(x, y, definition.Radius) && !definition.IsCustomDrawn
	r.batches.Solid.AddCircleIf(screenX+2, screenY+definition.Radius*0.45, definition.Radius*1.05, shadowColor, 0.35, isDrawn)
}

func (r *Renderer) queueEnemyBody(g *Game, index int) {
	enemies := g.enemies
	radius := enemyTable[enemies.Kind[index]].Radius
	x, y := enemies.PositionX[index], enemies.PositionY[index]
	if !r.camera.IsVisible(x, y, radius) || enemyTable[enemies.Kind[index]].IsCustomDrawn {
		return
	}
	screenX, screenY := r.camera.ToScreen(x, y)
	tint := enemies.TintedColor(index)
	directionX, directionY := normalize(g.player.X-x, g.player.Y-y)
	r.queueStatusAuras(g, index, screenX, screenY, radius)
	r.batches.Solid.AddCircle(screenX, screenY, radius, tint, 1)
	r.batches.Solid.AddCircleIf(screenX+directionX*radius*0.4, screenY+directionY*radius*0.4, radius*0.38, mixColor(tint, [3]float32{1, 1, 1}, 0.5), 0.9, radius >= 12)
}

func (r *Renderer) queueStatusAuras(g *Game, index int, screenX, screenY, radius float32) {
	enemies := g.enemies
	flicker := auraFlickerBase + (1-auraFlickerBase)*sine(float32(g.frame)*auraFlickerSpeed+float32(index))
	for _, statusIndex := range leveledStatuses {
		hasStatus := enemies.Status[index]&(1<<statusIndex) != 0
		level := enemies.Levels[statusIndex][index] * uint8(boolToIndex(hasStatus))
		color := statusTable[statusIndex].AuraColor
		r.batches.Glow.AddCircleIf(screenX, screenY, radius*auraGlowScales[level], color, auraGlowAlphas[level]*flicker, level > 0)
		r.batches.Solid.AddCircleIf(screenX, screenY, radius+auraRimWidth, color, flicker, level == maximumStatusLevel)
	}
}

func (r *Renderer) queueProjectiles(g *Game) {
	projectiles := g.projectiles
	for index := range projectiles.Count {
		screenX, screenY := r.camera.ToScreen(projectiles.X[index], projectiles.Y[index])
		lift := projectiles.ArcHeight(index)
		isFlask := projectiles.Kind[index] == ProjectileFlask
		r.batches.Solid.AddCircleIf(screenX, screenY, 5, shadowColor, 0.35, isFlask)
		r.batches.Solid.AddCircle(screenX, screenY-lift, 4.5+2*boolToFloat(isFlask), mixColor(projectiles.Color[index], [3]float32{1, 1, 1}, 0.6), 1)
		r.batches.Glow.AddCircle(screenX, screenY-lift, 16, projectiles.Color[index], 0.75)
	}
}

func (r *Renderer) queuePlayer(g *Game) {
	player := g.player
	screenX, screenY := r.camera.ToScreen(player.X, player.Y)
	r.batches.Glow.AddCircle(screenX, screenY, playerRadius*4, [3]float32{0.4, 0.7, 1}, 0.14)
	r.queueTachikoma(g)
	r.queueAimReticle(player, screenX, screenY)
	for blade := range player.BladeCount {
		bladeX, bladeY := r.camera.ToScreen(player.BladePosition(blade))
		r.batches.Solid.AddCircle(bladeX, bladeY, 7, weaponTable[WeaponOrbitBlades].Color, 1)
		r.batches.Glow.AddCircle(bladeX, bladeY, 20, [3]float32{0.8, 0.85, 1}, 0.45)
	}
}

func (r *Renderer) queueMouseCrosshair(g *Game) {
	isShown := g.controls.HasMouseAim && !g.isDemo
	if !isShown {
		return
	}
	x, y := g.controls.CursorX/draw.Scale, g.controls.CursorY/draw.Scale
	tint := [2][3]float32{{0.9, 0.95, 1}, {1, 0.75, 0.3}}[boolToIndex(g.player.IsFiring)]
	r.batches.Rings.AddCircle(x, y, crosshairRadius, tint, 0.85)
	r.batches.Solid.AddCircle(x, y, 1.8, tint, 1)
	for _, offset := range crosshairTicks {
		r.batches.Solid.AddSegment(x+offset[0]*crosshairRadius*0.55, y+offset[1]*crosshairRadius*0.55, x+offset[0]*crosshairRadius*1.25, y+offset[1]*crosshairRadius*1.25, 2, tint, 0.9)
	}
}

func (r *Renderer) queueAimReticle(player *Player, screenX, screenY float32) {
	reticleColors := [2][3]float32{{0.9, 0.95, 1}, {1, 0.8, 0.25}}
	tint := reticleColors[boolToIndex(player.IsAimLocked)]
	alpha := 0.45 + 0.4*boolToFloat(player.IsFiring)
	for dot := range 3 {
		distance := float32(32 + dot*12)
		r.batches.Solid.AddCircle(screenX+player.AimX*distance, screenY+player.AimY*distance, 2.6-float32(dot)*0.4, tint, alpha)
	}
}
