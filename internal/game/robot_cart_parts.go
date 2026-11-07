package game

// ===== Constants =====

const (
	cartBasketRear      = -25
	cartBasketFront     = -5
	cartBasketHalfWidth = 11
	cartBagDrift        = -0.6

	cartArtCenterX       = 6
	cartArtCageFrontTop  = -78
	cartArtCageFrontLow  = -36
	cartArtCageFrontHalf = 32
	cartArtCageBackTop   = -102
	cartArtCageBackLow   = -60
	cartArtCageBackHalf  = 26
	cartArtCageBar       = 0.8
	cartArtCoreCapY      = -90
	cartArtCoreBaseY     = -48
	cartArtCoreRadiusX   = 15
	cartArtCoreRadiusY   = 7
	cartCoreRingRadius   = 3.6
	cartCoreLensRadius   = 2.4
	cartCoreGlowRadius   = 7
)

var (
	cartGridForwards = [2]float32{-18.5, -11.5}
	cartGridSides    = [2]float32{-3.7, 3.7}
	cartArtBands     = [2]float32{-68, -56}
	cartArtLights    = [3]float32{-6, 6, 18}
)

// ===== Screen parts =====

func (r *Renderer) queueSeatHead(player *Player) {
	frame := &r.robotFrame
	r.queueHeadBand(player, -3, 11, 19*frame.Breath, frame.Paint.Outline)
	r.queueHeadBand(player, -2, 10, 17*frame.Breath, frame.Body)
	r.queueHeadBand(player, -2, 1, 17*frame.Breath, frame.Paint.Highlight)
}

func (r *Renderer) queueHeadlightEyes(player *Player) {
	rig := &player.Rig
	alpha := r.robotFrame.Alpha
	for _, side := range [2]float32{-5.2, 5.2} {
		lampX, lampY := r.headPoint(player, 8, side)
		r.batches.Glow.AddCircle(lampX, lampY, scaledTsukumo(7), robotPaletteYellow, 0.25*alpha)
		r.batches.Solid.AddCircle(lampX, lampY, scaledTsukumo(3.6), robotPaletteRaisin, alpha)
		r.batches.Solid.AddCircle(lampX, lampY, scaledTsukumo(2.8), robotPaletteYellow, alpha)
		r.batches.Solid.AddCircle(lampX+rig.LookX, lampY+rig.LookY, scaledTsukumo(1.2), tsukumoPupil, alpha)
	}
}

func (r *Renderer) queueBasketAbdomen(player *Player) {
	rig := &player.Rig
	frame := &r.robotFrame
	alpha := frame.Alpha
	jointX, jointY := r.tsukumoPoint(player, -5, 0)
	frontX, frontY := r.shiftedAbdomenPoint(player, cartBasketFront, 0)
	bagX, bagY := r.shiftedAbdomenPoint(player, -15, 0)
	r.batches.Solid.AddSegment(jointX, jointY, frontX, frontY, scaledTsukumo(3), robotPaletteSilver, alpha)
	r.queueCrateBand(player, cartBasketRear, cartBasketFront, 2*cartBasketHalfWidth, robotPaletteRaisin, 0.35*alpha)
	r.batches.Solid.AddCircle(bagX+rig.AbdomenOffsetX*cartBagDrift, bagY+rig.AbdomenOffsetY*cartBagDrift, scaledTsukumo(7), frame.Paint.Outline, alpha)
	r.batches.Solid.AddCircle(bagX+rig.AbdomenOffsetX*cartBagDrift, bagY+rig.AbdomenOffsetY*cartBagDrift, scaledTsukumo(5.8), frame.Body, alpha)
	coreX, coreY := bagX+rig.AbdomenOffsetX*cartBagDrift, bagY+rig.AbdomenOffsetY*cartBagDrift
	r.batches.Solid.AddCircle(coreX, coreY, scaledTsukumo(cartCoreRingRadius), frame.Paint.Outline, alpha)
	r.batches.Solid.AddCircle(coreX, coreY, scaledTsukumo(cartCoreLensRadius), robotPaletteLens, alpha)
	r.batches.Glow.AddCircle(coreX, coreY, scaledTsukumo(cartCoreGlowRadius), robotPaletteLens, 0.3*alpha)
	for _, side := range [4]float32{-cartBasketHalfWidth, cartGridSides[0], cartGridSides[1], cartBasketHalfWidth} {
		r.queueBasketWire(player, cartBasketRear, side, cartBasketFront, side)
	}
	for _, forward := range [4]float32{cartBasketRear, cartGridForwards[0], cartGridForwards[1], cartBasketFront} {
		r.queueBasketWire(player, forward, -cartBasketHalfWidth, forward, cartBasketHalfWidth)
	}
}

