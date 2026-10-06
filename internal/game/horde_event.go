package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"hordefall/internal/audio"
	"hordefall/internal/i18n"
)

// ===== Types =====

type HordeEvent struct {
	IsActive  bool
	Total     int
	Remaining int
}

type BossEventKind uint8

type bossEventStarter func(game *Game)

// ===== Constants =====

const (
	BossSpider BossEventKind = iota
	BossHorde
	BossWorm
	bossEventKindCount
)

const (
	spidersBeforeHorde    = 3
	hordeEventBaseSize    = 500
	hordeShellPeriod      = 8
	hordePowerBonus       = 2
	hordeClearBonusPerFoe = 20
)

var hordeCoreKinds = []EnemyKind{
	EnemyBrute, EnemyFrostling, EnemyRunner, EnemyBrute,
	EnemySwarmer, EnemyFrostling, EnemyBrute,
}

var hordeShellKinds = []EnemyKind{EnemyBloater, EnemyEmberling}

var hordeLayerStrides = [2]int{1, hordeShellPeriod}

var hordeLayerKinds = [2][]EnemyKind{hordeCoreKinds, hordeShellKinds}

var hordeLayerDistances = [2][2]float32{{760, 1040}, {1090, 1180}}

var bossEventStarters = [bossEventKindCount]bossEventStarter{
	BossSpider: (*Game).spawnSpiderEvent,
	BossHorde:  (*Game).startHordeEvent,
	BossWorm:   (*Game).spawnWormEvent,
}

var bossEventCycle = []BossEventKind{BossSpider, BossSpider, BossSpider, BossHorde, BossWorm, BossHorde}

var hordeBarColor = [3]float32{0.75, 0.35, 0.95}

// ===== Internal =====

func (g *Game) startBossEvent() {
	kind := g.nextBossEvent()
	g.bossEventCount++
	bossEventStarters[kind](g)
}

func (g *Game) nextBossEvent() BossEventKind {
	return bossEventCycle[g.bossEventCount%len(bossEventCycle)]
}

func (g *Game) spawnSpiderEvent() {
	g.spawnSpiderAt(g.random.Angle(), spiderSpawnDistance)
}

func (g *Game) startHordeEvent() {
	g.hordeWaveCount++
	size := hordeEventBaseSize * g.hordeWaveCount
	spawnedCount := 0
	for spawned := range size {
		spawnedCount += boolToIndex(g.spawnHordeEnemy(spawned))
	}
	ongoing := [2]HordeEvent{{}, g.hordeEvent}[boolToIndex(g.hordeEvent.IsActive)]
	g.hordeEvent = HordeEvent{Total: ongoing.Total + spawnedCount, Remaining: ongoing.Remaining + spawnedCount}
	g.hordeEvent.IsActive = g.hordeEvent.Remaining > 0
	g.effects.AddPopup(g.player.X, g.player.Y-80, i18n.F("horde.approaches", spawnedCount), hordeBarColor)
	g.say(ChatterHordeIncoming)
	g.effects.AddShake(8)
	g.playSound(audio.SoundStomp)
}

func (g *Game) spawnHordeEnemy(spawned int) bool {
	layer := boolToIndex(spawned%hordeShellPeriod == hordeShellPeriod-1)
	kinds := hordeLayerKinds[layer]
	kind := kinds[spawned/hordeLayerStrides[layer]%len(kinds)]
	angle := g.random.Angle()
	distance := g.random.Between(hordeLayerDistances[layer][0], hordeLayerDistances[layer][1])
	id := g.spawnEnemyAt(kind, g.player.X+cosine(angle)*distance, g.player.Y+sine(angle)*distance)
	if id == 0 {
		return false
	}
	g.empowerHordeMember(g.enemies.Count - 1)
	return true
}

func (g *Game) empowerHordeMember(index int) {
	enemies := g.enemies
	enemies.IsHordeEvent[index] = true
	enemies.Health[index] *= hordePowerBonus
	enemies.Power[index] *= hordePowerBonus
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
	g.effects.AddPopup(g.player.X, g.player.Y-80, i18n.T("horde.vanquished"), accentColor)
	g.say(ChatterHordeCleared)
	g.effects.AddShake(6)
	g.playSound(audio.SoundLevelUp)
}

func (u *UI) drawHordeEventBar(g *Game, screen *ebiten.Image) {
	if !g.hordeEvent.IsActive {
		return
	}
	y := float32(spiderBarTop + (len(g.spiders)+boolToIndex(g.worm.IsActive))*(spiderBarHeight+spiderBarGap))
	fraction := float32(g.hordeEvent.Remaining) / float32(max(1, g.hordeEvent.Total))
	drawBar(screen, screenWidth/2-spiderBarWidth/2, y, spiderBarWidth, spiderBarHeight, fraction, hordeBarColor)
	label := i18n.F("horde.bar", g.hordeEvent.Remaining, g.hordeEvent.Total)
	u.drawText(screen, label, u.small, screenWidth/2, y, textColor, 1, text.AlignCenter)
}
