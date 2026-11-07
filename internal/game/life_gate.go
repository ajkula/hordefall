package game

import (
	"slices"

	"github.com/ajkula/ajkurig/rng"
)

// ===== Types =====

type SteamPuff struct {
	Angle     float32
	Remaining float32
}

type LifeGate struct {
	Seconds    float32
	NextPuffIn float32
	Puffs      [lifeGatePuffCount]SteamPuff
	random     rng.Random
}

// ===== Constants =====

const (
	lifeGateHealthFraction      = 0.7
	lifeGateHealth              = 1
	lifeGateInvulnerableSeconds = 1
	lifeGatePuffCount           = 3
	lifeGateSeed                = 0x57EA4
	steamPuffSeconds            = 0.45
	steamPuffIntervalMinimum    = 0.05
	steamPuffIntervalMaximum    = 0.14
	steamPuffCenterForward      = -2
	steamPuffStartDistance      = 27
	steamPuffTravelDistance     = 20
	steamPuffRadius             = 6
	steamPuffGrowth             = 0.4
	steamPuffBirthAlpha         = 0.05
	steamPuffPeakAlpha          = 0.4
	steamPuffFadeFraction       = 0.25
	steamSparkMinimum           = 1
	steamSparkMaximum           = 3
	steamSparkSpread            = 0.5
	steamSparkSpeed             = 150
)

var (
	steamColor = [3]float32{0.9, 0.93, 0.95}
	emberColor = [3]float32{1, 0.62, 0.22}
)

// ===== Public API =====

func NewLifeGate() LifeGate {
	return LifeGate{random: rng.New(lifeGateSeed)}
}

func (p *Player) IsLifeGated(amount float32) bool {
	return amount >= p.Health && p.Health > lifeGateHealthFraction*p.MaximumHealth
}

// ===== Internal =====

func (g *Game) openLifeGateIf(isGated bool) {
	if !isGated {
		return
	}
	g.player.LifeGate.Seconds = lifeGateInvulnerableSeconds
}

func (g *Game) updateLifeGate(deltaSeconds float32) {
	player := g.player
	gate := &player.LifeGate
	slot, isEmitted := gate.update(deltaSeconds)
	if !isEmitted {
		return
	}
	angle := gate.Puffs[slot].Angle
	x, y := player.Rig.LocalToWorld(player.X, player.Y, steamPuffCenterForward+cosine(angle)*steamPuffStartDistance, sine(angle)*steamPuffStartDistance)
	sparks := steamSparkMinimum + gate.random.Below(steamSparkMaximum-steamSparkMinimum+1)
	g.effects.SpawnSparksToward(x, y, sparks, player.Rig.Heading+angle, steamSparkSpread, emberColor, steamSparkSpeed)
}

func (gate *LifeGate) update(deltaSeconds float32) (slot int, isEmitted bool) {
	gate.Seconds = max(0, gate.Seconds-deltaSeconds)
	gate.NextPuffIn = max(0, gate.NextPuffIn-deltaSeconds)
	for index := range gate.Puffs {
		gate.Puffs[index].Remaining = max(0, gate.Puffs[index].Remaining-deltaSeconds)
	}
	if gate.Seconds == 0 || gate.NextPuffIn > 0 {
		return 0, false
	}
	gate.NextPuffIn = gate.random.Between(steamPuffIntervalMinimum, steamPuffIntervalMaximum)
	slot = slices.IndexFunc(gate.Puffs[:], func(puff SteamPuff) bool { return puff.Remaining == 0 })
	if slot < 0 {
		return 0, false
	}
	gate.Puffs[slot] = SteamPuff{Angle: gate.random.Angle(), Remaining: steamPuffSeconds}
	return slot, true
}

func (r *Renderer) queueSteamPuffs(player *Player) {
	for index := range player.LifeGate.Puffs {
		r.queueSteamPuff(player, &player.LifeGate.Puffs[index])
	}
}

func (r *Renderer) queueSteamPuff(player *Player, puff *SteamPuff) {
	if puff.Remaining == 0 {
		return
	}
	progress := 1 - puff.Remaining/steamPuffSeconds
	distance := steamPuffStartDistance + steamPuffTravelDistance*progress*(2-progress)
	x, y := r.tsukumoPoint(player, steamPuffCenterForward+cosine(puff.Angle)*distance, sine(puff.Angle)*distance)
	fadeOut := clamp((1-progress)/steamPuffFadeFraction, 0, 1)
	alpha := (steamPuffBirthAlpha + (steamPuffPeakAlpha-steamPuffBirthAlpha)*progress) * fadeOut
	r.batches.Solid.AddCircle(x, y, steamPuffRadius*(1+steamPuffGrowth*progress), steamColor, alpha)
}
