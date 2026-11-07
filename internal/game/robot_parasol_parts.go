package game

// ===== Constants =====

const (
	parasolRibCount      = 8
	parasolOpenRadius    = 15
	parasolClosedRadius  = 6
	parasolSpinRate      = 0.4
	parasolRibAngleStep  = 6.2831855 / parasolRibCount
	parasolArtDomeTop    = -76
	parasolArtDomeCenter = -56
	parasolArtDomeHeight = 20
	parasolArtDomeWidth  = 38
)

var parasolArtFolds = [5]float32{-30, -15, 0, 15, 30}

// ===== Screen parts =====

func (r *Renderer) queueCrookHead(player *Player) {
	frame := &r.robotFrame
	headX, headY := r.headPoint(player, 4, 0)
	backX, backY := r.headPoint(player, -3, 0)
	bendX, bendY := r.headPoint(player, -7, 3)
	tipX, tipY := r.headPoint(player, -5, 7)
	r.batches.Solid.AddSegment(backX, backY, bendX, bendY, scaledTsukumo(2.2), robotPaletteSilver, frame.Alpha)
	r.batches.Solid.AddSegment(bendX, bendY, tipX, tipY, scaledTsukumo(2.2), robotPaletteSilver, frame.Alpha)
	r.batches.Solid.AddCircle(headX, headY, scaledTsukumo(7.2*frame.Breath), frame.Paint.Outline, frame.Alpha)
	r.batches.Solid.AddCircle(headX, headY, scaledTsukumo(6*frame.Breath), frame.Body, frame.Alpha)
}

func (r *Renderer) queueCanopyAbdomen(player *Player) {
	rig := &player.Rig
	frame := &r.robotFrame
	alpha := frame.Alpha
	radius := (parasolOpenRadius + (parasolClosedRadius-parasolOpenRadius)*rig.TuckAmount) * frame.Breath
	centerX, centerY := r.shiftedAbdomenPoint(player, -13, 0)
	jointX, jointY := r.tsukumoPoint(player, -5, 0)
	r.batches.Solid.AddSegment(jointX, jointY, centerX, centerY, scaledTsukumo(2.4), robotPaletteSilver, alpha)
	r.batches.Solid.AddCircle(centerX, centerY, scaledTsukumo(radius+1.2), frame.Paint.Outline, alpha)
	r.batches.Solid.AddCircle(centerX, centerY, scaledTsukumo(radius), frame.Body, alpha)
	for rib := range parasolRibCount {
		r.queueCanopyRib(player, centerX, centerY, radius, rib)
	}
	r.batches.Solid.AddCircle(centerX, centerY, scaledTsukumo(2), robotPaletteSilver, alpha)
}

func (r *Renderer) queueCanopyRib(player *Player, centerX, centerY, radius float32, rib int) {
	frame := &r.robotFrame
	angle := player.Rig.BobPhase*parasolSpinRate + float32(rib)*parasolRibAngleStep
	forward, side := rotateOffset(radius, 0, angle)
	tipX, tipY := r.shiftedAbdomenPoint(player, -13+forward, side)
	r.batches.Solid.AddSegment(centerX, centerY, tipX, tipY, scaledTsukumo(radius*0.42), frame.Paint.Highlight, 0.28*frame.Alpha*boolToFloat(rib%2 == 0))
	r.batches.Solid.AddSegment(centerX, centerY, tipX, tipY, scaledTsukumo(0.9), frame.Paint.Outline, frame.Alpha)
	r.batches.Solid.AddCircle(tipX, tipY, scaledTsukumo(1), robotPaletteSilver, frame.Alpha)
}

func (r *Renderer) queueRibLeg(player *Player, leg int) {
	alpha, outline := r.robotFrame.Alpha, r.robotFrame.Paint.Outline
	hipX, hipY, kneeX, kneeY, footX, footY := r.legPoints(player, leg)
	r.batches.Solid.AddCircle(footX+2, footY+4, scaledTsukumo(3.6), blackColor, 0.3*alpha)
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(2.6), outline, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(2.6), outline, alpha)
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(1.3), robotPaletteSilver, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(1.3), robotPaletteSilver, alpha)
	r.batches.Solid.AddCircle(kneeX, kneeY, scaledTsukumo(1.7), outline, alpha)
	r.queueWheel(footX, footY, player.Rig.Legs[leg].WheelSpin, alpha)
}

func (r *Renderer) queueRibArms(player *Player) {
	alpha, outline := r.robotFrame.Alpha, r.robotFrame.Paint.Outline
	reach := 15 - 3*player.Rig.ArmRecoil
	for _, side := range [2]float32{-1, 1} {
		shoulderX, shoulderY := r.headPoint(player, 4, 6*side)
		tipX, tipY := r.headPoint(player, reach, 4.5*side)
		r.batches.Solid.AddSegment(shoulderX, shoulderY, tipX, tipY, scaledTsukumo(2.2), outline, alpha)
		r.batches.Solid.AddSegment(shoulderX, shoulderY, tipX, tipY, scaledTsukumo(1), robotPaletteSilver, alpha)
		r.batches.Solid.AddCircle(tipX, tipY, scaledTsukumo(1.2), robotPaletteSilver, alpha)
	}
}

// ===== Art parts =====

func (c *ArtCanvas) paintRibRearLegs(pose ArtPose, _ *RobotPaint) {
	for _, limb := range tsukumoRearLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{1.3, 1.3}, [2]float32{1.2, 1.2}
		c.paintLimb(pose, limb, robotFarStrut, robotFarStrut)
		c.paintWheel(pose, limb.Foot, 7.5)
	}
}

func (c *ArtCanvas) paintRibFrontLegs(pose ArtPose, paint *RobotPaint) {
	for _, limb := range tsukumoFrontLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{1.5, 1.5}, [2]float32{1.4, 1.4}
		c.paintLimb(pose, limb, robotStrut, paint.Dark)
		c.paintWheel(pose, limb.Foot, 11.25)
	}
}

func (c *ArtCanvas) paintCanopyAbdomen(pose ArtPose, paint *RobotPaint) {
	c.paintBox(pose, 4, parasolArtDomeTop-3, 1.5, 4, paint.Dark)
	c.paintEllipsoid(pose, 4, parasolArtDomeCenter, parasolArtDomeWidth, parasolArtDomeHeight, paint.Shell)
	for _, fold := range parasolArtFolds {
		across := fold / parasolArtDomeWidth
		rimY := parasolArtDomeCenter + parasolArtDomeHeight*sqrt(1-across*across)
		c.paintCapsule(pose, 4, parasolArtDomeTop, 4+fold, rimY, 0.9, 0.9, paint.Dark)
	}
	c.paintCapsule(pose, 4, -38, 2, -30, 2, 2, robotStrut)
}

func (c *ArtCanvas) paintCrookHead(pose ArtPose, paint *RobotPaint) {
	c.paintCapsule(pose, -13, -36, -20, -42, 1.8, 1.8, robotStrut)
	c.paintCapsule(pose, -20, -42, -24, -36, 1.8, 1.8, robotStrut)
	c.paintEllipsoid(pose, 0, -36, 15, 13, paint.Head)
}

func (c *ArtCanvas) paintRibArms(pose ArtPose, _ *RobotPaint) {
	for _, side := range [2]float32{-1, 1} {
		c.paintCapsule(pose, 12*side, -27, 20*side, -15, 1.2, 1.2, robotStrut)
		c.paintFlatDisc(pose, 20*side, -15, 1.6, robotFlatSilver)
	}
}