func (r *Renderer) queueBasketWire(player *Player, fromForward, fromSide, toForward, toSide float32) {
	fromX, fromY := r.shiftedAbdomenPoint(player, fromForward, fromSide)
	toX, toY := r.shiftedAbdomenPoint(player, toForward, toSide)
	r.batches.Solid.AddSegment(fromX, fromY, toX, toY, scaledTsukumo(1.1), robotPaletteSilver, r.robotFrame.Alpha)
}

func (r *Renderer) queueTubeLeg(player *Player, leg int) {
	frame := &r.robotFrame
	alpha, outline := frame.Alpha, frame.Paint.Outline
	hipX, hipY, kneeX, kneeY, footX, footY := r.legPoints(player, leg)
	r.batches.Solid.AddCircle(footX+2, footY+4, scaledTsukumo(3.6), blackColor, 0.3*alpha)
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(3.6), outline, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(3.6), outline, alpha)
	r.batches.Solid.AddSegment(hipX, hipY, kneeX, kneeY, scaledTsukumo(2), robotPaletteSilver, alpha)
	r.batches.Solid.AddSegment(kneeX, kneeY, footX, footY, scaledTsukumo(2), robotPaletteSilver, alpha)
	r.batches.Solid.AddCircle(kneeX, kneeY, scaledTsukumo(2.4), outline, alpha)
	r.batches.Solid.AddCircle(kneeX, kneeY, scaledTsukumo(1.3), frame.Body, alpha)
	r.queueWheel(footX, footY, player.Rig.Legs[leg].WheelSpin, alpha)
}

func (r *Renderer) queueHandleArms(player *Player) {
	frame := &r.robotFrame
	alpha, outline := frame.Alpha, frame.Paint.Outline
	reach := 16 - 3*player.Rig.ArmRecoil
	for _, side := range [2]float32{-1, 1} {
		shoulderX, shoulderY := r.headPoint(player, 4, 8*side)
		tipX, tipY := r.headPoint(player, reach, 8*side)
		r.batches.Solid.AddSegment(shoulderX, shoulderY, tipX, tipY, scaledTsukumo(2.6), outline, alpha)
		r.batches.Solid.AddSegment(shoulderX, shoulderY, tipX, tipY, scaledTsukumo(1.3), robotPaletteSilver, alpha)
	}
	barFromX, barFromY := r.headPoint(player, reach, -9)
	barToX, barToY := r.headPoint(player, reach, -3.5)
	r.batches.Solid.AddSegment(barFromX, barFromY, barToX, barToY, scaledTsukumo(2.6), frame.Body, alpha)
	barFromX, barFromY = r.headPoint(player, reach, 3.5)
	barToX, barToY = r.headPoint(player, reach, 9)
	r.batches.Solid.AddSegment(barFromX, barFromY, barToX, barToY, scaledTsukumo(2.6), frame.Body, alpha)
}

// ===== Art parts =====

func (c *ArtCanvas) paintTubeRearLegs(pose ArtPose, _ *RobotPaint) {
	for _, limb := range tsukumoRearLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{1.8, 1.8}, [2]float32{1.7, 1.7}
		c.paintLimb(pose, limb, robotFarStrut, robotFarStrut)
		c.paintWheel(pose, limb.Foot, 7.5)
	}
}

