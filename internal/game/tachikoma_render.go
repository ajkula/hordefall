package game

// ===== Constants =====

const (
	dropletSteps = 4
)

var (
	tachikomaBlue      = [3]float32{0.22, 0.52, 1}
	tachikomaDeepBlue  = [3]float32{0.03, 0.07, 0.2}
	tachikomaHighlight = [3]float32{0.62, 0.8, 1}
	tachikomaWhite     = [3]float32{0.95, 0.97, 1}
	tachikomaLegShade  = [3]float32{0.5, 0.56, 0.68}
	tachikomaPupil     = [3]float32{0.06, 0.08, 0.14}
	tachikomaWheel     = [3]float32{0.14, 0.15, 0.18}
)

var tachikomaEyes = [3][2]float32{{10, 0}, {6.2, -4.6}, {6.2, 4.6}}

// ===== Internal =====

func (r *Renderer) queueTachikoma(g *Game) {
	player := g.player
	rig := &player.Rig
	isBlinking := player.InvulnerableSeconds > 0 && (g.frame/4)%2 == 0
	alpha := 1 - 0.5*boolToFloat(isBlinking)
	hurt := clamp(player.DamageFlashSeconds*4, 0, 1)
	body := mixColor(tachikomaBlue, hurtColor, hurt)
	breath := 1 + 0.045*sine(rig.BobPhase*3.14159265)
	r.queueTachikomaShadow(player, rig)
	for leg := range tachikomaLegCount {
		r.queueTachikomaLeg(player, rig, leg, alpha)
	}
	r.queueTachikomaAbdomen(player, rig, body, breath, alpha)
	r.queueNeckBearing(player, rig, alpha)
	r.queueTachikomaBarrel(player, rig, alpha)
	r.queueTachikomaArms(player, rig, alpha)
	r.queueTachikomaHead(player, rig, body, breath, alpha)
}

func (r *Renderer) tachikomaPoint(player *Player, forward, side float32) (float32, float32) {
	return r.ToScreen(player.Rig.LocalToWorld(player.X, player.Y, forward, side))
}

func (r *Renderer) headPoint(player *Player, forward, side float32) (float32, float32) {
	return r.ToScreen(player.Rig.HeadToWorld(player.X, player.Y, forward, side))
}

func (r *Renderer) abdomenPoint(player *Player, forward, side float32) (float32, float32) {
	return r.ToScreen(player.Rig.AbdomenToWorld(player.X, player.Y, forward, side))
}

func (r *Renderer) queueTachikomaShadow(player *Player, rig *TachikomaRig) {
	abdomenX, abdomenY := r.abdomenPoint(player, -13, 0)
	headX, headY := r.headPoint(player, 4, 0)
	r.solid.AddCircle(abdomenX+rig.AbdomenOffsetX+3, abdomenY+rig.AbdomenOffsetY+6, scaledTachikoma(12), blackColor, 0.3)
	r.solid.AddCircle(headX+3, headY+6, scaledTachikoma(9), blackColor, 0.3)
}

func (r *Renderer) queueTachikomaLeg(player *Player, rig *TachikomaRig, leg int, alpha float32) {
	layout := &tachikomaLegLayouts[leg]
	state := &rig.Legs[leg]
	hipWorldX, hipWorldY := rig.LocalToWorld(player.X, player.Y, layout.HipForward, layout.HipSide)
	kneeWorldX, kneeWorldY := rig.KneePosition(hipWorldX, hipWorldY, state.FootX, state.FootY, player.X, player.Y)
	hipX, hipY := r.ToScreen(hipWorldX, hipWorldY)
	kneeX, kneeY := r.ToScreen(kneeWorldX, kneeWorldY)
	footX, footY := r.ToScreen(state.FootX, state.FootY)
	r.solid.AddCircle(footX+2, footY+4, scaledTachikoma(3.6), blackColor, 0.3*alpha)
	r.queueDroplet(hipX, hipY, kneeX, kneeY, scaledTachikoma(2.2), scaledTachikoma(3.6), tachikomaDeepBlue, alpha)
	r.queueDroplet(kneeX, kneeY, footX, footY, scaledTachikoma(3.6), scaledTachikoma(2), tachikomaDeepBlue, alpha)
	r.queueDroplet(hipX, hipY, kneeX, kneeY, scaledTachikoma(1.5), scaledTachikoma(2.9), tachikomaWhite, alpha)
	r.queueDroplet(kneeX, kneeY, footX, footY, scaledTachikoma(2.9), scaledTachikoma(1.3), tachikomaWhite, alpha)
	r.solid.AddCircle(kneeX-0.7, kneeY-0.9, scaledTachikoma(1), tachikomaLegShade, alpha)
	r.queueWheel(footX, footY, state.WheelSpin, alpha)
}

