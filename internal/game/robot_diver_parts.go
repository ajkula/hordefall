package game

// ===== Constants =====

const (
	diverRivetCount     = 8
	diverRivetAngleStep = 6.2831855 / diverRivetCount
	diverBubbleCount    = 4
	diverBubbleRate     = 0.5
	diverHoseRibCount   = 3
)

var diverArtTankSides = [2]float32{-14, 22}

// ===== Screen parts =====

func (r *Renderer) queueHelmetHead(player *Player) {
	frame := &r.robotFrame
	headX, headY := r.headPoint(player, 4, 0)
	r.batches.Solid.AddCircle(headX, headY, scaledTsukumo(8.9*frame.Breath), frame.Paint.Outline, frame.Alpha)
	r.batches.Solid.AddCircle(headX, headY, scaledTsukumo(7.7*frame.Breath), frame.Body, frame.Alpha)
	for rivet := range diverRivetCount {
		forward, side := rotateOffset(6.6, 0, float32(rivet)*diverRivetAngleStep)
		rivetX, rivetY := r.headPoint(player, 4+forward, side)
		r.batches.Solid.AddCircle(rivetX, rivetY, scaledTsukumo(0.7), robotPaletteRaisin, frame.Alpha)
	}
}

func (r *Renderer) queuePortholeEyes(player *Player) {
	rig := &player.Rig
	alpha := r.robotFrame.Alpha
	eyeX, eyeY := r.headPoint(player, 8.5, 0)
	r.batches.Solid.AddCircle(eyeX, eyeY, scaledTsukumo(5), robotPaletteRaisin, alpha)
	r.batches.Solid.AddCircle(eyeX, eyeY, scaledTsukumo(4.2), robotPaletteSilver, alpha)
	r.batches.Solid.AddCircle(eyeX, eyeY, scaledTsukumo(3.2), robotPaletteLens, alpha)
	r.batches.Solid.AddCircle(eyeX+rig.LookX*1.6, eyeY+rig.LookY*1.6, scaledTsukumo(1.6), tsukumoPupil, alpha)
}

func (r *Renderer) queueTankAbdomen(player *Player) {
	frame := &r.robotFrame
	alpha, breath := frame.Alpha, frame.Breath
	jointX, jointY := r.tsukumoPoint(player, -5, 0)
	centerX, centerY := r.shiftedAbdomenPoint(player, -13, 0)
	r.batches.Solid.AddSegment(jointX, jointY, centerX, centerY, scaledTsukumo(5), frame.Paint.Outline, alpha)
	for _, side := range [2]float32{-5.4, 5.4} {
		rearX, rearY := r.shiftedAbdomenPoint(player, -22, side)
		frontX, frontY := r.shiftedAbdomenPoint(player, -7, side)
		r.queueDroplet(rearX, rearY, frontX, frontY, scaledTsukumo(6.2*breath), scaledTsukumo(6.2*breath), frame.Paint.Outline, alpha)
		r.queueDroplet(rearX, rearY, frontX, frontY, scaledTsukumo(5*breath), scaledTsukumo(5*breath), frame.Body, alpha)
	}
	r.queueCrateBand(player, -18, -16, 22, robotPaletteRaisin, alpha)
	r.queueCrateBand(player, -12, -10, 22, robotPaletteRaisin, alpha)
	for bubble := range diverBubbleCount * boolToIndex(frame.IsOnWater) {
		r.queueDiverBubble(player, bubble)
	}
}

func (r *Renderer) queueDiverBubble(player *Player, bubble int) {
	phase := player.Rig.BobPhase*diverBubbleRate + float32(bubble)/diverBubbleCount
	age := phase - float32(int(phase))
	x, y := r.shiftedAbdomenPoint(player, -24-16*age, 5*sine(phase*5+float32(bubble)))
	opacity := (1 - age) * r.robotFrame.Alpha
	r.batches.Solid.AddCircle(x, y, scaledTsukumo(1.4+1.8*age), robotPaletteSilver, 0.7*opacity)
	r.batches.Solid.AddCircle(x, y, scaledTsukumo(0.8+1.5*age), robotPaletteLens, 0.5*opacity)
}

func (r *Renderer) queueHoseLeg(player *Player, leg int) {
	alpha := r.robotFrame.Alpha
	outline := r.robotFrame.Paint.Outline
	hipX, hipY, kneeX, kneeY, footX, footY := r.legPoints(player, leg)
	r.batches.Solid.AddCircle(footX+2, footY+4, scaledTsukumo(3.6), blackColor, 0.3*alpha)
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(4), outline, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(4), outline, alpha)
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(2.4), robotPaletteSilver, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(2.4), robotPaletteSilver, alpha)
	r.queueHoseRibs(hipX, hipY, kneeX, kneeY)
	r.queueHoseRibs(kneeX, kneeY, footX, footY)
	r.queueWheel(footX, footY, player.Rig.Legs[leg].WheelSpin, alpha)
}

