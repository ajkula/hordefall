package game

// ===== Constants =====

const worksiteBoxEdge = 0.75

var (
	robotPaletteYellow = [3]float32{0.992, 0.961, 0}
	robotPaletteRaisin = [3]float32{0.153, 0.161, 0.196}
	robotPaletteSilver = [3]float32{0.82, 0.773, 0.753}
	robotPaletteLens   = [3]float32{0.216, 0.922, 0.953}

	robotFlatSilver = Material{Base: robotPaletteSilver}
	robotFlatRaisin = Material{Base: robotPaletteRaisin}
	robotFlatLens   = Material{Base: robotPaletteLens}
	robotFlatLamp   = Material{Base: robotPaletteYellow}
	robotStrut      = Material{Base: robotPaletteSilver, Rim: redRim, RimSide: 1, IsShaded: true}
	robotFarStrut   = Material{Base: [3]float32{0.5, 0.47, 0.46}, IsShaded: true}
)

// ===== Screen parts =====

func (r *Renderer) queueTurretHead(player *Player) {
	frame := &r.robotFrame
	rearX, rearY := r.headPoint(player, -3, 0)
	frontX, frontY := r.headPoint(player, 11, 0)
	innerRearX, innerRearY := r.headPoint(player, -2, 0)
	innerFrontX, innerFrontY := r.headPoint(player, 10, 0)
	noseX, noseY := r.headPoint(player, 13, 0)
	innerNoseX, innerNoseY := r.headPoint(player, 12, 0)
	r.batches.Solid.AddSegment(frontX, frontY, noseX, noseY, scaledTsukumo(13), frame.Paint.Outline, frame.Alpha)
	r.batches.Solid.AddSegment(rearX, rearY, frontX, frontY, scaledTsukumo(19*frame.Breath), frame.Paint.Outline, frame.Alpha)
	r.batches.Solid.AddSegment(innerFrontX, innerFrontY, innerNoseX, innerNoseY, scaledTsukumo(11), frame.Body, frame.Alpha)
	r.batches.Solid.AddSegment(innerRearX, innerRearY, innerFrontX, innerFrontY, scaledTsukumo(17*frame.Breath), frame.Body, frame.Alpha)
	stripeFromX, stripeFromY := r.headPoint(player, -1, -5.5)
	stripeToX, stripeToY := r.headPoint(player, 6, -5.5)
	r.batches.Solid.AddSegment(stripeFromX, stripeFromY, stripeToX, stripeToY, scaledTsukumo(1.6), frame.Paint.Highlight, 0.6*frame.Alpha)
}

func (r *Renderer) queueVisorEyes(player *Player) {
	rig := &player.Rig
	alpha := r.robotFrame.Alpha
	leftX, leftY := r.headPoint(player, 9, -6.5)
	rightX, rightY := r.headPoint(player, 9, 6.5)
	r.batches.Solid.AddSegment(leftX, leftY, rightX, rightY, scaledTsukumo(3.8), robotPaletteRaisin, alpha)
	_, lookSide := rotateOffset(rig.LookX, rig.LookY, -rig.HeadAngle())
	lensX, lensY := r.headPoint(player, 9, clamp(lookSide*4, -5, 5))
	r.batches.Solid.AddCircle(lensX, lensY, scaledTsukumo(1.5), robotPaletteLens, alpha)
	r.batches.Glow.AddCircle(lensX, lensY, scaledTsukumo(4), robotPaletteLens, 0.35*alpha)
}

func (r *Renderer) queueCrateAbdomen(player *Player) {
	rig := &player.Rig
	frame := &r.robotFrame
	shiftX, shiftY := rig.AbdomenOffsetX, rig.AbdomenOffsetY
	jointX, jointY := r.tsukumoPoint(player, -5, 0)
	centerX, centerY := r.abdomenPoint(player, -13, 0)
	r.batches.Solid.AddSegment(jointX, jointY, centerX+shiftX, centerY+shiftY, scaledTsukumo(7), frame.Paint.Outline, frame.Alpha)
	r.queueCrateBand(player, -24, -5, 21*frame.Breath, frame.Paint.Outline, frame.Alpha)
	r.queueCrateBand(player, -23, -6, 19*frame.Breath, frame.Body, frame.Alpha)
	for _, forward := range [3]float32{-20, -14.5, -9} {
		r.queueCrateBand(player, forward-1, forward+1, 19*frame.Breath, robotPaletteRaisin, frame.Alpha)
	}
	r.batches.Solid.AddCircle(jointX, jointY, scaledTsukumo(3.2), robotPaletteSilver, frame.Alpha)
}

