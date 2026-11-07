package game

import (
	"bytes"
	"reflect"
	"slices"
	"testing"

	"github.com/ajkula/ajkurig/draw"
)

// ===== Frozen reference =====

var (
	referenceBlue      = [3]float32{0.22, 0.52, 1}
	referenceDeepBlue  = [3]float32{0.03, 0.07, 0.2}
	referenceHighlight = [3]float32{0.62, 0.8, 1}
	referenceWhite     = [3]float32{0.95, 0.97, 1}
	referenceLegShade  = [3]float32{0.5, 0.56, 0.68}
	referencePupil     = [3]float32{0.06, 0.08, 0.14}
	referenceRedRim    = [3]float32{1, 0.22, 0.12}

	referenceEyes = [3][2]float32{{10, 0}, {6.2, -4.6}, {6.2, 4.6}}

	referenceShell = Material{Base: referenceBlue, Rim: referenceRedRim, RimSide: 1, IsShaded: true, IsHighlight: true}
	referenceHead  = Material{Base: [3]float32{0.12, 0.3, 0.66}, Rim: referenceRedRim, RimSide: 1, IsShaded: true, IsHighlight: true}
	referenceDark  = Material{Base: [3]float32{0.08, 0.14, 0.32}, Rim: referenceRedRim, RimSide: 1, IsShaded: true}
	referenceLimb  = Material{Base: referenceWhite, Rim: referenceRedRim, RimSide: 1, IsShaded: true}
	referenceFar   = Material{Base: [3]float32{0.62, 0.66, 0.74}, IsShaded: true}

	referenceRearLimbs = [2]ArtLimb{
		{Hip: [2]float32{-6, -22}, Knee: [2]float32{-16, -27.5}, Foot: [2]float32{-19, -12}, UpperWidth: [2]float32{3.2, 2.5}, LowerWidth: [2]float32{2.5, 1.8}},
		{Hip: [2]float32{6, -22}, Knee: [2]float32{16, -27.5}, Foot: [2]float32{19, -12}, UpperWidth: [2]float32{3.2, 2.5}, LowerWidth: [2]float32{2.5, 1.8}},
	}
	referenceFrontLimbs = [2]ArtLimb{
		{Hip: [2]float32{-17, -25}, Knee: [2]float32{-46, -36}, Foot: [2]float32{-40, -11}, UpperWidth: [2]float32{3.8, 3}, LowerWidth: [2]float32{3, 2.2}},
		{Hip: [2]float32{17, -25}, Knee: [2]float32{46, -36}, Foot: [2]float32{40, -11}, UpperWidth: [2]float32{3.8, 3}, LowerWidth: [2]float32{3, 2.2}},
	}
)

func (c *ArtCanvas) paintReferenceRobot(pose ArtPose) {
	for _, limb := range referenceRearLimbs {
		c.paintLimb(pose, limb, referenceFar, referenceFar)
		c.paintWheel(pose, limb.Foot, 7.5)
	}
	c.paintEllipsoid(pose, 4, -44, 36, 26, referenceShell)
	c.paintEllipsoid(pose, -21, -50, 5, 4, referenceDark)
	c.paintEllipsoid(pose, 24, -56, 4, 3, referenceDark)
	c.paintCapsule(pose, 2, -30, 0, -36, 8, 8, referenceDark)
	c.paintEllipsoid(pose, 0, -36, 22, 15, referenceHead)
	for _, eye := range tsukumoArtEyes {
		c.paintEye(pose, eye[0], eye[1], eye[2])
	}
	for _, side := range [2]float32{-1, 1} {
		c.paintCapsule(pose, 14*side, -27, 20*side, -16, 1.8, 1.6, referenceLimb)
		c.paintEllipsoid(pose, 21*side, -15, 2.5, 2.5, referenceDark)
	}
	c.paintCapsule(pose, tsukumoGunTopX, tsukumoGunTopY, 0, -18, 2.4, 2.4, referenceDark)
	c.paintFlatDisc(pose, 0, -17.5, 1.4, flatPupil)
	for _, limb := range referenceFrontLimbs {
		c.paintLimb(pose, limb, referenceLimb, referenceDark)
		c.paintWheel(pose, limb.Foot, 11.25)
	}
}

