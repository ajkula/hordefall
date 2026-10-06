package game

import (
	"cmp"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
)

// ===== Types =====

type wormPhaseDrawer func(renderer *Renderer, game *Game, worm *Sandworm)

type WormPalette struct {
	Dark     [3]float32
	Mid      [3]float32
	Light    [3]float32
	Specular [3]float32
}

type WormPiece struct {
	X           float32
	Y           float32
	Z           float32
	AxisX       float32
	AxisY       float32
	GroundAxisX float32
	GroundAxisY float32
	GroundSpan  float32
	Health      float32
	EnemyID     uint32
}

// ===== Constants =====

const (
	wormMoundRadius       = 30
	wormSegmentRadius     = 22
	wormHeadRadius        = 32
	wormPebbleCount       = 5
	wormPebbleSpin        = 0.12
	wormCrackCount        = 9
	wormCrackWidth        = 3
	wormWarningPulse      = 0.35
	wormCraterRadius      = 46
	wormCraterRockCount   = 10
	wormShadowOffsetX     = 8
	wormShadowOffsetY     = 11
	wormShadowOpacity     = 0.34
	wormShadowFadeHeight  = 400
	wormOutlineWidth      = 4
	wormSegmentHalfLength = 12
	wormHeadHalfLength    = 22
	wormJointWidthScale   = 1.15
	wormRibSpacing        = 7
	wormGaugeLength       = 30
	wormGaugeHeight       = 5
	wormGaugeLift         = 14
	wormCharring          = 0.7
	wormStatusTint        = 0.35
	wormPerspectiveGain   = 0.0013
)

var wormGroundDrawers = [wormPhaseCount]wormPhaseDrawer{
	WormBurrowing:   (*Renderer).queueWormMound,
	WormWarning:     (*Renderer).queueWormWarning,
	WormEmerging:    (*Renderer).queueWormBodyGround,
	WormRearing:     (*Renderer).queueWormBodyGround,
	WormAiming:      (*Renderer).queueWormBodyGround,
	WormDiving:      (*Renderer).queueWormBodyGround,
	WormHeadAiming:  (*Renderer).queueWormHeadShadow,
	WormHeadRolling: (*Renderer).queueWormHeadShadow,
	WormHeadResting: (*Renderer).queueWormHeadShadow,
}

var wormBodyDrawers = [wormPhaseCount]wormPhaseDrawer{
	WormBurrowing:   func(*Renderer, *Game, *Sandworm) {},
	WormWarning:     func(*Renderer, *Game, *Sandworm) {},
	WormEmerging:    (*Renderer).queueWormBody,
	WormRearing:     (*Renderer).queueWormBody,
	WormAiming:      (*Renderer).queueWormBody,
	WormDiving:      (*Renderer).queueWormBody,
	WormHeadAiming:  (*Renderer).queueWormLoneHead,
	WormHeadRolling: (*Renderer).queueWormLoneHead,
	WormHeadResting: (*Renderer).queueWormLoneHead,
}

var wormArmor = WormPalette{
	Dark:     [3]float32{0.17, 0.19, 0.24},
	Mid:      [3]float32{0.38, 0.42, 0.5},
	Light:    [3]float32{0.62, 0.68, 0.76},
	Specular: [3]float32{0.94, 0.97, 1},
}

var (
	wormArmorColor   = wormArmor.Light
	wormOutlineColor = [3]float32{0.04, 0.04, 0.06}
	wormCharColor    = [3]float32{0.1, 0.08, 0.07}
	wormVentColor    = [3]float32{0.3, 0.85, 1}
	wormSensorColor  = [3]float32{1, 0.15, 0.1}
	wormHoleColor    = [3]float32{0.1, 0.07, 0.05}
	wormRockColor    = [3]float32{0.36, 0.27, 0.17}
	wormGaugeEmpty   = [3]float32{0.05, 0.05, 0.05}
	wormGaugeFull    = [3]float32{0.3, 1, 0.35}
	wormGaugeLow     = [3]float32{1, 0.2, 0.1}
	wormLightX       = float32(-0.6)
	wormLightY       = float32(-0.8)
)

// ===== Internal =====

func (r *Renderer) queueSandwormGround(g *Game) {
	r.queueSandwormLayer(g, &wormGroundDrawers)
}

func (r *Renderer) queueSandworm(g *Game) {
	r.queueSandwormLayer(g, &wormBodyDrawers)
}

func (r *Renderer) queueSandwormLayer(g *Game, drawers *[wormPhaseCount]wormPhaseDrawer) {
	worm := &g.worm
	if !worm.IsActive {
		return
	}
	drawers[worm.Phase](r, g, worm)
}

