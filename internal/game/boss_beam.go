package game

import "hordefall/internal/audio"

// ===== Types =====

type LaserPhase uint8

type BeamProfile struct {
	Durations             [laserPhaseCount]float32
	FirstShotDelay        float32
	CooldownMinimum       float32
	CooldownMaximum       float32
	HalfWidth             float32
	Range                 float32
	WidthScale            float32
	PlayerDamagePerSecond float32
	EnemyDamagePerSecond  float32
	Status                StatusFlags
	Stimulus              GroundStimulus
	Sounds                [laserPhaseCount]audio.SoundKind
	TargetingColor        [3]float32
	CoreColor             [3]float32
	BeamColor             [3]float32
	HotColor              [3]float32
}

type BossBeam struct {
	Profile   *BeamProfile
	Phase     LaserPhase
	Timer     float32
	BeamAngle float32
	OriginX   float32
	OriginY   float32
	Motes     [chargeMoteCount]ChargeMote
	NextMote  int
}

type beamPhaseBehavior func(game *Game, beam *BossBeam, power, deltaSeconds float32)

// ===== Constants =====

const (
	LaserCooldown LaserPhase = iota
	LaserCharging
	LaserLocked
	LaserFiring
	laserPhaseCount
)

const (
	laserMuzzleForward       = 60
	laserGroundSampleSpacing = 18
	laserFiringShake         = 14
	laserFiringRing          = 90
)

var spiderBeamProfile = BeamProfile{
	Durations:      [laserPhaseCount]float32{LaserCharging: 2.2, LaserLocked: 0.4, LaserFiring: 0.35},
	FirstShotDelay: 1, CooldownMinimum: 6, CooldownMaximum: 9,
	HalfWidth: 20, Range: 1700, WidthScale: 1,
	PlayerDamagePerSecond: 140, EnemyDamagePerSecond: 420,
	Status: StatusBurning, Stimulus: StimulusScorch,
	Sounds:         [laserPhaseCount]audio.SoundKind{LaserCharging: audio.SoundLaserCharge, LaserFiring: audio.SoundLaserFire},
	TargetingColor: [3]float32{1, 0.25, 0.15}, CoreColor: [3]float32{1, 0.95, 0.8},
	BeamColor: [3]float32{1, 0.55, 0.2}, HotColor: [3]float32{1, 0.85, 0.4},
}

var frostBeamProfile = BeamProfile{
	Durations:      [laserPhaseCount]float32{LaserCharging: 3.4, LaserLocked: 1, LaserFiring: 0.35},
	FirstShotDelay: 0.6, CooldownMinimum: 1.6, CooldownMaximum: 1.6,
	HalfWidth: 40, Range: 1700, WidthScale: 2,
	PlayerDamagePerSecond: 140, EnemyDamagePerSecond: 420,
	Status: StatusChilled, Stimulus: StimulusChill,
	Sounds:         [laserPhaseCount]audio.SoundKind{LaserCharging: audio.SoundFrostCharge, LaserFiring: audio.SoundFrostFire},
	TargetingColor: [3]float32{0.35, 0.8, 1}, CoreColor: [3]float32{0.9, 1, 1},
	BeamColor: [3]float32{0.3, 0.7, 1}, HotColor: [3]float32{0.65, 0.95, 1},
}

var beamPhaseBehaviors = [laserPhaseCount]beamPhaseBehavior{
	LaserCooldown: func(*Game, *BossBeam, float32, float32) {},
	LaserCharging: (*Game).chargeBeam,
	LaserLocked:   (*Game).holdBeam,
	LaserFiring:   (*Game).fireBeam,
}

// ===== Public API =====

func (beam *BossBeam) Direction() (float32, float32) {
	return cosine(beam.BeamAngle), sine(beam.BeamAngle)
}

func (beam *BossBeam) IsTurretFrozen() bool {
	return beam.Phase == LaserLocked || beam.Phase == LaserFiring
}

func (beam *BossBeam) PhaseProgress() float32 {
	duration := max(beam.Profile.Durations[beam.Phase], 0.001)
	return clamp(1-beam.Timer/duration, 0, 1)
}

// ===== Internal =====

func newBossBeam(profile *BeamProfile) BossBeam {
	return BossBeam{Profile: profile, Phase: LaserCooldown, Timer: profile.FirstShotDelay}
}

