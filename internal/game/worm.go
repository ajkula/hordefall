package game

import (
	"math"

	"hordefall/internal/audio"
	"hordefall/internal/i18n"
)

// ===== Types =====

type WormPhase uint8

type Point3 struct {
	X float32
	Y float32
	Z float32
}

type WormSegment struct {
	X             float32
	Y             float32
	Z             float32
	EnemyID       uint32
	Health        float32
	IsAlive       bool
	DamageSeconds float32
}

type WormPath struct {
	Points  [wormPathCapacity]Point3
	Lengths [wormPathCapacity]float32
	Count   int
	Arch    float32
}

type Sandworm struct {
	IsActive             bool
	Phase                WormPhase
	PhaseSeconds         float32
	X                    float32
	Y                    float32
	HoleX                float32
	HoleY                float32
	HeadX                float32
	HeadY                float32
	HeadZ                float32
	Angle                float32
	Travel               float32
	HasFired             bool
	LingerSeconds        float32
	Controls             [4]Point3
	Aim                  Point3
	AimHistory           [wormAimHistory]Point3
	AimCursor            int
	Bend                 float32
	Joints               [wormSegmentCount + 1]Point3
	JointCount           int
	DiveControls         [4]Point3
	Path                 WormPath
	Segments             [wormSegmentCount]WormSegment
	SegmentMaximumHealth float32
	HeadEnemyID          uint32
	HeadHealth           float32
	HeadMaximumHealth    float32
	Power                float32
	Beam                 BossBeam
	DustSeconds          float32
	RollFromX            float32
	RollFromY            float32
	RollSpin             float32
	BlastHits            int
	BlastSeconds         float32
	HasRollHit           bool
}

type WormCrater struct {
	X       float32
	Y       float32
	Variant int
	Life    float32
}

type wormPhaseUpdater func(game *Game, worm *Sandworm, deltaSeconds float32)

// ===== Constants =====

const (
	WormBurrowing WormPhase = iota
	WormWarning
	WormEmerging
	WormRearing
	WormAiming
	WormDiving
	WormHeadAiming
	WormHeadRolling
	WormHeadResting
	wormPhaseCount
)

const (
	wormSpawnDistance        = 760
	wormBurrowSpeed          = 250
	wormStrikeDistance       = 50
	wormMinimumBurrowSeconds = 1.6
	wormMaximumBurrowSeconds = 3.5
	wormWarningSeconds       = 1.1
	wormSegmentCount         = 14
	wormSegmentSpacing       = 28
	wormHeadLength           = 46
	wormBodyLength           = wormHeadLength + wormSegmentCount*wormSegmentSpacing
	wormEmergeSpeed          = 520
	wormRearSeconds          = 0.8
	wormRearAimTurnRate      = 3.5
	wormAimPitch             = 0.35
	wormAimHistory           = 64
	wormJointDelayTicks      = 3
	wormBendExponent         = 1.6
	wormAimTurnRate          = 1.4
	wormDiveSpeed            = 420
	wormLingerSeconds        = 1.4
	wormPathSamples          = 40
	wormPathCapacity         = 2*wormPathSamples + wormSegmentCount + 3
	wormMuzzleForward        = 30
	wormDiveReach            = 1.4
	wormDiveLift             = 0.6
	wormEruptionRadius       = 110
	wormEruptionDamage       = 60
	wormEruptionPlayerHurt   = 30
	wormDustInterval         = 0.07
	wormCraterSeconds        = 8
	wormCraterFadeSeconds    = 2
	wormEruptionDust         = 34
	wormBlastReach           = 1.3
	wormBlastHits            = 5
	wormBlastInterval        = 0.11
	wormBlastDamage          = 5
	wormBlastShove           = 950
	wormWakeClods            = 2
	wormArenaMargin          = 120
	wormPowerGrowthSeconds   = 150
	wormHeadAimSeconds       = 1.2
	wormHeadLockSeconds      = 0.3
	wormHeadTurnRate         = 6
	wormHeadRollSeconds      = 1
	wormHeadCycleSeconds     = 4
	wormRollDistance         = 720
	wormRollRadius           = 48
	wormRollPlayerHurt       = 40
	wormRollEnemyDamage      = 900
	wormRollSpinRate         = 14
	wormSparkThreshold       = 0.75
	wormSmokeThreshold       = 0.5
	wormFireThreshold        = 0.25
	wormDamageInterval       = 0.12
)

