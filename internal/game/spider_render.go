package game

// ===== Constants =====

const (
	spiderKneeHeight    = 16
	spiderShadowOffsetX = 7
	spiderShadowOffsetY = 11
)

var laserDrawers = [laserPhaseCount]func(renderer *Renderer, laser *SpiderLaser, frame uint32){
	LaserCooldown: func(*Renderer, *SpiderLaser, uint32) {},
	LaserCharging: (*Renderer).drawTargetingBeam,
	LaserLocked:   (*Renderer).drawLockedBeam,
	LaserFiring:   (*Renderer).drawFiringBeam,
}

var (
	jointColor    = [3]float32{0.16, 0.17, 0.18}
	sensorRed     = [3]float32{1, 0.18, 0.12}
	blackColor    = [3]float32{0, 0, 0}
	whiteColor    = [3]float32{1, 1, 1}
	shadowOpacity = float32(0.32)
)

// ===== Internal =====

func (r *Renderer) queueSpiders(g *Game) {
	for index := range g.spiders {
		r.queueSpider(g, &g.spiders[index])
	}
}

func (r *Renderer) queueSpider(g *Game, rig *SpiderRig) {
	enemyIndex := g.enemies.IndexOfID(rig.EnemyID)
	if enemyIndex < 0 || !r.isVisible(rig.X, rig.Y, 140*spiderScale) {
		return
	}
	armor := mixColor(enemyTable[EnemySpiderTank].Color, g.enemies.TintedColor(enemyIndex), 0.5)
	for leg := range spiderLegCount {
		r.queueLegShadow(rig, leg)
	}
	r.queueBodyShadow(rig)
	for leg := range spiderLegCount {
		r.queueLeg(rig, leg, armor)
	}
	r.queueChassis(rig, armor)
	r.queueSensor(rig, g.frame)
}

func (r *Renderer) spiderPoint(rig *SpiderRig, forward, side float32) (float32, float32) {
	return r.ToScreen(rig.LocalToWorld(forward, side))
}

func (r *Renderer) queueLegShadow(rig *SpiderRig, leg int) {
	hipX, hipY := r.ToScreen(rig.HipPosition(leg))
	kneeX, kneeY := r.ToScreen(rig.KneePosition(leg))
	footX, footY := r.ToScreen(rig.Legs[leg].FootX, rig.Legs[leg].FootY)
	offsetX, offsetY := scaled(spiderShadowOffsetX), scaled(spiderShadowOffsetY)
	r.solid.AddSegment(hipX+offsetX, hipY+offsetY, kneeX+offsetX*1.6, kneeY+offsetY*1.6, scaled(9), blackColor, shadowOpacity)
	r.solid.AddSegment(kneeX+offsetX*1.6, kneeY+offsetY*1.6, footX+2, footY+3, scaled(7), blackColor, shadowOpacity)
	r.solid.AddCircle(footX+2, footY+3, scaled(7), blackColor, shadowOpacity)
}

func (r *Renderer) queueBodyShadow(rig *SpiderRig) {
	rearX, rearY := r.spiderPoint(rig, -16, 0)
	frontX, frontY := r.spiderPoint(rig, 20, 0)
	r.solid.AddCircle(rearX+scaled(spiderShadowOffsetX*1.5), rearY+scaled(spiderShadowOffsetY*1.5), scaled(35), blackColor, shadowOpacity)
	r.solid.AddCircle(frontX+scaled(spiderShadowOffsetX*1.5), frontY+scaled(spiderShadowOffsetY*1.5), scaled(25), blackColor, shadowOpacity)
}

func (r *Renderer) queueLeg(rig *SpiderRig, leg int, armor [3]float32) {
	dark := mixColor(armor, blackColor, 0.5)
	light := mixColor(armor, whiteColor, 0.25)
	lift := rig.FootLift(leg)
	hipX, hipY := r.ToScreen(rig.HipPosition(leg))
	kneeX, kneeY := r.ToScreen(rig.KneePosition(leg))
	kneeY -= scaled(spiderKneeHeight) + lift*0.5
	footX, footY := r.ToScreen(rig.Legs[leg].FootX, rig.Legs[leg].FootY)
	footY -= lift
	r.solid.AddSegment(hipX, hipY, kneeX, kneeY, scaled(15), dark, 1)
	r.solid.AddSegment(hipX, hipY, kneeX, kneeY, scaled(9), armor, 1)
	r.solid.AddSegment(kneeX, kneeY, footX, footY, scaled(11), dark, 1)
	r.solid.AddSegment(kneeX, kneeY, footX, footY, scaled(5), light, 1)
	r.solid.AddCircle(kneeX, kneeY, scaled(6.5), jointColor, 1)
	r.solid.AddCircle(kneeX, kneeY, scaled(3), light, 1)
	r.queueFoot(rig, leg, footX, footY, lift, dark, armor)
}

