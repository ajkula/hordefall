package main

// ===== Types =====

type Ring struct {
	X      float32
	Y      float32
	Radius float32
	Life   float32
	Color  [3]float32
}

type LightningBolt struct {
	Points [lightningSegments + 1][2]float32
	Life   float32
}

type Popup struct {
	X     float32
	Y     float32
	Text  string
	Color [3]float32
	Life  float32
}

type Effects struct {
	ParticleCount int
	ParticleX     []float32
	ParticleY     []float32
	VelocityX     []float32
	VelocityY     []float32
	Life          []float32
	MaxLife       []float32
	Size          []float32
	Color         [][3]float32
	Rings         []Ring
	Bolts         []LightningBolt
	Popups        []Popup
	ShakeTrauma   float32
	ShakeX        float32
	ShakeY        float32
	ShakeFactor   float32
	Density       float32
	Skids         SkidMarks
	random        Random
}

// ===== Constants =====

const (
	maximumParticles     = 10000
	maximumRings         = 96
	maximumBolts         = 64
	maximumPopups        = 28
	lightningSegments    = 7
	lightningJitter      = 14
	ringLifeSeconds      = 0.35
	lightningLifeSeconds = 0.14
	popupLifeSeconds     = 0.9
	popupRiseSpeed       = 46
	particleDrag         = 0.9
	shakeDecayPerSecond  = 2.2
)

// ===== Public API =====

func NewEffects() *Effects {
	return &Effects{
		ParticleX: make([]float32, maximumParticles), ParticleY: make([]float32, maximumParticles),
		VelocityX: make([]float32, maximumParticles), VelocityY: make([]float32, maximumParticles),
		Life: make([]float32, maximumParticles), MaxLife: make([]float32, maximumParticles),
		Size: make([]float32, maximumParticles), Color: make([][3]float32, maximumParticles),
		Rings:       make([]Ring, 0, maximumRings),
		Bolts:       make([]LightningBolt, 0, maximumBolts),
		Popups:      make([]Popup, 0, maximumPopups),
		ShakeFactor: 1,
		Density:     1,
		random:      NewRandom(0xBEEF),
	}
}

func (e *Effects) SpawnSparks(x, y float32, count int, color [3]float32, speed float32) {
	for range count {
		angle := e.random.Angle()
		velocity := e.random.Between(0.3, 1) * speed
		e.spawnParticle(x, y, cosine(angle)*velocity, sine(angle)*velocity, e.random.Between(0.25, 0.6), e.random.Between(2, 4), color)
	}
}

func (e *Effects) SpawnRain(x, y, radius float32, color [3]float32) {
	for range 40 {
		angle := e.random.Angle()
		distance := e.random.Float() * radius
		e.spawnParticle(x+cosine(angle)*distance, y+sine(angle)*distance-120, -20, 520, 0.25, 2, color)
	}
}

func (e *Effects) SpawnTrail(x, y float32, color [3]float32) {
	e.spawnParticle(x+e.random.Between(-4, 4), y+e.random.Between(-4, 4), 0, 0, 0.3, 5, color)
}

func (e *Effects) AddRing(x, y, radius float32, color [3]float32) {
	if radius <= 0 || len(e.Rings) >= maximumRings {
		return
	}
	e.Rings = append(e.Rings, Ring{X: x, Y: y, Radius: radius, Life: ringLifeSeconds, Color: color})
}

func (e *Effects) AddLightning(fromX, fromY, toX, toY float32) {
	if len(e.Bolts) >= maximumBolts {
		return
	}
	bolt := LightningBolt{Life: lightningLifeSeconds}
	for segment := range lightningSegments + 1 {
		progress := float32(segment) / lightningSegments
		jitter := lightningJitter * boolToFloat(segment > 0 && segment < lightningSegments)
		bolt.Points[segment] = [2]float32{
			fromX + (toX-fromX)*progress + e.random.Between(-jitter, jitter),
			fromY + (toY-fromY)*progress + e.random.Between(-jitter, jitter),
		}
	}
	e.Bolts = append(e.Bolts, bolt)
}