var wormPhaseUpdaters = [wormPhaseCount]wormPhaseUpdater{
	WormBurrowing:   (*Game).updateWormBurrowing,
	WormWarning:     (*Game).updateWormWarning,
	WormEmerging:    (*Game).updateWormEmerging,
	WormRearing:     (*Game).updateWormRearing,
	WormAiming:      (*Game).updateWormAiming,
	WormDiving:      (*Game).updateWormDiving,
	WormHeadAiming:  (*Game).updateWormHeadAiming,
	WormHeadRolling: (*Game).updateWormHeadRolling,
	WormHeadResting: (*Game).updateWormHeadResting,
}

var wormBodyPhases = [wormPhaseCount]bool{WormEmerging: true, WormRearing: true, WormAiming: true, WormDiving: true}

var wormHeadPhases = [wormPhaseCount]bool{WormHeadAiming: true, WormHeadRolling: true, WormHeadResting: true}

var wormPathBuilders = [wormPhaseCount]func(worm *Sandworm){
	WormBurrowing:   func(*Sandworm) {},
	WormWarning:     func(*Sandworm) {},
	WormEmerging:    appendColumnPath,
	WormRearing:     appendChainPath,
	WormAiming:      appendChainPath,
	WormDiving:      appendChainPath,
	WormHeadAiming:  func(*Sandworm) {},
	WormHeadRolling: func(*Sandworm) {},
	WormHeadResting: func(*Sandworm) {},
}

var (
	wormDirtColor  = [3]float32{0.48, 0.36, 0.22}
	wormSparkColor = [3]float32{1, 0.85, 0.45}
	wormFireColor  = [3]float32{1, 0.45, 0.1}
)

// ===== Public API =====

func (w *Sandworm) Direction() (float32, float32) {
	return cosine(w.Angle), sine(w.Angle)
}

func (w *Sandworm) IsBodyOut() bool {
	return w.IsActive && wormBodyPhases[w.Phase]
}

func (w *Sandworm) IsHeadAlone() bool {
	return w.IsActive && wormHeadPhases[w.Phase]
}

func (w *Sandworm) AliveSegmentCount() int {
	count := 0
	for index := range w.Segments {
		count += boolToIndex(w.Segments[index].IsAlive)
	}
	return count
}

func (w *Sandworm) HeadAlong() float32 {
	return w.Path.Arch + w.Travel*boolToFloat(w.Phase == WormDiving)
}

func (w *Sandworm) SlotAlong(slot int) float32 {
	return w.HeadAlong() - wormHeadLength - float32(slot)*wormSegmentSpacing
}

func (w *Sandworm) IsAlongAboveGround(along float32) bool {
	return w.IsBodyOut() && along >= 0 && along <= w.Path.Total()
}

func (w *Sandworm) IsHeadAboveGround() bool {
	return w.IsHeadAlone() || w.IsAlongAboveGround(w.HeadAlong()-wormHeadLength/2)
}

func (w *Sandworm) BodyLength() float32 {
	return wormHeadLength + float32(w.AliveSegmentCount())*wormSegmentSpacing
}

func (w *Sandworm) HealthFraction() float32 {
	total := w.HeadHealth
	for index := range w.Segments {
		total += w.Segments[index].Health * boolToFloat(w.Segments[index].IsAlive)
	}
	return total / (w.HeadMaximumHealth + w.SegmentMaximumHealth*wormSegmentCount)
}

func (p *WormPath) Total() float32 {
	return p.Lengths[max(0, p.Count-1)]
}

func (p *WormPath) At(along float32) (Point3, Point3) {
	if p.Count < 2 {
		return p.Points[0], Point3{0, 0, 1}
	}
	clamped := clamp(along, 0, p.Total())
	index := 1
	for index < p.Count-1 && p.Lengths[index] < clamped {
		index++
	}
	from, to := p.Points[index-1], p.Points[min(index, p.Count-1)]
	span := max(p.Lengths[min(index, p.Count-1)]-p.Lengths[index-1], 0.001)
	weight := clamp((clamped-p.Lengths[index-1])/span, 0, 1)
	tangentX, tangentY, tangentZ := normalizeVector3(to.X-from.X, to.Y-from.Y, to.Z-from.Z+boolToFloat(from == to))
	return lerpPoint(from, to, weight), Point3{tangentX, tangentY, tangentZ}
}

// ===== Internal =====