func (r *Renderer) queueFoot(rig *SpiderRig, leg int, footX, footY, lift float32, dark, armor [3]float32) {
	outwardX, outwardY := normalize(rig.Legs[leg].FootX-rig.X, rig.Legs[leg].FootY-rig.Y)
	padRadius := scaled(8) + lift*0.18
	r.solid.AddSegment(footX, footY, footX+outwardX*scaled(14), footY+outwardY*scaled(14), scaled(4), jointColor, 1)
	r.solid.AddCircle(footX, footY, padRadius, dark, 1)
	r.solid.AddCircle(footX, footY, padRadius*0.55, armor, 1)
}

func (r *Renderer) queueChassis(rig *SpiderRig, armor [3]float32) {
	dark := mixColor(armor, blackColor, 0.5)
	light := mixColor(armor, whiteColor, 0.28)
	r.addLocalSegment(rig, -2, 0, spiderTurretPivot, 0, scaled(26), dark)
	r.addLocalCircle(rig, -16, 0, scaled(34), dark)
	r.addLocalCircle(rig, -16, 0, scaled(30), armor)
	r.addLocalSegment(rig, -40, -13, -2, -13, scaled(2.5), dark)
	r.addLocalSegment(rig, -40, 13, -2, 13, scaled(2.5), dark)
	r.addLocalSegment(rig, -44, 0, -30, 0, scaled(3), dark)
	r.addLocalCircle(rig, -16, 0, scaled(10), light)
	for leg := range spiderLegCount {
		layout := &spiderLegLayouts[leg]
		r.addLocalCircle(rig, layout.HipForward, layout.HipSide, scaled(8), jointColor)
		r.addLocalCircle(rig, layout.HipForward, layout.HipSide, scaled(4), light)
	}
	r.queueTurretBearing(rig, dark, light)
	r.queueTurret(rig, armor, dark, light)
}

func (r *Renderer) queueTurretBearing(rig *SpiderRig, dark, light [3]float32) {
	r.addLocalCircle(rig, spiderTurretPivot, 0, scaled(17), jointColor)
	r.addLocalCircle(rig, spiderTurretPivot, 0, scaled(15), dark)
	for notch := range 8 {
		r.addTurretCircle(rig, cosine(float32(notch)*0.785)*13, sine(float32(notch)*0.785)*13, scaled(1.3), light)
	}
}

func (r *Renderer) queueTurret(rig *SpiderRig, armor, dark, light [3]float32) {
	r.addTurretSegment(rig, 14, 0, 60, 0, scaled(8), dark)
	r.addTurretSegment(rig, 14, 0, 58, 0, scaled(3.5), jointColor)
	r.addTurretCircle(rig, 60, 0, scaled(4.5), dark)
	r.addTurretSegment(rig, 36, -5, 36, 5, scaled(3), jointColor)
	r.addTurretCircle(rig, 12, 0, scaled(24), dark)
	r.addTurretCircle(rig, 12, 0, scaled(20), light)
	r.addTurretCircle(rig, 6, 0, scaled(9), armor)
	r.addTurretSegment(rig, 23, -13, 23, 13, scaled(5), jointColor)
}

func (r *Renderer) addTurretCircle(rig *SpiderRig, forward, side, radius float32, tint [3]float32) {
	x, y := r.ToScreen(rig.TurretToWorld(forward, side))
	r.solid.AddCircle(x, y, radius, tint, 1)
}

