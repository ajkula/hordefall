package game

import (
	"hordefall/internal/audio"

	"hordefall/internal/i18n"
)

// ===== Types =====

type LegRig struct {
	FootX        float32
	FootY        float32
	FromX        float32
	FromY        float32
	ToX          float32
	ToY          float32
	StepProgress float32
	IsStepping   bool
}

type LegLayout struct {
	HipForward  float32
	HipSide     float32
	RestForward float32
	RestSide    float32
	GaitGroup   int
}

type SpiderRig struct {
	EnemyID       uint32
	X             float32
	Y             float32
	VelocityX     float32
	VelocityY     float32
	Heading       float32
	MaximumHealth float32
	TurretServo   AngularServo
	Power         float32
	Laser         BossBeam
	Legs          [spiderLegCount]LegRig
}

// ===== Constants =====

const (
	spiderLegCount         = 4
	spiderScale            = 1.6
	spiderThighLength      = 50 * spiderScale
	spiderShinLength       = 56 * spiderScale
	spiderStepThreshold    = 34 * spiderScale
	spiderStepSeconds      = 0.26
	spiderStepLead         = 0.35
	spiderStepOvershoot    = 0.35
	spiderStepLift         = 16 * spiderScale
	spiderTurnRate         = 2.2
	spiderSpacingFactor    = 1.5
	spiderStompRadius      = 52
	spiderStompDamage      = 22
	spiderStompPlayerHurt  = 18
	spiderSpawnDistance    = 840
	bossKillsBase          = 300
	bossKillsGrowthSeconds = 120
	spiderTurretPivot      = 8
	maximumSpidersAlive    = 3
	spiderTurretLimit      = 2.3
	spiderTurretStiffness  = 34
	spiderTurretDamping    = 8
)

var spiderLegLayouts = [spiderLegCount]LegLayout{
	{HipForward: 14, HipSide: -20, RestForward: 56, RestSide: -62, GaitGroup: 0},
	{HipForward: 14, HipSide: 20, RestForward: 56, RestSide: 62, GaitGroup: 1},
	{HipForward: -22, HipSide: -20, RestForward: -50, RestSide: -66, GaitGroup: 1},
	{HipForward: -22, HipSide: 20, RestForward: -50, RestSide: 66, GaitGroup: 0},
}

var (
	stompDustColor = [3]float32{0.6, 0.55, 0.45}
	warningColor   = [3]float32{1, 0.3, 0.25}
)

// ===== Public API =====

func (rig *SpiderRig) LocalToWorld(forward, side float32) (float32, float32) {
	cosineHeading, sineHeading := cosine(rig.Heading), sine(rig.Heading)
	forward, side = forward*spiderScale, side*spiderScale
	return rig.X + forward*cosineHeading - side*sineHeading, rig.Y + forward*sineHeading + side*cosineHeading
}

func (rig *SpiderRig) TurretAngle() float32 {
	return rig.Heading + rig.TurretServo.Angle
}

func (rig *SpiderRig) TurretToWorld(forward, side float32) (float32, float32) {
	pivotX, pivotY := rig.LocalToWorld(spiderTurretPivot, 0)
	offsetX, offsetY := rotateOffset(forward*spiderScale, side*spiderScale, rig.TurretAngle())
	return pivotX + offsetX, pivotY + offsetY
}

func (rig *SpiderRig) HipPosition(leg int) (float32, float32) {
	return rig.LocalToWorld(spiderLegLayouts[leg].HipForward, spiderLegLayouts[leg].HipSide)
}

func (rig *SpiderRig) FootLift(leg int) float32 {
	return sine(rig.Legs[leg].StepProgress*3.14159265) * spiderStepLift * boolToFloat(rig.Legs[leg].IsStepping)
}

func (rig *SpiderRig) KneePosition(leg int) (float32, float32) {
	hipX, hipY := rig.HipPosition(leg)
	footX, footY := rig.Legs[leg].FootX, rig.Legs[leg].FootY
	firstX, firstY, secondX, secondY := solveKneeCandidates(hipX, hipY, footX, footY)
	isFirstOutward := distanceSquared(firstX, firstY, rig.X, rig.Y) >= distanceSquared(secondX, secondY, rig.X, rig.Y)
	weight := boolToFloat(isFirstOutward)
	return secondX + (firstX-secondX)*weight, secondY + (firstY-secondY)*weight
}

// ===== Internal =====