func (g *Game) spawnWormEvent() {
	angle := g.random.Angle()
	power := 1 + g.elapsedSeconds/wormPowerGrowthSeconds
	g.worm = Sandworm{
		IsActive: true, Phase: WormBurrowing, Power: power,
		X:                    clampToArena(g.player.X + cosine(angle)*wormSpawnDistance),
		Y:                    clampToArena(g.player.Y + sine(angle)*wormSpawnDistance),
		SegmentMaximumHealth: enemyTable[EnemySandwormSegment].Health * power,
		HeadMaximumHealth:    enemyTable[EnemySandworm].Health * power,
	}
	worm := &g.worm
	worm.HeadHealth = worm.HeadMaximumHealth
	for index := range worm.Segments {
		worm.Segments[index] = WormSegment{IsAlive: true, Health: worm.SegmentMaximumHealth}
	}
	g.effects.AddPopup(g.player.X, g.player.Y-80, i18n.T("worm.inbound"), warningColor)
	g.say(ChatterWormInbound)
	g.effects.AddShake(6)
	g.playSound(audio.SoundWormRoar)
}

func (g *Game) updateWorm(deltaSeconds float32) {
	g.ageWormCraters(deltaSeconds)
	worm := &g.worm
	if !worm.IsActive {
		return
	}
	worm.PhaseSeconds += deltaSeconds
	g.updateWormBlast(worm, deltaSeconds)
	g.collectWormLosses(worm)
	if !worm.IsActive {
		return
	}
	wormPhaseUpdaters[worm.Phase](g, worm, deltaSeconds)
	g.rebuildWormPathIf(worm, worm.IsBodyOut())
	g.updateWormSegments(worm, deltaSeconds)
}

func (g *Game) enterWormPhase(worm *Sandworm, phase WormPhase) {
	worm.Phase = phase
	worm.PhaseSeconds = 0
}

// ===== Internal: underground =====

func (g *Game) updateWormBurrowing(worm *Sandworm, deltaSeconds float32) {
	directionX, directionY := normalize(g.player.X-worm.X, g.player.Y-worm.Y)
	worm.X = clampToArena(worm.X + directionX*wormBurrowSpeed*deltaSeconds)
	worm.Y = clampToArena(worm.Y + directionY*wormBurrowSpeed*deltaSeconds)
	g.kickUpWormDust(worm, worm.X, worm.Y, deltaSeconds)
	isClose := distanceSquared(worm.X, worm.Y, g.player.X, g.player.Y) < wormStrikeDistance*wormStrikeDistance
	isReady := isClose && worm.PhaseSeconds >= wormMinimumBurrowSeconds || worm.PhaseSeconds >= wormMaximumBurrowSeconds
	if !isReady {
		return
	}
	worm.X, worm.Y = clampToArena(g.player.X), clampToArena(g.player.Y)
	g.enterWormPhase(worm, WormWarning)
	g.playSound(audio.SoundWormRumble)
}

func (g *Game) updateWormWarning(worm *Sandworm, deltaSeconds float32) {
	g.kickUpWormDust(worm, worm.X, worm.Y, deltaSeconds)
	if worm.PhaseSeconds < wormWarningSeconds {
		return
	}
	worm.HoleX, worm.HoleY = worm.X, worm.Y
	worm.Angle = atan2(g.player.Y-worm.HoleY, g.player.X-worm.HoleX)
	worm.Travel, worm.HasFired, worm.LingerSeconds = 0, false, 0
	worm.Controls = wormColumnControls(worm, 0)
	g.eruptWorm(worm)
	g.enterWormPhase(worm, WormEmerging)
}

// ===== Internal: body out =====

func (g *Game) updateWormEmerging(worm *Sandworm, deltaSeconds float32) {
	riseHeight := worm.BodyLength()
	worm.Travel = min(worm.Travel+wormEmergeSpeed*deltaSeconds, riseHeight)
	worm.Controls = wormColumnControls(worm, worm.Travel)
	g.kickUpWormDust(worm, worm.HoleX, worm.HoleY, deltaSeconds)
	if worm.Travel < riseHeight {
		return
	}
	worm.Travel = 0
	g.raiseWormChain(worm)
	g.enterWormPhase(worm, WormRearing)
}

func (g *Game) raiseWormChain(worm *Sandworm) {
	worm.Aim, worm.Bend = Point3{0, 0, 1}, 0
	for index := range worm.AimHistory {
		worm.AimHistory[index] = worm.Aim
	}
	buildWormChain(worm)
}

func (g *Game) updateWormRearing(worm *Sandworm, deltaSeconds float32) {
	g.steerWorm(worm, deltaSeconds, true)
	if worm.PhaseSeconds < wormRearSeconds {
		return
	}
	worm.Beam = newBossBeam(&frostBeamProfile)
	g.enterWormPhase(worm, WormAiming)
}

