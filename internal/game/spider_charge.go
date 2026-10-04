package game

// ===== Types =====

type ChargeMote struct {
	OffsetX float32
	OffsetY float32
	Age     float32
	Life    float32
	Size    float32
}

// ===== Constants =====

const (
	chargeMoteCount        = 64
	chargeMoteNearest      = 70
	chargeMoteFarthest     = 170
	chargeMoteLifeMinimum  = 0.35
	chargeMoteLifeMaximum  = 0.65
	chargeMoteStretch      = 2.6
	chargeMergeStart       = 0.85
	chargeMergeEnd         = 1.15
	chargeSwellPerMote     = 0.9
	chargeSwellMaximum     = 5
	chargeMotesPerTickMax  = 3
	chargeSphereMinimum    = 3
	chargeSphereGrowth     = 17
	chargeHaloScale        = 2.6
	dischargeWindow        = 0.6
	dischargeBoltChance    = 0.55
	dischargeBoltReachMin  = 50
	dischargeBoltReachMax  = 130
	dischargeBounceCount   = 2
	dischargeBounceTurn    = 1.3
	dischargeBounceShrink  = 0.65
	dischargeImpactSparks  = 4
	dischargeImpactSpeed   = 120
	dischargeSphereFlicker = 0.35
)

var chargeColor = [3]float32{1, 1, 1}

// ===== Internal =====

func (g *Game) drawInChargeMotes(laser *SpiderLaser, deltaSeconds float32) {
	spawnCount := 1 + int(laser.PhaseProgress()*chargeMotesPerTickMax)
	for range spawnCount {
		angle := g.random.Angle()
		distance := g.random.Between(chargeMoteNearest, chargeMoteFarthest)
		laser.Motes[laser.NextMote] = ChargeMote{
			OffsetX: cosine(angle) * distance, OffsetY: sine(angle) * distance,
			Life: g.random.Between(chargeMoteLifeMinimum, chargeMoteLifeMaximum), Size: g.random.Between(2.2, 4.2),
		}
		laser.NextMote = (laser.NextMote + 1) % chargeMoteCount
	}
	ageChargeMotes(laser, deltaSeconds)
}

func ageChargeMotes(laser *SpiderLaser, deltaSeconds float32) {
	for index := range laser.Motes {
		laser.Motes[index].Age += deltaSeconds
	}
}

func (g *Game) radiateDischarge(laser *SpiderLaser, deltaSeconds float32) {
	ageChargeMotes(laser, deltaSeconds)
	isRadiating := laser.PhaseProgress() < dischargeWindow && g.random.Chance(dischargeBoltChance)
	if !isRadiating {
		return
	}
	g.ricochetBolt(laser.OriginX, laser.OriginY, g.random.Angle(), g.random.Between(dischargeBoltReachMin, dischargeBoltReachMax))
}

func (g *Game) ricochetBolt(fromX, fromY, angle, reach float32) {
	for range dischargeBounceCount + 1 {
		toX, toY := fromX+cosine(angle)*reach, fromY+sine(angle)*reach
		g.effects.AddLightning(fromX, fromY, toX, toY)
		g.effects.SpawnSparks(toX, toY, dischargeImpactSparks, chargeColor, dischargeImpactSpeed)
		fromX, fromY = toX, toY
		angle += g.random.Between(-dischargeBounceTurn, dischargeBounceTurn)
		reach *= dischargeBounceShrink
	}
}

func (r *Renderer) drawChargeMotes(laser *SpiderLaser, sphereRadius float32) {
	originX, originY := r.ToScreen(laser.OriginX, laser.OriginY)
	for index := range laser.Motes {
		r.drawChargeMote(&laser.Motes[index], originX, originY, sphereRadius)
	}
}

func (r *Renderer) drawChargeMote(mote *ChargeMote, originX, originY, sphereRadius float32) {
	progress := mote.Age / max(mote.Life, 0.001)
	isAlive := progress < 1 && mote.Life > 0
	if !isAlive {
		return
	}
	remaining := 1 - progress*progress
	distance := length(mote.OffsetX, mote.OffsetY) * remaining
	directionX, directionY := normalize(mote.OffsetX, mote.OffsetY)
	stretch := 1 + chargeMoteStretch*progress*progress
	fusion := clamp(distance/max(sphereRadius, 1), 0, 1)
	radius := mote.Size * fusion
	x, y := originX+directionX*distance, originY+directionY*distance
	alpha := clamp(progress*3, 0, 1)
	r.glow.AddEllipse(x, y, radius*stretch*2.4, radius*2.4/sqrt(stretch), directionX, directionY, chargeColor, 0.45*alpha)
	r.solid.AddEllipse(x, y, radius*stretch, radius/sqrt(stretch), directionX, directionY, chargeColor, alpha)
}

func chargeSphereSwell(laser *SpiderLaser) float32 {
	merging := 0
	for index := range laser.Motes {
		mote := &laser.Motes[index]
		progress := mote.Age / max(mote.Life, 0.001)
		merging += boolToIndex(mote.Life > 0 && progress >= chargeMergeStart && progress < chargeMergeEnd)
	}
	return min(chargeSwellMaximum, float32(merging)*chargeSwellPerMote)
}

func (r *Renderer) drawChargeSphere(laser *SpiderLaser, radius, brightness float32) {
	originX, originY := r.ToScreen(laser.OriginX, laser.OriginY)
	r.glow.AddCircle(originX, originY, radius*chargeHaloScale, chargeColor, 0.35*brightness)
	r.glow.AddCircle(originX, originY, radius*1.4, chargeColor, 0.6*brightness)
	r.solid.AddCircle(originX, originY, radius, chargeColor, brightness)
}
