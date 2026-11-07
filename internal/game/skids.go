package game

import (
	"github.com/ajkula/ajkurig/colorx"
	"github.com/ajkula/ajkurig/fx"
)

// ===== Constants =====

const (
	skidWidthPerWheel = 0.8
	skidDustChance    = 0.3
)

var (
	effectsPalette = fx.Palette{
		Smoke: colorx.RGB{0.2, 0.2, 0.22}, Dust: colorx.RGB{0.56, 0.45, 0.32},
		BoltGlow: colorx.RGB{1, 1, 0.4}, BoltCore: colorx.RGB{1, 1, 1}, SkidColor: colorx.RGB{0.95, 0.97, 1},
	}
	dustColor = [3]float32{0.75, 0.78, 0.82}
)

// ===== Internal =====

func newEffects() *fx.Effects {
	effects := fx.NewEffects(effectsRandomSeed)
	effects.Palette = effectsPalette
	return effects
}

func (g *Game) updateSkid(rig *TsukumoRig, accelerationX, accelerationY float32) {
	for leg := range tsukumoLegCount {
		g.markWheel(&rig.Legs[leg], accelerationX, accelerationY)
	}
}

func (g *Game) markWheel(state *TsukumoLeg, accelerationX, accelerationY float32) {
	state.Skid.Steer(accelerationX, accelerationY)
	wheelDiameter := float32(2 * tsukumoWheelRadius * tsukumoScale)
	isMarked := state.Skid.Mark(&g.effects.Skids, state.FootX, state.FootY, wheelDiameter*skidWidthPerWheel)
	g.raiseDustIf(state.FootX, state.FootY, isMarked && g.random.Chance(state.Skid.Amount*skidDustChance))
}

func (g *Game) raiseDustIf(x, y float32, shouldRaise bool) {
	if !shouldRaise {
		return
	}
	g.effects.SpawnSparks(x, y, 1, dustColor, 40)
}