func (g *Game) updateWormAiming(worm *Sandworm, deltaSeconds float32) {
	isTracking := !worm.Beam.IsTurretFrozen() && !worm.HasFired
	g.steerWorm(worm, deltaSeconds, isTracking)
	g.rebuildWormPathIf(worm, true)
	g.aimWormBeamIf(worm, isTracking)
	wasFiring := worm.Beam.Phase == LaserFiring
	g.updateBossBeam(&worm.Beam, worm.Power, deltaSeconds)
	worm.HasFired = worm.HasFired || wasFiring && worm.Beam.Phase == LaserCooldown
	worm.LingerSeconds += deltaSeconds * boolToFloat(worm.HasFired)
	if worm.LingerSeconds < wormLingerSeconds {
		return
	}
	worm.Travel = 0
	worm.DiveControls = wormDiveControls(worm)
	g.playSound(audio.SoundWormRumble)
	g.enterWormPhase(worm, WormDiving)
}

func (g *Game) steerWorm(worm *Sandworm, deltaSeconds float32, isTracking bool) {
	tracking := boolToFloat(isTracking)
	aimRate := [2]float32{wormAimTurnRate, wormRearAimTurnRate}[boolToIndex(worm.Phase == WormRearing)]
	worm.Aim = turnVectorToward(worm.Aim, wormAimAt(worm.HeadPoint(), g.player.X, g.player.Y), aimRate*deltaSeconds*tracking)
	worm.Bend = min(1, worm.Bend+deltaSeconds/wormRearSeconds)
	worm.AimCursor = (worm.AimCursor + 1) % wormAimHistory
	worm.AimHistory[worm.AimCursor] = worm.Aim
	buildWormChain(worm)
}

func wormAimAt(head Point3, targetX, targetY float32) Point3 {
	screenX, screenY := normalize(targetX-head.X+0.0001, targetY-(head.Y-head.Z))
	along := screenY*wormAimPitch + sqrt(screenY*screenY*wormAimPitch*wormAimPitch-2*wormAimPitch*wormAimPitch+1)
	return Point3{along * screenX, along*screenY - wormAimPitch, -wormAimPitch}
}

func (w *Sandworm) HeadPoint() Point3 {
	neck := w.Joints[max(0, w.JointCount-1)]
	return Point3{neck.X + w.Aim.X*wormHeadLength, neck.Y + w.Aim.Y*wormHeadLength, max(0, neck.Z+w.Aim.Z*wormHeadLength)}
}

func buildWormChain(worm *Sandworm) {
	links := worm.AliveSegmentCount()
	worm.JointCount = links + 1
	worm.Joints[0] = Point3{worm.HoleX, worm.HoleY, 0}
	for link := range links {
		closeness := float32(link+1) / float32(links)
		delayed := worm.AimHistory[(worm.AimCursor-(links-1-link)*wormJointDelayTicks+wormAimHistory*wormSegmentCount)%wormAimHistory]
		weight := worm.Bend * float32(math.Pow(float64(closeness), wormBendExponent))
		direction := lerpPoint(Point3{0, 0, 1}, delayed, weight)
		worm.Joints[link+1] = pointAtDistance(worm.Joints[link], Point3{worm.Joints[link].X + direction.X, worm.Joints[link].Y + direction.Y, worm.Joints[link].Z + direction.Z}, wormSegmentSpacing)
	}
}

func (g *Game) aimWormBeamIf(worm *Sandworm, shouldAim bool) {
	if !shouldAim {
		return
	}
	head := worm.HeadPoint()
	aimX, aimY := normalize(worm.Aim.X, worm.Aim.Y-worm.Aim.Z)
	worm.Beam.OriginX = head.X + aimX*wormMuzzleForward
	worm.Beam.OriginY = head.Y - head.Z + aimY*wormMuzzleForward
	worm.Beam.BeamAngle = atan2(aimY, aimX)
}

func (g *Game) updateWormDiving(worm *Sandworm, deltaSeconds float32) {
	worm.Travel += wormDiveSpeed * deltaSeconds
	diveEnd := worm.DiveControls[3]
	isHeadDown := worm.HeadAlong() >= worm.Path.Total()
	g.kickUpWormDust(worm, diveEnd.X, diveEnd.Y, deltaSeconds*boolToFloat(isHeadDown))
	if worm.HeadAlong()-worm.BodyLength() < worm.Path.Total() {
		return
	}
	worm.X, worm.Y = clampToArena(diveEnd.X), clampToArena(diveEnd.Y)
	g.addWormCrater(worm.X, worm.Y)
	g.raiseWormDust(worm.X, worm.Y, wormEruptionDust/2)
	g.enterWormPhase(worm, WormBurrowing)
}

func wormColumnControls(worm *Sandworm, height float32) [4]Point3 {
	base := Point3{worm.HoleX, worm.HoleY, 0}
	return [4]Point3{base, {base.X, base.Y, height / 3}, {base.X, base.Y, height * 2 / 3}, {base.X, base.Y, height}}
}