func (r *Renderer) queueHoseRibs(fromX, fromY, toX, toY float32) {
	for rib := range diverHoseRibCount {
		progress := (float32(rib) + 1) / (diverHoseRibCount + 1)
		r.batches.Solid.AddCircle(fromX+(toX-fromX)*progress, fromY+(toY-fromY)*progress, scaledTsukumo(1.2), robotPaletteRaisin, r.robotFrame.Alpha)
	}
}

func (r *Renderer) queuePincerArms(player *Player) {
	alpha, outline := r.robotFrame.Alpha, r.robotFrame.Paint.Outline
	recoil := player.Rig.ArmRecoil
	opening := 1.5 + 3*recoil
	for _, side := range [2]float32{-1, 1} {
		shoulderX, shoulderY := r.headPoint(player, 4, 7*side)
		wristX, wristY := r.headPoint(player, 11-3*recoil, 9.5*side)
		innerX, innerY := r.headPoint(player, 16-3*recoil, (9.5-opening)*side)
		outerX, outerY := r.headPoint(player, 16-3*recoil, (9.5+opening)*side)
		r.batches.Solid.AddSegment(shoulderX, shoulderY, wristX, wristY, scaledTsukumo(3), outline, alpha)
		r.batches.Solid.AddSegment(shoulderX, shoulderY, wristX, wristY, scaledTsukumo(1.6), robotPaletteSilver, alpha)
		r.batches.Solid.AddSegment(wristX, wristY, innerX, innerY, scaledTsukumo(1.5), robotPaletteSilver, alpha)
		r.batches.Solid.AddSegment(wristX, wristY, outerX, outerY, scaledTsukumo(1.5), robotPaletteSilver, alpha)
		r.batches.Solid.AddCircle(wristX, wristY, scaledTsukumo(1.8), outline, alpha)
	}
}

// ===== Art parts =====

func (c *ArtCanvas) paintHoseRearLegs(pose ArtPose, _ *RobotPaint) {
	for _, limb := range tsukumoRearLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{2.2, 2.2}, [2]float32{2, 2}
		c.paintLimb(pose, limb, robotFarStrut, robotFarStrut)
		c.paintWheel(pose, limb.Foot, 7.5)
	}
}

func (c *ArtCanvas) paintHoseFrontLegs(pose ArtPose, paint *RobotPaint) {
	for _, limb := range tsukumoFrontLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{2.8, 2.8}, [2]float32{2.6, 2.6}
		c.paintLimb(pose, limb, robotStrut, paint.Dark)
		c.paintFlatDisc(pose, (limb.Hip[0]+limb.Knee[0])/2, (limb.Hip[1]+limb.Knee[1])/2, 1.6, robotFlatRaisin)
		c.paintFlatDisc(pose, (limb.Knee[0]+limb.Foot[0])/2, (limb.Knee[1]+limb.Foot[1])/2, 1.6, robotFlatRaisin)
		c.paintWheel(pose, limb.Foot, 11.25)
	}
}

func (c *ArtCanvas) paintTankAbdomen(pose ArtPose, paint *RobotPaint) {
	for _, side := range diverArtTankSides {
		c.paintEllipsoid(pose, side, -80, 2.4, 3, robotStrut)
		c.paintCapsule(pose, side, -68, side, -42, 11, 11, paint.Shell)
		c.paintBox(pose, side, -62, 11, 1.6, robotFlatRaisin)
	}
}

func (c *ArtCanvas) paintHelmetHead(pose ArtPose, paint *RobotPaint) {
	c.paintCapsule(pose, 2, -24, 0, -30, 11, 11, paint.Dark)
	c.paintEllipsoid(pose, 0, -38, 20, 18, paint.Head)
	for _, side := range [2]float32{-1, 1} {
		c.paintFlatDisc(pose, 16*side, -38, 1.3, robotFlatRaisin)
		c.paintFlatDisc(pose, 11*side, -50, 1.3, robotFlatRaisin)
	}
}

func (c *ArtCanvas) paintPortholeEyes(pose ArtPose, _ *RobotPaint) {
	c.paintEllipsoid(pose, 0, -38, 11, 11, robotStrut)
	c.paintFlatDisc(pose, 0, -38, 8.2, robotFlatLens)
	c.paintFlatDisc(pose, 2, -38, 3.4, flatPupil)
}

func (c *ArtCanvas) paintPincerArms(pose ArtPose, _ *RobotPaint) {
	for _, side := range [2]float32{-1, 1} {
		c.paintCapsule(pose, 14*side, -25, 23*side, -21, 1.8, 1.8, robotStrut)
		c.paintCapsule(pose, 23*side, -21, 27*side, -14, 1.3, 1, robotStrut)
		c.paintCapsule(pose, 23*side, -21, 19*side, -14, 1.3, 1, robotStrut)
	}
}
