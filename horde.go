package main

// ===== Types =====

type SpawnDirector struct {
	accumulator     float32
	nextWaveSeconds float32
}

// ===== Constants =====

const (
	separationResponse     = 0.7
	separationFrameSplit   = 2
	separationNeighborCap  = 14
	knockbackDamping       = 0.86
	groundContactInterval  = 12
	burnContagionPerSecond = 0.35
	burnContagionRadius    = 22
	spawnDistance          = 780
	waveIntervalSeconds    = 45
	waveRingDistance       = 560
)

// ===== Public API =====

func NewSpawnDirector() SpawnDirector {
	return SpawnDirector{nextWaveSeconds: waveIntervalSeconds}
}

// ===== Internal =====

func (g *Game) updateEnemies(deltaSeconds float32) {
	for index := 0; index < g.enemies.Count; index++ {
		g.updateEnemy(index, deltaSeconds)
	}
}

func (g *Game) updateEnemy(index int, deltaSeconds float32) {
	enemies := g.enemies
	definition := &enemyTable[enemies.Kind[index]]
	enemies.TickStatuses(index, deltaSeconds)
	g.moveEnemy(index, definition, deltaSeconds)
	g.applyGroundContact(index)
	g.spreadBurning(index, deltaSeconds)
	g.damagePlayerOnContact(index, definition)
	enemies.HitFlash[index] = max(0, enemies.HitFlash[index]-deltaSeconds)
	enemies.BladeCooldown[index] = max(0, enemies.BladeCooldown[index]-deltaSeconds)
}

func (g *Game) moveEnemy(index int, definition *EnemyDefinition, deltaSeconds float32) {
	enemies := g.enemies
	x, y := enemies.PositionX[index], enemies.PositionY[index]
	directionX, directionY := normalize(g.player.X-x, g.player.Y-y)
	speed := definition.Speed * enemies.SpeedFactor(index)
	separationX, separationY := g.computeSeparation(index, definition.Radius)
	velocityX := directionX*speed + enemies.KnockbackX[index]
	velocityY := directionY*speed + enemies.KnockbackY[index]
	enemies.PositionX[index] = clamp(x+velocityX*deltaSeconds+separationX*separationResponse, 0, arenaSize)
	enemies.PositionY[index] = clamp(y+velocityY*deltaSeconds+separationY*separationResponse, 0, arenaSize)
	enemies.KnockbackX[index] *= knockbackDamping
	enemies.KnockbackY[index] *= knockbackDamping
}

func (g *Game) computeSeparation(index int, radius float32) (float32, float32) {
	enemies := g.enemies
	isSkipped := (uint32(index)+g.frame)%separationFrameSplit != 0 || enemyTable[enemies.Kind[index]].IsHeavy
	if isSkipped {
		return 0, 0
	}
	g.separationCandidates = g.grid.AppendNearbyLimited(enemies.PositionX[index], enemies.PositionY[index], radius*2, g.separationCandidates[:0], separationNeighborCap)
	pushX, pushY := float32(0), float32(0)
	for _, other := range g.separationCandidates {
		deltaX, deltaY := g.pushBetween(index, int(other), radius)
		pushX += deltaX
		pushY += deltaY
	}
	return pushX, pushY
}

func (g *Game) pushBetween(index, other int, radius float32) (float32, float32) {
	enemies := g.enemies
	deltaX := enemies.PositionX[index] - enemies.PositionX[other]
	deltaY := enemies.PositionY[index] - enemies.PositionY[other]
	minimumDistance := radius + enemyTable[enemies.Kind[other]].Radius
	distanceSquared := deltaX*deltaX + deltaY*deltaY
	if distanceSquared >= minimumDistance*minimumDistance || other == index {
		return 0, 0
	}
	distance := length(deltaX, deltaY) + 0.001
	overlap := minimumDistance - distance
	return deltaX / distance * overlap, deltaY / distance * overlap
}