func (g *Game) updateBossBeam(beam *BossBeam, power, deltaSeconds float32) {
	beam.Timer -= deltaSeconds
	beamPhaseBehaviors[beam.Phase](g, beam, power, deltaSeconds)
	if beam.Timer > 0 {
		return
	}
	g.advanceBeamPhase(beam)
}

func (g *Game) advanceBeamPhase(beam *BossBeam) {
	beam.Phase = (beam.Phase + 1) % laserPhaseCount
	beam.Timer = beam.Profile.Durations[beam.Phase]
	g.enterBeamPhase(beam)
}

func (g *Game) enterBeamPhase(beam *BossBeam) {
	profile := beam.Profile
	isCooldown := beam.Phase == LaserCooldown
	beam.Timer += g.random.Between(profile.CooldownMinimum, profile.CooldownMaximum) * boolToFloat(isCooldown)
	isFiring := beam.Phase == LaserFiring
	g.effects.AddShake(laserFiringShake * boolToFloat(isFiring))
	g.playSound(profile.Sounds[beam.Phase])
	g.effects.AddRing(beam.OriginX, beam.OriginY, laserFiringRing*profile.WidthScale*boolToFloat(isFiring), profile.BeamColor)
}

func (g *Game) chargeBeam(beam *BossBeam, power, deltaSeconds float32) {
	g.drawInChargeMotes(beam, deltaSeconds)
}

func (g *Game) holdBeam(beam *BossBeam, power, deltaSeconds float32) {
	g.radiateDischarge(beam, deltaSeconds)
}

func (g *Game) fireBeam(beam *BossBeam, power, deltaSeconds float32) {
	profile := beam.Profile
	directionX, directionY := beam.Direction()
	g.burnPlayerInBeam(beam, power, directionX, directionY, deltaSeconds)
	for index := range g.enemies.Count {
		g.burnEnemyInBeam(index, beam, directionX, directionY, deltaSeconds)
	}
	for distance := float32(0); distance < profile.Range; distance += laserGroundSampleSpacing {
		g.stimulateBeamGround(beam, beam.OriginX+directionX*distance, beam.OriginY+directionY*distance, -directionY, directionX)
	}
	sparkDistance := g.random.Float() * profile.Range
	g.effects.SpawnSparks(beam.OriginX+directionX*sparkDistance, beam.OriginY+directionY*sparkDistance, 3, profile.BeamColor, 220)
}

func (g *Game) stimulateBeamGround(beam *BossBeam, x, y, normalX, normalY float32) {
	spread := beam.Profile.HalfWidth - groundCellSize
	for offset := -spread; offset <= spread; offset += groundCellSize {
		g.ground.StimulateAt(x+normalX*offset, y+normalY*offset, beam.Profile.Stimulus)
	}
}

func (g *Game) burnPlayerInBeam(beam *BossBeam, power, directionX, directionY, deltaSeconds float32) {
	player := g.player
	profile := beam.Profile
	distance := distanceToRay(player.X, player.Y, beam.OriginX, beam.OriginY, directionX, directionY, profile.Range)
	isHit := distance < profile.HalfWidth+playerRadius && player.DashSeconds == 0
	if !isHit {
		return
	}
	player.Health -= profile.PlayerDamagePerSecond * sqrt(power) * deltaSeconds
	player.DamageFlashSeconds = 0.25
	g.effects.AddShake(1)
}

func (g *Game) burnEnemyInBeam(index int, beam *BossBeam, directionX, directionY, deltaSeconds float32) {
	enemies := g.enemies
	profile := beam.Profile
	definition := &enemyTable[enemies.Kind[index]]
	distance := distanceToRay(enemies.PositionX[index], enemies.PositionY[index], beam.OriginX, beam.OriginY, directionX, directionY, profile.Range)
	isHit := distance < profile.HalfWidth+definition.Radius && !definition.IsHeavy
	if !isHit {
		return
	}
	enemies.Health[index] -= profile.EnemyDamagePerSecond * deltaSeconds
	enemies.HitFlash[index] = hitFlashSeconds
	enemies.LastHitByBoss[index] = true
	enemies.ApplyStatus(index, profile.Status)
}

func distanceToRay(pointX, pointY, originX, originY, directionX, directionY, rayLength float32) float32 {
	along := clamp((pointX-originX)*directionX+(pointY-originY)*directionY, 0, rayLength)
	return length(pointX-originX-directionX*along, pointY-originY-directionY*along)
}