func solveKneeCandidates(hipX, hipY, footX, footY float32) (float32, float32, float32, float32) {
	deltaX, deltaY := footX-hipX, footY-hipY
	distance := clamp(length(deltaX, deltaY), 1, spiderThighLength+spiderShinLength-0.5)
	alongX, alongY := deltaX/distance, deltaY/distance
	along := (spiderThighLength*spiderThighLength - spiderShinLength*spiderShinLength + distance*distance) / (2 * distance)
	height := sqrt(max(0, spiderThighLength*spiderThighLength-along*along))
	baseX, baseY := hipX+alongX*along, hipY+alongY*along
	return baseX - alongY*height, baseY + alongX*height, baseX + alongY*height, baseY - alongX*height
}

func (g *Game) startBossEventIfDue() {
	isSpiderCapped := g.nextBossEvent() == BossSpider && len(g.spiders) >= maximumSpidersAlive
	if g.bossProgressKills < g.bossKillsRequired || isSpiderCapped || g.worm.IsActive {
		return
	}
	g.bossProgressKills = 0
	g.bossKillsRequired = bossKillsRequiredAt(g.elapsedSeconds)
	g.startBossEvent()
}

func bossKillsRequiredAt(seconds float32) int {
	return int(bossKillsBase * (1 + seconds/bossKillsGrowthSeconds))
}

func (g *Game) spawnSpiderAt(angle, distance float32) {
	x := clamp(g.player.X+cosine(angle)*distance, 80, arenaSize-80)
	y := clamp(g.player.Y+sine(angle)*distance, 80, arenaSize-80)
	id := g.spawnEnemyAt(EnemySpiderTank, x, y)
	rig := SpiderRig{EnemyID: id, X: x, Y: y, Heading: angle + 3.14159265, Laser: newBossBeam(&spiderBeamProfile)}
	rig.Power = 1 + g.elapsedSeconds/150
	rig.MaximumHealth = enemyTable[EnemySpiderTank].Health * rig.Power
	for leg := range spiderLegCount {
		rig.Legs[leg].FootX, rig.Legs[leg].FootY = rig.LocalToWorld(spiderLegLayouts[leg].RestForward, spiderLegLayouts[leg].RestSide)
	}
	g.spiders = append(g.spiders, rig)
	g.effects.AddPopup(g.player.X, g.player.Y-80, i18n.T("spider.inbound"), warningColor)
	g.say(ChatterSpiderInbound)
	g.effects.AddShake(6)
}

func (g *Game) updateSpiders(deltaSeconds float32) {
	g.separateSpiders()
	kept := g.spiders[:0]
	for index := range g.spiders {
		enemyIndex := g.enemies.IndexOfID(g.spiders[index].EnemyID)
		g.updateSpider(&g.spiders[index], enemyIndex, deltaSeconds)
		kept = appendIf(kept, g.spiders[index], enemyIndex >= 0)
	}
	g.spiders = kept
}

func (g *Game) separateSpiders() {
	for first := range g.spiders {
		for second := first + 1; second < len(g.spiders); second++ {
			g.pushSpidersApart(g.enemies.IndexOfID(g.spiders[first].EnemyID), g.enemies.IndexOfID(g.spiders[second].EnemyID))
		}
	}
}

func (g *Game) pushSpidersApart(first, second int) {
	if first < 0 || second < 0 {
		return
	}
	enemies := g.enemies
	offsetX, offsetY := enemies.PositionX[second]-enemies.PositionX[first], enemies.PositionY[second]-enemies.PositionY[first]
	distance := length(offsetX, offsetY)
	spacing := (enemyTable[enemies.Kind[first]].Radius + enemyTable[enemies.Kind[second]].Radius) * spiderSpacingFactor
	if distance >= spacing {
		return
	}
	directionX, directionY := normalize(offsetX+boolToFloat(distance == 0), offsetY)
	push := (spacing - distance) / 2
	enemies.PositionX[first] = clamp(enemies.PositionX[first]-directionX*push, 0, arenaSize)
	enemies.PositionY[first] = clamp(enemies.PositionY[first]-directionY*push, 0, arenaSize)
	enemies.PositionX[second] = clamp(enemies.PositionX[second]+directionX*push, 0, arenaSize)
	enemies.PositionY[second] = clamp(enemies.PositionY[second]+directionY*push, 0, arenaSize)
}

