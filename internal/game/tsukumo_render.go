package game

// ===== Constants =====

const (
	dropletSteps = 4
)

var (
	tsukumoBlue      = [3]float32{0.22, 0.52, 1}
	tsukumoDeepBlue  = [3]float32{0.03, 0.07, 0.2}
	tsukumoHighlight = [3]float32{0.62, 0.8, 1}
	tsukumoWhite     = [3]float32{0.95, 0.97, 1}
	tsukumoLegShade  = [3]float32{0.5, 0.56, 0.68}
	tsukumoPupil     = [3]float32{0.06, 0.08, 0.14}
	tsukumoWheel     = [3]float32{0.14, 0.15, 0.18}
)

var tsukumoEyes = [3][2]float32{{10, 0}, {6.2, -4.6}, {6.2, 4.6}}

// ===== Internal =====

func (r *Renderer) queueTsukumo(g *Game) {
	isOnWater := g.ground.MaterialAt(g.player.X, g.player.Y) == GroundWater
	r.queueRobot(g.player, &g.robotLook, g.frame, isOnWater)
}

func (r *Renderer) queueRobot(player *Player, look *RobotLook, frame uint32, isOnWater bool) {
	rig := &player.Rig
	paint := &robotPaints[look.Color]
	isBlinking := player.InvulnerableSeconds > 0 && (frame/4)%2 == 0
	alpha := 1 - 0.5*boolToFloat(isBlinking)
	hurt := clamp(player.DamageFlashSeconds*4, 0, 1)
	body := mixColor(mixColor(paint.Body, hurtColor, hurt), tsukumoWhite, clamp(player.LevelFlashSeconds/volleyBurstFlashSeconds, 0, 1))
	breath := 1 + 0.045*sine(rig.BobPhase*3.14159265)
	r.robotFrame = RobotFrame{IsOnWater: isOnWater, Paint: paint, Body: body, Breath: breath, Alpha: alpha}
	r.queueTsukumoShadow(player, rig)
	for leg := range tsukumoLegCount {
		robotScreenLegs[look.Legs](r, player, leg)
	}
	robotScreenAbdomens[look.Abdomen](r, player)
	r.queueNeckBearing(player)
	r.queueTsukumoBarrel(player)
	robotScreenArms[look.Arms](r, player)
	robotScreenHeads[look.Head](r, player)
	robotScreenEyes[look.Eyes](r, player)
}

func (r *Renderer) tsukumoPoint(player *Player, forward, side float32) (float32, float32) {
	return r.camera.ToScreen(player.Rig.LocalToWorld(player.X, player.Y, forward, side))
}

func (r *Renderer) headPoint(player *Player, forward, side float32) (float32, float32) {
	return r.camera.ToScreen(player.Rig.HeadToWorld(player.X, player.Y, forward, side))
}

func (r *Renderer) abdomenPoint(player *Player, forward, side float32) (float32, float32) {
	return r.camera.ToScreen(player.Rig.AbdomenToWorld(player.X, player.Y, forward, side))
}

func (r *Renderer) queueTsukumoShadow(player *Player, rig *TsukumoRig) {
	abdomenX, abdomenY := r.abdomenPoint(player, -13, 0)
	headX, headY := r.headPoint(player, 4, 0)
	r.batches.Solid.AddCircle(abdomenX+rig.AbdomenOffsetX+3, abdomenY+rig.AbdomenOffsetY+6, scaledTsukumo(12), blackColor, 0.3)
	r.batches.Solid.AddCircle(headX+3, headY+6, scaledTsukumo(9), blackColor, 0.3)
}

func (r *Renderer) queueDropletLeg(player *Player, leg int) {
	rig := &player.Rig
	alpha, outline := r.robotFrame.Alpha, r.robotFrame.Paint.Outline
	layout := &tsukumoLegLayouts[leg]
	state := &rig.Legs[leg]
	hipWorldX, hipWorldY := rig.LocalToWorld(player.X, player.Y, layout.HipForward, layout.HipSide)
	kneeWorldX, kneeWorldY := rig.KneePosition(hipWorldX, hipWorldY, state.FootX, state.FootY, player.X, player.Y)
	hipX, hipY := r.camera.ToScreen(hipWorldX, hipWorldY)
	kneeX, kneeY := r.camera.ToScreen(kneeWorldX, kneeWorldY)
	footX, footY := r.camera.ToScreen(state.FootX, state.FootY)
	r.batches.Solid.AddCircle(footX+2, footY+4, scaledTsukumo(3.6), blackColor, 0.3*alpha)
	r.queueDroplet(hipX, hipY, kneeX, kneeY, scaledTsukumo(2.2), scaledTsukumo(3.6), outline, alpha)
	r.queueDroplet(kneeX, kneeY, footX, footY, scaledTsukumo(3.6), scaledTsukumo(2), outline, alpha)
	r.queueDroplet(hipX, hipY, kneeX, kneeY, scaledTsukumo(1.5), scaledTsukumo(2.9), tsukumoWhite, alpha)
	r.queueDroplet(kneeX, kneeY, footX, footY, scaledTsukumo(2.9), scaledTsukumo(1.3), tsukumoWhite, alpha)
	r.batches.Solid.AddCircle(kneeX-0.7, kneeY-0.9, scaledTsukumo(1), tsukumoLegShade, alpha)
	r.queueWheel(footX, footY, state.WheelSpin, alpha)
}