// ===== Internal: underground =====

func (r *Renderer) queueWormMound(g *Game, worm *Sandworm) {
	screenX, screenY := r.ToScreen(worm.X, worm.Y)
	r.solid.AddCircle(screenX, screenY, wormMoundRadius, mixColor(wormDirtColor, blackColor, 0.35), 0.9)
	r.solid.AddCircle(screenX, screenY-4, wormMoundRadius*0.72, wormDirtColor, 0.95)
	for pebble := range wormPebbleCount {
		angle := float32(pebble)*6.2831853/wormPebbleCount + float32(g.frame)*wormPebbleSpin
		r.solid.AddCircle(screenX+cosine(angle)*wormMoundRadius*0.9, screenY+sine(angle)*wormMoundRadius*0.6, 4, wormRockColor, 0.9)
	}
}

func (r *Renderer) queueWormWarning(g *Game, worm *Sandworm) {
	progress := clamp(worm.PhaseSeconds/wormWarningSeconds, 0, 1)
	screenX, screenY := r.ToScreen(worm.X, worm.Y)
	pulse := 0.5 + 0.5*sine(worm.PhaseSeconds*24)
	r.solid.AddCircle(screenX, screenY, wormEruptionRadius, warningColor, 0.1+wormWarningPulse*progress*pulse)
	for crack := range wormCrackCount {
		angle := float32(crack)*6.2831853/wormCrackCount + sine(float32(crack)*12.9898)*0.4
		reach := wormEruptionRadius * progress * (0.7 + 0.3*abs(sine(float32(crack)*78.233)))
		r.solid.AddSegment(screenX, screenY, screenX+cosine(angle)*reach, screenY+sine(angle)*reach, wormCrackWidth, wormHoleColor, 0.85)
	}
	r.queueWormMound(g, worm)
}

func (r *Renderer) queueWormCrater(x, y float32) {
	screenX, screenY := r.ToScreen(x, y)
	r.solid.AddCircle(screenX, screenY, wormCraterRadius, mixColor(wormDirtColor, blackColor, 0.25), 0.9)
	r.solid.AddCircle(screenX, screenY+3, wormCraterRadius*0.72, wormHoleColor, 1)
	for rock := range wormCraterRockCount {
		angle := float32(rock) * 6.2831853 / wormCraterRockCount
		size := 6 + 3*abs(sine(float32(rock)*7.31))
		rockX, rockY := screenX+cosine(angle)*wormCraterRadius*0.92, screenY+sine(angle)*wormCraterRadius*0.8
		r.solid.AddCircle(rockX, rockY, size+1.5, wormOutlineColor, 0.9)
		r.solid.AddCircle(rockX-1, rockY-1, size, wormRockColor, 1)
	}
}

// ===== Internal: body =====

func (r *Renderer) wormPieces(g *Game, worm *Sandworm) []WormPiece {
	pieces := make([]WormPiece, 0, wormSegmentCount)
	slot := 0
	for index := range worm.Segments {
		segment := &worm.Segments[index]
		along := worm.SlotAlong(slot)
		isShown := segment.IsAlive && worm.IsAlongAboveGround(along)
		slot += boolToIndex(segment.IsAlive)
		if !isShown {
			continue
		}
		_, tangent := worm.Path.At(along)
		pieces = append(pieces, newWormPiece(segment.X, segment.Y, segment.Z, tangent, segment.Health, segment.EnemyID))
	}
	return pieces
}

func newWormPiece(x, y, z float32, tangent Point3, health float32, enemyID uint32) WormPiece {
	axisX, axisY := normalize(tangent.X, tangent.Y-tangent.Z)
	groundX, groundY := normalize(tangent.X+0.0001, tangent.Y)
	return WormPiece{
		X: x, Y: y, Z: z, AxisX: axisX, AxisY: axisY, GroundAxisX: groundX, GroundAxisY: groundY,
		GroundSpan: length(tangent.X, tangent.Y), Health: health, EnemyID: enemyID,
	}
}

func (r *Renderer) wormHeadPiece(worm *Sandworm) WormPiece {
	_, tangent := worm.Path.At(worm.HeadAlong())
	return newWormPiece(worm.HeadX, worm.HeadY, worm.HeadZ, tangent, worm.HeadHealth, worm.HeadEnemyID)
}

