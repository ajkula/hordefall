package game

import (
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ajkula/ajkurig/world"
)

// ===== Types =====

type WeatherKind = world.WeatherID

type WeatherSoaking struct {
	EnemyStatus           StatusFlags
	ClearedStatus         StatusFlags
	StatusChancePerSecond float32
}

type precipitationQueue func(r *Renderer, style *world.Precipitation, x, y, size, alpha float32)

// ===== Constants =====

const (
	WeatherClear WeatherKind = world.WeatherClear + iota
	WeatherStorm
	WeatherHeatwave
	WeatherBlizzard
	weatherKindCount
)

const (
	weatherCalmMinimumSeconds = 180
	weatherCalmMaximumSeconds = 240
	weatherActiveSeconds      = 30
	weatherFadeSeconds        = 3
	weatherGroundRadiusCells  = 50
	strikeFlashSeconds        = 0.35
	weatherParticleMargin     = 40
	weatherLabelTop           = 44
	strikeMinimumDistance     = 140
	strikeMaximumDistance     = 520
	strikeRadius              = 60
	strikeDamage              = 26
	strikeHeight              = 640
	strikeShake               = 4
)

var weatherDefinitions = []world.WeatherDefinition{
	WeatherClear: {Multiply: [3]float32{1, 1, 1}},
	WeatherStorm: {
		Key: "storm", LabelColor: [3]float32{0.6, 0.75, 1},
		Multiply: [3]float32{0.62, 0.68, 0.82}, Lift: [3]float32{0, 0.01, 0.03},
		GroundStimulus: StimulusDouse, GroundCellsPerTick: 160, StrikeChancePerSecond: 0.3,
		Precipitation: world.Precipitation{
			Style: world.StyleStreak, Count: 260, VelocityX: -160, VelocityY: 1100, Size: 22, SizeSpread: 10,
			Color: [3]float32{0.65, 0.75, 1}, Alpha: 0.4,
		},
	},
	WeatherHeatwave: {
		Key: "heatwave", LabelColor: [3]float32{1, 0.6, 0.2},
		Multiply: [3]float32{1, 0.86, 0.66}, Lift: [3]float32{0.1, 0.04, 0},
		GroundStimulus: StimulusHeat, GroundCellsPerTick: 30, GroundInnerCells: 5,
		Precipitation: world.Precipitation{
			Style: world.StyleEmber, Count: 90, VelocityX: 15, VelocityY: -45, Size: 1.5, SizeSpread: 1.2, Sway: 10,
			Color: [3]float32{1, 0.55, 0.15}, Alpha: 0.8,
		},
	},
	WeatherBlizzard: {
		Key: "blizzard", LabelColor: [3]float32{0.85, 0.95, 1},
		Multiply: [3]float32{0.7, 0.84, 1}, Lift: [3]float32{0.12, 0.16, 0.22},
		GroundStimulus: StimulusChill, GroundCellsPerTick: 30,
		Precipitation: world.Precipitation{
			Style: world.StyleFlake, Count: 320, VelocityX: -50, VelocityY: 90, Size: 1.6, SizeSpread: 1.6, Sway: 18,
			Color: [3]float32{1, 1, 1}, Alpha: 0.85,
		},
	},
}

var weatherTiming = world.WeatherTiming{
	CalmMinimumSeconds: weatherCalmMinimumSeconds, CalmMaximumSeconds: weatherCalmMaximumSeconds,
	ActiveSeconds: weatherActiveSeconds, FadeSeconds: weatherFadeSeconds,
	GroundRadiusCells: weatherGroundRadiusCells, FlashSeconds: strikeFlashSeconds,
}

var weatherStartSounds = [weatherKindCount]SoundKind{WeatherStorm: SoundRain, WeatherHeatwave: SoundHiss, WeatherBlizzard: SoundFreeze}

var weatherSoakings = [weatherKindCount]WeatherSoaking{
	WeatherStorm:    {EnemyStatus: StatusWet, ClearedStatus: StatusBurning, StatusChancePerSecond: 0.35},
	WeatherBlizzard: {EnemyStatus: StatusChilled, StatusChancePerSecond: 0.4},
}

var precipitationQueues = [3]precipitationQueue{world.StyleStreak: queueRainStreak, world.StyleFlake: queueSnowflake, world.StyleEmber: queueEmber}

var (
	multiplyBlend = ebiten.Blend{
		BlendFactorSourceRGB: ebiten.BlendFactorDestinationColor, BlendFactorSourceAlpha: ebiten.BlendFactorZero,
		BlendFactorDestinationRGB: ebiten.BlendFactorZero, BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
		BlendOperationRGB: ebiten.BlendOperationAdd, BlendOperationAlpha: ebiten.BlendOperationAdd,
	}
	flashColor = [3]float32{1, 1, 1}
)

// ===== Internal =====

func newWeather(soundIDs SoundIDs) world.Weather {
	definitions := slices.Clone(weatherDefinitions)
	for kind := range definitions {
		definitions[kind].StartSound = soundIDs[weatherStartSounds[kind]]
	}
	return world.Weather{Definitions: definitions, Timing: weatherTiming}
}

func (g *Game) resetWeather() {
	g.weather.Reset(&g.random)
}

func (g *Game) updateWeather(deltaSeconds float32) {
	events := g.weather.Advance(deltaSeconds, &g.random)
	g.announceWeatherIf(events.HasStart)
	intensity := g.weather.Intensity()
	soaking := &weatherSoakings[g.weather.Kind]
	g.soakEnemies(soaking, soaking.StatusChancePerSecond*deltaSeconds*intensity)
	g.stimulateWeatherGroundIf(g.frame%groundTickFrames == 0)
	g.strikeLightningIf(g.weather.RollStrike(deltaSeconds, &g.random))
}

