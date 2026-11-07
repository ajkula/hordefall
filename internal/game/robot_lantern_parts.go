package game

// ===== Constants =====

var lanternRibOffsets = [5]float32{-7, -3.5, 0, 3.5, 7}

var lanternArtRibRows = [5]float32{-68, -59, -50, -41, -32}

// ===== Screen parts =====

func (r *Renderer) queueCapHead(player *Player) {
	frame := &r.robotFrame
	headX, headY := r.headPoint(player, 4, 0)
	r.batches.Solid.AddCircle(headX, headY, scaledTsukumo(8.6*frame.Breath), frame.Paint.Outline, frame.Alpha)
	r.batches.Solid.AddCircle(headX, headY, scaledTsukumo(7.4*frame.Breath), frame.Body, frame.Alpha)
	r.batches.Solid.AddCircle(headX, headY, scaledTsukumo(4.6*frame.Breath), robotPaletteRaisin, frame.Alpha)
}

func (r *Renderer) queueCyclopsEyes(player *Player) {
	rig := &player.Rig
	alpha := r.robotFrame.Alpha
	lookForward, lookSide := rotateOffset(rig.LookX, rig.LookY, -rig.HeadAngle())
	eyeX, eyeY := r.headPoint(player, 9+lookForward*0.6, lookSide*0.8)
	r.batches.Solid.AddCircle(eyeX, eyeY, scaledTsukumo(4.8), r.robotFrame.Paint.Outline, alpha)
	r.batches.Solid.AddCircle(eyeX, eyeY, scaledTsukumo(4), tsukumoWhite, alpha)
	r.batches.Solid.AddCircle(eyeX+rig.LookX*1.8, eyeY+rig.LookY*1.8, scaledTsukumo(2), tsukumoPupil, alpha)
}

func (r *Renderer) queuePaperAbdomen(player *Player) {
	frame := &r.robotFrame
	alpha, breath := frame.Alpha, frame.Breath
	jointX, jointY := r.tsukumoPoint(player, -5, 0)
	centerX, centerY := r.shiftedAbdomenPoint(player, -13, 0)
	r.batches.Glow.AddCircle(centerX, centerY, scaledTsukumo(24*breath), frame.Paint.Highlight, 0.22*alpha)
	r.batches.Solid.AddSegment(jointX, jointY, centerX, centerY, scaledTsukumo(4), frame.Paint.Outline, alpha)
	r.batches.Solid.AddCircle(centerX, centerY, scaledTsukumo(12.5*breath), frame.Paint.Outline, alpha)
	r.batches.Solid.AddCircle(centerX, centerY, scaledTsukumo(11.2*breath), frame.Body, alpha)
	r.batches.Solid.AddCircle(centerX, centerY, scaledTsukumo(6.5*breath), frame.Paint.Highlight, 0.7*alpha)
	for _, offset := range lanternRibOffsets {
		halfWidth := sqrt(121 - offset*offset)
		fromX, fromY := r.shiftedAbdomenPoint(player, -13+offset, -halfWidth)
		toX, toY := r.shiftedAbdomenPoint(player, -13+offset, halfWidth)
		r.batches.Solid.AddSegment(fromX, fromY, toX, toY, scaledTsukumo(0.9), frame.Paint.Outline, 0.7*alpha)
	}
	r.queueLanternTassel(player)
}

func (r *Renderer) queueLanternTassel(player *Player) {
	rig := &player.Rig
	alpha := r.robotFrame.Alpha
	rootX, rootY := r.shiftedAbdomenPoint(player, -25, 0)
	tipX, tipY := r.abdomenPoint(player, -33, 0)
	tipX, tipY = tipX+rig.AbdomenOffsetX*2.2, tipY+rig.AbdomenOffsetY*2.2
	r.batches.Solid.AddSegment(rootX, rootY, tipX, tipY, scaledTsukumo(1.2), robotPaletteSilver, alpha)
	r.batches.Solid.AddCircle(tipX, tipY, scaledTsukumo(1.9), robotPaletteYellow, alpha)
}

