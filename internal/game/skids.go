package game

// ===== Types =====

type SkidMark struct {
	FromX    float32
	FromY    float32
	ToX      float32
	ToY      float32
	Width    float32
	Strength float32
	Age      float32
}

type SkidMarks struct {
	Marks [maximumSkidMarks]SkidMark
	next  int
}

// ===== Constants =====

const (
	maximumSkidMarks      = 1024
	skidLifeSeconds       = 1.6
	skidAccelerationStart = 420
	skidAccelerationFull  = 1900
	skidRiseSharpness     = 0.6
	skidFallSharpness     = 0.12
	skidMinimumStrength   = 0.06
	skidMinimumStep       = 0.6
	skidWidthPerWheel     = 0.8
	skidMaximumOpacity    = 0.75
	skidDustChance        = 0.3
)

var (
	skidColor = [3]float32{0.95, 0.97, 1}
	dustColor = [3]float32{0.75, 0.78, 0.82}
)

// ===== Public API =====

func (s *SkidMarks) Add(mark SkidMark) {
	s.Marks[s.next] = mark
	s.next = (s.next + 1) % maximumSkidMarks
}

func (s *SkidMarks) Update(deltaSeconds float32) {
	for index := range s.Marks {
		s.Marks[index].Age += deltaSeconds
	}
}

func (s *SkidMarks) Clear() {
	s.Marks = [maximumSkidMarks]SkidMark{}
	s.next = 0
}

func SkidOpacity(mark *SkidMark) float32 {
	remaining := clamp(1-mark.Age/skidLifeSeconds, 0, 1)
	return mark.Strength * remaining * remaining * remaining * skidMaximumOpacity
}

// ===== Internal =====

func (g *Game) updateSkid(rig *TachikomaRig, accelerationX, accelerationY float32) {
	target := clamp((length(accelerationX, accelerationY)-skidAccelerationStart)/(skidAccelerationFull-skidAccelerationStart), 0, 1)
	sharpness := skidFallSharpness + (skidRiseSharpness-skidFallSharpness)*boolToFloat(target > rig.SkidAmount)
	rig.SkidAmount += (target - rig.SkidAmount) * sharpness
	for leg := range tachikomaLegCount {
		g.markWheel(rig, leg)
	}
}

func (g *Game) markWheel(rig *TachikomaRig, leg int) {
	state := &rig.Legs[leg]
	fromX, fromY := state.TrailX, state.TrailY
	state.TrailX, state.TrailY = state.FootX, state.FootY
	hasMoved := distanceSquared(fromX, fromY, state.FootX, state.FootY) > skidMinimumStep*skidMinimumStep
	if rig.SkidAmount < skidMinimumStrength || !hasMoved {
		return
	}
	wheelDiameter := float32(2 * tachikomaWheelRadius * tachikomaScale)
	g.effects.Skids.Add(SkidMark{
		FromX: fromX, FromY: fromY, ToX: state.FootX, ToY: state.FootY,
		Width:    wheelDiameter * skidWidthPerWheel * (0.6 + 0.4*rig.SkidAmount),
		Strength: rig.SkidAmount,
	})
	g.raiseDustIf(state.FootX, state.FootY, g.random.Chance(rig.SkidAmount*skidDustChance))
}

func (g *Game) raiseDustIf(x, y float32, shouldRaise bool) {
	if !shouldRaise {
		return
	}
	g.effects.SpawnSparks(x, y, 1, dustColor, 40)
}

func (r *Renderer) queueSkidMarks(g *Game) {
	for index := range g.effects.Skids.Marks {
		r.queueSkidMark(&g.effects.Skids.Marks[index])
	}
}

func (r *Renderer) queueSkidMark(mark *SkidMark) {
	opacity := SkidOpacity(mark)
	if opacity <= 0.005 || !r.isVisible(mark.ToX, mark.ToY, mark.Width) {
		return
	}
	fromX, fromY := r.ToScreen(mark.FromX, mark.FromY)
	toX, toY := r.ToScreen(mark.ToX, mark.ToY)
	r.glow.AddSegment(fromX, fromY, toX, toY, mark.Width, skidColor, opacity)
}
