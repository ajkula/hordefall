package game

// ===== Constants =====

const (
	kettleSteamPuffCount = 3
	kettleSteamRate      = 0.7

	kettleArtBodyX         = 4
	kettleArtBodyY         = -48
	kettleArtBodyRadiusX   = 33
	kettleArtBodyRadiusY   = 26
	kettleArtBailRise      = 9.0
	kettleArtBailSteps     = 32
	kettleArtBailAngleStep = 6.2831855 / kettleArtBailSteps
)

var (
	kettleStudOffsets    = [3]float32{-6, 0, 6}
	kettleArtStudColumns = [4]float32{-16, -3, 11, 24}
	kettleArtStudRows    = [3]float32{-62, -50, -38}
)

// ===== Screen parts =====

func (r *Renderer) queueLidHead(player *Player) {
	frame := &r.robotFrame
	headX, headY := r.headPoint(player, 4, 0)
	knobX, knobY := r.headPoint(player, 1, 0)
	r.batches.Solid.AddCircle(headX, headY, scaledTsukumo(8.6*frame.Breath), frame.Paint.Outline, frame.Alpha)
	r.batches.Solid.AddCircle(headX, headY, scaledTsukumo(7.4*frame.Breath), frame.Body, frame.Alpha)
	r.batches.Solid.AddCircle(knobX, knobY, scaledTsukumo(2.6), frame.Paint.Outline, frame.Alpha)
	r.batches.Solid.AddCircle(knobX, knobY, scaledTsukumo(1.3), robotPaletteSilver, frame.Alpha)
}

func (r *Renderer) queueDuoEyes(player *Player) {
	rig := &player.Rig
	lookForward, lookSide := rotateOffset(rig.LookX, rig.LookY, -rig.HeadAngle())
	for _, side := range [2]float32{-3.8, 3.8} {
		r.queueRoundEye(player, 9.5+lookForward*0.6, side+lookSide*0.8)
	}
}

func (r *Renderer) queueIronAbdomen(player *Player) {
	frame := &r.robotFrame
	alpha, breath := frame.Alpha, frame.Breath
	jointX, jointY := r.tsukumoPoint(player, -5, 0)
	centerX, centerY := r.shiftedAbdomenPoint(player, -13, 0)
	spoutFromX, spoutFromY := r.shiftedAbdomenPoint(player, -22, 0)
	spoutToX, spoutToY := r.shiftedAbdomenPoint(player, -29, 0)
	r.batches.Solid.AddSegment(jointX, jointY, centerX, centerY, scaledTsukumo(6), frame.Paint.Outline, alpha)
	r.batches.Solid.AddSegment(spoutFromX, spoutFromY, spoutToX, spoutToY, scaledTsukumo(5.4), frame.Paint.Outline, alpha)
	r.batches.Solid.AddSegment(spoutFromX, spoutFromY, spoutToX, spoutToY, scaledTsukumo(3.4), frame.Body, alpha)
	r.batches.Solid.AddCircle(centerX, centerY, scaledTsukumo(12.5*breath), frame.Paint.Outline, alpha)
	r.batches.Solid.AddCircle(centerX, centerY, scaledTsukumo(11.2*breath), frame.Body, alpha)
	for _, forward := range kettleStudOffsets {
		for _, side := range kettleStudOffsets {
			studX, studY := r.shiftedAbdomenPoint(player, -13+forward, side)
			r.batches.Solid.AddCircle(studX, studY, scaledTsukumo(1), frame.Paint.Outline, alpha)
		}
	}
	bailFromX, bailFromY := r.shiftedAbdomenPoint(player, -13, -12)
	bailToX, bailToY := r.shiftedAbdomenPoint(player, -13, 12)
	r.batches.Solid.AddSegment(bailFromX, bailFromY, bailToX, bailToY, scaledTsukumo(1.8), robotPaletteSilver, alpha)
	for puff := range kettleSteamPuffCount {
		r.queueKettlePuff(player, puff)
	}
}

func (r *Renderer) queueKettlePuff(player *Player, puff int) {
	rig := &player.Rig
	phase := rig.BobPhase*kettleSteamRate + float32(puff)/kettleSteamPuffCount
	age := phase - float32(int(phase))
	x, y := r.shiftedAbdomenPoint(player, -30-14*age, 3*sine(phase*6))
	opacity := (1 - age) * (0.25 + 0.6*rig.ArmRecoil) * r.robotFrame.Alpha
	r.batches.Solid.AddCircle(x, y, scaledTsukumo(1.5+3*age), robotPaletteSilver, opacity)
}

func (r *Renderer) queueStubLeg(player *Player, leg int) {
	frame := &r.robotFrame
	alpha, outline := frame.Alpha, frame.Paint.Outline
	hipX, hipY, kneeX, kneeY, footX, footY := r.legPoints(player, leg)
	r.batches.Solid.AddCircle(footX+2, footY+4, scaledTsukumo(3.6), blackColor, 0.3*alpha)
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(6), outline, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(6), outline, alpha)
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(4), frame.Body, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(4), frame.Body, alpha)
	r.batches.Solid.AddCircle(kneeX, kneeY, scaledTsukumo(3.6), outline, alpha)
	r.batches.Solid.AddCircle(kneeX, kneeY, scaledTsukumo(1.8), robotPaletteSilver, alpha)
	r.queueWheel(footX, footY, player.Rig.Legs[leg].WheelSpin, alpha)
}