func (r *Renderer) queueBambooLeg(player *Player, leg int) {
	rig := &player.Rig
	alpha, outline := r.robotFrame.Alpha, r.robotFrame.Paint.Outline
	hipX, hipY, kneeX, kneeY, footX, footY := r.legPoints(player, leg)
	r.batches.Solid.AddCircle(footX+2, footY+4, scaledTsukumo(3.6), blackColor, 0.3*alpha)
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(3.4), outline, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(3.4), outline, alpha)
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(2), robotPaletteSilver, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(2), robotPaletteSilver, alpha)
	r.batches.Solid.AddCircle((hipX+kneeX)/2, (hipY+kneeY)/2, scaledTsukumo(1.6), outline, alpha)
	r.batches.Solid.AddCircle((kneeX+footX)/2, (kneeY+footY)/2, scaledTsukumo(1.6), outline, alpha)
	r.batches.Solid.AddCircle(kneeX, kneeY, scaledTsukumo(2.1), outline, alpha)
	r.queueWheel(footX, footY, rig.Legs[leg].WheelSpin, alpha)
}

func (r *Renderer) queueHookArms(player *Player) {
	alpha, outline := r.robotFrame.Alpha, r.robotFrame.Paint.Outline
	pullBack := 3 * player.Rig.ArmRecoil
	for _, side := range [2]float32{-1, 1} {
		shoulderX, shoulderY := r.headPoint(player, 4, 7*side)
		elbowX, elbowY := r.headPoint(player, 11-pullBack, 10.5*side)
		tipX, tipY := r.headPoint(player, 14-pullBack, 6.5*side)
		r.batches.Solid.AddSegment(shoulderX, shoulderY, elbowX, elbowY, scaledTsukumo(2.8), outline, alpha)
		r.batches.Solid.AddSegment(elbowX, elbowY, tipX, tipY, scaledTsukumo(2.8), outline, alpha)
		r.batches.Solid.AddSegment(shoulderX, shoulderY, elbowX, elbowY, scaledTsukumo(1.4), robotPaletteSilver, alpha)
		r.batches.Solid.AddSegment(elbowX, elbowY, tipX, tipY, scaledTsukumo(1.4), robotPaletteSilver, alpha)
	}
}

// ===== Art parts =====

func (c *ArtCanvas) paintBambooRearLegs(pose ArtPose, _ *RobotPaint) {
	for _, limb := range tsukumoRearLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{1.8, 1.8}, [2]float32{1.6, 1.6}
		c.paintLimb(pose, limb, robotFarStrut, robotFarStrut)
		c.paintWheel(pose, limb.Foot, 7.5)
	}
}

func (c *ArtCanvas) paintBambooFrontLegs(pose ArtPose, paint *RobotPaint) {
	for _, limb := range tsukumoFrontLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{2.2, 2.2}, [2]float32{2, 2}
		c.paintLimb(pose, limb, robotStrut, paint.Dark)
		c.paintEllipsoid(pose, (limb.Hip[0]+limb.Knee[0])/2, (limb.Hip[1]+limb.Knee[1])/2, 2.6, 2.6, paint.Dark)
		c.paintEllipsoid(pose, (limb.Knee[0]+limb.Foot[0])/2, (limb.Knee[1]+limb.Foot[1])/2, 2.6, 2.6, paint.Dark)
		c.paintWheel(pose, limb.Foot, 11.25)
	}
}

func (c *ArtCanvas) paintPaperAbdomen(pose ArtPose, paint *RobotPaint) {
	c.paintBox(pose, 4, -77, 13, 3, paint.Dark)
	c.paintEllipsoid(pose, 4, -50, 32, 26, paint.Shell)
	for _, row := range lanternArtRibRows {
		offset := (row + 50) / 26
		halfWidth := 32 * sqrt(1-offset*offset)
		c.paintCapsule(pose, 4-halfWidth, row, 4+halfWidth, row, 0.9, 0.9, paint.Dark)
	}
}

func (c *ArtCanvas) paintCapHead(pose ArtPose, paint *RobotPaint) {
	c.paintCapsule(pose, 2, -30, 0, -36, 8, 8, paint.Dark)
	c.paintEllipsoid(pose, 0, -36, 19, 13, paint.Head)
}

func (c *ArtCanvas) paintCyclopsEyes(pose ArtPose, _ *RobotPaint) {
	c.paintEye(pose, 0, -37, 9)
}

func (c *ArtCanvas) paintHookArms(pose ArtPose, _ *RobotPaint) {
	for _, side := range [2]float32{-1, 1} {
		c.paintCapsule(pose, 14*side, -27, 24*side, -20, 1.6, 1.6, robotStrut)
		c.paintCapsule(pose, 24*side, -20, 20*side, -13, 1.6, 1.2, robotStrut)
	}
}