func wormDiveControls(worm *Sandworm) [4]Point3 {
	head := worm.HeadPoint()
	directionX, directionY := normalize(worm.Aim.X+0.0001, worm.Aim.Y)
	reach := max(head.Z*wormDiveReach, wormHeadLength)
	end := Point3{clampToArena(head.X + directionX*reach), clampToArena(head.Y + directionY*reach), 0}
	lift := max(head.Z*wormDiveLift, wormHeadLength)
	return [4]Point3{head, {head.X + worm.Aim.X*lift, head.Y + worm.Aim.Y*lift, head.Z + max(worm.Aim.Z, 0)*lift + lift/2}, {end.X, end.Y, lift}, end}
}

func (g *Game) rebuildWormPathIf(worm *Sandworm, shouldRebuild bool) {
	if !shouldRebuild {
		return
	}
	path := &worm.Path
	path.Points[0], path.Lengths[0], path.Count = Point3{worm.HoleX, worm.HoleY, 0}, 0, 1
	wormPathBuilders[worm.Phase](worm)
	path.Arch = path.Total()
	appendDivePathIf(worm, worm.Phase == WormDiving)
	head, _ := path.At(worm.HeadAlong())
	worm.HeadX, worm.HeadY, worm.HeadZ = head.X, head.Y, head.Z
}

func appendColumnPath(worm *Sandworm) {
	appendBezierSamples(&worm.Path, worm.Controls)
}

func appendChainPath(worm *Sandworm) {
	for joint := 1; joint < worm.JointCount; joint++ {
		appendPathPoint(&worm.Path, worm.Joints[joint])
	}
	appendPathPoint(&worm.Path, worm.HeadPoint())
}

func appendDivePathIf(worm *Sandworm, shouldAppend bool) {
	if !shouldAppend {
		return
	}
	appendBezierSamples(&worm.Path, worm.DiveControls)
}

func appendBezierSamples(path *WormPath, controls [4]Point3) {
	for step := 1; step <= wormPathSamples; step++ {
		appendPathPoint(path, bezierPoint(controls, float32(step)/wormPathSamples))
	}
}

func appendPathPoint(path *WormPath, point Point3) {
	previous := path.Points[path.Count-1]
	path.Points[path.Count] = point
	path.Lengths[path.Count] = path.Lengths[path.Count-1] + length3(point.X-previous.X, point.Y-previous.Y, point.Z-previous.Z)
	path.Count++
}

func (g *Game) updateWormSegments(worm *Sandworm, deltaSeconds float32) {
	slot := 0
	for index := range worm.Segments {
		segment := &worm.Segments[index]
		g.updateWormSegment(worm, segment, slot, deltaSeconds)
		slot += boolToIndex(segment.IsAlive)
	}
}

func (g *Game) updateWormSegment(worm *Sandworm, segment *WormSegment, slot int, deltaSeconds float32) {
	if !segment.IsAlive {
		return
	}
	along := worm.SlotAlong(slot)
	point, _ := worm.Path.At(along)
	segment.X, segment.Y, segment.Z = point.X, point.Y, point.Z
	isUp := worm.IsAlongAboveGround(along)
	g.surfaceWormSegmentIf(worm, segment, isUp && segment.EnemyID == 0)
	g.buryWormSegmentIf(segment, !isUp && segment.EnemyID != 0)
	g.placeWormSegmentEnemy(segment)
	g.smoulderWormSegment(worm, segment, deltaSeconds)
}

func (g *Game) surfaceWormSegmentIf(worm *Sandworm, segment *WormSegment, shouldSurface bool) {
	if !shouldSurface {
		return
	}
	segment.EnemyID = g.enemies.Spawn(EnemySandwormSegment, segment.X, segment.Y-segment.Z, worm.Power)
	g.setEnemyHealthIfPresent(segment.EnemyID, segment.Health)
}

func (g *Game) buryWormSegmentIf(segment *WormSegment, shouldBury bool) {
	if !shouldBury {
		return
	}
	index := g.enemies.IndexOfID(segment.EnemyID)
	segment.EnemyID = 0
	if index < 0 {
		return
	}
	segment.Health = g.enemies.Health[index]
	g.enemies.Remove(index)
}

func (g *Game) placeWormSegmentEnemy(segment *WormSegment) {
	index := g.enemies.IndexOfID(segment.EnemyID)
	if index < 0 || segment.EnemyID == 0 {
		return
	}
	g.enemies.PositionX[index], g.enemies.PositionY[index] = segment.X, segment.Y-segment.Z
	g.enemies.KnockbackX[index], g.enemies.KnockbackY[index] = 0, 0
	segment.Health = g.enemies.Health[index]
}

