package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// ===== Types =====

type SpriteBatch struct {
	texture     *ebiten.Image
	textureSize float32
	vertices    []ebiten.Vertex
	indices     []uint32
	options     ebiten.DrawTrianglesOptions
}

type Renderer struct {
	solid       *SpriteBatch
	glow        *SpriteBatch
	rings       *SpriteBatch
	groundImage *ebiten.Image
	wormSprites *WormSprites
	CameraX     float32
	CameraY     float32
}

// ===== Constants =====

const (
	spriteTextureSize = 64
	ringTextureSize   = 256
	batchQuadCapacity = 32768
	cullMargin        = 40
	auraRimWidth      = 1.2
	auraFlickerBase   = 0.75
	auraFlickerSpeed  = 0.55
	smokeOpacity      = 0.5
	crosshairRadius   = 11
	mouseAimDeadzone  = 6
)

var (
	auraGlowScales = [maximumStatusLevel + 1]float32{0, 1.6, 2.1, 2.6}
	auraGlowAlphas = [maximumStatusLevel + 1]float32{0, 0.16, 0.28, 0.42}
	shadowColor    = [3]float32{0, 0, 0}
	hurtColor      = [3]float32{1, 0.25, 0.25}
	fireCellGlow   = [3]float32{1, 0.5, 0.15}
	crosshairTicks = [4][2]float32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	smokeColor     = [3]float32{0.2, 0.2, 0.22}
)

// ===== Public API =====

func NewRenderer(groundColumns, groundRows int) *Renderer {
	return &Renderer{
		solid:       NewSpriteBatch(newRadialTexture(spriteTextureSize, solidFalloff), ebiten.BlendSourceOver),
		glow:        NewSpriteBatch(newRadialTexture(spriteTextureSize, glowFalloff), ebiten.BlendLighter),
		rings:       NewSpriteBatch(newRadialTexture(ringTextureSize, ringFalloff), ebiten.BlendLighter),
		groundImage: ebiten.NewImage(groundColumns, groundRows),
		wormSprites: NewWormSprites(),
	}
}

func NewSpriteBatch(texture *ebiten.Image, blend ebiten.Blend) *SpriteBatch {
	batch := &SpriteBatch{
		texture:     texture,
		textureSize: float32(texture.Bounds().Dx()),
		vertices:    make([]ebiten.Vertex, 0, batchQuadCapacity*4),
		indices:     make([]uint32, 0, batchQuadCapacity*6),
	}
	batch.options.Blend = blend
	return batch
}

func (b *SpriteBatch) AddCircle(x, y, radius float32, tint [3]float32, alpha float32) {
	base := uint32(len(b.vertices))
	for corner := range 4 {
		offsetX, offsetY := float32(corner&1), float32(corner>>1)
		b.vertices = append(b.vertices, ebiten.Vertex{
			DstX: x + (offsetX*2-1)*radius, DstY: y + (offsetY*2-1)*radius,
			SrcX: offsetX * b.textureSize, SrcY: offsetY * b.textureSize,
			ColorR: tint[0], ColorG: tint[1], ColorB: tint[2], ColorA: alpha,
		})
	}
	b.indices = append(b.indices, base, base+1, base+2, base+1, base+3, base+2)
}

func (b *SpriteBatch) AddEllipse(x, y, radiusAlong, radiusAcross, directionX, directionY float32, tint [3]float32, alpha float32) {
	base := uint32(len(b.vertices))
	alongX, alongY := directionX*radiusAlong, directionY*radiusAlong
	acrossX, acrossY := -directionY*radiusAcross, directionX*radiusAcross
	for corner := range 4 {
		offsetX, offsetY := float32(corner&1), float32(corner>>1)
		alongSign, acrossSign := offsetX*2-1, offsetY*2-1
		b.vertices = append(b.vertices, ebiten.Vertex{
			DstX: x + alongX*alongSign + acrossX*acrossSign, DstY: y + alongY*alongSign + acrossY*acrossSign,
			SrcX: offsetX * b.textureSize, SrcY: offsetY * b.textureSize,
			ColorR: tint[0], ColorG: tint[1], ColorB: tint[2], ColorA: alpha,
		})
	}
	b.indices = append(b.indices, base, base+1, base+2, base+1, base+3, base+2)
}

