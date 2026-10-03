package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ===== Types =====

type HordeEvent struct {
	IsActive  bool
	Total     int
	Remaining int
}

type bossEventStarter func(game *Game)

// ===== Constants =====

const (
	bossEventsPerHorde     = 3
	hordeEventBaseSize     = 500
	hordeRingInnerDistance = 760
	hordeRingOuterDistance = 1150
	hordeClearBonusPerFoe  = 20
)

var hordeEventKinds = []EnemyKind{EnemyBrute, EnemyBrute, EnemyBloater}

var bossEventStarters = [2]bossEventStarter{(*Game).spawnSpiderEvent, (*Game).startHordeEvent}

var hordeBarColor = [3]float32{0.75, 0.35, 0.95}

// ===== Internal =====

func (g *Game) startBossEvent() {
	g.bossEventCount++
	isHorde := g.bossEventCount%bossEventsPerHorde == 0
	bossEventStarters[boolToIndex(isHorde)](g)
}

func (g *Game) spawnSpiderEvent() {
	g.spawnSpiderAt(g.random.Angle(), spiderSpawnDistance)
}

func (g *Game) startHordeEvent() {
	size := hordeEventBaseSize * (g.bossEventCount / bossEventsPerHorde)
	spawnedCount := 0
	for spawned := range size {
		spawnedCount += boolToIndex(g.spawnHordeEnemy(spawned))
	}
	g.hordeEvent = HordeEvent{IsActive: spawnedCount > 0, Total: spawnedCount, Remaining: spawnedCount}
	g.effects.AddPopup(g.player.X, g.player.Y-80, fmt.Sprintf("THE HORDE APPROACHES  x%d", spawnedCount), hordeBarColor)
	g.effects.AddShake(8)
	g.playSound(SoundStomp)
}

func (g *Game) spawnHordeEnemy(spawned int) bool {
	angle := g.random.Angle()
	distance := g.random.Between(hordeRingInnerDistance, hordeRingOuterDistance)
	kind := hordeEventKinds[spawned%len(hordeEventKinds)]
	id := g.spawnEnemyAt(kind, g.player.X+cosine(angle)*distance, g.player.Y+sine(angle)*distance)
	if id == 0 {
		return false
	}
	g.enemies.IsHordeEvent[g.enemies.Count-1] = true
	return true
}

func (g *Game) recordHordeCasualtyIf(isHordeEnemy bool) {
	if !isHordeEnemy || !g.hordeEvent.IsActive {
		return
	}
	g.hordeEvent.Remaining--
	if g.hordeEvent.Remaining > 0 {
		return
	}
	g.finishHordeEvent()
}

func (g *Game) finishHordeEvent() {
	g.hordeEvent.IsActive = false
	g.killScore += g.hordeEvent.Total * hordeClearBonusPerFoe
	g.effects.AddPopup(g.player.X, g.player.Y-80, "HORDE VANQUISHED", accentColor)
	g.effects.AddShake(6)
	g.playSound(SoundLevelUp)
}

func (u *UI) drawHordeEventBar(g *Game, screen *ebiten.Image) {
	if !g.hordeEvent.IsActive {
		return
	}
	y := float32(spiderBarTop + len(g.spiders)*(spiderBarHeight+spiderBarGap))
	fraction := float32(g.hordeEvent.Remaining) / float32(max(1, g.hordeEvent.Total))
	drawBar(screen, screenWidth/2-spiderBarWidth/2, y, spiderBarWidth, spiderBarHeight, fraction, hordeBarColor)
	label := fmt.Sprintf("HORDE  %d / %d", g.hordeEvent.Remaining, g.hordeEvent.Total)
	u.drawText(screen, label, u.small, screenWidth/2, y, textColor, 1, text.AlignCenter)
}