func (g *Game) announceWeatherIf(hasStarted bool) {
	if !hasStarted {
		return
	}
	g.app.PlaySound(g.weather.Definition().StartSound)
	g.say(weatherChatter[g.weather.Kind])
}

func (g *Game) soakEnemies(soaking *WeatherSoaking, chance float32) {
	if soaking.EnemyStatus == 0 || chance <= 0 {
		return
	}
	enemies := g.enemies
	for index := range enemies.Count {
		isDry := enemies.Status[index]&soaking.EnemyStatus == 0
		g.soakEnemyIf(index, soaking, isDry && g.random.Chance(chance))
	}
}

func (g *Game) soakEnemyIf(index int, soaking *WeatherSoaking, shouldSoak bool) {
	if !shouldSoak {
		return
	}
	g.enemies.Status[index] &^= soaking.ClearedStatus
	g.enemies.ApplyStatus(index, soaking.EnemyStatus)
}

func (g *Game) stimulateWeatherGroundIf(isGroundTick bool) {
	if !isGroundTick {
		return
	}
	g.weather.StimulateGround(g.ground, g.player.X, g.player.Y, &g.random)
}

func (g *Game) strikeLightningIf(shouldStrike bool) {
	if !shouldStrike {
		return
	}
	angle := g.random.Angle()
	distance := g.random.Between(strikeMinimumDistance, strikeMaximumDistance)
	x := clamp(g.player.X+cosine(angle)*distance, 0, arenaSize)
	y := clamp(g.player.Y+sine(angle)*distance, 0, arenaSize)
	g.effects.AddLightning(x+g.random.Between(-80, 80), y-strikeHeight, x, y)
	g.effects.AddRing(x, y, strikeRadius, [3]float32(weatherDefinitions[WeatherStorm].LabelColor))
	g.effects.SpawnSparks(x, y, 12, flashColor, 200)
	g.effects.AddShake(strikeShake)
	g.playSound(SoundElectrocute)
	g.QueueBurst(Burst{X: x, Y: y, Radius: strikeRadius, Damage: strikeDamage, Element: ElementShock})
}

func (r *Renderer) queuePrecipitation(g *Game) {
	intensity := g.weather.Intensity()
	style := &g.weather.Definition().Precipitation
	if intensity <= 0 {
		return
	}
	spanX, spanY := float32(screenWidth+2*weatherParticleMargin), float32(screenHeight+2*weatherParticleMargin)
	clock := g.elapsedSeconds
	for particle := range style.Count {
		seedX, seedY, seedSize := particleSeed(particle, 0), particleSeed(particle, 1), particleSeed(particle, 2)
		sway := sine(clock*1.7+seedSize*6.28) * style.Sway
		x := wrap(seedX*spanX+style.VelocityX*clock+sway-r.CameraX, spanX) - weatherParticleMargin
		y := wrap(seedY*spanY+style.VelocityY*clock*(0.7+seedSize*0.6)-r.CameraY, spanY) - weatherParticleMargin
		precipitationQueues[style.Style](r, style, x, y, style.Size+style.SizeSpread*seedSize, style.Alpha*intensity)
	}
}

func queueRainStreak(r *Renderer, style *world.Precipitation, x, y, size, alpha float32) {
	directionX, directionY := normalize(style.VelocityX, style.VelocityY)
	r.glow.AddSegment(x, y, x+directionX*size, y+directionY*size, 1.5, [3]float32(style.Color), alpha)
}

func queueSnowflake(r *Renderer, style *world.Precipitation, x, y, size, alpha float32) {
	r.solid.AddCircle(x, y, size, [3]float32(style.Color), alpha)
}

func queueEmber(r *Renderer, style *world.Precipitation, x, y, size, alpha float32) {
	r.glow.AddCircle(x, y, size*3, [3]float32(style.Color), alpha*0.35)
	r.glow.AddCircle(x, y, size, [3]float32(style.Color), alpha)
}

func (r *Renderer) applyWeatherGrade(g *Game, screen *ebiten.Image) {
	if g.weather.Intensity() <= 0 && g.weather.FlashSeconds <= 0 {
		return
	}
	multiply, lift := g.weather.Grade()
	drawScreenQuad(screen, [3]float32(multiply), multiplyBlend)
	drawScreenQuad(screen, [3]float32(lift), ebiten.BlendLighter)
}

func drawScreenQuad(screen *ebiten.Image, tint [3]float32, blend ebiten.Blend) {
	width, height := float32(screen.Bounds().Dx()), float32(screen.Bounds().Dy())
	vertices := make([]ebiten.Vertex, 0, 4)
	for corner := range 4 {
		vertices = append(vertices, ebiten.Vertex{
			DstX: float32(corner&1) * width, DstY: float32(corner>>1) * height, SrcX: 1.5, SrcY: 1.5,
			ColorR: tint[0], ColorG: tint[1], ColorB: tint[2], ColorA: 1,
		})
	}
	solidPixel = ensureSolidPixel(solidPixel)
	screen.DrawTriangles32(vertices, []uint32{0, 1, 2, 1, 3, 2}, solidPixel, &ebiten.DrawTrianglesOptions{Blend: blend})
}

func particleSeed(particle, channel int) float32 {
	hash := uint32(particle)*2654435761 + uint32(channel)*40503 + 0x9E3779B9
	hash = (hash ^ (hash >> 15)) * 2246822519
	return float32((hash^(hash>>13))&0xFFFF) / 0xFFFF
}

func wrap(value, span float32) float32 {
	return value - span*float32(math.Floor(float64(value/span)))
}