func (g *Game) applyGroundContact(index int) {
	if (uint32(index)+g.frame)%groundContactInterval != 0 {
		return
	}
	element := groundTable[g.ground.KindAt(g.enemies.PositionX[index], g.enemies.PositionY[index])].ContactElement
	if element == ElementNone {
		return
	}
	g.HitEnemy(index, 0, element)
}

func (g *Game) spreadBurning(index int, deltaSeconds float32) {
	enemies := g.enemies
	isSpreading := enemies.Status[index]&StatusBurning != 0 && g.random.Chance(deltaSeconds*burnContagionPerSecond)
	if !isSpreading {
		return
	}
	x, y := enemies.PositionX[index], enemies.PositionY[index]
	g.ground.StimulateAt(x, y, StimulusHeat)
	g.candidates = g.grid.AppendNearby(x, y, burnContagionRadius, g.candidates[:0])
	if len(g.candidates) == 0 {
		return
	}
	g.HitEnemy(int(g.candidates[g.random.Below(len(g.candidates))]), 0, ElementFire)
}

func (g *Game) damagePlayerOnContact(index int, definition *EnemyDefinition) {
	enemies := g.enemies
	reach := definition.Radius + playerRadius
	isTouching := distanceSquared(enemies.PositionX[index], enemies.PositionY[index], g.player.X, g.player.Y) < reach*reach
	isFrozen := enemies.Status[index]&StatusFrozen != 0
	if !isTouching || isFrozen {
		return
	}
	g.DamagePlayer(definition.ContactDamage * enemies.Power[index])
}

func (g *Game) updateSpawning(deltaSeconds float32) {
	g.director.accumulator += spawnRateAt(g.elapsedSeconds) * deltaSeconds
	for g.director.accumulator >= 1 {
		g.director.accumulator--
		angle := g.random.Angle()
		g.spawnEnemyAt(g.chooseEnemyKind(), g.player.X+cosine(angle)*spawnDistance, g.player.Y+sine(angle)*spawnDistance)
	}
	g.spawnWaveIfDue()
}

func (g *Game) spawnWaveIfDue() {
	if g.elapsedSeconds < g.director.nextWaveSeconds {
		return
	}
	g.director.nextWaveSeconds += waveIntervalSeconds
	waveSize := 50 + int(g.elapsedSeconds/60*45)
	for spawned := range waveSize {
		angle := float32(spawned) / float32(waveSize) * 2 * 3.14159265
		g.spawnEnemyAt(g.chooseEnemyKind(), g.player.X+cosine(angle)*waveRingDistance, g.player.Y+sine(angle)*waveRingDistance)
	}
	g.effects.AddPopup(g.player.X, g.player.Y-60, "ENCIRCLED!", [3]float32{1, 0.3, 0.3})
}

func (g *Game) spawnEnemyAt(kind EnemyKind, x, y float32) uint32 {
	healthScale := 1 + g.elapsedSeconds/150
	jitterX, jitterY := g.random.Between(-6, 6), g.random.Between(-6, 6)
	return g.enemies.Spawn(kind, clamp(x+jitterX, 0, arenaSize), clamp(y+jitterY, 0, arenaSize), healthScale)
}

func (g *Game) chooseEnemyKind() EnemyKind {
	totalWeight := float32(0)
	for kind := range enemyKindCount {
		totalWeight += g.unlockedWeight(kind)
	}
	roll := g.random.Float() * totalWeight
	for kind := range enemyKindCount {
		roll -= g.unlockedWeight(kind)
		if roll < 0 {
			return kind
		}
	}
	return EnemySwarmer
}

func (g *Game) unlockedWeight(kind EnemyKind) float32 {
	definition := &enemyTable[kind]
	return definition.SpawnWeight * boolToFloat(g.elapsedSeconds >= definition.UnlockSeconds)
}

func spawnRateAt(seconds float32) float32 {
	minutes := seconds / 60
	return 1.4 + minutes*4 + minutes*minutes*2.5
}
