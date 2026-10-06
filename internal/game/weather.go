package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"hordefall/internal/audio"
)

// ===== Types =====

type WeatherKind uint8

type Precipitation struct {
	Count      int
	VelocityX  float32
	VelocityY  float32
	Size       float32
	SizeSpread float32
	Sway       float32
	Color      [3]float32
	Alpha      float32
	Queue      func(r *Renderer, style *Precipitation, x, y, size, alpha float32)
}

type WeatherDefinition struct {
	Key                   string
	LabelColor            [3]float32
	Multiply              [3]float32
	Lift                  [3]float32
	EnemyStatus           StatusFlags
	ClearedStatus         StatusFlags
	StatusChancePerSecond float32
	GroundStimulus        GroundStimulus
	GroundCellsPerTick    float32
	GroundInnerCells      float32
	StrikeChancePerSecond float32
	StartSound            audio.SoundKind
	Precipitation         Precipitation
}

type Weather struct {
	Kind          WeatherKind
	ActiveSeconds float32
	CalmSeconds   float32
	FlashSeconds  float32
}

// ===== Constants =====

const (
	WeatherClear WeatherKind = iota
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
	weatherParticleMargin     = 40
	weatherLabelTop           = 44
	strikeFlashSeconds        = 0.35
	strikeFlashStrength       = 0.45
	strikeMinimumDistance     = 140
	strikeMaximumDistance     = 520
	strikeRadius              = 60
	strikeDamage              = 26
	strikeHeight              = 640
	strikeShake               = 4
)

var weatherTable = [weatherKindCount]WeatherDefinition{
	WeatherClear: {Multiply: [3]float32{1, 1, 1}},
	WeatherStorm: {
		Key: "storm", LabelColor: [3]float32{0.6, 0.75, 1},
		Multiply: [3]float32{0.62, 0.68, 0.82}, Lift: [3]float32{0, 0.01, 0.03},
		EnemyStatus: StatusWet, ClearedStatus: StatusBurning, StatusChancePerSecond: 0.35,
		GroundStimulus: StimulusDouse, GroundCellsPerTick: 160,
		StrikeChancePerSecond: 0.3, StartSound: audio.SoundRain,
		Precipitation: Precipitation{
			Count: 260, VelocityX: -160, VelocityY: 1100, Size: 22, SizeSpread: 10,
			Color: [3]float32{0.65, 0.75, 1}, Alpha: 0.4, Queue: queueRainStreak,
		},
	},
	WeatherHeatwave: {
		Key: "heatwave", LabelColor: [3]float32{1, 0.6, 0.2},
		Multiply: [3]float32{1, 0.86, 0.66}, Lift: [3]float32{0.1, 0.04, 0},
		GroundStimulus: StimulusHeat, GroundCellsPerTick: 30, GroundInnerCells: 5,
		StartSound: audio.SoundHiss,
		Precipitation: Precipitation{
			Count: 90, VelocityX: 15, VelocityY: -45, Size: 1.5, SizeSpread: 1.2, Sway: 10,
			Color: [3]float32{1, 0.55, 0.15}, Alpha: 0.8, Queue: queueEmber,
		},
	},
	WeatherBlizzard: {
		Key: "blizzard", LabelColor: [3]float32{0.85, 0.95, 1},
		Multiply: [3]float32{0.7, 0.84, 1}, Lift: [3]float32{0.12, 0.16, 0.22},
		EnemyStatus: StatusChilled, StatusChancePerSecond: 0.4,
		GroundStimulus: StimulusChill, GroundCellsPerTick: 30,
		StartSound: audio.SoundFreeze,
		Precipitation: Precipitation{
			Count: 320, VelocityX: -50, VelocityY: 90, Size: 1.6, SizeSpread: 1.6, Sway: 18,
			Color: [3]float32{1, 1, 1}, Alpha: 0.85, Queue: queueSnowflake,
		},
	},
}

var (
	multiplyBlend = ebiten.Blend{
		BlendFactorSourceRGB: ebiten.BlendFactorDestinationColor, BlendFactorSourceAlpha: ebiten.BlendFactorZero,
		BlendFactorDestinationRGB: ebiten.BlendFactorZero, BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
		BlendOperationRGB: ebiten.BlendOperationAdd, BlendOperationAlpha: ebiten.BlendOperationAdd,
	}
	flashColor = [3]float32{1, 1, 1}
)

// ===== Public API =====

func (w *Weather) Intensity() float32 {
	elapsed := weatherActiveSeconds - w.ActiveSeconds
	fade := clamp(min(elapsed, w.ActiveSeconds)/weatherFadeSeconds, 0, 1)
	return fade * boolToFloat(w.Kind != WeatherClear)
}

// ===== Internal =====

func (g *Game) resetWeather() {
	g.weather = Weather{CalmSeconds: g.random.Between(weatherCalmMinimumSeconds, weatherCalmMaximumSeconds)}
}