func (r *Renderer) queueCrateBand(player *Player, fromForward, toForward, width float32, tint [3]float32, alpha float32) {
	rig := &player.Rig
	fromX, fromY := r.abdomenPoint(player, fromForward, 0)
	toX, toY := r.abdomenPoint(player, toForward, 0)
	r.batches.Solid.AddSegment(fromX+rig.AbdomenOffsetX, fromY+rig.AbdomenOffsetY, toX+rig.AbdomenOffsetX, toY+rig.AbdomenOffsetY, scaledTsukumo(width), tint, alpha)
}

func (r *Renderer) queueStrutLeg(player *Player, leg int) {
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
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(4.4), outline, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(4.4), outline, alpha)
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(2.4), robotPaletteSilver, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(2.4), robotPaletteSilver, alpha)
	r.batches.Solid.AddCircle(kneeX, kneeY, scaledTsukumo(3), outline, alpha)
	r.batches.Solid.AddCircle(kneeX, kneeY, scaledTsukumo(1.4), robotPaletteSilver, alpha)
	r.queueWheel(footX, footY, state.WheelSpin, alpha)
}

func (r *Renderer) queueLampArms(player *Player) {
	alpha := r.robotFrame.Alpha
	for _, side := range [2]float32{-11, 11} {
		lampX, lampY := r.headPoint(player, 7, side)
		r.batches.Solid.AddCircle(lampX, lampY, scaledTsukumo(2.4), robotPaletteRaisin, alpha)
		r.batches.Solid.AddCircle(lampX, lampY, scaledTsukumo(1.4), robotPaletteYellow, alpha)
	}
}

// ===== Art parts =====

func (c *ArtCanvas) paintStrutRearLegs(pose ArtPose, _ *RobotPaint) {
	for _, limb := range tsukumoRearLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{2, 2}, [2]float32{1.8, 1.8}
		c.paintLimb(pose, limb, robotFarStrut, robotFarStrut)
		c.paintWheel(pose, limb.Foot, 7.5)
	}
}

func (c *ArtCanvas) paintStrutFrontLegs(pose ArtPose, paint *RobotPaint) {
	for _, limb := range tsukumoFrontLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{2.4, 2.4}, [2]float32{2.2, 2.2}
		c.paintLimb(pose, limb, robotStrut, paint.Dark)
		c.paintWheel(pose, limb.Foot, 11.25)
	}
}

func (c *ArtCanvas) paintCrateAbdomen(pose ArtPose, paint *RobotPaint) {
	c.paintBox(pose, 4, -46, 34, 22, paint.Shell)
	for _, x := range [4]float32{-20, -6, 12, 28} {
		c.paintBox(pose, x, -60, 3, 7, robotFlatRaisin)
	}
}

func (c *ArtCanvas) paintTurretHead(pose ArtPose, paint *RobotPaint) {
	c.paintCapsule(pose, 2, -30, 0, -36, 8, 8, paint.Dark)
	c.paintBox(pose, 0, -38, 23, 12, paint.Head)
}

func (c *ArtCanvas) paintVisorEyes(pose ArtPose, _ *RobotPaint) {
	c.paintBox(pose, 0, -37, 17, 4.5, robotFlatRaisin)
	c.paintFlatDisc(pose, 5, -37, 2.8, robotFlatLens)
}

func (c *ArtCanvas) paintLampArms(pose ArtPose, paint *RobotPaint) {
	for _, side := range [2]float32{-1, 1} {
		c.paintEllipsoid(pose, 20*side, -24, 3.4, 3.4, paint.Dark)
		c.paintFlatDisc(pose, 20*side, -24, 2, robotFlatLamp)
	}
}

func (c *ArtCanvas) paintBox(pose ArtPose, x, y, halfWidth, halfHeight float32, material Material) {
	centerX, centerY := pose.point(x, y)
	scaledX, scaledY := max(0.5, halfWidth*pose.Scale), max(0.5, halfHeight*pose.Scale)
	for py := int(centerY - scaledY - 1); py <= int(centerY+scaledY+1); py++ {
		for px := int(centerX - scaledX - 1); px <= int(centerX+scaledX+1); px++ {
			acrossX, acrossY := (float32(px)+0.5-centerX)/scaledX, (float32(py)+0.5-centerY)/scaledY
			isInside := acrossX*acrossX <= 1 && acrossY*acrossY <= 1
			c.shadePixelIf(px, py, boxEdgeNormal(acrossX), boxEdgeNormal(acrossY), material, isInside)
		}
	}
}

func boxEdgeNormal(across float32) float32 {
	sign := 1 - 2*boolToFloat(across < 0)
	distance := max(0, across*sign-worksiteBoxEdge) / (1 - worksiteBoxEdge)
	return sign * distance * 0.7
}