func (r *Renderer) queueWheel(x, y, spin, alpha float32) {
	r.solid.AddCircle(x, y, scaledTachikoma(tachikomaWheelRadius), tachikomaWheel, alpha)
	r.solid.AddCircle(x, y, scaledTachikoma(1.4), tachikomaWhite, alpha)
	r.solid.AddSegment(x-cosine(spin)*3, y-sine(spin)*3, x+cosine(spin)*3, y+sine(spin)*3, scaledTachikoma(0.9), tachikomaLegShade, alpha)
}

func (r *Renderer) queueDroplet(fromX, fromY, toX, toY, fromRadius, toRadius float32, tint [3]float32, alpha float32) {
	r.solid.AddCircle(fromX, fromY, fromRadius, tint, alpha)
	r.solid.AddCircle(toX, toY, toRadius, tint, alpha)
	for step := range dropletSteps {
		startProgress := float32(step) / dropletSteps
		endProgress := float32(step+1) / dropletSteps
		width := 2 * (fromRadius + (toRadius-fromRadius)*(startProgress+endProgress)/2)
		r.solid.AddSegment(
			fromX+(toX-fromX)*startProgress, fromY+(toY-fromY)*startProgress,
			fromX+(toX-fromX)*endProgress, fromY+(toY-fromY)*endProgress, width, tint, alpha)
	}
}

func (r *Renderer) queueTachikomaAbdomen(player *Player, rig *TachikomaRig, body [3]float32, breath, alpha float32) {
	jointX, jointY := r.tachikomaPoint(player, -5, 0)
	centerX, centerY := r.abdomenPoint(player, -13, 0)
	tailX, tailY := r.abdomenPoint(player, -22, 0)
	shiftX, shiftY := rig.AbdomenOffsetX, rig.AbdomenOffsetY
	r.solid.AddSegment(jointX, jointY, centerX+shiftX, centerY+shiftY, scaledTachikoma(7), tachikomaDeepBlue, alpha)
	r.solid.AddCircle(centerX+shiftX, centerY+shiftY, scaledTachikoma(11.5*breath), tachikomaDeepBlue, alpha)
	r.solid.AddCircle(centerX+shiftX, centerY+shiftY, scaledTachikoma(10.2*breath), body, alpha)
	r.solid.AddSegment(centerX+shiftX, centerY+shiftY, tailX+shiftX, tailY+shiftY, scaledTachikoma(1.4), tachikomaDeepBlue, alpha)
	r.solid.AddCircle(centerX+shiftX-2, centerY+shiftY-3, scaledTachikoma(4*breath), tachikomaHighlight, 0.55*alpha)
	r.solid.AddCircle(jointX, jointY, scaledTachikoma(3.2), tachikomaWhite, alpha)
}

