package game

import (
	"github.com/ajkula/ajkurig/fx"
	"github.com/ajkula/ajkurig/procanim"
)

// ===== Types =====

type TsukumoLeg struct {
	FootX     float32
	FootY     float32
	VelocityX float32
	VelocityY float32
	WheelSpin float32
	Skid      fx.SkidTracker
}

type TsukumoLegLayout struct {
	HipForward  float32
	HipSide     float32
	RestForward float32
	RestSide    float32
}

type TsukumoRig struct {
	Heading           float32
	Legs              [tsukumoLegCount]TsukumoLeg
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
	HeadServo         procanim.Servo
	MuzzleX           float32
	MuzzleY           float32
}

// ===== Constants =====

const (
	tsukumoLegCount         = 4
	tsukumoScale            = 1.3
	tsukumoWheelRadius      = 3.4
	tsukumoThighLength      = 10 * tsukumoScale
	tsukumoShinLength       = 12 * tsukumoScale
	tsukumoFootStiffness    = 320
	tsukumoFootDamping      = 22
	tsukumoFootLead         = 0.05
	tsukumoAbdomenStiffness = 150
	tsukumoAbdomenDamping   = 13
	tsukumoAbdomenInertia   = 0.022
	tsukumoAbdomenMaxOffset = 6
	tsukumoTurnRate         = 11
	tsukumoDashTuck         = 0.45
	tsukumoIdleBeforeGlance = 1.2
	tsukumoGazeSharpness    = 9
	tsukumoRecoilRecovery   = 6
	tsukumoHeadLimit        = 1.9
	tsukumoHeadStiffness    = 300
	tsukumoHeadDamping      = 17
	tsukumoNeckForward      = 2.5
	tsukumoMuzzleForward    = 17
	tsukumoAbdomenCounter   = 0.22
)

var tsukumoLegLayouts = [tsukumoLegCount]TsukumoLegLayout{
	{HipForward: 4, HipSide: -6, RestForward: 15, RestSide: -16},
	{HipForward: 4, HipSide: 6, RestForward: 15, RestSide: 16},
	{HipForward: -7, HipSide: -6, RestForward: -15, RestSide: -17},
	{HipForward: -7, HipSide: 6, RestForward: -15, RestSide: 17},
}

// ===== Public API =====

func (rig *TsukumoRig) LocalToWorld(originX, originY, forward, side float32) (float32, float32) {
	offsetX, offsetY := rotateOffset(forward*tsukumoScale, side*tsukumoScale, rig.Heading)
	return originX + offsetX, originY + offsetY
}

func (rig *TsukumoRig) HeadAngle() float32 {
	return rig.Heading + rig.HeadServo.Angle
}

func (rig *TsukumoRig) HeadToWorld(originX, originY, forward, side float32) (float32, float32) {
	pivotX, pivotY := rig.LocalToWorld(originX, originY, tsukumoNeckForward, 0)
	offsetX, offsetY := rotateOffset(forward*tsukumoScale, side*tsukumoScale, rig.HeadAngle())
	return pivotX + offsetX, pivotY + offsetY
}

func (rig *TsukumoRig) AbdomenToWorld(originX, originY, forward, side float32) (float32, float32) {
	jointX, jointY := rig.LocalToWorld(originX, originY, -5, 0)
	offsetX, offsetY := rotateOffset((forward+5)*tsukumoScale, side*tsukumoScale, rig.Heading-rig.HeadServo.Angle*tsukumoAbdomenCounter)
	return jointX + offsetX, jointY + offsetY
}

func (rig *TsukumoRig) KneePosition(hipX, hipY, footX, footY, centerX, centerY float32) (float32, float32) {
	return procanim.SolveKneeAwayFrom(hipX, hipY, footX, footY, tsukumoThighLength, tsukumoShinLength, centerX, centerY)
}

// ===== Internal =====

func (g *Game) updateTsukumo(deltaSeconds float32) {
	player := g.player
	rig := &player.Rig
	speed := length(player.VelocityX, player.VelocityY)
	isMoving := speed > 5
	g.steerTsukumo(rig, isMoving, deltaSeconds)
	isDashing := player.DashSeconds > 0
	rig.TuckAmount += (boolToFloat(isDashing) - rig.TuckAmount) * min(1, deltaSeconds*14)
	accelerationX, accelerationY := g.measureAcceleration(rig, deltaSeconds)
	g.swingAbdomen(rig, accelerationX, accelerationY, deltaSeconds)
	for leg := range tsukumoLegCount {
		g.rollLeg(rig, leg, deltaSeconds)
	}
	g.updateSkid(rig, accelerationX, accelerationY)
	rig.BobPhase += deltaSeconds * (2.2 + speed*0.045)
	rig.ArmRecoil = max(0, rig.ArmRecoil-deltaSeconds*tsukumoRecoilRecovery)
	g.updateGaze(rig, isMoving, deltaSeconds)
}

func (g *Game) steerTsukumo(rig *TsukumoRig, isMoving bool, deltaSeconds float32) {
	player := g.player
	aimAngle := atan2(player.AimY, player.AimX)
	relativeAim := angleDifference(rig.Heading, aimAngle)
	excess := relativeAim - clamp(relativeAim, -tsukumoHeadLimit, tsukumoHeadLimit)
	hasExcess := boolToFloat(abs(excess) > 0.01)
	travelTurn := angleDifference(rig.Heading, atan2(player.VelocityY, player.VelocityX)) * boolToFloat(isMoving)
	bodyTarget := rig.Heading + travelTurn*(1-hasExcess) + excess*hasExcess
	rig.Heading = procanim.TurnToward(rig.Heading, bodyTarget, tsukumoTurnRate*deltaSeconds)
	headTarget := clamp(angleDifference(rig.Heading, aimAngle), -tsukumoHeadLimit, tsukumoHeadLimit)
	rig.HeadServo.Drive(headTarget, tsukumoHeadStiffness, tsukumoHeadDamping, deltaSeconds)
	rig.HeadServo.Angle = clamp(rig.HeadServo.Angle, -tsukumoHeadLimit-0.15, tsukumoHeadLimit+0.15)
	rig.MuzzleX, rig.MuzzleY = rig.HeadToWorld(player.X, player.Y, tsukumoMuzzleForward-4*rig.ArmRecoil, 0)
}

