package main

// ===== Types =====

type Burst struct {
	X             float32
	Y             float32
	Radius        float32
	Damage        float32
	Element       Element
	Requires      StatusFlags
	DamagesPlayer bool
}

// ===== Constants =====

const (
	maximumBurstsPerFrame = 384
	hitFlashSeconds       = 0.12
	burstKnockback        = 220
	playerBurstDamageRate = 0.35
)

// ===== Public API =====

func (g *Game) HitEnemy(index int, damage float32, element Element) {
	enemies := g.enemies
	if enemies.Health[index] <= 0 {
		return
	}
	enemies.LastHitByBoss[index] = false
	reaction := FindReaction(element, enemies.Status[index])
	definition := &reactionTable[reaction]
	enemies.ApplyStatus(index, elementStatus[element]&^definition.SuppressedStatus)
	enemies.Status[index] &^= definition.RemovesStatus
	enemies.ApplyStatus(index, definition.AddsStatus)
	finalDamage := damage * definition.DamageMultiplier
	enemies.Health[index] -= finalDamage
	enemies.HitFlash[index] = max(enemies.HitFlash[index], hitFlashSeconds*boolToFloat(finalDamage > 0))
	g.triggerReaction(reaction, enemies.PositionX[index], enemies.PositionY[index])
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
	for _, candidate := range g.candidates {
		g.burstHitIfInside(int(candidate), burst, radiusSquared)
	}
	g.burstHitPlayerIfInside(burst, radiusSquared)
}

// ===== Internal =====

func (g *Game) triggerReaction(reaction ReactionKind, x, y float32) {
	if reaction == ReactionNone {
		return
	}
	definition := &reactionTable[reaction]
	g.reactionCounts[reaction]++
	g.effects.AddPopup(x, y, definition.Name, definition.Color)
	g.effects.AddShake(definition.ShakeStrength)
	g.effects.AddRing(x, y, definition.AreaRadius, definition.Color)
	g.effects.SpawnSparks(x, y, 10, definition.Color, 170)
	g.ground.StimulateArea(x, y, definition.GroundRadiusCells, definition.GroundStimulus)
	g.QueueBurst(Burst{
		X: x, Y: y, Radius: definition.AreaRadius, Damage: definition.AreaDamage * g.player.DamageMultiplier,
		Element: definition.AreaElement, Requires: definition.AreaRequires, DamagesPlayer: definition.DamagesPlayer,
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
	isInside := distanceSquared(enemies.PositionX[index], enemies.PositionY[index], burst.X, burst.Y) <= radiusSquared
	hasRequiredStatus := enemies.Status[index]&burst.Requires == burst.Requires
	if !isInside || !hasRequiredStatus {
		return
	}
	g.HitEnemy(index, burst.Damage, burst.Element)
	g.PushEnemy(index, burst.X, burst.Y, burstKnockback*boolToFloat(burst.Damage > 0))
}

func (g *Game) burstHitPlayerIfInside(burst Burst, radiusSquared float32) {
	isInside := distanceSquared(g.player.X, g.player.Y, burst.X, burst.Y) <= radiusSquared
	if !burst.DamagesPlayer || !isInside {
		return
	}
	g.DamagePlayer(burst.Damage * playerBurstDamageRate)
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
	g.grantKillRewardsIf(x, y, definition, !enemies.LastHitByBoss[index])
	g.effects.SpawnSparks(x, y, 5, definition.Color, 110)
	g.ground.StimulateArea(x, y, definition.DeathGroundCells, definition.DeathStimulus)
	g.QueueBurst(Burst{X: x, Y: y, Radius: definition.DeathRadius, Damage: definition.DeathDamage, Element: definition.DeathElement})
	enemies.Remove(index)
}

func (g *Game) grantKillRewardsIf(x, y float32, definition *EnemyDefinition, isPlayerKill bool) {
	if !isPlayerKill {
		return
	}
	g.gems.Drop(x, y, definition.Experience)
	g.kills++
	g.HealFromKill()
}
