package main

// ===== Types =====

type TachikomaLeg struct {
	FootX     float32
	FootY     float32
	VelocityX float32
	VelocityY float32
	WheelSpin float32
	TrailX    float32
	TrailY    float32
}

type TachikomaLegLayout struct {
	HipForward  float32
	HipSide     float32
	RestForward float32
	RestSide    float32
}

type TachikomaRig struct {
	Heading           float32
	Legs              [tachikomaLegCount]TachikomaLeg
	AbdomenOffsetX    float32
	AbdomenOffsetY    float32
	AbdomenVelocityX  float32
	AbdomenVelocityY  float32
	PreviousVelocityX float32
	PreviousVelocityY float32
	LookX             float32
	LookY             float32
	GlanceX           float32
	GlanceY           float32
	GlanceSeconds     float32
	IdleSeconds       float32
	BobPhase          float32
	ArmRecoil         float32
	TuckAmount        float32
	SkidAmount        float32
	HeadServo         AngularServo
	MuzzleX           float32
	MuzzleY           float32
	IsPlaced          bool
}

// ===== Constants =====

const (
	tachikomaLegCount         = 4
	tachikomaScale            = 1.3
	tachikomaWheelRadius      = 3.4
	tachikomaThighLength      = 10 * tachikomaScale
	tachikomaShinLength       = 12 * tachikomaScale
	tachikomaFootStiffness    = 320
	tachikomaFootDamping      = 22
	tachikomaFootLead         = 0.05
	tachikomaAbdomenStiffness = 150
	tachikomaAbdomenDamping   = 13
	tachikomaAbdomenInertia   = 0.022
	tachikomaAbdomenMaxOffset = 6
	tachikomaTurnRate         = 11
	tachikomaDashTuck         = 0.45
	tachikomaIdleBeforeGlance = 1.2
	tachikomaGazeSharpness    = 9
	tachikomaRecoilRecovery   = 6
	tachikomaHeadLimit        = 1.9
	tachikomaHeadStiffness    = 300
	tachikomaHeadDamping      = 17
	tachikomaNeckForward      = 2.5
	tachikomaMuzzleForward    = 17
	tachikomaAbdomenCounter   = 0.22
)

var tachikomaLegLayouts = [tachikomaLegCount]TachikomaLegLayout{
	{HipForward: 4, HipSide: -6, RestForward: 15, RestSide: -16},
	{HipForward: 4, HipSide: 6, RestForward: 15, RestSide: 16},
	{HipForward: -7, HipSide: -6, RestForward: -15, RestSide: -17},
	{HipForward: -7, HipSide: 6, RestForward: -15, RestSide: 17},
}

// ===== Public API =====

func (rig *TachikomaRig) LocalToWorld(originX, originY, forward, side float32) (float32, float32) {
	offsetX, offsetY := rotateOffset(forward*tachikomaScale, side*tachikomaScale, rig.Heading)
	return originX + offsetX, originY + offsetY
}

func (rig *TachikomaRig) WorldToLocalDirection(x, y float32) (float32, float32) {
	cosineHeading, sineHeading := cosine(rig.Heading), sine(rig.Heading)
	return x*cosineHeading + y*sineHeading, -x*sineHeading + y*cosineHeading
}

func (rig *TachikomaRig) HeadAngle() float32 {
	return rig.Heading + rig.HeadServo.Angle
}

func (rig *TachikomaRig) HeadToWorld(originX, originY, forward, side float32) (float32, float32) {
	pivotX, pivotY := rig.LocalToWorld(originX, originY, tachikomaNeckForward, 0)
	offsetX, offsetY := rotateOffset(forward*tachikomaScale, side*tachikomaScale, rig.HeadAngle())
	return pivotX + offsetX, pivotY + offsetY
}

func (rig *TachikomaRig) AbdomenToWorld(originX, originY, forward, side float32) (float32, float32) {
	jointX, jointY := rig.LocalToWorld(originX, originY, -5, 0)
	offsetX, offsetY := rotateOffset((forward+5)*tachikomaScale, side*tachikomaScale, rig.Heading-rig.HeadServo.Angle*tachikomaAbdomenCounter)
	return jointX + offsetX, jointY + offsetY
}