func (player *Player) placeRig() {
	rig := &player.Rig
	for leg := range tsukumoLegCount {
		layout := &tsukumoLegLayouts[leg]
		rig.Legs[leg].FootX, rig.Legs[leg].FootY = rig.LocalToWorld(player.X, player.Y, layout.RestForward, layout.RestSide)
		rig.Legs[leg].Skid.TrailX, rig.Legs[leg].Skid.TrailY = rig.Legs[leg].FootX, rig.Legs[leg].FootY
	}
	rig.LookX, rig.LookY = player.AimX, player.AimY
	rig.MuzzleX, rig.MuzzleY = player.X, player.Y
}

func (g *Game) measureAcceleration(rig *TsukumoRig, deltaSeconds float32) (float32, float32) {
	player := g.player
	accelerationX := (player.VelocityX - rig.PreviousVelocityX) / deltaSeconds
	accelerationY := (player.VelocityY - rig.PreviousVelocityY) / deltaSeconds
	rig.PreviousVelocityX, rig.PreviousVelocityY = player.VelocityX, player.VelocityY
	return accelerationX, accelerationY
}

func (g *Game) swingAbdomen(rig *TsukumoRig, accelerationX, accelerationY, deltaSeconds float32) {
	forceX := -accelerationX*tsukumoAbdomenInertia - rig.AbdomenOffsetX*tsukumoAbdomenStiffness - rig.AbdomenVelocityX*tsukumoAbdomenDamping
	forceY := -accelerationY*tsukumoAbdomenInertia - rig.AbdomenOffsetY*tsukumoAbdomenStiffness - rig.AbdomenVelocityY*tsukumoAbdomenDamping
	rig.AbdomenVelocityX += forceX * deltaSeconds
	rig.AbdomenVelocityY += forceY * deltaSeconds
	rig.AbdomenOffsetX = clamp(rig.AbdomenOffsetX+rig.AbdomenVelocityX*deltaSeconds, -tsukumoAbdomenMaxOffset, tsukumoAbdomenMaxOffset)
	rig.AbdomenOffsetY = clamp(rig.AbdomenOffsetY+rig.AbdomenVelocityY*deltaSeconds, -tsukumoAbdomenMaxOffset, tsukumoAbdomenMaxOffset)
}

func (g *Game) rollLeg(rig *TsukumoRig, leg int, deltaSeconds float32) {
	player := g.player
	layout := &tsukumoLegLayouts[leg]
	state := &rig.Legs[leg]
	reach := 1 - tsukumoDashTuck*rig.TuckAmount
	restX, restY := rig.LocalToWorld(player.X, player.Y, layout.RestForward*reach, layout.RestSide*reach)
	targetX := restX + player.VelocityX*tsukumoFootLead
	targetY := restY + player.VelocityY*tsukumoFootLead
	state.VelocityX += ((targetX-state.FootX)*tsukumoFootStiffness - state.VelocityX*tsukumoFootDamping) * deltaSeconds
	state.VelocityY += ((targetY-state.FootY)*tsukumoFootStiffness - state.VelocityY*tsukumoFootDamping) * deltaSeconds
	state.FootX += state.VelocityX * deltaSeconds
	state.FootY += state.VelocityY * deltaSeconds
	hipX, hipY := rig.LocalToWorld(player.X, player.Y, layout.HipForward, layout.HipSide)
	deltaX, deltaY := state.FootX-hipX, state.FootY-hipY
	limit := min(1, (tsukumoThighLength+tsukumoShinLength-0.5)/max(0.001, length(deltaX, deltaY)))
	state.FootX, state.FootY = hipX+deltaX*limit, hipY+deltaY*limit
	state.WheelSpin += length(state.VelocityX, state.VelocityY) * deltaSeconds * 0.35
}

func (g *Game) updateGaze(rig *TsukumoRig, isMoving bool, deltaSeconds float32) {
	player := g.player
	isIdle := !isMoving && !player.IsFiring
	rig.IdleSeconds = (rig.IdleSeconds + deltaSeconds) * boolToFloat(isIdle)
	rig.GlanceSeconds -= deltaSeconds
	g.pickGlanceIfDue(rig)
	glanceWeight := boolToFloat(rig.IdleSeconds > tsukumoIdleBeforeGlance)
	targetX := player.AimX + (rig.GlanceX-player.AimX)*glanceWeight
	targetY := player.AimY + (rig.GlanceY-player.AimY)*glanceWeight
	sharpness := min(1, deltaSeconds*tsukumoGazeSharpness)
	rig.LookX, rig.LookY = normalize(rig.LookX+(targetX-rig.LookX)*sharpness, rig.LookY+(targetY-rig.LookY)*sharpness)
}

func (g *Game) pickGlanceIfDue(rig *TsukumoRig) {
	if rig.GlanceSeconds > 0 {
		return
	}
	angle := g.random.Angle()
	rig.GlanceX, rig.GlanceY = cosine(angle), sine(angle)
	rig.GlanceSeconds = g.random.Between(0.5, 1.6)
}