func (g *Game) smoulderWormSegment(worm *Sandworm, segment *WormSegment, deltaSeconds float32) {
	segment.DamageSeconds -= deltaSeconds
	if segment.EnemyID == 0 || segment.DamageSeconds > 0 {
		return
	}
	segment.DamageSeconds += wormDamageInterval
	fraction := segment.Health / worm.SegmentMaximumHealth
	screenY := segment.Y - segment.Z
	g.effects.SpawnSparks(segment.X, screenY, 2*boolToIndex(fraction < wormSparkThreshold && g.random.Chance(0.5)), wormSparkColor, 180)
	g.effects.AddSmokeIf(segment.X, screenY-8, 7, fraction < wormSmokeThreshold)
	g.effects.SpawnFlames(segment.X, screenY-6, 2*boolToIndex(fraction < wormFireThreshold), wormFireColor)
}

// ===== Internal: losses =====

func (g *Game) collectWormLosses(worm *Sandworm) {
	for index := range worm.Segments {
		g.collectWormSegmentLoss(worm, &worm.Segments[index])
	}
	isHeadLost := worm.HeadEnemyID != 0 && g.enemies.IndexOfID(worm.HeadEnemyID) < 0
	g.destroyWormIf(worm, isHeadLost)
	g.detachWormHeadIf(worm, worm.IsActive && worm.IsBodyOut() && worm.AliveSegmentCount() == 0)
}

func (g *Game) collectWormSegmentLoss(worm *Sandworm, segment *WormSegment) {
	isLost := segment.IsAlive && segment.EnemyID != 0 && g.enemies.IndexOfID(segment.EnemyID) < 0
	if !isLost {
		return
	}
	segment.IsAlive, segment.EnemyID = false, 0
	screenY := segment.Y - segment.Z
	g.effects.SpawnSparks(segment.X, screenY, 26, wormFireColor, 260)
	g.effects.SpawnFlames(segment.X, screenY, 10, wormFireColor)
	for range 6 {
		g.effects.AddSmokeIf(segment.X+g.random.Between(-20, 20), screenY+g.random.Between(-20, 20), 12, true)
	}
	g.effects.AddRing(segment.X, screenY, 70, wormFireColor)
	g.effects.AddShake(5)
}

func (g *Game) detachWormHeadIf(worm *Sandworm, shouldDetach bool) {
	if !shouldDetach {
		return
	}
	worm.Beam.Phase = LaserCooldown
	worm.HeadX, worm.HeadY, worm.HeadZ = clampToArena(worm.HeadX), clampToArena(worm.HeadY), 0
	worm.Angle = atan2(worm.Aim.Y, worm.Aim.X)
	worm.HeadEnemyID = g.enemies.Spawn(EnemySandworm, worm.HeadX, worm.HeadY, worm.Power)
	g.setEnemyHealthIfPresent(worm.HeadEnemyID, worm.HeadHealth)
	g.slamWorm(worm, worm.HeadX, worm.HeadY)
	g.playSound(audio.SoundWormRoar)
	g.enterWormPhase(worm, WormHeadResting)
	worm.PhaseSeconds = wormHeadCycleSeconds - wormHeadAimSeconds - wormHeadRollSeconds - 1
}

func (g *Game) destroyWormIf(worm *Sandworm, shouldDestroy bool) {
	if !shouldDestroy {
		return
	}
	g.effects.SpawnSparks(worm.HeadX, worm.HeadY, 40, wormFireColor, 320)
	g.effects.AddPopup(g.player.X, g.player.Y-80, i18n.T("worm.destroyed"), accentColor)
	g.say(ChatterWormDown)
	g.effects.AddShake(14)
	worm.IsActive = false
}

// ===== Internal: lone head =====

func (g *Game) updateWormHeadAiming(worm *Sandworm, deltaSeconds float32) {
	g.followWormHead(worm)
	target := atan2(g.player.Y-worm.HeadY, g.player.X-worm.HeadX)
	isLocked := worm.PhaseSeconds >= wormHeadAimSeconds-wormHeadLockSeconds
	turn := wormHeadTurnRate * deltaSeconds * boolToFloat(!isLocked)
	worm.Angle += clamp(angleDifference(worm.Angle, target), -turn, turn)
	if worm.PhaseSeconds < wormHeadAimSeconds {
		return
	}
	worm.RollFromX, worm.RollFromY, worm.HasRollHit = worm.HeadX, worm.HeadY, false
	g.playSound(audio.SoundWormRoll)
	g.enterWormPhase(worm, WormHeadRolling)
}

