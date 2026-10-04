package game

import "hordefall/internal/audio"

// ===== Types =====

type WeaponKind uint8

type WeaponDefinition struct {
	Name             string
	Description      string
	Element          Element
	IsAimed          bool
	FireSound        audio.SoundKind
	Color            [3]float32
	BaseCooldown     float32
	CooldownPerLevel float32
	BaseDamage       float32
	DamagePerLevel   float32
	BaseCount        int
	LevelsPerCount   int
	BaseRadius       float32
	RadiusPerLevel   float32
}

type WeaponState struct {
	Kind              WeaponKind
	Level             int
	CooldownRemaining float32
}

type weaponBehavior func(game *Game, weapon *WeaponState, definition *WeaponDefinition)

// ===== Constants =====

const (
	WeaponEmberBolt WeaponKind = iota
	WeaponFrostNova
	WeaponArcLightning
	WeaponOilFlask
	WeaponDownpour
	WeaponOrbitBlades
	WeaponSeismicHammer
	WeaponStaticMines
	weaponKindCount
)

const (
	maximumWeaponLevel  = 7
	maximumWeaponSlots  = 5
	boltSpeed           = 440
	boltSpreadRadians   = 0.17
	flaskFlightSeconds  = 0.55
	targetSearchRadius  = 420
	chainSearchRadius   = 150
	aimConeMinimumDot   = 0.72
	throwDistance       = 230
	rainDistance        = 250
	volleySpreadRadians = 0.35
	fizzleDistance      = 140
	bladeHitRadius      = 16
	bladeHitCooldown    = 0.4
	bladeKnockback      = 260
	bladeSpinSpeed      = 3.4
	hammerShake         = 3
	hammerDustSparks    = 30
	hammerInnerRing     = 0.55
	hammerQuakeReach    = 2.6
)

var hammerDustColor = [3]float32{0.75, 0.68, 0.55}

var weaponTable = [weaponKindCount]WeaponDefinition{
	WeaponEmberBolt: {
		Name: "Ember Bolt", Description: "Fires burning bolts where you aim.",
		Element: ElementFire, IsAimed: true, FireSound: audio.SoundPew, Color: [3]float32{1, 0.55, 0.15},
		BaseCooldown: 0.42, CooldownPerLevel: 0.07, BaseDamage: 11, DamagePerLevel: 4,
		BaseCount: 1, LevelsPerCount: 2,
	},
	WeaponFrostNova: {
		Name: "Frost Nova", Description: "Pulses cold around you.",
		Element: ElementFrost, FireSound: audio.SoundNova, Color: [3]float32{0.6, 0.85, 1},
		BaseCooldown: 2.6, CooldownPerLevel: 0.06, BaseDamage: 5, DamagePerLevel: 3,
		BaseCount: 1, LevelsPerCount: 99, BaseRadius: 95, RadiusPerLevel: 14,
	},
	WeaponArcLightning: {
		Name: "Arc Lightning", Description: "Strikes the enemy you aim at, then chains.",
		Element: ElementShock, IsAimed: true, FireSound: audio.SoundZap, Color: [3]float32{1, 1, 0.5},
		BaseCooldown: 0.9, CooldownPerLevel: 0.06, BaseDamage: 12, DamagePerLevel: 4,
		BaseCount: 3, LevelsPerCount: 1,
	},
	WeaponOilFlask: {
		Name: "Oil Flask", Description: "Lobs oil where you aim.",
		Element: ElementOil, IsAimed: true, Color: [3]float32{0.35, 0.25, 0.4},
		BaseCooldown: 3.2, CooldownPerLevel: 0.07, BaseDamage: 2, DamagePerLevel: 1,
		BaseCount: 1, LevelsPerCount: 3, BaseRadius: 55, RadiusPerLevel: 8,
	},
	WeaponDownpour: {
		Name: "Downpour", Description: "Calls rain where you aim.",
		Element: ElementWater, IsAimed: true, FireSound: audio.SoundRain, Color: [3]float32{0.35, 0.6, 1},
		BaseCooldown: 3.6, CooldownPerLevel: 0.07, BaseDamage: 3, DamagePerLevel: 1.5,
		BaseCount: 1, LevelsPerCount: 3, BaseRadius: 85, RadiusPerLevel: 12,
	},
	WeaponOrbitBlades: {
		Name: "Orbit Blades", Description: "Spinning blades.",
		Element: ElementPhysical, Color: [3]float32{0.9, 0.9, 0.95},
		BaseDamage: 9, DamagePerLevel: 4, BaseCount: 2, LevelsPerCount: 2, BaseRadius: 72, RadiusPerLevel: 6,
	},
	WeaponSeismicHammer: {
		Name: "Seismic Hammer", Description: "Slams the ground.",
		Element: ElementPhysical, FireSound: audio.SoundStomp, Color: [3]float32{0.85, 0.72, 0.5},
		BaseCooldown: 2.4, CooldownPerLevel: 0.06, BaseDamage: 14, DamagePerLevel: 5,
		BaseCount: 1, LevelsPerCount: 99, BaseRadius: 120, RadiusPerLevel: 12,
	},
	WeaponStaticMines: {
		Name: "Static Mines", Description: "Drops shock mines.",
		Element: ElementShock, Color: mineColor,
		BaseCooldown: 1.8, CooldownPerLevel: 0.06, BaseDamage: 10, DamagePerLevel: 4,
		BaseCount: 1, LevelsPerCount: 2, BaseRadius: 70, RadiusPerLevel: 8,
	},
}