func (rig *TachikomaRig) KneePosition(hipX, hipY, footX, footY, centerX, centerY float32) (float32, float32) {
	deltaX, deltaY := footX-hipX, footY-hipY
	distance := clamp(length(deltaX, deltaY), 1, tachikomaThighLength+tachikomaShinLength-0.5)
	alongX, alongY := deltaX/distance, deltaY/distance
	along := (tachikomaThighLength*tachikomaThighLength - tachikomaShinLength*tachikomaShinLength + distance*distance) / (2 * distance)
	height := sqrt(max(0, tachikomaThighLength*tachikomaThighLength-along*along))
	baseX, baseY := hipX+alongX*along, hipY+alongY*along
	firstX, firstY := baseX-alongY*height, baseY+alongX*height
	secondX, secondY := baseX+alongY*height, baseY-alongX*height
	weight := boolToFloat(distanceSquared(firstX, firstY, centerX, centerY) >= distanceSquared(secondX, secondY, centerX, centerY))
	return secondX + (firstX-secondX)*weight, secondY + (firstY-secondY)*weight
}

// ===== Internal =====

func (g *Game) updateTachikoma(deltaSeconds float32) {
	player := g.player
	rig := &player.Rig
	if !rig.IsPlaced {
		g.placeTachikoma(rig)
		return
	}
	speed := length(player.VelocityX, player.VelocityY)
	isMoving := speed > 5
	g.steerTachikoma(rig, isMoving, deltaSeconds)
	isDashing := player.DashSeconds > 0
	rig.TuckAmount += (boolToFloat(isDashing) - rig.TuckAmount) * min(1, deltaSeconds*14)
	accelerationX, accelerationY := g.measureAcceleration(rig, deltaSeconds)
	g.swingAbdomen(rig, accelerationX, accelerationY, deltaSeconds)
	for leg := range tachikomaLegCount {
		g.rollLeg(rig, leg, deltaSeconds)
	}
	g.updateSkid(rig, accelerationX, accelerationY)
	rig.BobPhase += deltaSeconds * (2.2 + speed*0.045)
	rig.ArmRecoil = max(0, rig.ArmRecoil-deltaSeconds*tachikomaRecoilRecovery)
	g.updateGaze(rig, isMoving, deltaSeconds)
}

func (g *Game) steerTachikoma(rig *TachikomaRig, isMoving bool, deltaSeconds float32) {
	player := g.player
	aimAngle := atan2(player.AimY, player.AimX)
	relativeAim := angleDifference(rig.Heading, aimAngle)
	excess := relativeAim - clamp(relativeAim, -tachikomaHeadLimit, tachikomaHeadLimit)
	hasExcess := boolToFloat(abs(excess) > 0.01)
	travelTurn := angleDifference(rig.Heading, atan2(player.VelocityY, player.VelocityX)) * boolToFloat(isMoving)
	bodyTarget := rig.Heading + travelTurn*(1-hasExcess) + excess*hasExcess
	rig.Heading = turnToward(rig.Heading, bodyTarget, tachikomaTurnRate*deltaSeconds)
	headTarget := clamp(angleDifference(rig.Heading, aimAngle), -tachikomaHeadLimit, tachikomaHeadLimit)
	rig.HeadServo.Drive(headTarget, tachikomaHeadStiffness, tachikomaHeadDamping, deltaSeconds)
	rig.HeadServo.Angle = clamp(rig.HeadServo.Angle, -tachikomaHeadLimit-0.15, tachikomaHeadLimit+0.15)
	rig.MuzzleX, rig.MuzzleY = rig.HeadToWorld(player.X, player.Y, tachikomaMuzzleForward-4*rig.ArmRecoil, 0)
}

func (g *Game) placeTachikoma(rig *TachikomaRig) {
	player := g.player
	for leg := range tachikomaLegCount {
		layout := &tachikomaLegLayouts[leg]
		rig.Legs[leg].FootX, rig.Legs[leg].FootY = rig.LocalToWorld(player.X, player.Y, layout.RestForward, layout.RestSide)
		rig.Legs[leg].TrailX, rig.Legs[leg].TrailY = rig.Legs[leg].FootX, rig.Legs[leg].FootY
	}
	rig.LookX, rig.LookY = player.AimX, player.AimY
	rig.MuzzleX, rig.MuzzleY = player.X, player.Y
	rig.IsPlaced = true
}

