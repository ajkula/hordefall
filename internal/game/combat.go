package game

import (
	"cmp"
	"slices"
)

// ===== Types =====

type Burst struct {
	X             float32
	Y             float32
	Radius        float32
	Damage        float32
	Element       Element
	Requires      StatusFlags
	DamagesPlayer bool
	SparesBodies  bool
	Source        string

	OriginPartID      uint32
	OriginBodyID      uint32
	IsBodySpreadSpent bool
	IsConducting      bool
	ConductionHop     int
}

type bodyPartHit struct {
	Index    int32
	BodyID   uint32
	Distance float32
}

// ===== Constants =====

const (
	maximumBurstsPerFrame = 384
	hitFlashSeconds       = 0.12
	burstKnockback        = 220
	playerBurstDamageRate = 0.35
)

const conductionHopsPerBody = 2

var bodySpreadDamageShares = [...]float32{1, 0.25, 0.0625}

// ===== Public API =====

func (g *Game) HitEnemy(index int, damage float32, element Element) {
	enemies := g.enemies
	if enemies.Health[index] <= 0 {
		return
	}
	enemies.LastHitByBoss[index] = false
	reaction := enemies.FindReaction(index, element)
	definition := &reactionTable[reaction]
	enemies.ApplyStatus(index, elementStatus[element]&^definition.SuppressedStatus)
	enemies.Status[index] &^= definition.RemovesStatus
	enemies.ApplyStatus(index, definition.AddsStatus)
	g.woundEnemy(index, damage*definition.DamageMultiplier)
	g.triggerReaction(reaction, index)
}

func (g *Game) PushEnemy(index int, fromX, fromY, strength float32) {
	directionX, directionY := normalize(g.enemies.PositionX[index]-fromX, g.enemies.PositionY[index]-fromY)
	resistance := 12 / enemyTable[g.enemies.Kind[index]].Radius
	g.enemies.KnockbackX[index] += directionX * strength * resistance
	g.enemies.KnockbackY[index] += directionY * strength * resistance
}

func (g *Game) QueueBurst(burst Burst) {
	if burst.Radius <= 0 || len(g.bursts) >= maximumBurstsPerFrame {
		return
	}
	g.bursts = append(g.bursts, burst)
}

func (g *Game) ApplyBurst(burst Burst) {
	g.candidates = g.grid.AppendNearby(burst.X, burst.Y, burst.Radius, g.candidates[:0])
	radiusSquared := burst.Radius * burst.Radius
	g.bodyPartHits = g.bodyPartHits[:0]
	for _, candidate := range g.candidates {
		g.burstHitIfInside(int(candidate), burst, radiusSquared)
	}
	g.spreadBurstAlongBodies(burst)
	g.burstHitPlayerIfInside(burst, radiusSquared)
}

// ===== Internal =====

func (g *Game) woundEnemy(index int, damage float32) {
	enemies := g.enemies
	enemies.Health[index] -= damage
	enemies.HitFlash[index] = max(enemies.HitFlash[index], hitFlashSeconds*boolToFloat(damage > 0))
}

func (g *Game) triggerReaction(reaction ReactionKind, index int) {
	if reaction == ReactionNone {
		return
	}
	definition := &reactionTable[reaction]
	x, y := g.enemies.PositionX[index], g.enemies.PositionY[index]
	isConducting := definition.AreaRequires&StatusWet != 0
	g.reactionCounts[reaction]++
	g.noteChatterReaction()
	g.playSound(definition.Sound)
	g.effects.AddPopup(x, y, definition.Name, definition.Color)
	g.effects.AddShake(definition.ShakeStrength)
	g.effects.AddRing(x, y, definition.AreaRadius, definition.Color)
	g.effects.SpawnSparks(x, y, 10, definition.Color, 170)
	g.ground.StimulateArea(x, y, definition.GroundRadiusCells, definition.GroundStimulus)
	g.QueueBurst(Burst{
		X: x, Y: y, Radius: definition.AreaRadius, Damage: definition.AreaDamage * g.player.DamageMultiplier,
		Element: definition.AreaElement, Requires: definition.AreaRequires, DamagesPlayer: definition.DamagesPlayer, Source: definition.Name,
		OriginPartID: g.enemies.ID[index], OriginBodyID: g.enemies.BodyID[index], IsBodySpreadSpent: g.isBodySpreadSpent,
		IsConducting: isConducting, ConductionHop: g.conductionHop + boolToIndex(isConducting),
	})
}

func (g *Game) processBursts() {
	for processed := 0; processed < len(g.bursts) && processed < maximumBurstsPerFrame; processed++ {
		g.ApplyBurst(g.bursts[processed])
	}
	g.bursts = g.bursts[:0]
}

func (g *Game) burstHitIfInside(index int, burst Burst, radiusSquared float32) {
	enemies := g.enemies
	distance := distanceSquared(enemies.PositionX[index], enemies.PositionY[index], burst.X, burst.Y)
	hasRequiredStatus := enemies.Status[index]&burst.Requires == burst.Requires
	if distance > radiusSquared || !hasRequiredStatus {
		return
	}
	if enemies.BodyID[index] != 0 {
		g.queueBodyPartHit(index, distance, &burst)
		return
	}
	g.HitEnemy(index, burst.Damage, burst.Element)
	g.PushEnemy(index, burst.X, burst.Y, burstKnockback*boolToFloat(burst.Damage > 0))
}