func (r *Renderer) queueWheel(x, y, spin, alpha float32) {
	r.batches.Solid.AddCircle(x, y, scaledTsukumo(tsukumoWheelRadius), tsukumoWheel, alpha)
	r.batches.Solid.AddCircle(x, y, scaledTsukumo(1.4), tsukumoWhite, alpha)
	r.batches.Solid.AddSegment(x-cosine(spin)*3, y-sine(spin)*3, x+cosine(spin)*3, y+sine(spin)*3, scaledTsukumo(0.9), tsukumoLegShade, alpha)
}

func (r *Renderer) queueDroplet(fromX, fromY, toX, toY, fromRadius, toRadius float32, tint [3]float32, alpha float32) {
	r.batches.Solid.AddCircle(fromX, fromY, fromRadius, tint, alpha)
	r.batches.Solid.AddCircle(toX, toY, toRadius, tint, alpha)
	for step := range dropletSteps {
		startProgress := float32(step) / dropletSteps
		endProgress := float32(step+1) / dropletSteps
		width := 2 * (fromRadius + (toRadius-fromRadius)*(startProgress+endProgress)/2)
		r.batches.Solid.AddSegment(
			fromX+(toX-fromX)*startProgress, fromY+(toY-fromY)*startProgress,
			fromX+(toX-fromX)*endProgress, fromY+(toY-fromY)*endProgress, width, tint, alpha)
	}
}

func (r *Renderer) queueBulbAbdomen(player *Player) {
	rig := &player.Rig
	frame := &r.robotFrame
	body, breath, alpha, outline := frame.Body, frame.Breath, frame.Alpha, frame.Paint.Outline
	jointX, jointY := r.tsukumoPoint(player, -5, 0)
	centerX, centerY := r.abdomenPoint(player, -13, 0)
	tailX, tailY := r.abdomenPoint(player, -22, 0)
	shiftX, shiftY := rig.AbdomenOffsetX, rig.AbdomenOffsetY
	r.batches.Solid.AddSegment(jointX, jointY, centerX+shiftX, centerY+shiftY, scaledTsukumo(7), outline, alpha)
	r.batches.Solid.AddCircle(centerX+shiftX, centerY+shiftY, scaledTsukumo(11.5*breath), outline, alpha)
	r.batches.Solid.AddCircle(centerX+shiftX, centerY+shiftY, scaledTsukumo(10.2*breath), body, alpha)
	r.batches.Solid.AddSegment(centerX+shiftX, centerY+shiftY, tailX+shiftX, tailY+shiftY, scaledTsukumo(1.4), outline, alpha)
	r.batches.Solid.AddCircle(centerX+shiftX-2, centerY+shiftY-3, scaledTsukumo(4*breath), frame.Paint.Highlight, 0.55*alpha)
	r.batches.Solid.AddCircle(jointX, jointY, scaledTsukumo(3.2), tsukumoWhite, alpha)
}

func (r *Renderer) queueNeckBearing(player *Player) {
	rig := &player.Rig
	alpha, outline := r.robotFrame.Alpha, r.robotFrame.Paint.Outline
	pivotX, pivotY := r.tsukumoPoint(player, tsukumoNeckForward, 0)
	neckX, neckY := r.tsukumoPoint(player, -5, 0)
	r.batches.Solid.AddSegment(neckX, neckY, pivotX, pivotY, scaledTsukumo(6), outline, alpha)
	r.batches.Solid.AddCircle(pivotX, pivotY, scaledTsukumo(5.4), outline, alpha)
	r.batches.Solid.AddCircle(pivotX, pivotY, scaledTsukumo(4.4), tsukumoLegShade, alpha)
	r.batches.Solid.AddCircle(pivotX, pivotY, scaledTsukumo(3.1), outline, alpha)
	for _, notch := range [2]float32{1.5708, -1.5708} {
		notchX, notchY := rotateOffset(3.8, 0, rig.HeadAngle()+notch)
		r.batches.Solid.AddCircle(pivotX+notchX, pivotY+notchY, scaledTsukumo(0.9), tsukumoWhite, alpha)
	}
}

