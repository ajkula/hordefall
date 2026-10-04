package main

// ===== Types =====

type LaserPhase uint8

type SpiderLaser struct {
	Phase     LaserPhase
	Timer     float32
	BeamAngle float32
	OriginX   float32
	OriginY   float32
	Motes     [chargeMoteCount]ChargeMote
	NextMote  int
}

type laserPhaseBehavior func(game *Game, rig *SpiderRig, deltaSeconds float32)

// ===== Constants =====

const (
	LaserCooldown LaserPhase = iota
	LaserCharging
	LaserLocked
	LaserFiring
	laserPhaseCount
)

const (
	laserFirstShotDelay      = 1
	laserCooldownMinimum     = 6
	laserCooldownMaximum     = 9
	laserChargeSeconds       = 2.2
	laserLockSeconds         = 0.4
	laserFireSeconds         = 0.35
	laserRange               = 1700
	laserMuzzleForward       = 60
	laserBeamHalfWidth       = 20
	laserPlayerDamagePerSec  = 140
	laserEnemyDamagePerSec   = 420
	laserGroundSampleSpacing = 18
)

var laserPhaseDurations = [laserPhaseCount]float32{
	LaserCharging: laserChargeSeconds,
	LaserLocked:   laserLockSeconds,
	LaserFiring:   laserFireSeconds,
}

var laserPhaseBehaviors = [laserPhaseCount]laserPhaseBehavior{
	LaserCooldown: func(*Game, *SpiderRig, float32) {},
	LaserCharging: (*Game).chargeLaser,
	LaserLocked:   (*Game).holdLaser,
	LaserFiring:   (*Game).fireLaser,
}

var laserPhaseSounds = [laserPhaseCount]SoundKind{
	LaserCharging: SoundLaserCharge,
	LaserFiring:   SoundLaserFire,
}

var (
	laserTargetingColor = [3]float32{1, 0.25, 0.15}
	laserCoreColor      = [3]float32{1, 0.95, 0.8}
	laserBeamColor      = [3]float32{1, 0.55, 0.2}
)

// ===== Public API =====

func (laser *SpiderLaser) Direction() (float32, float32) {
	return cosine(laser.BeamAngle), sine(laser.BeamAngle)
}

func (laser *SpiderLaser) IsTurretFrozen() bool {
	return laser.Phase == LaserLocked || laser.Phase == LaserFiring
}

func (laser *SpiderLaser) PhaseProgress() float32 {
	duration := max(laserPhaseDurations[laser.Phase], 0.001)
	return clamp(1-laser.Timer/duration, 0, 1)
}

// ===== Internal =====

func newSpiderLaser() SpiderLaser {
	return SpiderLaser{Phase: LaserCooldown, Timer: laserFirstShotDelay}
}

func (g *Game) updateSpiderLaser(rig *SpiderRig, deltaSeconds float32) {
	laser := &rig.Laser
	laser.Timer -= deltaSeconds
	laserPhaseBehaviors[laser.Phase](g, rig, deltaSeconds)
	if laser.Timer > 0 {
		return
	}
	g.advanceLaserPhase(rig)
}

func (g *Game) advanceLaserPhase(rig *SpiderRig) {
	laser := &rig.Laser
	laser.Phase = (laser.Phase + 1) % laserPhaseCount
	laser.Timer = laserPhaseDurations[laser.Phase]
	g.enterLaserPhase(rig)
}

func (g *Game) enterLaserPhase(rig *SpiderRig) {
	laser := &rig.Laser
	isCooldown := laser.Phase == LaserCooldown
	laser.Timer += g.random.Between(laserCooldownMinimum, laserCooldownMaximum) * boolToFloat(isCooldown)
	isFiring := laser.Phase == LaserFiring
	g.effects.AddShake(14 * boolToFloat(isFiring))
	g.playSound(laserPhaseSounds[laser.Phase])
	g.effects.AddRing(laser.OriginX, laser.OriginY, 90*boolToFloat(isFiring), laserBeamColor)
}

func (g *Game) aimLaser(rig *SpiderRig) {
	laser := &rig.Laser
	laser.OriginX, laser.OriginY = rig.TurretToWorld(laserMuzzleForward, 0)
	laser.BeamAngle = rig.TurretAngle()
}

func (g *Game) chargeLaser(rig *SpiderRig, deltaSeconds float32) {
	g.aimLaser(rig)
	g.drawInChargeMotes(&rig.Laser, deltaSeconds)
}

func (g *Game) holdLaser(rig *SpiderRig, deltaSeconds float32) {
	g.radiateDischarge(&rig.Laser, deltaSeconds)
}

func (g *Game) fireLaser(rig *SpiderRig, deltaSeconds float32) {
	laser := &rig.Laser
	directionX, directionY := laser.Direction()
	g.burnPlayerInBeam(rig, directionX, directionY, deltaSeconds)
	for index := range g.enemies.Count {
		g.burnEnemyInBeam(index, laser, directionX, directionY, deltaSeconds)
	}
	for distance := float32(0); distance < laserRange; distance += laserGroundSampleSpacing {
		g.ground.StimulateAt(laser.OriginX+directionX*distance, laser.OriginY+directionY*distance, StimulusScorch)
	}
	sparkDistance := g.random.Float() * laserRange
	g.effects.SpawnSparks(laser.OriginX+directionX*sparkDistance, laser.OriginY+directionY*sparkDistance, 3, laserBeamColor, 220)
}

func (g *Game) burnPlayerInBeam(rig *SpiderRig, directionX, directionY, deltaSeconds float32) {
	player := g.player
	laser := &rig.Laser
	distance := distanceToRay(player.X, player.Y, laser.OriginX, laser.OriginY, directionX, directionY, laserRange)
	isHit := distance < laserBeamHalfWidth+playerRadius && player.DashSeconds == 0
	if !isHit {
		return
	}
	player.Health -= laserPlayerDamagePerSec * sqrt(rig.Power) * deltaSeconds
	player.DamageFlashSeconds = 0.25
	g.effects.AddShake(1)
}

func (g *Game) burnEnemyInBeam(index int, laser *SpiderLaser, directionX, directionY, deltaSeconds float32) {
	enemies := g.enemies
	definition := &enemyTable[enemies.Kind[index]]
	distance := distanceToRay(enemies.PositionX[index], enemies.PositionY[index], laser.OriginX, laser.OriginY, directionX, directionY, laserRange)
	isHit := distance < laserBeamHalfWidth+definition.Radius && !definition.IsHeavy
	if !isHit {
		return
	}
	enemies.Health[index] -= laserEnemyDamagePerSec * deltaSeconds
	enemies.HitFlash[index] = hitFlashSeconds
	enemies.LastHitByBoss[index] = true
	enemies.ApplyStatus(index, StatusBurning)
}

func distanceToRay(pointX, pointY, originX, originY, directionX, directionY, rayLength float32) float32 {
	along := clamp((pointX-originX)*directionX+(pointY-originY)*directionY, 0, rayLength)
	return length(pointX-originX-directionX*along, pointY-originY-directionY*along)
}
