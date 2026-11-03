package game

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/ui"
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
	hordeEventBaseSize      = 500
	hordeShellPeriod        = 8
	hordePowerBonus         = 2
	hordeClearBonusPerFoe   = 20
	bossStackOpeningRepeats = 3
	hordeHuntNearDistance   = 300
	hordeHuntDistanceRange  = screenWidth
	hordeHuntDistanceBoost  = 2.5
	hordeHuntDwindleBoost   = 1.5
	hordeHuntPlayerShare    = 0.9
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

var bossDiscoveryList = []BossEventKind{BossSpider, BossHorde, BossWorm}

var bossStack = buildBossStack(bossDiscoveryList)

var bossCapacities = [2]int{1, maximumBossesAlive}

var hordeBarColor = [3]float32{0.75, 0.35, 0.95}

// ===== Internal =====

func (g *Game) startBossEvent() {
	kind := g.nextBossEvent()
	g.bossEventCount++
	bossEventStarters[kind](g)
}

func (g *Game) nextBossEvent() BossEventKind {
	stackPosition := g.bossEventCount - len(bossDiscoveryList)
	if stackPosition < 0 {
		return bossDiscoveryList[g.bossEventCount]
	}
	return bossStack[stackPosition%len(bossStack)]
}

func buildBossStack(discoveryList []BossEventKind) []BossEventKind {
	return append(slices.Repeat(discoveryList[:1], bossStackOpeningRepeats), discoveryList[1:]...)
}

func (g *Game) isBossCounterFrozen() bool {
	isStackPhase := g.bossEventCount > len(bossDiscoveryList)
	return g.livingBossCount() >= bossCapacities[boolToIndex(isStackPhase)]
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
	g.playSound(SoundStomp)
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
	g.playSound(SoundLevelUp)
}

func (g *Game) hordeHuntSpeed(index int, speed, distanceToPlayer float32) float32 {
	if !g.enemies.IsHordeEvent[index] {
		return speed
	}
	distanceWeight := clamp((distanceToPlayer-hordeHuntNearDistance)/hordeHuntDistanceRange, 0, 1)
	dwindle := 1 - float32(g.hordeEvent.Remaining)/float32(max(1, g.hordeEvent.Total))
	hunting := speed * (1 + hordeHuntDistanceBoost*distanceWeight + hordeHuntDwindleBoost*dwindle)
	return min(hunting, max(speed, hordeHuntPlayerShare*g.player.MoveSpeed))
}

func (u *UI) drawHordeEventBar(g *Game, screen *ebiten.Image) {
	if !g.hordeEvent.IsActive {
		return
	}
	y := float32(spiderBarTop + (len(g.spiders)+len(g.worms))*(spiderBarHeight+spiderBarGap))
	fraction := float32(g.hordeEvent.Remaining) / float32(max(1, g.hordeEvent.Total))
	drawBossBar(screen, y, fraction, hordeBarColor)
	label := i18n.F("horde.bar", g.hordeEvent.Remaining, g.hordeEvent.Total)
	u.kit.Text.Draw(screen, label, ui.FaceSmall, screenWidth/2, y, textColor, 1, text.AlignCenter)
}
