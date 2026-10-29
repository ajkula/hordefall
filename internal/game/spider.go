package game

import (
	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/mathx"
	"github.com/ajkula/ajkurig/procanim"
)

// ===== Types =====

type SpiderRig struct {
	procanim.Body
	EnemyID       uint32
	MaximumHealth float32
	TurretServo   procanim.Servo
	Power         float32
	Laser         BossBeam
	Gait          *procanim.Gait
}

// ===== Constants =====

const (
	spiderLegCount         = 4
	spiderScale            = 1.6
	spiderTurnRate         = 2.2
	spiderMinimumSpeed     = 5
	spiderSpacingFactor    = 1.5
	spiderStompRadius      = 52
	spiderStompDamage      = 22
	spiderStompPlayerHurt  = 18
	spiderStompSource      = "Spider Tank Stomp"
	spiderSpawnDistance    = 840
	bossKillsBase          = 300
	bossKillsGrowthSeconds = 120
	spiderTurretPivot      = 8
	maximumBossesAlive     = 3
	spiderTurretLimit      = 2.3
	spiderTurretStiffness  = 34
	spiderTurretDamping    = 8
)

var spiderLegLayouts = []procanim.LegLayout{
	{HipForward: 14, HipSide: -20, RestForward: 56, RestSide: -62, Group: 0},
	{HipForward: 14, HipSide: 20, RestForward: 56, RestSide: 62, Group: 1},
	{HipForward: -22, HipSide: -20, RestForward: -50, RestSide: -66, Group: 1},
	{HipForward: -22, HipSide: 20, RestForward: -50, RestSide: 66, Group: 0},
}

var spiderGait = procanim.GaitSettings{
	StepThreshold: 34, StepSeconds: 0.26, StepLead: 0.35, StepOvershoot: 0.35,
	StepLift: 16 * spiderScale, ThighLength: 50, ShinLength: 56,
}

var (
	stompDustColor = [3]float32{0.6, 0.55, 0.45}
	warningColor   = [3]float32{1, 0.3, 0.25}
)

// ===== Public API =====

func (rig *SpiderRig) TurretAngle() float32 {
	return rig.Heading + rig.TurretServo.Angle
}

func (rig *SpiderRig) TurretToWorld(forward, side float32) (float32, float32) {
	pivotX, pivotY := rig.LocalToWorld(spiderTurretPivot, 0)
	offsetX, offsetY := rotateOffset(forward*spiderScale, side*spiderScale, rig.TurretAngle())
	return pivotX + offsetX, pivotY + offsetY
}

func (rig *SpiderRig) HipPosition(leg int) (float32, float32) {
	return rig.Gait.HipPosition(&rig.Body, leg)
}

func (rig *SpiderRig) KneePosition(leg int) (float32, float32) {
	return rig.Gait.KneePosition(&rig.Body, leg)
}

// ===== Internal =====

func (g *Game) startBossEventIfDue() {
	next := g.nextBossEvent()
	isBossCapped := isBossCreature[next] && g.livingBossCount() >= maximumBossesAlive
	isWormCapped := next == BossWorm && g.worm.IsActive
	if g.bossProgressKills < g.bossKillsRequired || isBossCapped || isWormCapped {
		return
	}
	g.bossProgressKills = 0
	g.bossKillsRequired = bossKillsRequiredAt(g.elapsedSeconds)
	g.startBossEvent()
}

func (g *Game) livingBossCount() int {
	return len(g.spiders) + boolToIndex(g.worm.IsActive)
}

func bossKillsRequiredAt(seconds float32) int {
	return int(bossKillsBase * (1 + seconds/bossKillsGrowthSeconds))
}

func (g *Game) spawnSpiderAt(angle, distance float32) {
	x := clamp(g.player.X+cosine(angle)*distance, 80, arenaSize-80)
	y := clamp(g.player.Y+sine(angle)*distance, 80, arenaSize-80)
	id := g.spawnEnemyAt(EnemySpiderTank, x, y)
	rig := SpiderRig{EnemyID: id, Body: procanim.Body{X: x, Y: y, Heading: angle + 3.14159265, Scale: spiderScale}, Laser: newBossBeam(&spiderBeamProfile)}
	rig.Power = 1 + g.elapsedSeconds/150
	rig.MaximumHealth = enemyTable[EnemySpiderTank].Health * rig.Power
	rig.Gait = procanim.NewGait(spiderGait, spiderLegLayouts)
	rig.Gait.Plant(&rig.Body)
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
		kept = mathx.AppendIf(kept, g.spiders[index], enemyIndex >= 0)
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
	rig.Follow(g.enemies.PositionX[enemyIndex], g.enemies.PositionY[enemyIndex], spiderTurnRate, spiderMinimumSpeed, deltaSeconds)
	turretTarget := clamp(angleDifference(rig.Heading, atan2(g.player.Y-rig.Y, g.player.X-rig.X)), -spiderTurretLimit, spiderTurretLimit)
	rig.TurretServo.Drive(turretTarget, spiderTurretStiffness, spiderTurretDamping, deltaSeconds*boolToFloat(!rig.Laser.IsTurretFrozen()))
	g.aimSpiderBeamIf(rig, !rig.Laser.IsTurretFrozen())
	g.updateBossBeam(&rig.Laser, rig.Power, deltaSeconds)
	for _, leg := range rig.Gait.Update(&rig.Body, deltaSeconds) {
		g.stomp(rig.Gait.Legs[leg].FootX, rig.Gait.Legs[leg].FootY, rig.Power)
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

func (g *Game) stomp(x, y, power float32) {
	g.effects.SpawnSparks(x, y, 9, stompDustColor, 120)
	g.effects.AddRing(x, y, spiderStompRadius, stompDustColor)
	g.effects.AddShake(1.2)
	g.playSound(SoundStomp)
	g.QueueBurst(Burst{X: x, Y: y, Radius: spiderStompRadius, Damage: spiderStompDamage * power, Element: ElementPhysical})
	isPlayerCrushed := distanceSquared(x, y, g.player.X, g.player.Y) < spiderStompRadius*spiderStompRadius
	g.DamagePlayer(spiderStompPlayerHurt*power*boolToFloat(isPlayerCrushed), spiderStompSource)
}
