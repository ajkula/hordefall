package main

import "strings"

// ===== Types =====

type DemoSequence struct {
	Name              string
	Subtitle          string
	PassiveCount      int
	Weapons           []WeaponKind
	WeaponLevel       int
	EnemyKinds        []EnemyKind
	EnemyCount        int
	RingRadius        float32
	EnemyStatus       StatusFlags
	GroundStimulus    GroundStimulus
	GroundRadiusCells int
	SpiderCount       int
	ToughnessSeconds  float32
	FireRestFraction  float32
}

// ===== Constants =====

const (
	demoSequenceSeconds      = 5
	demoOrbitSpeed           = 0.6
	demoOrbitDrift           = 0.35
	demoAimSearch            = 520
	demoSpiderDistance       = 430
	demoSpiderChargeLag      = 1.1
	randomClipName           = "LIVE ACTION"
	randomSpiderChance       = 0.35
	demoSpiderStagger        = 0.6
	demoFirePeriod           = 1.2
	reinforceThreshold       = 0.4
	reinforceUrgentThreshold = 0.15
	reinforceShare           = 0.5
	reinforceInterval        = 0.5
	reinforceNear            = 1
	reinforceFar             = 1.3
)

var randomClipStatuses = []StatusFlags{0, StatusOiled, StatusWet, StatusBurning, StatusChilled}

var randomClipGrounds = []GroundStimulus{StimulusNone, StimulusOil, StimulusScorch, StimulusChill}

var randomClipEnemyKinds = []EnemyKind{EnemySwarmer, EnemyRunner, EnemyBloater, EnemyBrute, EnemyFrostling, EnemyEmberling}

var demoSequences = []DemoSequence{
	{
		Name: "INFERNO CHAIN", Weapons: []WeaponKind{WeaponEmberBolt, WeaponOilFlask}, WeaponLevel: 3,
		EnemyKinds: []EnemyKind{EnemySwarmer, EnemyBloater}, EnemyCount: 900, RingRadius: 520,
		EnemyStatus: StatusOiled, GroundStimulus: StimulusOil, GroundRadiusCells: 10, ToughnessSeconds: 900,
		FireRestFraction: 0.5,
	},
	{
		Name: "ELECTROCUTION", Weapons: []WeaponKind{WeaponArcLightning, WeaponDownpour}, WeaponLevel: 6,
		EnemyKinds: []EnemyKind{EnemySwarmer, EnemyRunner}, EnemyCount: 300, RingRadius: 260,
		EnemyStatus: StatusWet, ToughnessSeconds: 450,
	},
	{
		Name: "FREEZE & SHATTER", Weapons: []WeaponKind{WeaponFrostNova, WeaponOrbitBlades}, WeaponLevel: 6,
		EnemyKinds: []EnemyKind{EnemySwarmer, EnemyBrute}, EnemyCount: 320, RingRadius: 210,
		EnemyStatus: StatusWet, ToughnessSeconds: 450,
	},
	{
		Name: "WILDFIRE", Weapons: []WeaponKind{WeaponEmberBolt}, WeaponLevel: 6,
		EnemyKinds: []EnemyKind{EnemySwarmer, EnemyRunner, EnemyEmberling}, EnemyCount: 320, RingRadius: 300,
		EnemyStatus: StatusBurning, GroundStimulus: StimulusScorch, GroundRadiusCells: 3, ToughnessSeconds: 600,
	},
	{
		Name: "SPIDER TANK", Weapons: []WeaponKind{WeaponEmberBolt, WeaponArcLightning}, WeaponLevel: 4,
		EnemyKinds: []EnemyKind{EnemySwarmer, EnemyRunner}, EnemyCount: 160, RingRadius: 320,
		SpiderCount: 1, ToughnessSeconds: 300,
	},
	{
		Name: "HORDE x4000", Weapons: []WeaponKind{WeaponFrostNova, WeaponArcLightning, WeaponOrbitBlades}, WeaponLevel: 3,
		EnemyKinds: []EnemyKind{EnemySwarmer, EnemyRunner, EnemyBrute}, EnemyCount: 4000, RingRadius: 650,
		ToughnessSeconds: 1500,
	},
	{
		Name: "SIEGE: 3 SPIDER TANKS", Subtitle: "and a horde of 2000",
		Weapons: []WeaponKind{WeaponEmberBolt, WeaponArcLightning, WeaponFrostNova, WeaponOrbitBlades}, WeaponLevel: 6,
		EnemyKinds: []EnemyKind{EnemySwarmer, EnemyRunner, EnemyBrute, EnemyFrostling}, EnemyCount: 2000, RingRadius: 700,
		SpiderCount: 3, ToughnessSeconds: 900,
	},
	{
		Name: "FULL ARSENAL", Subtitle: "6 weapons at max level, 2500 soaked foes",
		Weapons: []WeaponKind{
			WeaponEmberBolt, WeaponFrostNova, WeaponArcLightning,
			WeaponOilFlask, WeaponDownpour, WeaponOrbitBlades,
		},
		WeaponLevel: maximumWeaponLevel, PassiveCount: 6,
		EnemyKinds: []EnemyKind{EnemySwarmer, EnemyRunner, EnemyBloater, EnemyBrute, EnemyFrostling, EnemyEmberling},
		EnemyCount: 2500, RingRadius: 650, EnemyStatus: StatusWet,
		GroundStimulus: StimulusOil, GroundRadiusCells: 6, ToughnessSeconds: 1500,
	},
}