var weaponBehaviors = [weaponKindCount]weaponBehavior{
	WeaponEmberBolt:     (*Game).fireEmberBolts,
	WeaponFrostNova:     (*Game).pulseFrostNova,
	WeaponArcLightning:  (*Game).castArcLightning,
	WeaponOilFlask:      (*Game).throwOilFlasks,
	WeaponDownpour:      (*Game).callDownpour,
	WeaponOrbitBlades:   (*Game).spinOrbitBlades,
	WeaponSeismicHammer: (*Game).slamSeismicHammer,
	WeaponStaticMines:   (*Game).layStaticMines,
}

// ===== Public API =====

func (d *WeaponDefinition) DamageAt(level int) float32 {
	return d.BaseDamage + d.DamagePerLevel*float32(level-1)
}

func (d *WeaponDefinition) CooldownAt(level int) float32 {
	return d.BaseCooldown * max(0.35, 1-d.CooldownPerLevel*float32(level-1))
}

func (d *WeaponDefinition) CountAt(level int) int {
	return d.BaseCount + (level-1)/d.LevelsPerCount
}

func (d *WeaponDefinition) RadiusAt(level int) float32 {
	return d.BaseRadius + d.RadiusPerLevel*float32(level-1)
}

// ===== Internal =====

func (g *Game) updateWeapons(deltaSeconds float32) {
	for slot := range g.player.Weapons {
		g.updateWeapon(&g.player.Weapons[slot], deltaSeconds)
	}
}

func (g *Game) updateWeapon(weapon *WeaponState, deltaSeconds float32) {
	definition := &weaponTable[weapon.Kind]
	weapon.CooldownRemaining = max(0, weapon.CooldownRemaining-deltaSeconds)
	isHeldBack := definition.IsAimed && !g.player.IsFiring
	if weapon.CooldownRemaining > 0 || isHeldBack {
		return
	}
	weapon.CooldownRemaining = definition.CooldownAt(weapon.Level) * g.player.CooldownMultiplier
	weaponBehaviors[weapon.Kind](g, weapon, definition)
	g.playSound(definition.FireSound)
	g.player.Rig.ArmRecoil = max(g.player.Rig.ArmRecoil, boolToFloat(definition.IsAimed))
}

func (g *Game) fireEmberBolts(weapon *WeaponState, definition *WeaponDefinition) {
	player := g.player
	baseAngle := atan2(player.AimY, player.AimX)
	count := definition.CountAt(weapon.Level)
	for shot := range count {
		angle := baseAngle + (float32(shot)-float32(count-1)/2)*boltSpreadRadians
		g.projectiles.SpawnBolt(player.Rig.MuzzleX, player.Rig.MuzzleY, angle, boltSpeed,
			definition.DamageAt(weapon.Level)*player.DamageMultiplier, definition.Element, int16(1+weapon.Level/3), definition.Color)
	}
}