func (r *Renderer) queueReferenceRobot(g *Game) {
	player := g.player
	rig := &player.Rig
	isBlinking := player.InvulnerableSeconds > 0 && (g.frame/4)%2 == 0
	alpha := 1 - 0.5*boolToFloat(isBlinking)
	hurt := clamp(player.DamageFlashSeconds*4, 0, 1)
	body := mixColor(mixColor(referenceBlue, hurtColor, hurt), referenceWhite, clamp(player.LevelFlashSeconds/volleyBurstFlashSeconds, 0, 1))
	breath := 1 + 0.045*sine(rig.BobPhase*3.14159265)
	r.queueTsukumoShadow(player, rig)
	for leg := range tsukumoLegCount {
		r.queueReferenceLeg(player, rig, leg, alpha)
	}
	r.queueReferenceAbdomen(player, rig, body, breath, alpha)
	r.queueReferenceNeck(player, rig, alpha)
	r.queueReferenceBarrel(player, rig, alpha)
	r.queueReferenceArms(player, rig, alpha)
	r.queueReferenceHead(player, rig, body, breath, alpha)
}

func (r *Renderer) queueReferenceLeg(player *Player, rig *TsukumoRig, leg int, alpha float32) {
	layout := &tsukumoLegLayouts[leg]
	state := &rig.Legs[leg]
	hipWorldX, hipWorldY := rig.LocalToWorld(player.X, player.Y, layout.HipForward, layout.HipSide)
	kneeWorldX, kneeWorldY := rig.KneePosition(hipWorldX, hipWorldY, state.FootX, state.FootY, player.X, player.Y)
	hipX, hipY := r.camera.ToScreen(hipWorldX, hipWorldY)
	kneeX, kneeY := r.camera.ToScreen(kneeWorldX, kneeWorldY)
	footX, footY := r.camera.ToScreen(state.FootX, state.FootY)
	r.batches.Solid.AddCircle(footX+2, footY+4, scaledTsukumo(3.6), blackColor, 0.3*alpha)
	r.queueDroplet(hipX, hipY, kneeX, kneeY, scaledTsukumo(2.2), scaledTsukumo(3.6), referenceDeepBlue, alpha)
	r.queueDroplet(kneeX, kneeY, footX, footY, scaledTsukumo(3.6), scaledTsukumo(2), referenceDeepBlue, alpha)
	r.queueDroplet(hipX, hipY, kneeX, kneeY, scaledTsukumo(1.5), scaledTsukumo(2.9), referenceWhite, alpha)
	r.queueDroplet(kneeX, kneeY, footX, footY, scaledTsukumo(2.9), scaledTsukumo(1.3), referenceWhite, alpha)
	r.batches.Solid.AddCircle(kneeX-0.7, kneeY-0.9, scaledTsukumo(1), referenceLegShade, alpha)
	r.queueWheel(footX, footY, state.WheelSpin, alpha)
}

func (r *Renderer) queueReferenceAbdomen(player *Player, rig *TsukumoRig, body [3]float32, breath, alpha float32) {
	jointX, jointY := r.tsukumoPoint(player, -5, 0)
	centerX, centerY := r.abdomenPoint(player, -13, 0)
	tailX, tailY := r.abdomenPoint(player, -22, 0)
	shiftX, shiftY := rig.AbdomenOffsetX, rig.AbdomenOffsetY
	r.batches.Solid.AddSegment(jointX, jointY, centerX+shiftX, centerY+shiftY, scaledTsukumo(7), referenceDeepBlue, alpha)
	r.batches.Solid.AddCircle(centerX+shiftX, centerY+shiftY, scaledTsukumo(11.5*breath), referenceDeepBlue, alpha)
	r.batches.Solid.AddCircle(centerX+shiftX, centerY+shiftY, scaledTsukumo(10.2*breath), body, alpha)
	r.batches.Solid.AddSegment(centerX+shiftX, centerY+shiftY, tailX+shiftX, tailY+shiftY, scaledTsukumo(1.4), referenceDeepBlue, alpha)
	r.batches.Solid.AddCircle(centerX+shiftX-2, centerY+shiftY-3, scaledTsukumo(4*breath), referenceHighlight, 0.55*alpha)
	r.batches.Solid.AddCircle(jointX, jointY, scaledTsukumo(3.2), referenceWhite, alpha)
}