// ===== Public API =====

func (g *Game) DemoSequenceName() string {
	return g.currentDemo.Name
}

func (g *Game) DemoSequenceSubtitle() string {
	return g.currentDemo.Subtitle
}

func (g *Game) DemoSequenceProgress() float32 {
	return clamp(g.demoSeconds/demoSequenceSeconds, 0, 1)
}

// ===== Internal =====

func (g *Game) startDemo() {
	g.demoSlot = -1
	g.startNextDemoSequence()
}

func (g *Game) startNextDemoSequence() {
	g.demoSlot++
	g.demoSequence = (g.demoSlot / 2) % len(demoSequences)
	candidates := [2]DemoSequence{demoSequences[g.demoSequence], g.randomDemoSequence()}
	g.currentDemo = candidates[g.demoSlot%2]
	sequence := &g.currentDemo
	g.resetRun()
	g.isDemo = true
	g.demoSeconds = 0
	g.director.accumulator = -1e9
	g.director.nextWaveSeconds = 1e9
	g.bossKillsRequired = 1 << 30
	g.elapsedSeconds = sequence.ToughnessSeconds
	g.equipDemoWeapons(sequence)
	for range sequence.PassiveCount {
		passiveTable[g.random.Below(int(passiveKindCount))].Apply(g.player)
	}
	g.ground.StimulateArea(g.player.X, g.player.Y, sequence.GroundRadiusCells, sequence.GroundStimulus)
	for spawned := range sequence.EnemyCount {
		g.spawnDemoEnemy(sequence, spawned)
	}
	g.demoReinforceCooldown = 0
	for spider := range sequence.SpiderCount {
		g.spawnDemoSpider(spider, sequence.SpiderCount)
	}
}

func (g *Game) randomDemoSequence() DemoSequence {
	weaponPicks := g.pickDistinct(2+g.random.Below(3), int(weaponKindCount))
	enemyPicks := g.pickDistinct(2+g.random.Below(3), len(randomClipEnemyKinds))
	weapons := make([]WeaponKind, len(weaponPicks))
	names := make([]string, len(weaponPicks))
	for index, pick := range weaponPicks {
		weapons[index] = WeaponKind(pick)
		names[index] = weaponTable[pick].Name
	}
	enemyKinds := make([]EnemyKind, len(enemyPicks))
	for index, pick := range enemyPicks {
		enemyKinds[index] = randomClipEnemyKinds[pick]
	}
	return DemoSequence{
		Name: randomClipName, Subtitle: strings.Join(names, " + "),
		Weapons: weapons, WeaponLevel: 3 + g.random.Below(5), PassiveCount: g.random.Below(4),
		EnemyKinds: enemyKinds, EnemyCount: 150 + g.random.Below(451), RingRadius: g.random.Between(250, 500),
		EnemyStatus:       randomClipStatuses[g.random.Below(len(randomClipStatuses))],
		GroundStimulus:    randomClipGrounds[g.random.Below(len(randomClipGrounds))],
		GroundRadiusCells: 4 + g.random.Below(9),
		SpiderCount:       boolToIndex(g.random.Chance(randomSpiderChance)),
		ToughnessSeconds:  g.random.Between(200, 700),
	}
}

