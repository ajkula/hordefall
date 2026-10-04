package game

import "hordefall/internal/audio"

// ===== Types =====

type StaticMine struct {
	X      float32
	Y      float32
	Age    float32
	Damage float32
	Radius float32
}

// ===== Constants =====

const (
	maximumMines       = 48
	mineArmSeconds     = 0.4
	mineLifeSeconds    = 10
	mineTriggerRadius  = 30
	mineDropScatter    = 36
	mineBlinkSpeed     = 9
	mineBodyRadius     = 5
	mineGlowRadius     = 15
	enemyMaximumRadius = 60
	mineDetonateSparks = 14
)

var (
	mineColor     = [3]float32{0.95, 0.85, 0.25}
	mineBodyColor = [3]float32{0.3, 0.28, 0.22}
)

// ===== Internal =====

func (g *Game) layStaticMines(weapon *WeaponState, definition *WeaponDefinition) {
	player := g.player
	for range definition.CountAt(weapon.Level) {
		mine := StaticMine{
			X:      clamp(player.X+g.random.Between(-mineDropScatter, mineDropScatter), 0, arenaSize),
			Y:      clamp(player.Y+g.random.Between(-mineDropScatter, mineDropScatter), 0, arenaSize),
			Damage: definition.DamageAt(weapon.Level) * player.DamageMultiplier,
			Radius: definition.RadiusAt(weapon.Level) * player.AreaMultiplier,
		}
		g.mines = appendIf(g.mines, mine, len(g.mines) < maximumMines)
	}
}

func (g *Game) updateMines(deltaSeconds float32) {
	for index := len(g.mines) - 1; index >= 0; index-- {
		g.updateMine(index, deltaSeconds)
	}
}

func (g *Game) updateMine(index int, deltaSeconds float32) {
	mine := &g.mines[index]
	mine.Age += deltaSeconds
	isExpired := mine.Age >= mineLifeSeconds
	isTriggered := mine.Age >= mineArmSeconds && g.hasEnemyWithin(mine.X, mine.Y, mineTriggerRadius)
	if !isExpired && !isTriggered {
		return
	}
	g.detonateMineIf(*mine, isTriggered)
	g.mines[index] = g.mines[len(g.mines)-1]
	g.mines = g.mines[:len(g.mines)-1]
}

func (g *Game) detonateMineIf(mine StaticMine, isTriggered bool) {
	if !isTriggered {
		return
	}
	g.ApplyBurst(Burst{X: mine.X, Y: mine.Y, Radius: mine.Radius, Damage: mine.Damage, Element: ElementShock})
	g.effects.AddRing(mine.X, mine.Y, mine.Radius, mineColor)
	g.effects.SpawnSparks(mine.X, mine.Y, mineDetonateSparks, mineColor, mine.Radius*2)
	g.playSound(audio.SoundZap)
}

func (g *Game) hasEnemyWithin(x, y, radius float32) bool {
	g.candidates = g.grid.AppendNearby(x, y, radius+enemyMaximumRadius, g.candidates[:0])
	for _, candidate := range g.candidates {
		index := int(candidate)
		reach := radius + enemyTable[g.enemies.Kind[index]].Radius
		if distanceSquared(g.enemies.PositionX[index], g.enemies.PositionY[index], x, y) < reach*reach {
			return true
		}
	}
	return false
}

func (r *Renderer) queueMines(g *Game) {
	for _, mine := range g.mines {
		screenX, screenY := r.ToScreen(mine.X, mine.Y)
		isVisible := r.isVisible(mine.X, mine.Y, mineGlowRadius)
		isArmed := mine.Age >= mineArmSeconds
		blink := 0.5 + 0.5*sine(mine.Age*mineBlinkSpeed)
		r.solid.AddCircleIf(screenX, screenY, mineBodyRadius, mineBodyColor, 1, isVisible)
		r.solid.AddCircleIf(screenX, screenY, mineBodyRadius*0.5, mineColor, 0.4+0.6*blink*boolToFloat(isArmed), isVisible)
		r.glow.AddCircleIf(screenX, screenY, mineGlowRadius, mineColor, 0.12+0.25*blink*boolToFloat(isArmed), isVisible)
	}
}