func (r *Renderer) queueReferenceNeck(player *Player, rig *TsukumoRig, alpha float32) {
	pivotX, pivotY := r.tsukumoPoint(player, tsukumoNeckForward, 0)
	neckX, neckY := r.tsukumoPoint(player, -5, 0)
	r.batches.Solid.AddSegment(neckX, neckY, pivotX, pivotY, scaledTsukumo(6), referenceDeepBlue, alpha)
	r.batches.Solid.AddCircle(pivotX, pivotY, scaledTsukumo(5.4), referenceDeepBlue, alpha)
	r.batches.Solid.AddCircle(pivotX, pivotY, scaledTsukumo(4.4), referenceLegShade, alpha)
	r.batches.Solid.AddCircle(pivotX, pivotY, scaledTsukumo(3.1), referenceDeepBlue, alpha)
	for _, notch := range [2]float32{1.5708, -1.5708} {
		notchX, notchY := rotateOffset(3.8, 0, rig.HeadAngle()+notch)
		r.batches.Solid.AddCircle(pivotX+notchX, pivotY+notchY, scaledTsukumo(0.9), referenceWhite, alpha)
	}
}

func (r *Renderer) queueReferenceBarrel(player *Player, rig *TsukumoRig, alpha float32) {
	recoil := 4 * rig.ArmRecoil
	breechX, breechY := r.headPoint(player, 4, 0)
	muzzleX, muzzleY := r.headPoint(player, tsukumoMuzzleForward-recoil, 0)
	r.batches.Solid.AddSegment(breechX, breechY, muzzleX, muzzleY, scaledTsukumo(3.4), referenceDeepBlue, alpha)
	r.batches.Solid.AddSegment(breechX, breechY, muzzleX, muzzleY, scaledTsukumo(1.6), referenceLegShade, alpha)
	r.batches.Solid.AddCircle(muzzleX, muzzleY, scaledTsukumo(1.9), referenceDeepBlue, alpha)
	r.batches.Glow.AddCircleIf(muzzleX, muzzleY, scaledTsukumo(9), [3]float32{1, 0.7, 0.3}, rig.ArmRecoil*0.8, rig.ArmRecoil > 0.4)
}

func (r *Renderer) queueReferenceHead(player *Player, rig *TsukumoRig, body [3]float32, breath, alpha float32) {
	headX, headY := r.headPoint(player, 4, 0)
	r.batches.Solid.AddCircle(headX, headY, scaledTsukumo(8.6*breath), referenceDeepBlue, alpha)
	r.batches.Solid.AddCircle(headX, headY, scaledTsukumo(7.5*breath), body, alpha)
	highlightX, highlightY := r.headPoint(player, 2.5, -2.5)
	r.batches.Solid.AddCircle(highlightX, highlightY, scaledTsukumo(2.8), referenceHighlight, 0.5*alpha)
	lookForward, lookSide := rotateOffset(rig.LookX, rig.LookY, -rig.HeadAngle())
	for _, eye := range referenceEyes {
		r.queueReferenceEye(player, eye[0]+lookForward*0.8, eye[1]+lookSide*1.2, rig, alpha)
	}
}

func (r *Renderer) queueReferenceEye(player *Player, forward, side float32, rig *TsukumoRig, alpha float32) {
	eyeX, eyeY := r.headPoint(player, forward, side)
	r.batches.Solid.AddCircle(eyeX, eyeY, scaledTsukumo(3), referenceDeepBlue, alpha)
	r.batches.Solid.AddCircle(eyeX, eyeY, scaledTsukumo(2.4), referenceWhite, alpha)
	r.batches.Solid.AddCircle(eyeX+rig.LookX*1, eyeY+rig.LookY*1, scaledTsukumo(1.25), referencePupil, alpha)
}