func (e *Effects) AddPopup(x, y float32, text string, color [3]float32) {
	if text == "" || len(e.Popups) >= maximumPopups {
		return
	}
	e.Popups = append(e.Popups, Popup{X: x, Y: y - 20, Text: text, Color: color, Life: popupLifeSeconds})
}

func (e *Effects) AddShake(strength float32) {
	e.ShakeTrauma = min(1, e.ShakeTrauma+strength*0.05)
}

func (e *Effects) ShakeOffset() (float32, float32) {
	return e.ShakeX, e.ShakeY
}

func (e *Effects) Update(deltaSeconds float32) {
	for index := e.ParticleCount - 1; index >= 0; index-- {
		e.updateParticle(index, deltaSeconds)
	}
	e.Rings = filterAlive(e.Rings, deltaSeconds, func(ring *Ring) *float32 { return &ring.Life })
	e.Bolts = filterAlive(e.Bolts, deltaSeconds, func(bolt *LightningBolt) *float32 { return &bolt.Life })
	e.Popups = filterAlive(e.Popups, deltaSeconds, func(popup *Popup) *float32 { return &popup.Life })
	for index := range e.Popups {
		e.Popups[index].Y -= popupRiseSpeed * deltaSeconds
	}
	e.ShakeTrauma = max(0, e.ShakeTrauma-shakeDecayPerSecond*deltaSeconds)
	e.Skids.Update(deltaSeconds)
	magnitude := e.ShakeTrauma * e.ShakeTrauma * 14 * e.ShakeFactor
	e.ShakeX, e.ShakeY = e.random.Between(-magnitude, magnitude), e.random.Between(-magnitude, magnitude)
}

func (e *Effects) Clear() {
	e.ParticleCount = 0
	e.Rings, e.Bolts, e.Popups = e.Rings[:0], e.Bolts[:0], e.Popups[:0]
	e.ShakeTrauma, e.ShakeX, e.ShakeY = 0, 0, 0
	e.Skids.Clear()
}

// ===== Internal =====

func (e *Effects) spawnParticle(x, y, velocityX, velocityY, life, size float32, color [3]float32) {
	isThinnedOut := e.random.Float() >= e.Density
	if e.ParticleCount >= maximumParticles || isThinnedOut {
		return
	}
	index := e.ParticleCount
	e.ParticleCount++
	e.ParticleX[index], e.ParticleY[index] = x, y
	e.VelocityX[index], e.VelocityY[index] = velocityX, velocityY
	e.Life[index], e.MaxLife[index], e.Size[index], e.Color[index] = life, life, size, color
}

func (e *Effects) updateParticle(index int, deltaSeconds float32) {
	e.ParticleX[index] += e.VelocityX[index] * deltaSeconds
	e.ParticleY[index] += e.VelocityY[index] * deltaSeconds
	e.VelocityX[index] *= particleDrag
	e.VelocityY[index] *= particleDrag
	e.Life[index] -= deltaSeconds
	if e.Life[index] > 0 {
		return
	}
	last := e.ParticleCount - 1
	e.ParticleX[index], e.ParticleY[index] = e.ParticleX[last], e.ParticleY[last]
	e.VelocityX[index], e.VelocityY[index] = e.VelocityX[last], e.VelocityY[last]
	e.Life[index], e.MaxLife[index], e.Size[index], e.Color[index] = e.Life[last], e.MaxLife[last], e.Size[last], e.Color[last]
	e.ParticleCount--
}

func filterAlive[T any](items []T, deltaSeconds float32, lifeOf func(item *T) *float32) []T {
	kept := items[:0]
	for index := range items {
		life := lifeOf(&items[index])
		*life -= deltaSeconds
		kept = appendIf(kept, items[index], *life > 0)
	}
	return kept
}