func (g *Game) pulseFrostNova(weapon *WeaponState, definition *WeaponDefinition) {
	player := g.player
	radius := definition.RadiusAt(weapon.Level) * player.AreaMultiplier
	g.ApplyBurst(Burst{
		X: player.X, Y: player.Y, Radius: radius,
		Damage: definition.DamageAt(weapon.Level) * player.DamageMultiplier, Element: definition.Element,
	})
	g.ground.StimulateArea(player.X, player.Y, int(radius/groundCellSize), StimulusChill)
	g.effects.AddRing(player.X, player.Y, radius, definition.Color)
	g.effects.SpawnSparks(player.X, player.Y, 24, definition.Color, radius*2)
}

func (g *Game) castArcLightning(weapon *WeaponState, definition *WeaponDefinition) {
	player := g.player
	current := g.nearestEnemyInAim(player.X, player.Y, targetSearchRadius)
	if current < 0 {
		g.effects.AddLightning(player.Rig.MuzzleX, player.Rig.MuzzleY, player.X+player.AimX*fizzleDistance, player.Y+player.AimY*fizzleDistance)
		return
	}
	fromX, fromY := player.Rig.MuzzleX, player.Rig.MuzzleY
	g.chainedEnemies = g.chainedEnemies[:0]
	damage := definition.DamageAt(weapon.Level) * player.DamageMultiplier
	for hop := 0; hop < definition.CountAt(weapon.Level) && current >= 0; hop++ {
		toX, toY := g.enemies.PositionX[current], g.enemies.PositionY[current]
		g.effects.AddLightning(fromX, fromY, toX, toY)
		g.chainedEnemies = append(g.chainedEnemies, int32(current))
		g.HitEnemy(current, damage, definition.Element)
		fromX, fromY = toX, toY
		current = g.nearestEnemyWithin(toX, toY, chainSearchRadius, true)
	}
}

func (g *Game) throwOilFlasks(weapon *WeaponState, definition *WeaponDefinition) {
	player := g.player
	count := definition.CountAt(weapon.Level)
	for volley := range count {
		targetX, targetY := g.aimPoint(volley, count, throwDistance)
		g.projectiles.SpawnFlask(player.Rig.MuzzleX, player.Rig.MuzzleY, targetX, targetY, flaskFlightSeconds,
			definition.DamageAt(weapon.Level)*player.DamageMultiplier, definition.RadiusAt(weapon.Level)*player.AreaMultiplier, definition.Color)
	}
}

func (g *Game) callDownpour(weapon *WeaponState, definition *WeaponDefinition) {
	count := definition.CountAt(weapon.Level)
	for volley := range count {
		targetX, targetY := g.aimPoint(volley, count, rainDistance)
		g.rainAt(targetX, targetY, definition, weapon.Level)
	}
}

func (g *Game) aimPoint(volley, count int, distance float32) (float32, float32) {
	player := g.player
	angle := atan2(player.AimY, player.AimX) + (float32(volley)-float32(count-1)/2)*volleySpreadRadians
	return player.X + cosine(angle)*distance, player.Y + sine(angle)*distance
}

func (g *Game) rainAt(x, y float32, definition *WeaponDefinition, level int) {
	radius := definition.RadiusAt(level) * g.player.AreaMultiplier
	g.ApplyBurst(Burst{X: x, Y: y, Radius: radius, Damage: definition.DamageAt(level) * g.player.DamageMultiplier, Element: definition.Element})
	g.ground.StimulateArea(x, y, int(radius/groundCellSize), StimulusDouse)
	g.effects.AddRing(x, y, radius, definition.Color)
	g.effects.SpawnRain(x, y, radius, definition.Color)
}