func (g *Game) updateWeather(deltaSeconds float32) {
	weather := &g.weather
	weather.FlashSeconds = max(0, weather.FlashSeconds-deltaSeconds)
	weather.CalmSeconds -= deltaSeconds * boolToFloat(weather.Kind == WeatherClear)
	weather.ActiveSeconds = max(0, weather.ActiveSeconds-deltaSeconds)
	g.startWeatherIfDue()
	g.endWeatherIfOver()
	definition := &weatherTable[weather.Kind]
	intensity := weather.Intensity()
	g.soakEnemies(definition, definition.StatusChancePerSecond*deltaSeconds*intensity)
	g.stimulateWeatherGround(definition, intensity)
	g.strikeLightningIf(g.random.Chance(definition.StrikeChancePerSecond * deltaSeconds * intensity))
}

func (g *Game) startWeatherIfDue() {
	weather := &g.weather
	if weather.Kind != WeatherClear || weather.CalmSeconds > 0 {
		return
	}
	weather.Kind = WeatherKind(1 + g.random.Below(int(weatherKindCount)-1))
	weather.ActiveSeconds = weatherActiveSeconds
	g.playSound(weatherTable[weather.Kind].StartSound)
}

func (g *Game) endWeatherIfOver() {
	if g.weather.Kind == WeatherClear || g.weather.ActiveSeconds > 0 {
		return
	}
	g.resetWeather()
}

func (g *Game) soakEnemies(definition *WeatherDefinition, chance float32) {
	if definition.EnemyStatus == 0 || chance <= 0 {
		return
	}
	enemies := g.enemies
	for index := range enemies.Count {
		isDry := enemies.Status[index]&definition.EnemyStatus == 0
		g.soakEnemyIf(index, definition, isDry && g.random.Chance(chance))
	}
}

func (g *Game) soakEnemyIf(index int, definition *WeatherDefinition, shouldSoak bool) {
	if !shouldSoak {
		return
	}
	g.enemies.Status[index] &^= definition.ClearedStatus
	g.enemies.ApplyStatus(index, definition.EnemyStatus)
}

func (g *Game) stimulateWeatherGround(definition *WeatherDefinition, intensity float32) {
	isGroundTick := g.frame%groundTickFrames == 0
	count := int(definition.GroundCellsPerTick*intensity) * boolToIndex(isGroundTick)
	for range count {
		angle := g.random.Angle()
		distance := g.random.Between(definition.GroundInnerCells, weatherGroundRadiusCells) * groundCellSize
		g.ground.StimulateAt(g.player.X+cosine(angle)*distance, g.player.Y+sine(angle)*distance, definition.GroundStimulus)
	}
}

func (g *Game) strikeLightningIf(shouldStrike bool) {
	if !shouldStrike {
		return
	}
	angle := g.random.Angle()
	distance := g.random.Between(strikeMinimumDistance, strikeMaximumDistance)
	x := clamp(g.player.X+cosine(angle)*distance, 0, arenaSize)
	y := clamp(g.player.Y+sine(angle)*distance, 0, arenaSize)
	g.weather.FlashSeconds = strikeFlashSeconds
	g.effects.AddLightning(x+g.random.Between(-80, 80), y-strikeHeight, x, y)
	g.effects.AddRing(x, y, strikeRadius, weatherTable[WeatherStorm].LabelColor)
	g.effects.SpawnSparks(x, y, 12, flashColor, 200)
	g.effects.AddShake(strikeShake)
	g.playSound(audio.SoundElectrocute)
	g.QueueBurst(Burst{X: x, Y: y, Radius: strikeRadius, Damage: strikeDamage, Element: ElementShock})
}

func (r *Renderer) queuePrecipitation(g *Game) {
	intensity := g.weather.Intensity()
	style := &weatherTable[g.weather.Kind].Precipitation
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
		style.Queue(r, style, x, y, style.Size+style.SizeSpread*seedSize, style.Alpha*intensity)
	}
}

func queueRainStreak(r *Renderer, style *Precipitation, x, y, size, alpha float32) {
	directionX, directionY := normalize(style.VelocityX, style.VelocityY)
	r.glow.AddSegment(x, y, x+directionX*size, y+directionY*size, 1.5, style.Color, alpha)
}

func queueSnowflake(r *Renderer, style *Precipitation, x, y, size, alpha float32) {
	r.solid.AddCircle(x, y, size, style.Color, alpha)
}

func queueEmber(r *Renderer, style *Precipitation, x, y, size, alpha float32) {
	r.glow.AddCircle(x, y, size*3, style.Color, alpha*0.35)
	r.glow.AddCircle(x, y, size, style.Color, alpha)
}

func (r *Renderer) applyWeatherGrade(g *Game, screen *ebiten.Image) {
	intensity := g.weather.Intensity()
	flash := g.weather.FlashSeconds / strikeFlashSeconds * strikeFlashStrength
	if intensity <= 0 && flash <= 0 {
		return
	}
	definition := &weatherTable[g.weather.Kind]
	multiply := mixColor([3]float32{1, 1, 1}, definition.Multiply, intensity)
	lift := mixColor([3]float32{0, 0, 0}, definition.Lift, intensity)
	drawScreenQuad(screen, multiply, multiplyBlend)
	drawScreenQuad(screen, mixColor(lift, flashColor, flash), ebiten.BlendLighter)
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