func (g *Game) measureAcceleration(rig *TachikomaRig, deltaSeconds float32) (float32, float32) {
	player := g.player
	accelerationX := (player.VelocityX - rig.PreviousVelocityX) / deltaSeconds
	accelerationY := (player.VelocityY - rig.PreviousVelocityY) / deltaSeconds
	rig.PreviousVelocityX, rig.PreviousVelocityY = player.VelocityX, player.VelocityY
	return accelerationX, accelerationY
}

func (g *Game) swingAbdomen(rig *TachikomaRig, accelerationX, accelerationY, deltaSeconds float32) {
	forceX := -accelerationX*tachikomaAbdomenInertia - rig.AbdomenOffsetX*tachikomaAbdomenStiffness - rig.AbdomenVelocityX*tachikomaAbdomenDamping
	forceY := -accelerationY*tachikomaAbdomenInertia - rig.AbdomenOffsetY*tachikomaAbdomenStiffness - rig.AbdomenVelocityY*tachikomaAbdomenDamping
	rig.AbdomenVelocityX += forceX * deltaSeconds
	rig.AbdomenVelocityY += forceY * deltaSeconds
	rig.AbdomenOffsetX = clamp(rig.AbdomenOffsetX+rig.AbdomenVelocityX*deltaSeconds, -tachikomaAbdomenMaxOffset, tachikomaAbdomenMaxOffset)
	rig.AbdomenOffsetY = clamp(rig.AbdomenOffsetY+rig.AbdomenVelocityY*deltaSeconds, -tachikomaAbdomenMaxOffset, tachikomaAbdomenMaxOffset)
}

func (g *Game) rollLeg(rig *TachikomaRig, leg int, deltaSeconds float32) {
	player := g.player
	layout := &tachikomaLegLayouts[leg]
	state := &rig.Legs[leg]
	reach := 1 - tachikomaDashTuck*rig.TuckAmount
	restX, restY := rig.LocalToWorld(player.X, player.Y, layout.RestForward*reach, layout.RestSide*reach)
	targetX := restX + player.VelocityX*tachikomaFootLead
	targetY := restY + player.VelocityY*tachikomaFootLead
	state.VelocityX += ((targetX-state.FootX)*tachikomaFootStiffness - state.VelocityX*tachikomaFootDamping) * deltaSeconds
	state.VelocityY += ((targetY-state.FootY)*tachikomaFootStiffness - state.VelocityY*tachikomaFootDamping) * deltaSeconds
	state.FootX += state.VelocityX * deltaSeconds
	state.FootY += state.VelocityY * deltaSeconds
	hipX, hipY := rig.LocalToWorld(player.X, player.Y, layout.HipForward, layout.HipSide)
	deltaX, deltaY := state.FootX-hipX, state.FootY-hipY
	limit := min(1, (tachikomaThighLength+tachikomaShinLength-0.5)/max(0.001, length(deltaX, deltaY)))
	state.FootX, state.FootY = hipX+deltaX*limit, hipY+deltaY*limit
	state.WheelSpin += length(state.VelocityX, state.VelocityY) * deltaSeconds * 0.35
}

func (g *Game) updateGaze(rig *TachikomaRig, isMoving bool, deltaSeconds float32) {
	player := g.player
	isIdle := !isMoving && !player.IsFiring
	rig.IdleSeconds = (rig.IdleSeconds + deltaSeconds) * boolToFloat(isIdle)
	rig.GlanceSeconds -= deltaSeconds
	g.pickGlanceIfDue(rig)
	glanceWeight := boolToFloat(rig.IdleSeconds > tachikomaIdleBeforeGlance)
	targetX := player.AimX + (rig.GlanceX-player.AimX)*glanceWeight
	targetY := player.AimY + (rig.GlanceY-player.AimY)*glanceWeight
	sharpness := min(1, deltaSeconds*tachikomaGazeSharpness)
	rig.LookX, rig.LookY = normalize(rig.LookX+(targetX-rig.LookX)*sharpness, rig.LookY+(targetY-rig.LookY)*sharpness)
}

func (g *Game) pickGlanceIfDue(rig *TachikomaRig) {
	if rig.GlanceSeconds > 0 {
		return
	}
	angle := g.random.Angle()
	rig.GlanceX, rig.GlanceY = cosine(angle), sine(angle)
	rig.GlanceSeconds = g.random.Between(0.5, 1.6)
}