func (g *Game) queueBodyPartHit(index int, distance float32, burst *Burst) {
	bodyID := g.enemies.BodyID[index]
	isOriginBody := bodyID == burst.OriginBodyID
	isAlreadyHit := isOriginBody && ((burst.IsBodySpreadSpent && !burst.IsConducting) || g.enemies.ID[index] == burst.OriginPartID)
	if burst.SparesBodies || isAlreadyHit {
		return
	}
	g.bodyPartHits = append(g.bodyPartHits, bodyPartHit{Index: int32(index), BodyID: bodyID, Distance: distance})
}

func (g *Game) spreadBurstAlongBodies(burst Burst) {
	slices.SortFunc(g.bodyPartHits, compareBodyPartHits)
	rank, previousBodyID := 0, uint32(0)
	for _, hit := range g.bodyPartHits {
		rank = (rank + 1) * boolToIndex(hit.BodyID == previousBodyID)
		previousBodyID = hit.BodyID
		g.hitBodyPartFromBurst(hit, rank, &burst)
	}
	g.dryBodyIf(burst.OriginBodyID, g.isConductionOver(&burst))
}

func (g *Game) hitBodyPartFromBurst(hit bodyPartHit, rank int, burst *Burst) {
	isOriginBody := hit.BodyID == burst.OriginBodyID
	if !burst.IsConducting || !isOriginBody {
		g.hitBodyPartAtRank(int(hit.Index), rank+boolToIndex(isOriginBody), burst.Damage, burst.Element)
		return
	}
	if rank == 0 {
		g.conductionHop = burst.ConductionHop
		g.HitEnemy(int(hit.Index), burst.Damage, burst.Element)
		g.conductionHop = 0
	}
}

func (g *Game) isConductionOver(burst *Burst) bool {
	hasConductedPart := slices.ContainsFunc(g.bodyPartHits, func(hit bodyPartHit) bool { return hit.BodyID == burst.OriginBodyID })
	return burst.ConductionHop > 0 && (!burst.IsConducting || burst.ConductionHop >= conductionHopsPerBody || !hasConductedPart)
}

func (g *Game) dryBodyIf(bodyID uint32, shouldDry bool) {
	if !shouldDry || bodyID == 0 {
		return
	}
	enemies := g.enemies
	for index := range enemies.Count {
		enemies.Status[index] &^= StatusWet * StatusFlags(boolToIndex(enemies.BodyID[index] == bodyID))
	}
}

func compareBodyPartHits(first, second bodyPartHit) int {
	return cmp.Or(cmp.Compare(first.BodyID, second.BodyID), cmp.Compare(first.Distance, second.Distance))
}

func (g *Game) hitBodyPartAtRank(index, rank int, damage float32, element Element) {
	if rank == 0 {
		g.isBodySpreadSpent = true
		g.HitEnemy(index, damage, element)
		g.isBodySpreadSpent = false
		return
	}
	if rank < len(bodySpreadDamageShares) {
		g.woundEnemy(index, damage*bodySpreadDamageShares[rank])
		g.enemies.LastHitByBoss[index] = false
	}
}

func (g *Game) burstHitPlayerIfInside(burst Burst, radiusSquared float32) {
	isInside := distanceSquared(g.player.X, g.player.Y, burst.X, burst.Y) <= radiusSquared
	if !burst.DamagesPlayer || !isInside {
		return
	}
	g.DamagePlayer(burst.Damage*playerBurstDamageRate, burst.Source)
}

func (g *Game) removeDeadEnemies() {
	for index := g.enemies.Count - 1; index >= 0; index-- {
		g.resolveDeathIfDead(index)
	}
}

func (g *Game) resolveDeathIfDead(index int) {
	enemies := g.enemies
	if enemies.Health[index] > 0 {
		return
	}
	definition := &enemyTable[enemies.Kind[index]]
	x, y := enemies.PositionX[index], enemies.PositionY[index]
	isHordeEnemy := enemies.IsHordeEvent[index]
	g.grantKillRewardsIf(x, y, definition, !enemies.LastHitByBoss[index] && !g.isDemo, isHordeEnemy)
	g.recordHordeCasualtyIf(isHordeEnemy)
	g.effects.SpawnSparks(x, y, 5, definition.Color, 110)
	g.playSound(definition.DeathSound)
	g.ground.StimulateArea(x, y, definition.DeathGroundCells, definition.DeathStimulus)
	g.QueueBurst(Burst{X: x, Y: y, Radius: definition.DeathRadius, Damage: definition.DeathDamage, Element: definition.DeathElement})
	enemies.Remove(index)
}

func (g *Game) grantKillRewardsIf(x, y float32, definition *EnemyDefinition, isPlayerKill, isHordeEnemy bool) {
	if !isPlayerKill {
		return
	}
	g.gems.Drop(x, y, definition.Experience)
	g.dropHealOrbIf(x, y)
	g.awardKillScore(definition.Experience)
	g.kills++
	g.countRobotProgress(definition)
	g.bossProgressKills += boolToIndex(!isHordeEnemy && !g.isBossCounterFrozen())
	g.HealFromKill()
}