func (b *SpriteBatch) AddSegment(fromX, fromY, toX, toY, width float32, tint [3]float32, alpha float32) {
	directionX, directionY := normalize(toX-fromX, toY-fromY)
	normalX, normalY := -directionY*width/2, directionX*width/2
	center := b.textureSize / 2
	base := uint32(len(b.vertices))
	corners := [4][2]float32{
		{fromX + normalX, fromY + normalY}, {toX + normalX, toY + normalY},
		{fromX - normalX, fromY - normalY}, {toX - normalX, toY - normalY},
	}
	for _, corner := range corners {
		b.vertices = append(b.vertices, ebiten.Vertex{
			DstX: corner[0], DstY: corner[1], SrcX: center, SrcY: center,
			ColorR: tint[0], ColorG: tint[1], ColorB: tint[2], ColorA: alpha,
		})
	}
	b.indices = append(b.indices, base, base+1, base+2, base+1, base+3, base+2)
}

func (b *SpriteBatch) AddCircleIf(x, y, radius float32, tint [3]float32, alpha float32, shouldAdd bool) {
	if !shouldAdd {
		return
	}
	b.AddCircle(x, y, radius, tint, alpha)
}

func (b *SpriteBatch) Flush(screen *ebiten.Image) {
	if len(b.indices) == 0 {
		return
	}
	for index := range b.vertices {
		b.vertices[index].DstX *= renderScale
		b.vertices[index].DstY *= renderScale
	}
	screen.DrawTriangles32(b.vertices, b.indices, b.texture, &b.options)
	b.vertices, b.indices = b.vertices[:0], b.indices[:0]
}

func (r *Renderer) DrawWorld(g *Game, screen *ebiten.Image) {
	r.updateCamera(g)
	r.drawGround(g, screen)
	r.queueGroundGlow(g)
	r.queueSkidMarks(g)
	r.queueGems(g)
	r.queueHealOrbs(g)
	r.queueMines(g)
	r.queueSandwormGround(g)
	r.queueEnemies(g)
	r.queuePlayer(g)
	r.queueSpiders(g)
	r.queueSandworm(g)
	r.queueProjectiles(g)
	r.queueSmokes(g)
	r.queueParticles(g)
	r.queueSpiderBeams(g)
	r.queueRings(g)
	r.queueLightning(g)
	r.queuePrecipitation(g)
	r.queueMouseCrosshair(g)
	r.solid.Flush(screen)
	r.drawSandwormSprites(g, screen)
	r.glow.Flush(screen)
	r.rings.Flush(screen)
	r.applyWeatherGrade(g, screen)
	g.post.ApplyBloom(bloomLevels[g.settings.BloomLevel].Factor)
}

func (r *Renderer) ToScreen(x, y float32) (float32, float32) {
	return x - r.CameraX, y - r.CameraY
}

func (r *Renderer) ScreenToWorld(x, y float32) (float32, float32) {
	return x + r.CameraX, y + r.CameraY
}

// ===== Internal =====

func newRadialTexture(size int, falloff func(distance, half float32) float32) *ebiten.Image {
	pixels := make([]byte, size*size*4)
	half := float32(size) / 2
	for index := range size * size {
		offsetX := float32(index%size) + 0.5 - half
		offsetY := float32(index/size) + 0.5 - half
		intensity := byte(falloff(length(offsetX, offsetY)/half, half) * 255)
		copy(pixels[index*4:], []byte{intensity, intensity, intensity, intensity})
	}
	texture := ebiten.NewImage(size, size)
	texture.WritePixels(pixels)
	return texture
}

func solidFalloff(distance, half float32) float32 {
	return clamp((1-distance)*half, 0, 1)
}

func ringFalloff(distance, half float32) float32 {
	bandCenter := 1 - 6/half
	return clamp(4.5-abs(distance-bandCenter)*half, 0, 1)
}

func glowFalloff(distance, half float32) float32 {
	remaining := clamp(1-distance, 0, 1)
	return remaining * remaining
}

func (r *Renderer) updateCamera(g *Game) {
	shakeX, shakeY := g.effects.ShakeOffset()
	r.CameraX = clamp(g.player.X-screenWidth/2, 0, arenaSize-screenWidth) + shakeX
	r.CameraY = clamp(g.player.Y-screenHeight/2, 0, arenaSize-screenHeight) + shakeY
}

func (r *Renderer) isVisible(x, y, radius float32) bool {
	screenX, screenY := r.ToScreen(x, y)
	reach := radius + cullMargin
	return screenX > -reach && screenY > -reach && screenX < screenWidth+reach && screenY < screenHeight+reach
}

func (r *Renderer) drawGround(g *Game, screen *ebiten.Image) {
	g.ground.RefreshPixels(g.frame)
	r.groundImage.WritePixels(g.ground.Pixels)
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(groundCellSize, groundCellSize)
	options.GeoM.Translate(float64(-r.CameraX), float64(-r.CameraY))
	options.GeoM.Scale(float64(renderScale), float64(renderScale))
	screen.DrawImage(r.groundImage, options)
}