func (r *Renderer) queueReferenceArms(player *Player, rig *TsukumoRig, alpha float32) {
	reach := 10 - 4*rig.ArmRecoil
	for _, side := range [2]float32{-6.5, 6.5} {
		shoulderX, shoulderY := r.headPoint(player, 5, side)
		tipX, tipY := r.headPoint(player, 5+reach, side*0.85)
		r.queueDroplet(shoulderX, shoulderY, tipX, tipY, scaledTsukumo(1.4), scaledTsukumo(0.8), referenceWhite, alpha)
		r.batches.Solid.AddCircle(tipX, tipY, scaledTsukumo(1.2), referenceLegShade, alpha)
	}
}

// ===== Tests =====

func TestComposedRobotArtMatchesTheReferencePixelForPixel(t *testing.T) {
	for _, zoom := range [...]float32{artZoomStart, 0.35, 0.5, 0.75, 0.9, 1} {
		pose := zoomedPose(tsukumoArtX, tsukumoArtY, zoom)
		composed, reference := NewArtCanvas(), NewArtCanvas()
		composed.beginLayer()
		composed.paintRobot(pose)
		composed.commitLayer()
		reference.beginLayer()
		reference.paintReferenceRobot(pose)
		reference.commitLayer()
		if !slices.ContainsFunc(reference.Final, func(value byte) bool { return value != 0 }) {
			t.Fatalf("zoom %v: the reference painted nothing", zoom)
		}
		if !bytes.Equal(composed.Final, reference.Final) {
			t.Fatalf("zoom %v: the composed robot differs from the reference", zoom)
		}
	}
}

func TestComposedRobotOnScreenMatchesTheReference(t *testing.T) {
	game := newHeadlessGame(t)
	game.applyRobotLook(RobotLook{})
	states := [...]struct {
		Name string
		Set  func(*Player)
	}{
		{"rest", func(*Player) {}},
		{"recoil", func(player *Player) { player.Rig.ArmRecoil = 0.8 }},
		{"hurt", func(player *Player) { player.DamageFlashSeconds = 0.1 }},
		{"level flash", func(player *Player) { player.LevelFlashSeconds = volleyBurstFlashSeconds / 2 }},
		{"invulnerable", func(player *Player) { player.InvulnerableSeconds = 1 }},
		{"swaying", func(player *Player) {
			player.Rig.AbdomenOffsetX, player.Rig.AbdomenOffsetY, player.Rig.LookX, player.Rig.LookY, player.Rig.BobPhase = 4, -3, 0.7, -0.4, 0.3
		}},
	}
	for range 30 {
		stepHeadlessIdle(game)
	}
	for _, state := range states {
		state.Set(game.player)
		for _, frame := range [...]uint32{0, 4} {
			game.frame = frame
			composed, reference := rendererOnFreshBatches(game), rendererOnFreshBatches(game)
			composed.queueTsukumo(game)
			reference.queueReferenceRobot(game)
			requireSameVertices(t, state.Name, "solid", composed.batches.Solid, reference.batches.Solid)
			requireSameVertices(t, state.Name, "glow", composed.batches.Glow, reference.batches.Glow)
		}
	}
}

// ===== Helpers =====

func rendererOnFreshBatches(game *Game) *Renderer {
	renderer := *game.renderer
	renderer.batches = draw.NewBatches()
	return &renderer
}

func requireSameVertices(t *testing.T, state, name string, composed, reference *draw.SpriteBatch) {
	t.Helper()
	composedValues, referenceValues := batchVertexValues(composed), batchVertexValues(reference)
	if name == "solid" && len(referenceValues) == 0 {
		t.Fatalf("%s: the reference queued nothing", state)
	}
	if !slices.Equal(composedValues, referenceValues) {
		t.Fatalf("%s: %s vertices differ from the reference", state, name)
	}
}

func batchVertexValues(batch *draw.SpriteBatch) []float64 {
	vertices := reflect.ValueOf(batch).Elem().FieldByName("vertices")
	values := make([]float64, 0, vertices.Len()*8)
	for index := range vertices.Len() {
		vertex := vertices.Index(index)
		for field := range vertex.NumField() {
			values = append(values, vertex.Field(field).Float())
		}
	}
	return values
}
