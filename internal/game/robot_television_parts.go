package game

// ===== Constants =====

const televisionStaticBarCount = 4

var televisionEyeSpans = [2]float32{2.2, 0.6}

// ===== Screen parts =====

func (r *Renderer) queueScreenHead(player *Player) {
	frame := &r.robotFrame
	r.queueHeadBand(player, -3, 11, 20*frame.Breath, frame.Paint.Outline)
	r.queueHeadBand(player, -2, 10, 18*frame.Breath, frame.Body)
	r.queueHeadBand(player, -0.5, 8.5, 13.5*frame.Breath, robotPaletteRaisin)
}

func (r *Renderer) queueHeadBand(player *Player, fromForward, toForward, width float32, tint [3]float32) {
	fromX, fromY := r.headPoint(player, fromForward, 0)
	toX, toY := r.headPoint(player, toForward, 0)
	r.batches.Solid.AddSegment(fromX, fromY, toX, toY, scaledTsukumo(width), tint, r.robotFrame.Alpha)
}

func (r *Renderer) queuePixelEyes(player *Player) {
	rig := &player.Rig
	alpha := r.robotFrame.Alpha
	isHurt := player.DamageFlashSeconds > 0
	span := televisionEyeSpans[boolToIndex(isHurt)]
	lookForward, lookSide := rotateOffset(rig.LookX, rig.LookY, -rig.HeadAngle())
	for _, side := range [2]float32{-3.4, 3.4} {
		forward, across := 4+lookForward*1.2, side+lookSide*0.8
		fromX, fromY := r.headPoint(player, forward-span, across)
		toX, toY := r.headPoint(player, forward+span, across)
		r.batches.Solid.AddSegment(fromX, fromY, toX, toY, scaledTsukumo(3.2), robotPaletteLens, alpha)
	}
	glowX, glowY := r.headPoint(player, 4, 0)
	r.batches.Glow.AddCircleIf(glowX, glowY, scaledTsukumo(11), robotPaletteLens, 0.18*alpha, !isHurt)
	for bar := range televisionStaticBarCount * boolToIndex(isHurt) {
		r.queueStaticBar(player, bar)
	}
}

func (r *Renderer) queueStaticBar(player *Player, bar int) {
	forward := 4 + 4*sine(player.Rig.BobPhase*23+float32(bar)*2.1)
	fromX, fromY := r.headPoint(player, forward, -6.5)
	toX, toY := r.headPoint(player, forward, 6.5)
	r.batches.Solid.AddSegment(fromX, fromY, toX, toY, scaledTsukumo(0.8), robotPaletteSilver, 0.8*r.robotFrame.Alpha)
}

func (r *Renderer) queueTapeAbdomen(player *Player) {
	rig := &player.Rig
	frame := &r.robotFrame
	jointX, jointY := r.tsukumoPoint(player, -5, 0)
	centerX, centerY := r.shiftedAbdomenPoint(player, -13, 0)
	r.batches.Solid.AddSegment(jointX, jointY, centerX, centerY, scaledTsukumo(5), frame.Paint.Outline, frame.Alpha)
	r.queueCrateBand(player, -25, -5, 22*frame.Breath, frame.Paint.Outline, frame.Alpha)
	r.queueCrateBand(player, -24, -6, 20*frame.Breath, frame.Body, frame.Alpha)
	r.queueCrateBand(player, -22, -20, 14, robotPaletteRaisin, frame.Alpha)
	for _, side := range [2]float32{-4.6, 4.6} {
		reelX, reelY := r.shiftedAbdomenPoint(player, -11.5, side)
		spokeX, spokeY := rotateOffset(scaledTsukumo(2.4), 0, rig.BobPhase*3)
		r.batches.Solid.AddCircle(reelX, reelY, scaledTsukumo(3.6), robotPaletteRaisin, frame.Alpha)
		r.batches.Solid.AddCircle(reelX, reelY, scaledTsukumo(2.6), robotPaletteSilver, frame.Alpha)
		r.batches.Solid.AddSegment(reelX-spokeX, reelY-spokeY, reelX+spokeX, reelY+spokeY, scaledTsukumo(0.9), robotPaletteRaisin, frame.Alpha)
	}
}