func (c *ArtCanvas) paintTubeFrontLegs(pose ArtPose, paint *RobotPaint) {
	for _, limb := range tsukumoFrontLimbs {
		limb.UpperWidth, limb.LowerWidth = [2]float32{2.2, 2.2}, [2]float32{2, 2}
		c.paintLimb(pose, limb, robotStrut, paint.Dark)
		c.paintWheel(pose, limb.Foot, 11.25)
	}
}

func (c *ArtCanvas) paintBasketAbdomen(pose ArtPose, paint *RobotPaint) {
	c.paintCageBase(pose, paint)
	c.paintCageBack(pose)
	c.paintCore(pose, paint)
	c.paintCageRoof(pose)
	c.paintCageSides(pose)
	c.paintCageFrontFrame(pose)
	c.paintCageMounts(pose, paint)
}

func (c *ArtCanvas) paintCageBase(pose ArtPose, paint *RobotPaint) {
	centerY := float32(cartArtCageFrontLow+cartArtCageBackLow) / 2
	halfDepth := float32(cartArtCageFrontLow-cartArtCageBackLow) / 2
	c.paintBox(pose, cartArtCenterX, centerY, cartArtCageBackHalf, halfDepth, paint.Dark)
}

func (c *ArtCanvas) paintCageMounts(pose ArtPose, paint *RobotPaint) {
	for _, side := range [2]float32{-1, 1} {
		c.paintBox(pose, cartArtCenterX+cartArtCageFrontHalf*side, cartArtCageFrontLow, 3.2, 2.4, paint.Dark)
		c.paintBox(pose, cartArtCenterX+cartArtCageBackHalf*side, cartArtCageBackLow, 2.6, 2, paint.Dark)
	}
}

func (c *ArtCanvas) paintCageBack(pose ArtPose) {
	left, right := float32(cartArtCenterX-cartArtCageBackHalf), float32(cartArtCenterX+cartArtCageBackHalf)
	c.paintCageBar(pose, left, cartArtCageBackTop, left, cartArtCageBackLow)
	c.paintCageBar(pose, right, cartArtCageBackTop, right, cartArtCageBackLow)
	c.paintCageBar(pose, left, cartArtCageBackLow, right, cartArtCageBackLow)
}

func (c *ArtCanvas) paintCore(pose ArtPose, paint *RobotPaint) {
	c.paintUprightCylinder(pose, cartArtCenterX, cartArtCoreCapY, cartArtCoreBaseY, cartArtCoreRadiusX, cartArtCoreRadiusY, paint.Shell)
	c.paintEllipsoid(pose, cartArtCenterX, cartArtCoreCapY, cartArtCoreRadiusX, cartArtCoreRadiusY, paint.Head)
	c.paintEllipsoid(pose, cartArtCenterX, cartArtCoreCapY, 6, 3, paint.Dark)
	c.paintEllipsoid(pose, cartArtCenterX, cartArtCoreCapY, 4, 2, robotFlatLens)
	for _, band := range cartArtBands {
		c.paintCapsule(pose, cartArtCenterX-cartArtCoreRadiusX+1, band, cartArtCenterX+cartArtCoreRadiusX-1, band, 1, 1, robotFlatLens)
	}
	for _, light := range cartArtLights {
		c.paintFlatDisc(pose, light, -62, 1.5, robotFlatLamp)
	}
}

func (c *ArtCanvas) paintUprightCylinder(pose ArtPose, x, capY, baseY, radiusX, radiusY float32, material Material) {
	centerX, capScreenY := pose.point(x, capY)
	_, baseScreenY := pose.point(x, baseY)
	scaledX, scaledY := max(0.5, radiusX*pose.Scale), max(0.5, radiusY*pose.Scale)
	for py := int(capScreenY); py <= int(baseScreenY+scaledY+1); py++ {
		for px := int(centerX - scaledX - 1); px <= int(centerX+scaledX+1); px++ {
			across := (float32(px) + 0.5 - centerX) / scaledX
			isAcross := across*across <= 1
			lowestY := baseScreenY + scaledY*sqrt(max(0, 1-across*across))
			c.shadePixelIf(px, py, across*0.98, 0, material, isAcross && float32(py)+0.5 <= lowestY)
		}
	}
}

