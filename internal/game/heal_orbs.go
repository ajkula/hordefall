package game

import "hordefall/internal/audio"

// ===== Types =====

type HealOrb struct {
	X           float32
	Y           float32
	Age         float32
	IsAttracted bool
}

// ===== Constants =====

const (
	healOrbDropChance   = 0.02
	healOrbAmount       = 20
	maximumHealOrbs     = 64
	healOrbRadius       = 6
	healOrbGlowScale    = 3.4
	healOrbPulseSpeed   = 5
	healOrbCrossLength  = 3.6
	healOrbCrossWidth   = 1.6
	healOrbPopupRise    = 20
	healOrbDropScatter  = 10
	healOrbPulseDepth   = 0.18
	healOrbGlowStrength = 0.4
)

var (
	healOrbColor  = [3]float32{0.25, 0.95, 0.4}
	healCrossTint = [3]float32{1, 1, 1}
)

// ===== Internal =====

func (g *Game) dropHealOrbIf(x, y float32) {
	isDropping := g.random.Chance(healOrbDropChance) && len(g.healOrbs) < maximumHealOrbs
	if !isDropping {
		return
	}
	g.healOrbs = append(g.healOrbs, HealOrb{
		X: x + g.random.Between(-healOrbDropScatter, healOrbDropScatter),
		Y: y + g.random.Between(-healOrbDropScatter, healOrbDropScatter),
	})
}

func (g *Game) updateHealOrbs(deltaSeconds float32) {
	for index := len(g.healOrbs) - 1; index >= 0; index-- {
		g.updateHealOrb(index, deltaSeconds)
	}
}

func (g *Game) updateHealOrb(index int, deltaSeconds float32) {
	player := g.player
	orb := &g.healOrbs[index]
	orb.Age += deltaSeconds
	distance := length(player.X-orb.X, player.Y-orb.Y)
	orb.IsAttracted = orb.IsAttracted || distance < player.MagnetRadius
	directionX, directionY := normalize(player.X-orb.X, player.Y-orb.Y)
	step := gemAttractSpeed * deltaSeconds * boolToFloat(orb.IsAttracted)
	orb.X += directionX * step
	orb.Y += directionY * step
	if distance > gemCollectRadius {
		return
	}
	g.collectHealOrb(index)
}

func (g *Game) collectHealOrb(index int) {
	player := g.player
	player.Health = min(player.MaximumHealth, player.Health+healOrbAmount)
	g.effects.AddPopup(player.X, player.Y-healOrbPopupRise, "+20 HP", healOrbColor)
	g.playSound(audio.SoundGem)
	g.healOrbs[index] = g.healOrbs[len(g.healOrbs)-1]
	g.healOrbs = g.healOrbs[:len(g.healOrbs)-1]
}

func (r *Renderer) queueHealOrbs(g *Game) {
	for _, orb := range g.healOrbs {
		screenX, screenY := r.ToScreen(orb.X, orb.Y)
		isVisible := r.isVisible(orb.X, orb.Y, healOrbRadius*healOrbGlowScale)
		radius := healOrbRadius * (1 + healOrbPulseDepth*sine(orb.Age*healOrbPulseSpeed))
		r.glow.AddCircleIf(screenX, screenY, radius*healOrbGlowScale, healOrbColor, healOrbGlowStrength, isVisible)
		r.solid.AddCircleIf(screenX, screenY, radius, healOrbColor, 1, isVisible)
		r.queueHealCrossIf(screenX, screenY, isVisible)
	}
}

func (r *Renderer) queueHealCrossIf(x, y float32, isVisible bool) {
	if !isVisible {
		return
	}
	r.solid.AddSegment(x-healOrbCrossLength, y, x+healOrbCrossLength, y, healOrbCrossWidth, healCrossTint, 1)
	r.solid.AddSegment(x, y-healOrbCrossLength, x, y+healOrbCrossLength, healOrbCrossWidth, healCrossTint, 1)
}