func (g *Game) pickDistinct(count, total int) []int {
	pool := make([]int, total)
	for index := range pool {
		pool[index] = index
	}
	for index := range count {
		swap := index + g.random.Below(total-index)
		pool[index], pool[swap] = pool[swap], pool[index]
	}
	return pool[:count]
}

func (g *Game) equipDemoWeapons(sequence *DemoSequence) {
	g.player.Weapons = g.player.Weapons[:0]
	for _, kind := range sequence.Weapons {
		g.player.Weapons = append(g.player.Weapons, WeaponState{Kind: kind, Level: sequence.WeaponLevel})
	}
}

func (g *Game) spawnDemoEnemy(sequence *DemoSequence, spawned int) {
	g.spawnDemoEnemyBetween(sequence, spawned, 0.35, 1, sequence.EnemyStatus)
}

func (g *Game) spawnDemoEnemyBetween(sequence *DemoSequence, spawned int, nearFraction, farFraction float32, status StatusFlags) {
	angle := g.random.Angle()
	distance := sequence.RingRadius * g.random.Between(nearFraction, farFraction)
	kind := sequence.EnemyKinds[spawned%len(sequence.EnemyKinds)]
	g.spawnEnemyAt(kind, g.player.X+cosine(angle)*distance, g.player.Y+sine(angle)*distance)
	g.enemies.ApplyStatus(g.enemies.Count-1, status)
}

func (g *Game) spawnDemoSpider(spider, spiderCount int) {
	angle := (float32(spider)/float32(spiderCount))*2*3.14159265 + g.random.Between(-0.4, 0.4)
	g.spawnSpiderAt(angle, demoSpiderDistance)
	laser := &g.spiders[len(g.spiders)-1].Laser
	laser.Phase, laser.Timer = LaserCharging, demoSpiderChargeLag+float32(spider)*demoSpiderStagger
}

func (g *Game) reinforceDemoIfThin() {
	sequence := &g.currentDemo
	g.demoReinforceCooldown = max(0, g.demoReinforceCooldown-deltaSeconds)
	alive := float32(g.enemies.Count - len(g.spiders))
	isThin := alive < float32(sequence.EnemyCount)*reinforceThreshold
	isNearlyEmpty := alive < float32(sequence.EnemyCount)*reinforceUrgentThreshold
	if !isThin || (g.demoReinforceCooldown > 0 && !isNearlyEmpty) {
		return
	}
	g.demoReinforceCooldown = reinforceInterval
	for spawned := range int(float32(sequence.EnemyCount) * reinforceShare) {
		g.spawnDemoEnemyBetween(sequence, spawned, reinforceNear, reinforceFar, 0)
	}
}

func (g *Game) updateDemo() {
	g.demoSeconds += deltaSeconds
	if g.demoSeconds > demoSequenceSeconds {
		g.startNextDemoSequence()
	}
	menuControls := g.controls
	g.controls = g.autopilotControls()
	g.aimAtNearestEnemy()
	g.reinforceDemoIfThin()
	g.simulate()
	g.player.Health = g.player.MaximumHealth
	g.player.PendingLevelUps = 0
	g.controls = menuControls
}

func (g *Game) autopilotControls() Controls {
	angle := g.demoSeconds * demoOrbitSpeed
	firePhase := g.demoSeconds/demoFirePeriod - float32(int(g.demoSeconds/demoFirePeriod))
	isFiring := firePhase >= g.currentDemo.FireRestFraction
	held := ActionAimLock | ActionFire*Action(boolToIndex(isFiring))
	return Controls{MoveX: -sine(angle) * demoOrbitDrift, MoveY: cosine(angle) * demoOrbitDrift, Held: held}
}

func (g *Game) aimAtNearestEnemy() {
	player := g.player
	target := g.nearestEnemyWithin(player.X, player.Y, demoAimSearch, false)
	if target < 0 {
		return
	}
	player.AimX, player.AimY = normalize(g.enemies.PositionX[target]-player.X, g.enemies.PositionY[target]-player.Y)
}