func (r *Renderer) queueWormBodyGround(g *Game, worm *Sandworm) {
	r.queueWormCrater(worm.HoleX, worm.HoleY)
	r.queueWormDiveCraterIf(worm, worm.Phase == WormDiving)
	for _, piece := range r.wormPieces(g, worm) {
		r.queueWormShadow(&piece, wormSegmentHalfLength, wormSegmentRadius)
	}
	head := r.wormHeadPiece(worm)
	r.queueWormHeadShadowIf(&head, worm.IsHeadAboveGround())
}

func (r *Renderer) queueWormDiveCraterIf(worm *Sandworm, shouldDraw bool) {
	if !shouldDraw {
		return
	}
	r.queueWormCrater(worm.DiveControls[3].X, worm.DiveControls[3].Y)
}

func (r *Renderer) queueWormBody(g *Game, worm *Sandworm) {
	pieces := r.wormPieces(g, worm)
	joint := r.wormHeadPiece(worm)
	for index := range pieces {
		r.queueWormJoint(&joint, &pieces[index])
		joint = pieces[index]
	}
	r.queueBossBeam(&worm.Beam, g.frame)
}

func (r *Renderer) drawSandwormSprites(g *Game, screen *ebiten.Image) {
	worm := &g.worm
	isShown := worm.IsBodyOut() || worm.IsHeadAlone()
	if !isShown {
		return
	}
	pieces := r.wormPieces(g, worm)
	head := r.wormSpriteHead(worm)
	isHeadShown := worm.IsHeadAboveGround()
	isHeadDrawn := !isHeadShown
	for _, index := range sortedByHeight(pieces) {
		isHeadFirst := !isHeadDrawn && head.Z < pieces[index].Z
		r.drawWormSpriteIf(g, screen, &head, worm.HeadMaximumHealth, r.wormSprites.Heads[:], isHeadFirst)
		isHeadDrawn = isHeadDrawn || isHeadFirst
		r.drawWormSpriteIf(g, screen, &pieces[index], worm.SegmentMaximumHealth, r.wormSprites.Segments[:], true)
	}
	r.drawWormSpriteIf(g, screen, &head, worm.HeadMaximumHealth, r.wormSprites.Heads[:], !isHeadDrawn)
	for index := range pieces {
		r.drawWormGauge(screen, &pieces[index], worm.SegmentMaximumHealth)
	}
}

func (r *Renderer) wormSpriteHead(worm *Sandworm) WormPiece {
	pieces := [2]WormPiece{r.wormHeadPiece(worm), r.wormLoneHeadPiece(worm)}
	return pieces[boolToIndex(worm.IsHeadAlone())]
}

func (r *Renderer) drawWormSpriteIf(g *Game, screen *ebiten.Image, piece *WormPiece, maximumHealth float32, frames []*ebiten.Image, shouldDraw bool) {
	if !shouldDraw {
		return
	}
	frame := frames[wormSpriteAngleIndex(piece.AxisX, piece.AxisY)]
	size := float32(frame.Bounds().Dx())
	scale := wormSpritePixel * wormDepthScale(piece.Z)
	screenX, screenY := r.ToScreen(piece.X, piece.Y-piece.Z)
	options := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	options.GeoM.Translate(float64(-size/2), float64(-size/2))
	options.GeoM.Scale(float64(scale*renderScale), float64(scale*renderScale))
	options.GeoM.Translate(float64(float32(int(screenX))*renderScale), float64(float32(int(screenY))*renderScale))
	tint := r.wormSpriteTint(g, piece, maximumHealth)
	options.ColorScale.Scale(tint[0], tint[1], tint[2], 1)
	screen.DrawImage(frame, options)
}

func (r *Renderer) wormSpriteTint(g *Game, piece *WormPiece, maximumHealth float32) [3]float32 {
	damage := 1 - clamp(piece.Health/maximumHealth, 0, 1)
	enemyIndex := g.enemies.IndexOfID(piece.EnemyID)
	hasEnemy := enemyIndex >= 0 && piece.EnemyID != 0
	status := [2][3]float32{whiteColor, g.enemies.TintedColor(max(0, enemyIndex))}[boolToIndex(hasEnemy)]
	base := mixColor(whiteColor, mixColor(whiteColor, status, wormStatusTint), 1)
	return mixColor(base, wormCharColor, damage*wormCharring)
}

