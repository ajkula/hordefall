package main

// ===== Types =====

type ProjectileKind uint8

type ProjectileStore struct {
	Count      int
	X          []float32
	Y          []float32
	VelocityX  []float32
	VelocityY  []float32
	Damage     []float32
	Life       []float32
	MaxLife    []float32
	AreaRadius []float32
	Element    []Element
	Kind       []ProjectileKind
	Pierce     []int16
	LastHit    []int32
	Color      [][3]float32
}

type projectileBehavior func(game *Game, index int)

// ===== Constants =====

const (
	ProjectileBolt ProjectileKind = iota
	ProjectileFlask
	projectileKindCount
)

const (
	maximumProjectiles   = 2048
	boltLifeSeconds      = 1.6
	boltHitRadius        = 6
	boltKnockback        = 120
	flaskArcHeight       = 70
	pierceToughnessRatio = 3
)

var projectileBehaviors = [projectileKindCount]projectileBehavior{
	ProjectileBolt:  (*Game).updateBolt,
	ProjectileFlask: (*Game).updateFlask,
}

// ===== Public API =====

func NewProjectileStore(capacity int) *ProjectileStore {
	return &ProjectileStore{
		X: make([]float32, capacity), Y: make([]float32, capacity),
		VelocityX: make([]float32, capacity), VelocityY: make([]float32, capacity),
		Damage: make([]float32, capacity), Life: make([]float32, capacity), MaxLife: make([]float32, capacity),
		AreaRadius: make([]float32, capacity), Element: make([]Element, capacity), Kind: make([]ProjectileKind, capacity),
		Pierce: make([]int16, capacity), LastHit: make([]int32, capacity), Color: make([][3]float32, capacity),
	}
}

func (s *ProjectileStore) SpawnBolt(x, y, angle, speed, damage float32, element Element, pierce int16, color [3]float32) {
	index, hasSlot := s.allocate()
	if !hasSlot {
		return
	}
	s.X[index], s.Y[index] = x, y
	s.VelocityX[index], s.VelocityY[index] = cosine(angle)*speed, sine(angle)*speed
	s.Damage[index], s.Element[index], s.Pierce[index], s.Color[index] = damage, element, pierce, color
	s.Life[index], s.MaxLife[index] = boltLifeSeconds, boltLifeSeconds
	s.Kind[index], s.LastHit[index], s.AreaRadius[index] = ProjectileBolt, -1, 0
}

func (s *ProjectileStore) SpawnFlask(fromX, fromY, toX, toY, flightSeconds, damage, areaRadius float32, color [3]float32) {
	index, hasSlot := s.allocate()
	if !hasSlot {
		return
	}
	s.X[index], s.Y[index] = fromX, fromY
	s.VelocityX[index], s.VelocityY[index] = (toX-fromX)/flightSeconds, (toY-fromY)/flightSeconds
	s.Damage[index], s.Element[index], s.Pierce[index], s.Color[index] = damage, ElementOil, 0, color
	s.Life[index], s.MaxLife[index] = flightSeconds, flightSeconds
	s.Kind[index], s.LastHit[index], s.AreaRadius[index] = ProjectileFlask, -1, areaRadius
}

func (s *ProjectileStore) ArcHeight(index int) float32 {
	progress := 1 - s.Life[index]/s.MaxLife[index]
	return flaskArcHeight * 4 * progress * (1 - progress) * boolToFloat(s.Kind[index] == ProjectileFlask)
}

// ===== Internal =====

func (s *ProjectileStore) allocate() (int, bool) {
	if s.Count >= len(s.X) {
		return 0, false
	}
	s.Count++
	return s.Count - 1, true
}

func (s *ProjectileStore) remove(index int) {
	last := s.Count - 1
	s.X[index], s.Y[index] = s.X[last], s.Y[last]
	s.VelocityX[index], s.VelocityY[index] = s.VelocityX[last], s.VelocityY[last]
	s.Damage[index], s.Life[index], s.MaxLife[index] = s.Damage[last], s.Life[last], s.MaxLife[last]
	s.AreaRadius[index], s.Element[index], s.Kind[index] = s.AreaRadius[last], s.Element[last], s.Kind[last]
	s.Pierce[index], s.LastHit[index], s.Color[index] = s.Pierce[last], s.LastHit[last], s.Color[last]
	s.Count--
}

func (g *Game) updateProjectiles(deltaSeconds float32) {
	projectiles := g.projectiles
	for index := projectiles.Count - 1; index >= 0; index-- {
		projectiles.X[index] += projectiles.VelocityX[index] * deltaSeconds
		projectiles.Y[index] += projectiles.VelocityY[index] * deltaSeconds
		projectiles.Life[index] -= deltaSeconds
		projectileBehaviors[projectiles.Kind[index]](g, index)
		g.removeProjectileIfSpent(index)
	}
}

func (g *Game) removeProjectileIfSpent(index int) {
	if g.projectiles.Life[index] > 0 {
		return
	}
	g.projectiles.remove(index)
}

func (g *Game) updateBolt(index int) {
	projectiles := g.projectiles
	target := g.firstEnemyTouching(projectiles.X[index], projectiles.Y[index], boltHitRadius, projectiles.LastHit[index])
	if target < 0 {
		return
	}
	isTooTough := g.enemies.Health[target] >= projectiles.Damage[index]*pierceToughnessRatio || enemyTable[g.enemies.Kind[target]].IsHeavy
	g.HitEnemy(target, projectiles.Damage[index], projectiles.Element[index])
	g.PushEnemy(target, projectiles.X[index], projectiles.Y[index], boltKnockback)
	g.effects.SpawnSparks(projectiles.X[index], projectiles.Y[index], 4, projectiles.Color[index], 150)
	projectiles.LastHit[index] = int32(target)
	projectiles.Pierce[index]--
	projectiles.Life[index] *= boolToFloat(projectiles.Pierce[index] >= 0 && !isTooTough)
}

func (g *Game) updateFlask(index int) {
	projectiles := g.projectiles
	if projectiles.Life[index] > 0 {
		return
	}
	x, y, radius := projectiles.X[index], projectiles.Y[index], projectiles.AreaRadius[index]
	g.ApplyBurst(Burst{X: x, Y: y, Radius: radius, Damage: projectiles.Damage[index], Element: ElementOil})
	g.ground.StimulateArea(x, y, int(radius/groundCellSize), StimulusOil)
	g.playSound(SoundSplash)
	g.effects.AddRing(x, y, radius, projectiles.Color[index])
	g.effects.SpawnSparks(x, y, 14, [3]float32{0.2, 0.15, 0.25}, 160)
}

func (g *Game) firstEnemyTouching(x, y, radius float32, excluded int32) int {
	g.candidates = g.grid.AppendNearby(x, y, radius+20, g.candidates[:0])
	for _, candidate := range g.candidates {
		if g.isEnemyTouching(int(candidate), x, y, radius) && candidate != excluded {
			return int(candidate)
		}
	}
	return -1
}

func (g *Game) isEnemyTouching(index int, x, y, radius float32) bool {
	enemies := g.enemies
	reach := radius + enemyTable[enemies.Kind[index]].Radius
	isAlive := enemies.Health[index] > 0
	return isAlive && distanceSquared(enemies.PositionX[index], enemies.PositionY[index], x, y) < reach*reach
}