func (g *Game) updateWormHeadRolling(worm *Sandworm, deltaSeconds float32) {
	progress := clamp(worm.PhaseSeconds/wormHeadRollSeconds, 0, 1)
	eased := progress * progress * (3 - 2*progress)
	directionX, directionY := worm.Direction()
	worm.HeadX = clampToArena(worm.RollFromX + directionX*wormRollDistance*eased)
	worm.HeadY = clampToArena(worm.RollFromY + directionY*wormRollDistance*eased)
	worm.RollSpin += wormRollSpinRate * deltaSeconds * (1 - abs(2*progress-1))
	g.followWormHead(worm)
	g.crushUnderWormHead(worm, deltaSeconds)
	g.kickUpWormDust(worm, worm.HeadX, worm.HeadY, deltaSeconds)
	if progress < 1 {
		return
	}
	g.enterWormPhase(worm, WormHeadResting)
}

func (g *Game) updateWormHeadResting(worm *Sandworm, deltaSeconds float32) {
	g.followWormHead(worm)
	if worm.PhaseSeconds < wormHeadCycleSeconds-wormHeadAimSeconds-wormHeadRollSeconds {
		return
	}
	g.enterWormPhase(worm, WormHeadAiming)
}

func (g *Game) followWormHead(worm *Sandworm) {
	index := g.enemies.IndexOfID(worm.HeadEnemyID)
	if index < 0 {
		return
	}
	g.enemies.PositionX[index], g.enemies.PositionY[index] = worm.HeadX, worm.HeadY
	g.enemies.KnockbackX[index], g.enemies.KnockbackY[index] = 0, 0
	worm.HeadHealth = g.enemies.Health[index]
}

func (g *Game) crushUnderWormHead(worm *Sandworm, deltaSeconds float32) {
	g.ApplyBurst(Burst{X: worm.HeadX, Y: worm.HeadY, Radius: wormRollRadius, Damage: wormRollEnemyDamage * worm.Power * deltaSeconds, Element: ElementPhysical})
	reach := float32(wormRollRadius + playerRadius)
	isPlayerHit := !worm.HasRollHit && distanceSquared(worm.HeadX, worm.HeadY, g.player.X, g.player.Y) < reach*reach
	worm.HasRollHit = worm.HasRollHit || isPlayerHit
	g.DamagePlayer(wormRollPlayerHurt * worm.Power * boolToFloat(isPlayerHit))
	g.effects.AddShake(6 * boolToFloat(isPlayerHit))
}

// ===== Internal: shared =====

func (g *Game) slamWorm(worm *Sandworm, x, y float32) {
	g.ApplyBurst(Burst{X: x, Y: y, Radius: wormEruptionRadius, Damage: wormEruptionDamage * worm.Power, Element: ElementPhysical})
	isPlayerHit := distanceSquared(x, y, g.player.X, g.player.Y) < wormEruptionRadius*wormEruptionRadius
	g.DamagePlayer(wormEruptionPlayerHurt * worm.Power * boolToFloat(isPlayerHit))
	g.effects.SpawnSparks(x, y, 18, wormDirtColor, 220)
	g.effects.AddRing(x, y, wormEruptionRadius, wormDirtColor)
	g.effects.AddShake(7)
	g.playSound(audio.SoundStomp)
}

func (g *Game) kickUpWormDust(worm *Sandworm, x, y, deltaSeconds float32) {
	worm.DustSeconds -= deltaSeconds
	if worm.DustSeconds > 0 {
		return
	}
	worm.DustSeconds += wormDustInterval
	g.effects.AddDust(x+g.random.Between(-18, 18), y+g.random.Between(-12, 12), g.random.Between(8, 14))
	g.effects.SpawnSparks(x, y, wormWakeClods, wormDirtColor, 110)
}

func (g *Game) eruptWorm(worm *Sandworm) {
	x, y := worm.HoleX, worm.HoleY
	g.ApplyBurst(Burst{X: x, Y: y, Radius: wormEruptionRadius, Damage: wormEruptionDamage * worm.Power, Element: ElementPhysical})
	g.addWormCrater(x, y)
	g.raiseWormDust(x, y, wormEruptionDust)
	g.effects.SpawnSparks(x, y, 30, wormDirtColor, 320)
	g.effects.AddRing(x, y, wormEruptionRadius, wormDirtColor)
	g.effects.AddShake(10)
	g.playSound(audio.SoundStomp)
	g.playSound(audio.SoundWormRoar)
	reach := float32(wormEruptionRadius * wormBlastReach)
	isPlayerCaught := distanceSquared(x, y, g.player.X, g.player.Y) < reach*reach
	g.blastPlayerIf(worm, x, y, isPlayerCaught)
}