func (r *Renderer) addTurretSegment(rig *SpiderRig, fromForward, fromSide, toForward, toSide, width float32, tint [3]float32) {
	fromX, fromY := r.ToScreen(rig.TurretToWorld(fromForward, fromSide))
	toX, toY := r.ToScreen(rig.TurretToWorld(toForward, toSide))
	r.solid.AddSegment(fromX, fromY, toX, toY, width, tint, 1)
}

func (r *Renderer) queueSensor(rig *SpiderRig, frame uint32) {
	flicker := 0.75 + 0.25*sine(float32(frame)*0.3)
	sensorX, sensorY := r.ToScreen(rig.TurretToWorld(28, 0))
	r.glow.AddCircle(sensorX, sensorY, scaled(22), sensorRed, 0.7*flicker)
	r.solid.AddCircle(sensorX, sensorY, scaled(4.5), sensorRed, 1)
	r.solid.AddCircle(sensorX, sensorY, scaled(1.8), whiteColor, flicker)
}

func (r *Renderer) addLocalCircle(rig *SpiderRig, forward, side, radius float32, tint [3]float32) {
	x, y := r.spiderPoint(rig, forward, side)
	r.solid.AddCircle(x, y, radius, tint, 1)
}

func (r *Renderer) addLocalSegment(rig *SpiderRig, fromForward, fromSide, toForward, toSide, width float32, tint [3]float32) {
	fromX, fromY := r.spiderPoint(rig, fromForward, fromSide)
	toX, toY := r.spiderPoint(rig, toForward, toSide)
	r.solid.AddSegment(fromX, fromY, toX, toY, width, tint, 1)
}

func (r *Renderer) queueSpiderLasers(g *Game) {
	for index := range g.spiders {
		laser := &g.spiders[index].Laser
		laserDrawers[laser.Phase](r, laser, g.frame)
	}
}

func (r *Renderer) drawTargetingBeam(laser *SpiderLaser, frame uint32) {
	progress := laser.PhaseProgress()
	flicker := 0.7 + 0.3*sine(float32(frame)*0.9)
	r.queueBeam(laser, 8, laserTargetingColor, (0.12+0.25*progress)*flicker)
	r.queueBeam(laser, 1.6, laserCoreColor, (0.5+0.5*progress)*flicker)
	sphereRadius := chargeSphereMinimum + chargeSphereGrowth*progress + chargeSphereSwell(laser)
	r.drawChargeMotes(laser, sphereRadius)
	r.drawChargeSphere(laser, sphereRadius, 0.6+0.4*progress)
}

func (r *Renderer) drawLockedBeam(laser *SpiderLaser, frame uint32) {
	blink := 0.45 + 0.55*float32((frame/3)%2)
	r.queueBeam(laser, 14, laserTargetingColor, 0.4*blink)
	r.queueBeam(laser, 3, laserCoreColor, blink)
	sphereRadius := chargeSphereMinimum + chargeSphereGrowth + chargeSphereSwell(laser)
	flicker := 1 - dischargeSphereFlicker*float32((frame/2)%2)
	r.drawChargeMotes(laser, sphereRadius)
	r.drawChargeSphere(laser, sphereRadius, flicker)
}

func (r *Renderer) drawFiringBeam(laser *SpiderLaser, frame uint32) {
	fade := clamp(laser.Timer/laserFireSeconds, 0, 1)
	pulse := 1 + 0.15*sine(float32(frame)*1.7)
	r.queueBeam(laser, 96*pulse, laserBeamColor, 0.22*fade)
	r.queueBeam(laser, 56*pulse, laserBeamColor, 0.4*fade)
	r.queueBeam(laser, 28*pulse, [3]float32{1, 0.85, 0.4}, 0.75*fade)
	r.queueBeam(laser, 10*pulse, laserCoreColor, fade)
	originX, originY := r.ToScreen(laser.OriginX, laser.OriginY)
	r.glow.AddCircle(originX, originY, 80*pulse, laserCoreColor, 0.8*fade)
}

func (r *Renderer) queueBeam(laser *SpiderLaser, width float32, tint [3]float32, alpha float32) {
	directionX, directionY := laser.Direction()
	originX, originY := r.ToScreen(laser.OriginX, laser.OriginY)
	r.glow.AddSegment(originX, originY, originX+directionX*laserRange, originY+directionY*laserRange, width, tint, alpha)
}

func scaled(value float32) float32 {
	return value * spiderScale
}