func (r *Renderer) drawWormGauge(screen *ebiten.Image, piece *WormPiece, maximumHealth float32) {
	fraction := clamp(piece.Health/maximumHealth, 0, 1)
	scale := wormDepthScale(piece.Z)
	screenX, screenY := r.ToScreen(piece.X, piece.Y-piece.Z-(wormSegmentRadius+wormGaugeLift)*scale)
	left := float32(int(screenX - wormGaugeLength/2))
	top := float32(int(screenY))
	fillRect(screen, left-1, top-1, wormGaugeLength+2, wormGaugeHeight+2, toColor(wormGaugeEmpty, 0.9))
	fillRect(screen, left, top, wormGaugeLength*fraction, wormGaugeHeight, toColor(mixColor(wormGaugeLow, wormGaugeFull, fraction), 1))
}

func sortedByHeight(pieces []WormPiece) []int {
	order := make([]int, len(pieces))
	for index := range order {
		order[index] = index
	}
	slices.SortStableFunc(order, func(first, second int) int { return cmp.Compare(pieces[first].Z, pieces[second].Z) })
	return order
}

func wormPerspective(height float32) float32 {
	return 1 + height*wormPerspectiveGain
}

func (r *Renderer) queueWormShadow(piece *WormPiece, halfLength, radius float32) {
	screenX, screenY := r.ToScreen(piece.X+wormShadowOffsetX, piece.Y+wormShadowOffsetY)
	reach := halfLength * piece.GroundSpan
	fade := clamp(1-piece.Z/wormShadowFadeHeight, 0.35, 1)
	r.solid.AddSegment(screenX-piece.GroundAxisX*reach, screenY-piece.GroundAxisY*reach, screenX+piece.GroundAxisX*reach, screenY+piece.GroundAxisY*reach, radius*2+2, blackColor, wormShadowOpacity*fade)
	r.solid.AddCircle(screenX, screenY, radius+1, blackColor, wormShadowOpacity*fade*0.6)
}

func (r *Renderer) queueWormJoint(from, to *WormPiece) {
	startX, startY := r.ToScreen(from.X, from.Y-from.Z)
	endX, endY := r.ToScreen(to.X, to.Y-to.Z)
	width := wormSegmentRadius * wormJointWidthScale * wormPerspective((from.Z+to.Z)/2)
	r.solid.AddSegment(startX, startY, endX, endY, width+wormOutlineWidth, wormOutlineColor, 1)
	r.solid.AddSegment(startX, startY, endX, endY, width, wormArmor.Dark, 1)
	axisX, axisY := normalize(endX-startX, endY-startY)
	reach := length(endX-startX, endY-startY)
	for along := float32(wormRibSpacing); along < reach; along += wormRibSpacing {
		ribX, ribY := startX+axisX*along, startY+axisY*along
		r.solid.AddSegment(ribX-axisY*width/2, ribY+axisX*width/2, ribX+axisY*width/2, ribY-axisX*width/2, 2, wormArmor.Mid, 0.9)
	}
}

func (r *Renderer) queueWormHeadShadowIf(head *WormPiece, shouldDraw bool) {
	if !shouldDraw {
		return
	}
	r.queueWormShadow(head, wormHeadHalfLength+14, wormHeadRadius)
}

func (r *Renderer) queueWormHeadShadow(g *Game, worm *Sandworm) {
	head := r.wormLoneHeadPiece(worm)
	r.queueWormHeadShadowIf(&head, true)
}

func (r *Renderer) wormLoneHeadPiece(worm *Sandworm) WormPiece {
	directionX, directionY := worm.Direction()
	return newWormPiece(worm.HeadX, worm.HeadY, 0, Point3{directionX, directionY, 0}, worm.HeadHealth, worm.HeadEnemyID)
}

func (r *Renderer) queueWormLoneHead(g *Game, worm *Sandworm) {
	directionX, directionY := worm.Direction()
	r.queueWormAimLaserIf(worm, directionX, directionY, worm.Phase == WormHeadAiming)
}

func (r *Renderer) queueWormAimLaserIf(worm *Sandworm, directionX, directionY float32, shouldDraw bool) {
	if !shouldDraw {
		return
	}
	isBlinking := worm.PhaseSeconds > wormHeadAimSeconds-wormHeadLockSeconds
	alpha := 1 - 0.6*boolToFloat(isBlinking && int(worm.PhaseSeconds*20)%2 == 0)
	originX, originY := r.ToScreen(worm.HeadX+directionX*wormMuzzleForward, worm.HeadY+directionY*wormMuzzleForward)
	endX, endY := originX+directionX*wormRollDistance, originY+directionY*wormRollDistance
	r.glow.AddSegment(originX, originY, endX, endY, 7, wormSensorColor, 0.25*alpha)
	r.glow.AddSegment(originX, originY, endX, endY, 2, wormSensorColor, 0.95*alpha)
}
