package game

import (
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ajkula/ajkurig/colorx"
	"github.com/ajkula/ajkurig/draw"
)

// ===== Types =====

type NukeBlast struct {
	X           float32
	Y           float32
	Seconds     float32
	FrontRadius float32
	Stamp       uint32
	SteamPuffs  int
	hitBossIDs  []uint32
}

// ===== Constants =====

const (
	nukeRadius             = screenWidth / 2
	nukeClearingRadius     = 130
	nukeKillOverkill       = 2
	nukeBossDamageFraction = 0.15
	nukeGrowthSeconds      = 0.3
	nukeDissolveSeconds    = 0.5
	nukeDissolveGrowth     = 0.08
	nukeAuraScale          = 1.15
	nukeAuraAlpha          = 0.25
	nukeSeedStep           = 3.7
	nukeGoldenAngle        = 2.39996
	nukeFrontFlamesPerTick = 20
	nukeInnerFlamesPerTick = 16
	nukeSmokesPerTick      = 1
	nukeSmokeSize          = 20
	nukeSteamCellStride    = 2
	nukeSteamPuffMaximum   = 240
	nukeSteamPuffSize      = 24
	nukeSteamOpacity       = 0.15
	nukeSteamJitterModulus = 17
)

var (
	nukeFlameColor = colorx.RGB{1, 0.42, 0.05}
	nukeSteamColor = colorx.RGB{0.92, 0.95, 1}
)

const fireballSource = `//kage:unit pixels
package main

var Center vec2
var Radius float
var Clock float
var Dissolve float
var Seed float

func hash(p vec2) float {
	return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453)
}

func noise(p vec2) float {
	cell := floor(p)
	blend := fract(p)
	blend = blend * blend * (3.0 - 2.0*blend)
	bottom := mix(hash(cell), hash(cell+vec2(1.0, 0.0)), blend.x)
	top := mix(hash(cell+vec2(0.0, 1.0)), hash(cell+vec2(1.0, 1.0)), blend.x)
	return mix(bottom, top, blend.y)
}

func fractal(p vec2) float {
	total := 0.0
	weight := 0.5
	for octave := 0; octave < 4; octave++ {
		total += weight * noise(p)
		p = p * 2.03
		weight *= 0.5
	}
	return total
}

func fireColor(heat float) vec3 {
	shade := mix(vec3(0.16, 0.02, 0.01), vec3(0.85, 0.12, 0.02), smoothstep(0.0, 0.3, heat))
	shade = mix(shade, vec3(1.0, 0.5, 0.06), smoothstep(0.25, 0.55, heat))
	shade = mix(shade, vec3(1.0, 0.86, 0.25), smoothstep(0.5, 0.78, heat))
	return mix(shade, vec3(1.0, 0.98, 0.88), smoothstep(0.78, 1.0, heat))
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	relative := (dstPos.xy - imageDstOrigin() - Center) / Radius
	distance := length(relative)
	dome := sqrt(max(0.0, 1.0-min(distance*distance, 1.0)))
	surface := relative / (1.0 + dome)
	flow := fractal(surface*4.0 + vec2(Seed, Seed*1.7-Clock*1.4))
	curls := fractal(surface*9.0 - vec2(Clock*0.9, Seed) + flow*2.0)
	turbulence := flow*0.65 + curls*0.35
	body := 1.0 - smoothstep(0.78, 0.98, distance+(turbulence-0.45)*0.32)
	heat := dome*0.5 - relative.y*0.14 + (turbulence-0.45)*1.3 + 0.1 - Dissolve*1.15
	burn := Dissolve * 1.25
	holes := smoothstep(burn-0.12, burn+0.12, turbulence+dome*0.25+0.05)
	alpha := body * holes * 0.88
	return vec4(fireColor(clamp(heat, 0.0, 1.0))*alpha, alpha)
}
`