func (r *Renderer) queueNeckBearing(player *Player, rig *TachikomaRig, alpha float32) {
	pivotX, pivotY := r.tachikomaPoint(player, tachikomaNeckForward, 0)
	neckX, neckY := r.tachikomaPoint(player, -5, 0)
	r.solid.AddSegment(neckX, neckY, pivotX, pivotY, scaledTachikoma(6), tachikomaDeepBlue, alpha)
	r.solid.AddCircle(pivotX, pivotY, scaledTachikoma(5.4), tachikomaDeepBlue, alpha)
	r.solid.AddCircle(pivotX, pivotY, scaledTachikoma(4.4), tachikomaLegShade, alpha)
	r.solid.AddCircle(pivotX, pivotY, scaledTachikoma(3.1), tachikomaDeepBlue, alpha)
	for _, notch := range [2]float32{1.5708, -1.5708} {
		notchX, notchY := rotateOffset(3.8, 0, rig.HeadAngle()+notch)
		r.solid.AddCircle(pivotX+notchX, pivotY+notchY, scaledTachikoma(0.9), tachikomaWhite, alpha)
	}
}

func (r *Renderer) queueTachikomaBarrel(player *Player, rig *TachikomaRig, alpha float32) {
	recoil := 4 * rig.ArmRecoil
	breechX, breechY := r.headPoint(player, 4, 0)
	muzzleX, muzzleY := r.headPoint(player, tachikomaMuzzleForward-recoil, 0)
	r.solid.AddSegment(breechX, breechY, muzzleX, muzzleY, scaledTachikoma(3.4), tachikomaDeepBlue, alpha)
	r.solid.AddSegment(breechX, breechY, muzzleX, muzzleY, scaledTachikoma(1.6), tachikomaLegShade, alpha)
	r.solid.AddCircle(muzzleX, muzzleY, scaledTachikoma(1.9), tachikomaDeepBlue, alpha)
	r.glow.AddCircleIf(muzzleX, muzzleY, scaledTachikoma(9), [3]float32{1, 0.7, 0.3}, rig.ArmRecoil*0.8, rig.ArmRecoil > 0.4)
}

func (r *Renderer) queueTachikomaHead(player *Player, rig *TachikomaRig, body [3]float32, breath, alpha float32) {
	headX, headY := r.headPoint(player, 4, 0)
	r.solid.AddCircle(headX, headY, scaledTachikoma(8.6*breath), tachikomaDeepBlue, alpha)
	r.solid.AddCircle(headX, headY, scaledTachikoma(7.5*breath), body, alpha)
	highlightX, highlightY := r.headPoint(player, 2.5, -2.5)
	r.solid.AddCircle(highlightX, highlightY, scaledTachikoma(2.8), tachikomaHighlight, 0.5*alpha)
	lookForward, lookSide := rotateOffset(rig.LookX, rig.LookY, -rig.HeadAngle())
	for _, eye := range tachikomaEyes {
		r.queueTachikomaEye(player, eye[0]+lookForward*0.8, eye[1]+lookSide*1.2, rig, alpha)
	}
}

func (r *Renderer) queueTachikomaEye(player *Player, forward, side float32, rig *TachikomaRig, alpha float32) {
	eyeX, eyeY := r.headPoint(player, forward, side)
	r.solid.AddCircle(eyeX, eyeY, scaledTachikoma(3), tachikomaDeepBlue, alpha)
	r.solid.AddCircle(eyeX, eyeY, scaledTachikoma(2.4), tachikomaWhite, alpha)
	r.solid.AddCircle(eyeX+rig.LookX*1, eyeY+rig.LookY*1, scaledTachikoma(1.25), tachikomaPupil, alpha)
}

func (r *Renderer) queueTachikomaArms(player *Player, rig *TachikomaRig, alpha float32) {
	reach := 10 - 4*rig.ArmRecoil
	for _, side := range [2]float32{-6.5, 6.5} {
		shoulderX, shoulderY := r.headPoint(player, 5, side)
		tipX, tipY := r.headPoint(player, 5+reach, side*0.85)
		r.queueDroplet(shoulderX, shoulderY, tipX, tipY, scaledTachikoma(1.4), scaledTachikoma(0.8), tachikomaWhite, alpha)
		r.solid.AddCircle(tipX, tipY, scaledTachikoma(1.2), tachikomaLegShade, alpha)
	}
}

func scaledTachikoma(value float32) float32 {
	return value * tachikomaScale
}