func (r *Renderer) queueLoopArms(player *Player) {
	alpha, outline := r.robotFrame.Alpha, r.robotFrame.Paint.Outline
	pullBack := 3 * player.Rig.ArmRecoil
	for _, side := range [2]float32{-1, 1} {
		shoulderX, shoulderY := r.headPoint(player, 4, 6*side)
		ringX, ringY := r.headPoint(player, 11-pullBack, 10*side)
		r.batches.Solid.AddSegment(shoulderX, shoulderY, ringX, ringY, scaledTsukumo(2.2), outline, alpha)
		r.batches.Solid.AddCircle(ringX, ringY, scaledTsukumo(3.4), outline, alpha)
		r.batches.Solid.AddCircle(ringX, ringY, scaledTsukumo(2.5), robotPaletteSilver, alpha)
		r.batches.Solid.AddCircle(ringX, ringY, scaledTsukumo(1.2), outline, alpha)
	}
}

// ===== Art parts =====

func (c *ArtCanvas) paintStubRearLegs(pose ArtPose, _ *RobotPaint) {
	for _, limb := range tsukumoRearLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{4, 3.6}, [2]float32{3.6, 3}
		c.paintLimb(pose, limb, robotFarStrut, robotFarStrut)
		c.paintWheel(pose, limb.Foot, 7.5)
	}
}

func (c *ArtCanvas) paintStubFrontLegs(pose ArtPose, paint *RobotPaint) {
	for _, limb := range tsukumoFrontLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{5, 4.5}, [2]float32{4.5, 3.5}
		c.paintLimb(pose, limb, paint.Head, paint.Dark)
		c.paintWheel(pose, limb.Foot, 11.25)
	}
}

func (c *ArtCanvas) paintIronAbdomen(pose ArtPose, paint *RobotPaint) {
	c.paintCapsule(pose, 36, -48, 50, -58, 4.5, 3, paint.Shell)
	c.paintFlatDisc(pose, 54, -66, 2.4, robotFlatSilver)
	c.paintFlatDisc(pose, 58, -73, 1.6, robotFlatSilver)
	c.paintKettleBailRing(pose)
	c.paintEllipsoid(pose, kettleArtBodyX, kettleArtBodyY, kettleArtBodyRadiusX, kettleArtBodyRadiusY, paint.Shell)
	for _, column := range kettleArtStudColumns {
		for _, row := range kettleArtStudRows {
			c.paintFlatDisc(pose, column, row, 1.4, robotFlatRaisin)
		}
	}
	halfRise := kettleArtBailRise / 2 / float32(kettleArtBodyRadiusY)
	attachOffsetX := kettleArtBodyRadiusX * sqrt(1-halfRise*halfRise)
	for _, side := range [2]float32{-1, 1} {
		c.paintEllipsoid(pose, kettleArtBodyX+attachOffsetX*side, kettleArtBodyY-kettleArtBailRise/2, 2.4, 2.4, paint.Dark)
	}
}

func (c *ArtCanvas) paintKettleBailRing(pose ArtPose) {
	ringY := float32(kettleArtBodyY - kettleArtBailRise)
	for step := range kettleArtBailSteps {
		fromAngle := float32(step) * kettleArtBailAngleStep
		toAngle := fromAngle + kettleArtBailAngleStep
		c.paintCapsule(pose,
			kettleArtBodyX+kettleArtBodyRadiusX*cosine(fromAngle), ringY+kettleArtBodyRadiusY*sine(fromAngle),
			kettleArtBodyX+kettleArtBodyRadiusX*cosine(toAngle), ringY+kettleArtBodyRadiusY*sine(toAngle),
			1.6, 1.6, robotStrut)
	}
}

func (c *ArtCanvas) paintLidHead(pose ArtPose, paint *RobotPaint) {
	c.paintCapsule(pose, 2, -30, 0, -36, 8, 8, paint.Dark)
	c.paintEllipsoid(pose, 0, -36, 21, 12, paint.Head)
	c.paintEllipsoid(pose, 0, -49, 3.5, 3, paint.Dark)
}

func (c *ArtCanvas) paintDuoEyes(pose ArtPose, _ *RobotPaint) {
	for _, side := range [2]float32{-1, 1} {
		c.paintEye(pose, 8*side, -35, 5.5)
	}
}

func (c *ArtCanvas) paintLoopArms(pose ArtPose, paint *RobotPaint) {
	for _, side := range [2]float32{-1, 1} {
		c.paintCapsule(pose, 14*side, -28, 21*side, -24, 1.6, 1.6, paint.Dark)
		c.paintEllipsoid(pose, 22*side, -23, 3.6, 3.6, robotStrut)
		c.paintFlatDisc(pose, 22*side, -23, 1.7, robotFlatRaisin)
	}
}