func (r *Renderer) queueCableLeg(player *Player, leg int) {
	alpha := r.robotFrame.Alpha
	hipX, hipY, kneeX, kneeY, footX, footY := r.legPoints(player, leg)
	r.batches.Solid.AddCircle(footX+2, footY+4, scaledTsukumo(3.6), blackColor, 0.3*alpha)
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(2.8), robotPaletteSilver, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(2.8), robotPaletteSilver, alpha)
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(1.5), robotPaletteRaisin, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(1.5), robotPaletteRaisin, alpha)
	r.batches.Solid.AddCircle(kneeX, kneeY, scaledTsukumo(1.9), robotPaletteSilver, alpha)
	r.batches.Solid.AddCircle(kneeX, kneeY, scaledTsukumo(1), robotPaletteRaisin, alpha)
	r.queueWheel(footX, footY, player.Rig.Legs[leg].WheelSpin, alpha)
}

func (r *Renderer) queueAntennaArms(player *Player) {
	alpha := r.robotFrame.Alpha
	reach := 15 - 3*player.Rig.ArmRecoil
	for _, side := range [2]float32{-1, 1} {
		rootX, rootY := r.headPoint(player, 2, 6*side)
		tipX, tipY := r.headPoint(player, reach, 15*side)
		r.batches.Solid.AddSegment(rootX, rootY, tipX, tipY, scaledTsukumo(1.3), robotPaletteSilver, alpha)
		r.batches.Solid.AddCircle(tipX, tipY, scaledTsukumo(1.7), robotPaletteRaisin, alpha)
		r.batches.Solid.AddCircle(tipX, tipY, scaledTsukumo(1), robotPaletteYellow, alpha)
	}
}

// ===== Art parts =====

func (c *ArtCanvas) paintCableRearLegs(pose ArtPose, _ *RobotPaint) {
	for _, limb := range tsukumoRearLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{1.4, 1.4}, [2]float32{1.3, 1.3}
		c.paintLimb(pose, limb, robotFarStrut, robotFarStrut)
		c.paintWheel(pose, limb.Foot, 7.5)
	}
}

func (c *ArtCanvas) paintCableFrontLegs(pose ArtPose, paint *RobotPaint) {
	for _, limb := range tsukumoFrontLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{1.8, 1.8}, [2]float32{1.6, 1.6}
		c.paintLimb(pose, limb, robotStrut, paint.Dark)
		c.paintWheel(pose, limb.Foot, 11.25)
	}
}

func (c *ArtCanvas) paintTapeAbdomen(pose ArtPose, paint *RobotPaint) {
	c.paintBox(pose, 4, -56, 31, 15, paint.Shell)
	c.paintBox(pose, 4, -65, 19, 2, robotFlatRaisin)
	for _, side := range [2]float32{-1, 1} {
		c.paintFlatDisc(pose, 4+22*side, -58, 3.6, robotFlatRaisin)
		c.paintFlatDisc(pose, 4+22*side, -58, 1.8, robotFlatSilver)
	}
}

func (c *ArtCanvas) paintScreenHead(pose ArtPose, paint *RobotPaint) {
	c.paintCapsule(pose, 2, -30, 0, -36, 8, 8, paint.Dark)
	c.paintBox(pose, 0, -37, 24, 15, paint.Head)
	c.paintBox(pose, 0, -37, 19, 11, robotFlatRaisin)
}

func (c *ArtCanvas) paintPixelEyes(pose ArtPose, _ *RobotPaint) {
	for _, side := range [2]float32{-1, 1} {
		c.paintBox(pose, 7*side, -39, 3, 3.5, robotFlatLens)
	}
	c.paintBox(pose, 0, -31.5, 4, 1, robotFlatLens)
}

func (c *ArtCanvas) paintAntennaArms(pose ArtPose, _ *RobotPaint) {
	for _, side := range [2]float32{-1, 1} {
		c.paintCapsule(pose, 8*side, -51, 26*side, -78, 1, 1, robotStrut)
		c.paintFlatDisc(pose, 26*side, -78, 2.2, robotFlatLamp)
	}
}