var nukeGroundNeighbors = [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

var puddleReactions = [groundKindCount]func(g *Game, blast *NukeBlast, column, row int){
	GroundOil:   (*Game).igniteOilCell,
	GroundWater: (*Game).steamWaterCell,
	GroundIce:   (*Game).meltIceCell,
}

// ===== Public API =====

func (g *Game) detonateNuke() {
	player := g.player
	g.nukeStamp++
	g.nukeBlasts = append(g.nukeBlasts, NukeBlast{X: player.X, Y: player.Y, Stamp: g.nukeStamp})
	g.effects.AddRing(player.X, player.Y, nukeRadius, logoNukeColor)
	g.effects.AddShake(14)
	g.effects.SpawnSparks(player.X, player.Y, 40, logoNukeColor, 420)
	g.playSound(SoundExplosion)
}

func (blast *NukeBlast) DissolveProgress() float32 {
	return clamp((blast.Seconds-nukeGrowthSeconds)/nukeDissolveSeconds, 0, 1)
}

// ===== Internal =====

func (g *Game) updateNukeBlasts(deltaSeconds float32) {
	for index := range g.nukeBlasts {
		g.updateNukeBlast(&g.nukeBlasts[index], deltaSeconds)
	}
	g.nukeBlasts = slices.DeleteFunc(g.nukeBlasts, func(blast NukeBlast) bool {
		return blast.DissolveProgress() >= 1
	})
}

func (g *Game) updateNukeBlast(blast *NukeBlast, deltaSeconds float32) {
	blast.Seconds += deltaSeconds
	previousFront := blast.FrontRadius
	progress := clamp(blast.Seconds/nukeGrowthSeconds, 0, 1)
	blast.FrontRadius = nukeRadius * progress * (2 - progress)
	if blast.FrontRadius <= previousFront {
		g.spawnInnerFire(blast)
		return
	}
	g.burnEnemiesInside(blast)
	g.touchPuddlesBetween(blast, previousFront)
	g.spawnFrontFlames(blast)
}

func (g *Game) burnEnemiesInside(blast *NukeBlast) {
	enemies := g.enemies
	for index := range enemies.Count {
		isInside := distanceSquared(enemies.PositionX[index], enemies.PositionY[index], blast.X, blast.Y) < blast.FrontRadius*blast.FrontRadius
		if isInside {
			g.burnEnemy(blast, index)
		}
	}
}

func (g *Game) burnEnemy(blast *NukeBlast, index int) {
	enemies := g.enemies
	kind := enemies.Kind[index]
	if kind < EnemySpiderTank {
		g.HitEnemy(index, nukeKillOverkill*enemies.Health[index], ElementFire)
		return
	}
	if slices.Contains(blast.hitBossIDs, enemies.ID[index]) {
		return
	}
	blast.hitBossIDs = append(blast.hitBossIDs, enemies.ID[index])
	g.HitEnemy(index, nukeBossDamageFraction*enemyTable[kind].Health*enemies.Power[index], ElementFire)
}

func (g *Game) touchPuddlesBetween(blast *NukeBlast, innerRadius float32) {
	ground := g.ground
	reach := int(blast.FrontRadius)/groundCellSize + 1
	centerColumn, centerRow := int(blast.X)/groundCellSize, int(blast.Y)/groundCellSize
	for row := max(0, centerRow-reach); row <= min(ground.Rows-1, centerRow+reach); row++ {
		for column := max(0, centerColumn-reach); column <= min(ground.Columns-1, centerColumn+reach); column++ {
			g.touchPuddleIfCrossed(blast, column, row, innerRadius)
		}
	}
}

func (g *Game) touchPuddleIfCrossed(blast *NukeBlast, column, row int, innerRadius float32) {
	x, y := cellCenter(column, row)
	distance := distanceSquared(x, y, blast.X, blast.Y)
	isCrossed := distance >= innerRadius*innerRadius && distance < blast.FrontRadius*blast.FrontRadius
	if !isCrossed {
		return
	}
	g.floodPuddle(blast, column, row)
	if distance >= nukeClearingRadius*nukeClearingRadius {
		g.ground.StimulateAt(x, y, StimulusScorch)
	}
}

func (g *Game) floodPuddle(blast *NukeBlast, column, row int) {
	ground := g.ground
	material := ground.MaterialAtCell(column, row)
	reaction := puddleReactions[material]
	index := row*ground.Columns + column
	if reaction == nil || g.puddleMarks[index] == blast.Stamp {
		return
	}
	g.puddleMarks[index] = blast.Stamp
	g.puddleQueue = append(g.puddleQueue[:0], int32(index))
	for head := 0; head < len(g.puddleQueue); head++ {
		cell := int(g.puddleQueue[head])
		cellColumn, cellRow := cell%ground.Columns, cell/ground.Columns
		g.queuePuddleNeighbors(blast, cellColumn, cellRow, material)
		reaction(g, blast, cellColumn, cellRow)
	}
	g.playSoundIf(SoundHiss, material != GroundOil)
}

func (g *Game) queuePuddleNeighbors(blast *NukeBlast, column, row int, material GroundKind) {
	ground := g.ground
	for _, step := range nukeGroundNeighbors {
		nextColumn, nextRow := column+step[0], row+step[1]
		isInside := nextColumn >= 0 && nextRow >= 0 && nextColumn < ground.Columns && nextRow < ground.Rows
		if !isInside {
			continue
		}
		index := nextRow*ground.Columns + nextColumn
		if ground.MaterialAtCell(nextColumn, nextRow) != material || g.puddleMarks[index] == blast.Stamp {
			continue
		}
		g.puddleMarks[index] = blast.Stamp
		g.puddleQueue = append(g.puddleQueue, int32(index))
	}
}

func (g *Game) igniteOilCell(_ *NukeBlast, column, row int) {
	x, y := cellCenter(column, row)
	g.ground.StimulateAt(x, y, StimulusHeat)
}

func (g *Game) meltIceCell(blast *NukeBlast, column, row int) {
	x, y := cellCenter(column, row)
	g.ground.StimulateAt(x, y, StimulusScorch)
	g.steamWaterCell(blast, column, row)
}

func (g *Game) steamWaterCell(blast *NukeBlast, column, row int) {
	isSampled := column%nukeSteamCellStride == 0 && row%nukeSteamCellStride == 0
	if !isSampled || blast.SteamPuffs >= nukeSteamPuffMaximum {
		return
	}
	blast.SteamPuffs++
	x, y := cellCenter(column, row)
	x += float32((column*7+row*13)%nukeSteamJitterModulus) - nukeSteamJitterModulus/2
	y += float32((column*11+row*5)%nukeSteamJitterModulus) - nukeSteamJitterModulus/2
	g.effects.AddCloud(x, y, nukeSteamPuffSize, nukeSteamOpacity, nukeSteamColor)
}

func (g *Game) spawnFrontFlames(blast *NukeBlast) {
	for flame := range nukeFrontFlamesPerTick {
		angle := float32(flame) * fullTurn / nukeFrontFlamesPerTick
		g.effects.SpawnFlames(blast.X+cosine(angle)*blast.FrontRadius, blast.Y+sine(angle)*blast.FrontRadius, 1, nukeFlameColor)
	}
}

func (g *Game) spawnInnerFire(blast *NukeBlast) {
	for flame := range nukeInnerFlamesPerTick {
		x, y := g.innerFirePoint(blast, flame)
		g.effects.SpawnFlames(x, y, 1, nukeFlameColor)
	}
	for smoke := range nukeSmokesPerTick {
		x, y := g.innerFirePoint(blast, smoke)
		g.effects.AddSmokeIf(x, y, nukeSmokeSize, true)
	}
}

func (g *Game) innerFirePoint(blast *NukeBlast, index int) (x, y float32) {
	seed := float64(g.frame)*0.37 + float64(index)*0.618034
	fraction := float32(seed - math.Floor(seed))
	angle := float32(g.frame+uint32(index)) * nukeGoldenAngle
	distance := blast.FrontRadius * float32(math.Sqrt(float64(fraction)))
	return blast.X + cosine(angle)*distance, blast.Y + sine(angle)*distance
}

func cellCenter(column, row int) (x, y float32) {
	return float32(column*groundCellSize) + groundCellSize/2, float32(row*groundCellSize) + groundCellSize/2
}

func (r *Renderer) warmFireballShaderOnce() {
	if r.isFireballWarm {
		return
	}
	r.isFireballWarm = true
	ebiten.NewImage(1, 1).DrawRectShader(1, 1, r.fireballShader, &ebiten.DrawRectShaderOptions{})
}

func (r *Renderer) drawNukeBlasts(g *Game, screen *ebiten.Image) {
	r.warmFireballShaderOnce()
	if len(g.nukeBlasts) == 0 {
		return
	}
	r.batches.Solid.Flush(screen)
	for index := range g.nukeBlasts {
		r.drawNukeBlast(screen, &g.nukeBlasts[index])
	}
}

func (r *Renderer) drawNukeBlast(screen *ebiten.Image, blast *NukeBlast) {
	dissolve := blast.DissolveProgress()
	worldRadius := blast.FrontRadius * (1 + nukeDissolveGrowth*dissolve)
	screenX, screenY := r.camera.ToScreen(blast.X, blast.Y)
	r.batches.Glow.AddCircle(screenX, screenY, worldRadius*nukeAuraScale, nukeFlameColor, nukeAuraAlpha*(1-dissolve))
	centerX, centerY, radius := screenX*draw.Scale, screenY*draw.Scale, worldRadius*draw.Scale
	bounds := screen.Bounds()
	left, top := max(0, int(centerX-radius)), max(0, int(centerY-radius))
	right, bottom := min(bounds.Dx(), int(centerX+radius)+1), min(bounds.Dy(), int(centerY+radius)+1)
	if right <= left || bottom <= top {
		return
	}
	options := &ebiten.DrawRectShaderOptions{Uniforms: r.fireballValues}
	options.GeoM.Translate(float64(left), float64(top))
	r.fireballValues["Center"] = [2]float32{centerX, centerY}
	r.fireballValues["Radius"] = radius
	r.fireballValues["Clock"] = blast.Seconds
	r.fireballValues["Dissolve"] = dissolve
	r.fireballValues["Seed"] = float32(blast.Stamp) * nukeSeedStep
	screen.DrawRectShader(right-left, bottom-top, r.fireballShader, options)
}

func mustCompileFireball() *ebiten.Shader {
	shader, err := ebiten.NewShader([]byte(fireballSource))
	if err != nil {
		panic(err)
	}
	return shader
}