func (c *ArtCanvas) paintCageRoof(pose ArtPose) {
	for step := float32(0); step <= 3; step++ {
		progress := step / 3
		c.paintCageBar(pose, cartArtCenterX-cartArtCageFrontHalf+2*cartArtCageFrontHalf*progress, cartArtCageFrontTop,
			cartArtCenterX-cartArtCageBackHalf+2*cartArtCageBackHalf*progress, cartArtCageBackTop)
		row := cartArtCageFrontTop + (cartArtCageBackTop-cartArtCageFrontTop)*progress
		half := cartArtCageFrontHalf + (cartArtCageBackHalf-cartArtCageFrontHalf)*progress
		c.paintCageBar(pose, cartArtCenterX-half, row, cartArtCenterX+half, row)
	}
}

func (c *ArtCanvas) paintCageSides(pose ArtPose) {
	for _, side := range [2]float32{-1, 1} {
		frontX, backX := cartArtCenterX+cartArtCageFrontHalf*side, cartArtCenterX+cartArtCageBackHalf*side
		for step := float32(1); step <= 2; step++ {
			progress := step / 2
			c.paintCageBar(pose, frontX, cartArtCageFrontTop+(cartArtCageFrontLow-cartArtCageFrontTop)*progress,
				backX, cartArtCageBackTop+(cartArtCageBackLow-cartArtCageBackTop)*progress)
		}
	}
}

func (c *ArtCanvas) paintCageFrontFrame(pose ArtPose) {
	left, right := float32(cartArtCenterX-cartArtCageFrontHalf), float32(cartArtCenterX+cartArtCageFrontHalf)
	c.paintCageBar(pose, left, cartArtCageFrontTop, left, cartArtCageFrontLow)
	c.paintCageBar(pose, right, cartArtCageFrontTop, right, cartArtCageFrontLow)
	c.paintCageBar(pose, left, cartArtCageFrontLow, right, cartArtCageFrontLow)
}

func (c *ArtCanvas) paintCageBar(pose ArtPose, fromX, fromY, toX, toY float32) {
	c.paintCapsule(pose, fromX, fromY, toX, toY, cartArtCageBar, cartArtCageBar, robotFlatSilver)
}

func (c *ArtCanvas) paintSeatHead(pose ArtPose, paint *RobotPaint) {
	c.paintEllipsoid(pose, 0, -41, 20, 14.5, robotFlatRaisin)
	c.paintEllipsoid(pose, 0, -41, 18, 12.5, paint.Head)
	c.paintBox(pose, 0, -32, 20, 9, paint.Head)
	c.paintBox(pose, 0, -41.5, 21, 1.6, robotFlatRaisin)
	c.paintEllipsoid(pose, 0, -49, 3.6, 2.6, robotFlatRaisin)
	c.paintFlatDisc(pose, 0, -49, 2.2, robotFlatLens)
}

func (c *ArtCanvas) paintHeadlightEyes(pose ArtPose, paint *RobotPaint) {
	for _, side := range [2]float32{-1, 1} {
		c.paintEllipsoid(pose, 10*side, -33, 6, 6, paint.Dark)
		c.paintFlatDisc(pose, 10*side, -33, 4.4, robotFlatLamp)
		c.paintFlatDisc(pose, 10*side+1, -33, 1.8, flatPupil)
	}
}

func (c *ArtCanvas) paintHandleArms(pose ArtPose, _ *RobotPaint) {
	for _, side := range [2]float32{-1, 1} {
		c.paintCapsule(pose, 18*side, -27, 24*side, -19, 1.5, 1.5, robotStrut)
		c.paintFlatDisc(pose, 24*side, -19, 2, robotFlatSilver)
	}
}