func (r *Renderer) queueTsukumoBarrel(player *Player) {
	rig := &player.Rig
	alpha, outline := r.robotFrame.Alpha, r.robotFrame.Paint.Outline
	recoil := 4 * rig.ArmRecoil
	breechX, breechY := r.headPoint(player, 4, 0)
	muzzleX, muzzleY := r.headPoint(player, tsukumoMuzzleForward-recoil, 0)
	r.batches.Solid.AddSegment(breechX, breechY, muzzleX, muzzleY, scaledTsukumo(3.4), outline, alpha)
	r.batches.Solid.AddSegment(breechX, breechY, muzzleX, muzzleY, scaledTsukumo(1.6), tsukumoLegShade, alpha)
	r.batches.Solid.AddCircle(muzzleX, muzzleY, scaledTsukumo(1.9), outline, alpha)
	r.batches.Glow.AddCircleIf(muzzleX, muzzleY, scaledTsukumo(9), [3]float32{1, 0.7, 0.3}, rig.ArmRecoil*0.8, rig.ArmRecoil > 0.4)
}

func (r *Renderer) queueRoundHead(player *Player) {
	frame := &r.robotFrame
	headX, headY := r.headPoint(player, 4, 0)
	r.batches.Solid.AddCircle(headX, headY, scaledTsukumo(8.6*frame.Breath), frame.Paint.Outline, frame.Alpha)
	r.batches.Solid.AddCircle(headX, headY, scaledTsukumo(7.5*frame.Breath), frame.Body, frame.Alpha)
	highlightX, highlightY := r.headPoint(player, 2.5, -2.5)
	r.batches.Solid.AddCircle(highlightX, highlightY, scaledTsukumo(2.8), frame.Paint.Highlight, 0.5*frame.Alpha)
}

func (r *Renderer) queueTrioEyes(player *Player) {
	rig := &player.Rig
	lookForward, lookSide := rotateOffset(rig.LookX, rig.LookY, -rig.HeadAngle())
	for _, eye := range tsukumoEyes {
		r.queueRoundEye(player, eye[0]+lookForward*0.8, eye[1]+lookSide*1.2)
	}
}

func (r *Renderer) queueRoundEye(player *Player, forward, side float32) {
	rig := &player.Rig
	alpha := r.robotFrame.Alpha
	eyeX, eyeY := r.headPoint(player, forward, side)
	r.batches.Solid.AddCircle(eyeX, eyeY, scaledTsukumo(3), r.robotFrame.Paint.Outline, alpha)
	r.batches.Solid.AddCircle(eyeX, eyeY, scaledTsukumo(2.4), tsukumoWhite, alpha)
	r.batches.Solid.AddCircle(eyeX+rig.LookX*1, eyeY+rig.LookY*1, scaledTsukumo(1.25), tsukumoPupil, alpha)
}

func (r *Renderer) queueManipulatorArms(player *Player) {
	alpha := r.robotFrame.Alpha
	reach := 10 - 4*player.Rig.ArmRecoil
	for _, side := range [2]float32{-6.5, 6.5} {
		shoulderX, shoulderY := r.headPoint(player, 5, side)
		tipX, tipY := r.headPoint(player, 5+reach, side*0.85)
		r.queueDroplet(shoulderX, shoulderY, tipX, tipY, scaledTsukumo(1.4), scaledTsukumo(0.8), tsukumoWhite, alpha)
		r.batches.Solid.AddCircle(tipX, tipY, scaledTsukumo(1.2), tsukumoLegShade, alpha)
	}
}

func scaledTsukumo(value float32) float32 {
	return value * tsukumoScale
}

func (r *Renderer) shiftedAbdomenPoint(player *Player, forward, side float32) (float32, float32) {
	x, y := r.abdomenPoint(player, forward, side)
	return x + player.Rig.AbdomenOffsetX, y + player.Rig.AbdomenOffsetY
}

func (r *Renderer) legPoints(player *Player, leg int) (float32, float32, float32, float32, float32, float32) {
	rig := &player.Rig
	layout := &tsukumoLegLayouts[leg]
	state := &rig.Legs[leg]
	hipWorldX, hipWorldY := rig.LocalToWorld(player.X, player.Y, layout.HipForward, layout.HipSide)
	kneeWorldX, kneeWorldY := rig.KneePosition(hipWorldX, hipWorldY, state.FootX, state.FootY, player.X, player.Y)
	hipX, hipY := r.camera.ToScreen(hipWorldX, hipWorldY)
	kneeX, kneeY := r.camera.ToScreen(kneeWorldX, kneeWorldY)
	footX, footY := r.camera.ToScreen(state.FootX, state.FootY)
	return hipX, hipY, kneeX, kneeY, footX, footY
}