func (r *Renderer) queueGroundGlow(g *Game) {
	firstColumn := clampInt(int(r.CameraX)/groundCellSize-1, 0, g.ground.Columns-1)
	lastColumn := clampInt(int(r.CameraX+screenWidth)/groundCellSize+1, 0, g.ground.Columns-1)
	firstRow := clampInt(int(r.CameraY)/groundCellSize-1, 0, g.ground.Rows-1)
	lastRow := clampInt(int(r.CameraY+screenHeight)/groundCellSize+1, 0, g.ground.Rows-1)
	for row := firstRow; row <= lastRow; row++ {
		r.queueGroundRowGlow(g, row, firstColumn, lastColumn)
	}
}

func (r *Renderer) queueGroundRowGlow(g *Game, row, firstColumn, lastColumn int) {
	for column := firstColumn; column <= lastColumn; column++ {
		isFire := g.ground.cells[row*g.ground.Columns+column] == GroundFire
		screenX, screenY := r.ToScreen(float32(column*groundCellSize+groundCellSize/2), float32(row*groundCellSize+groundCellSize/2))
		r.glow.AddCircleIf(screenX, screenY, groundCellSize*1.6, fireCellGlow, 0.22, isFire)
	}
}

func (r *Renderer) queueGems(g *Game) {
	gems := g.gems
	for index := range gems.Count {
		tier := GemTierFor(gems.Value[index])
		screenX, screenY := r.ToScreen(gems.X[index], gems.Y[index])
		isVisible := r.isVisible(gems.X[index], gems.Y[index], tier.Radius)
		r.solid.AddCircleIf(screenX, screenY, tier.Radius, tier.Color, 1, isVisible)
		r.glow.AddCircleIf(screenX, screenY, tier.Radius*3, tier.Color, 0.25, isVisible)
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
	screenX, screenY := r.ToScreen(x, y)
	isDrawn := r.isVisible(x, y, definition.Radius) && !definition.IsCustomDrawn
	r.solid.AddCircleIf(screenX+2, screenY+definition.Radius*0.45, definition.Radius*1.05, shadowColor, 0.35, isDrawn)
}

func (r *Renderer) queueEnemyBody(g *Game, index int) {
	enemies := g.enemies
	radius := enemyTable[enemies.Kind[index]].Radius
	x, y := enemies.PositionX[index], enemies.PositionY[index]
	if !r.isVisible(x, y, radius) || enemyTable[enemies.Kind[index]].IsCustomDrawn {
		return
	}
	screenX, screenY := r.ToScreen(x, y)
	tint := enemies.TintedColor(index)
	directionX, directionY := normalize(g.player.X-x, g.player.Y-y)
	r.queueStatusAuras(g, index, screenX, screenY, radius)
	r.solid.AddCircle(screenX, screenY, radius, tint, 1)
	r.solid.AddCircleIf(screenX+directionX*radius*0.4, screenY+directionY*radius*0.4, radius*0.38, mixColor(tint, [3]float32{1, 1, 1}, 0.5), 0.9, radius >= 12)
}

func (r *Renderer) queueStatusAuras(g *Game, index int, screenX, screenY, radius float32) {
	enemies := g.enemies
	flicker := auraFlickerBase + (1-auraFlickerBase)*sine(float32(g.frame)*auraFlickerSpeed+float32(index))
	for _, statusIndex := range leveledStatuses {
		hasStatus := enemies.Status[index]&(1<<statusIndex) != 0
		level := enemies.StatusLevels[statusIndex][index] * uint8(boolToIndex(hasStatus))
		color := statusTable[statusIndex].AuraColor
		r.glow.AddCircleIf(screenX, screenY, radius*auraGlowScales[level], color, auraGlowAlphas[level]*flicker, level > 0)
		r.solid.AddCircleIf(screenX, screenY, radius+auraRimWidth, color, flicker, level == maximumStatusLevel)
	}
}

func (r *Renderer) queueProjectiles(g *Game) {
	projectiles := g.projectiles
	for index := range projectiles.Count {
		screenX, screenY := r.ToScreen(projectiles.X[index], projectiles.Y[index])
		lift := projectiles.ArcHeight(index)
		isFlask := projectiles.Kind[index] == ProjectileFlask
		r.solid.AddCircleIf(screenX, screenY, 5, shadowColor, 0.35, isFlask)
		r.solid.AddCircle(screenX, screenY-lift, 4.5+2*boolToFloat(isFlask), mixColor(projectiles.Color[index], [3]float32{1, 1, 1}, 0.6), 1)
		r.glow.AddCircle(screenX, screenY-lift, 16, projectiles.Color[index], 0.75)
	}
}

func (r *Renderer) queuePlayer(g *Game) {
	player := g.player
	screenX, screenY := r.ToScreen(player.X, player.Y)
	r.glow.AddCircle(screenX, screenY, playerRadius*4, [3]float32{0.4, 0.7, 1}, 0.14)
	r.queueTachikoma(g)
	r.queueAimReticle(player, screenX, screenY)
	for blade := range player.BladeCount {
		bladeX, bladeY := r.ToScreen(player.BladePosition(blade))
		r.solid.AddCircle(bladeX, bladeY, 7, weaponTable[WeaponOrbitBlades].Color, 1)
		r.glow.AddCircle(bladeX, bladeY, 20, [3]float32{0.8, 0.85, 1}, 0.45)
	}
}

func (r *Renderer) queueMouseCrosshair(g *Game) {
	isShown := g.controls.HasMouseAim && !g.isDemo
	if !isShown {
		return
	}
	x, y := g.controls.CursorX/renderScale, g.controls.CursorY/renderScale
	tint := [2][3]float32{{0.9, 0.95, 1}, {1, 0.75, 0.3}}[boolToIndex(g.player.IsFiring)]
	r.rings.AddCircle(x, y, crosshairRadius, tint, 0.85)
	r.solid.AddCircle(x, y, 1.8, tint, 1)
	for _, offset := range crosshairTicks {
		r.solid.AddSegment(x+offset[0]*crosshairRadius*0.55, y+offset[1]*crosshairRadius*0.55, x+offset[0]*crosshairRadius*1.25, y+offset[1]*crosshairRadius*1.25, 2, tint, 0.9)
	}
}

func (r *Renderer) queueAimReticle(player *Player, screenX, screenY float32) {
	reticleColors := [2][3]float32{{0.9, 0.95, 1}, {1, 0.8, 0.25}}
	tint := reticleColors[boolToIndex(player.IsAimLocked)]
	alpha := 0.45 + 0.4*boolToFloat(player.IsFiring)
	for dot := range 3 {
		distance := float32(32 + dot*12)
		r.solid.AddCircle(screenX+player.AimX*distance, screenY+player.AimY*distance, 2.6-float32(dot)*0.4, tint, alpha)
	}
}

func (r *Renderer) queueSmokes(g *Game) {
	for index := range g.effects.Smokes {
		smoke := &g.effects.Smokes[index]
		screenX, screenY := r.ToScreen(smoke.X, smoke.Y)
		fade := smoke.Life / smoke.MaxLife
		r.solid.AddCircle(screenX, screenY, smoke.Size*(2-fade), smokeColor, smokeOpacity*fade)
	}
}

func (r *Renderer) queueParticles(g *Game) {
	effects := g.effects
	for index := range effects.ParticleCount {
		screenX, screenY := r.ToScreen(effects.ParticleX[index], effects.ParticleY[index])
		fade := effects.Life[index] / effects.MaxLife[index]
		r.glow.AddCircle(screenX, screenY, effects.Size[index]*(1+fade), effects.Color[index], fade)
	}
}

func (r *Renderer) queueRings(g *Game) {
	for _, ring := range g.effects.Rings {
		progress := 1 - ring.Life/ringLifeSeconds
		screenX, screenY := r.ToScreen(ring.X, ring.Y)
		r.rings.AddCircle(screenX, screenY, ring.Radius*(0.4+0.6*progress), ring.Color, 1-progress)
	}
}

func (r *Renderer) queueLightning(g *Game) {
	for index := range g.effects.Bolts {
		r.queueBolt(&g.effects.Bolts[index])
	}
}

func (r *Renderer) queueBolt(bolt *LightningBolt) {
	fade := bolt.Life / lightningLifeSeconds
	for segment := range lightningSegments {
		fromX, fromY := r.ToScreen(bolt.Points[segment][0], bolt.Points[segment][1])
		toX, toY := r.ToScreen(bolt.Points[segment+1][0], bolt.Points[segment+1][1])
		r.glow.AddSegment(fromX, fromY, toX, toY, 6, [3]float32{1, 1, 0.4}, 0.45*fade)
		r.glow.AddSegment(fromX, fromY, toX, toY, 2, [3]float32{1, 1, 1}, fade)
	}
}

func toColor(rgb [3]float32, alpha float32) color.NRGBA {
	return color.NRGBA{
		R: uint8(clamp(rgb[0], 0, 1) * 255),
		G: uint8(clamp(rgb[1], 0, 1) * 255),
		B: uint8(clamp(rgb[2], 0, 1) * 255),
		A: uint8(clamp(alpha, 0, 1) * 255),
	}
}