func (g *Game) updateSpider(rig *SpiderRig, enemyIndex int, deltaSeconds float32) {
	if enemyIndex < 0 {
		g.effects.AddPopup(rig.X, rig.Y-60, i18n.T("spider.destroyed"), accentColor)
		g.say(ChatterSpiderDown)
		g.effects.AddShake(12)
		return
	}
	x, y := g.enemies.PositionX[enemyIndex], g.enemies.PositionY[enemyIndex]
	rig.VelocityX, rig.VelocityY = (x-rig.X)/deltaSeconds, (y-rig.Y)/deltaSeconds
	rig.X, rig.Y = x, y
	rig.Heading = turnToward(rig.Heading, atan2(rig.VelocityY, rig.VelocityX), spiderTurnRate*deltaSeconds*boolToFloat(length(rig.VelocityX, rig.VelocityY) > 5))
	turretTarget := clamp(angleDifference(rig.Heading, atan2(g.player.Y-rig.Y, g.player.X-rig.X)), -spiderTurretLimit, spiderTurretLimit)
	rig.TurretServo.Drive(turretTarget, spiderTurretStiffness, spiderTurretDamping, deltaSeconds*boolToFloat(!rig.Laser.IsTurretFrozen()))
	g.aimSpiderBeamIf(rig, !rig.Laser.IsTurretFrozen())
	g.updateBossBeam(&rig.Laser, rig.Power, deltaSeconds)
	for leg := range spiderLegCount {
		g.updateLeg(rig, leg, deltaSeconds)
	}
}

func (g *Game) aimSpiderBeamIf(rig *SpiderRig, shouldAim bool) {
	if !shouldAim {
		return
	}
	beam := &rig.Laser
	beam.OriginX, beam.OriginY = rig.TurretToWorld(laserMuzzleForward, 0)
	beam.BeamAngle = rig.TurretAngle()
}

func (g *Game) updateLeg(rig *SpiderRig, leg int, deltaSeconds float32) {
	state := &rig.Legs[leg]
	if state.IsStepping {
		g.advanceStep(state, rig.Power, deltaSeconds)
		return
	}
	layout := &spiderLegLayouts[leg]
	restX, restY := rig.LocalToWorld(layout.RestForward, layout.RestSide)
	desiredX, desiredY := restX+rig.VelocityX*spiderStepLead, restY+rig.VelocityY*spiderStepLead
	isTooFar := distanceSquared(state.FootX, state.FootY, desiredX, desiredY) > spiderStepThreshold*spiderStepThreshold
	if !isTooFar || rig.isGroupStepping(1-layout.GaitGroup) {
		return
	}
	state.FromX, state.FromY = state.FootX, state.FootY
	state.ToX = desiredX + (desiredX-state.FootX)*spiderStepOvershoot
	state.ToY = desiredY + (desiredY-state.FootY)*spiderStepOvershoot
	state.StepProgress, state.IsStepping = 0, true
}

func (g *Game) advanceStep(state *LegRig, power, deltaSeconds float32) {
	state.StepProgress = min(1, state.StepProgress+deltaSeconds/spiderStepSeconds)
	eased := smoothstep(state.StepProgress)
	state.FootX = state.FromX + (state.ToX-state.FromX)*eased
	state.FootY = state.FromY + (state.ToY-state.FromY)*eased
	if state.StepProgress < 1 {
		return
	}
	state.IsStepping = false
	g.stomp(state.FootX, state.FootY, power)
}

func (g *Game) stomp(x, y, power float32) {
	g.effects.SpawnSparks(x, y, 9, stompDustColor, 120)
	g.effects.AddRing(x, y, spiderStompRadius, stompDustColor)
	g.effects.AddShake(1.2)
	g.playSound(audio.SoundStomp)
	g.QueueBurst(Burst{X: x, Y: y, Radius: spiderStompRadius, Damage: spiderStompDamage * power, Element: ElementPhysical})
	isPlayerCrushed := distanceSquared(x, y, g.player.X, g.player.Y) < spiderStompRadius*spiderStompRadius
	g.DamagePlayer(spiderStompPlayerHurt * power * boolToFloat(isPlayerCrushed))
}

func (rig *SpiderRig) isGroupStepping(group int) bool {
	for leg := range spiderLegCount {
		if spiderLegLayouts[leg].GaitGroup == group && rig.Legs[leg].IsStepping {
			return true
		}
	}
	return false
}

func turnToward(current, target, maximumStep float32) float32 {
	return current + clamp(angleDifference(current, target), -maximumStep, maximumStep)
}