func (g *Game) slamSeismicHammer(weapon *WeaponState, definition *WeaponDefinition) {
	player := g.player
	radius := definition.RadiusAt(weapon.Level) * player.AreaMultiplier
	damage := definition.DamageAt(weapon.Level) * player.DamageMultiplier
	g.ApplyBurst(Burst{X: player.X, Y: player.Y, Radius: radius, Damage: damage, Element: definition.Element})
	g.QueueBurst(Burst{X: player.X, Y: player.Y, Radius: radius * hammerQuakeReach, Damage: damage, Element: definition.Element, Requires: StatusFrozen})
	g.effects.AddRing(player.X, player.Y, radius*hammerQuakeReach, hammerDustColor)
	g.effects.AddRing(player.X, player.Y, radius, definition.Color)
	g.effects.AddRing(player.X, player.Y, radius*hammerInnerRing, hammerDustColor)
	g.effects.SpawnSparks(player.X, player.Y, hammerDustSparks, hammerDustColor, radius*1.5)
	g.effects.AddShake(hammerShake)
}

func (g *Game) spinOrbitBlades(weapon *WeaponState, definition *WeaponDefinition) {
	player := g.player
	player.BladeCount = definition.CountAt(weapon.Level)
	player.BladeRadius = definition.RadiusAt(weapon.Level) * player.AreaMultiplier
	player.BladeAngle += bladeSpinSpeed / ticksPerSecond
	damage := definition.DamageAt(weapon.Level) * player.DamageMultiplier
	for blade := range player.BladeCount {
		bladeX, bladeY := player.BladePosition(blade)
		g.slashAround(bladeX, bladeY, damage, definition.Element)
	}
}

func (g *Game) slashAround(x, y, damage float32, element Element) {
	g.candidates = g.grid.AppendNearby(x, y, bladeHitRadius+20, g.candidates[:0])
	for _, candidate := range g.candidates {
		g.slashIfTouching(int(candidate), x, y, damage, element)
	}
}

func (g *Game) slashIfTouching(index int, x, y, damage float32, element Element) {
	enemies := g.enemies
	reach := bladeHitRadius + enemyTable[enemies.Kind[index]].Radius
	isTouching := distanceSquared(enemies.PositionX[index], enemies.PositionY[index], x, y) < reach*reach
	if !isTouching || enemies.BladeCooldown[index] > 0 {
		return
	}
	enemies.BladeCooldown[index] = bladeHitCooldown
	g.HitEnemy(index, damage, element)
	g.PushEnemy(index, g.player.X, g.player.Y, bladeKnockback)
	g.effects.SpawnSparks(enemies.PositionX[index], enemies.PositionY[index], 3, [3]float32{1, 1, 1}, 140)
}

func (g *Game) nearestEnemyWithin(x, y, radius float32, shouldSkipChained bool) int {
	g.candidates = g.grid.AppendNearby(x, y, radius, g.candidates[:0])
	nearest, nearestDistance := -1, radius*radius
	for _, candidate := range g.candidates {
		index := int(candidate)
		distance := distanceSquared(x, y, g.enemies.PositionX[index], g.enemies.PositionY[index])
		isEligible := g.enemies.Health[index] > 0 && !(shouldSkipChained && g.isChained(candidate))
		if isEligible && distance < nearestDistance {
			nearest, nearestDistance = index, distance
		}
	}
	return nearest
}

func (g *Game) nearestEnemyInAim(x, y, radius float32) int {
	g.candidates = g.grid.AppendNearby(x, y, radius, g.candidates[:0])
	nearest, nearestDistance := -1, radius*radius
	for _, candidate := range g.candidates {
		index := int(candidate)
		distance := distanceSquared(x, y, g.enemies.PositionX[index], g.enemies.PositionY[index])
		isEligible := g.enemies.Health[index] > 0 && g.isInsideAimCone(index, x, y)
		if isEligible && distance < nearestDistance {
			nearest, nearestDistance = index, distance
		}
	}
	return nearest
}

func (g *Game) isInsideAimCone(index int, x, y float32) bool {
	directionX, directionY := normalize(g.enemies.PositionX[index]-x, g.enemies.PositionY[index]-y)
	return directionX*g.player.AimX+directionY*g.player.AimY >= aimConeMinimumDot
}

func (g *Game) isChained(index int32) bool {
	for _, chained := range g.chainedEnemies {
		if chained == index {
			return true
		}
	}
	return false
}

func angleBetween(fromX, fromY, toX, toY float32) float32 {
	return atan2(toY-fromY, toX-fromX)
}