func (g *Game) blastPlayerIf(worm *Sandworm, x, y float32, isCaught bool) {
	if !isCaught {
		return
	}
	awayX, awayY := normalize(g.player.X-x+boolToFloat(g.player.X == x), g.player.Y-y)
	g.player.ShoveX, g.player.ShoveY = awayX*wormBlastShove, awayY*wormBlastShove
	worm.BlastHits, worm.BlastSeconds = wormBlastHits, 0
}

func (g *Game) updateWormBlast(worm *Sandworm, deltaSeconds float32) {
	worm.BlastSeconds -= deltaSeconds
	if worm.BlastHits == 0 || worm.BlastSeconds > 0 {
		return
	}
	worm.BlastSeconds += wormBlastInterval
	worm.BlastHits--
	player := g.player
	player.Health = max(min(player.Health, 1), player.Health-wormBlastDamage*worm.Power)
	player.DamageFlashSeconds = 0.2
	g.effects.SpawnSparks(player.X, player.Y, 6, wormDirtColor, 180)
	g.effects.AddDust(player.X, player.Y, 10)
	g.effects.AddShake(3)
	g.playSound(audio.SoundHurt)
}

func (g *Game) raiseWormDust(x, y float32, count int) {
	for range count {
		angle := g.random.Angle()
		distance := g.random.Between(0, wormEruptionRadius)
		g.effects.AddDust(x+cosine(angle)*distance, y+sine(angle)*distance*0.8, g.random.Between(12, 24))
	}
}

func (g *Game) addWormCrater(x, y float32) {
	g.wormCraters = append(g.wormCraters, WormCrater{X: x, Y: y, Variant: g.random.Below(wormCraterVariants), Life: wormCraterSeconds})
}

func (g *Game) ageWormCraters(deltaSeconds float32) {
	g.wormCraters = filterAlive(g.wormCraters, deltaSeconds, func(crater *WormCrater) *float32 { return &crater.Life })
}

func (g *Game) setEnemyHealthIfPresent(id uint32, health float32) {
	index := g.enemies.IndexOfID(id)
	if index < 0 || id == 0 {
		return
	}
	g.enemies.Health[index] = health
}

func clampToArena(value float32) float32 {
	return clamp(value, wormArenaMargin, arenaSize-wormArenaMargin)
}

func bezierPoint(controls [4]Point3, t float32) Point3 {
	inverse := 1 - t
	weights := [4]float32{inverse * inverse * inverse, 3 * inverse * inverse * t, 3 * inverse * t * t, t * t * t}
	var point Point3
	for index, control := range controls {
		point.X += control.X * weights[index]
		point.Y += control.Y * weights[index]
		point.Z += control.Z * weights[index]
	}
	return point
}

func bezierLength(controls [4]Point3) float32 {
	total := float32(0)
	previous := controls[0]
	for step := 1; step <= wormPathSamples; step++ {
		point := bezierPoint(controls, float32(step)/wormPathSamples)
		total += length3(point.X-previous.X, point.Y-previous.Y, point.Z-previous.Z)
		previous = point
	}
	return total
}

func lerpPoint(from, to Point3, weight float32) Point3 {
	return Point3{from.X + (to.X-from.X)*weight, from.Y + (to.Y-from.Y)*weight, from.Z + (to.Z-from.Z)*weight}
}

func length3(x, y, z float32) float32 {
	return sqrt(x*x + y*y + z*z)
}

func normalizeVector3(x, y, z float32) (float32, float32, float32) {
	size := max(length3(x, y, z), 0.0001)
	return x / size, y / size, z / size
}

func moveTowardPoint(from, to Point3, maximumStep float32) Point3 {
	distance := length3(to.X-from.X, to.Y-from.Y, to.Z-from.Z)
	weight := clamp(maximumStep/max(distance, 0.0001), 0, 1)
	return lerpPoint(from, to, weight)
}

func turnVectorToward(from, to Point3, maximumAngle float32) Point3 {
	cosineBetween := clamp(from.X*to.X+from.Y*to.Y+from.Z*to.Z, -1, 1)
	angle := acos(cosineBetween)
	weight := clamp(maximumAngle/max(angle, 0.0001), 0, 1)
	blended := lerpPoint(from, Point3{to.X + 0.001, to.Y, to.Z}, weight)
	x, y, z := normalizeVector3(blended.X, blended.Y, blended.Z)
	return Point3{x, y, z}
}

func pointAtDistance(anchor, toward Point3, distance float32) Point3 {
	x, y, z := normalizeVector3(toward.X-anchor.X, toward.Y-anchor.Y, toward.Z-anchor.Z)
	return Point3{anchor.X + x*distance, anchor.Y + y*distance, max(0, anchor.Z+z*distance)}
}
