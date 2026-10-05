package game

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// ===== Types =====

type Material struct {
	Base        [3]float32
	Rim         [3]float32
	RimSide     float32
	IsShaded    bool
	IsHighlight bool
}

type ArtLayer struct {
	Colors []byte
}

type ArtCanvas struct {
	Width  int
	Height int
	Final  []byte
	Layer  ArtLayer
	Image  *ebiten.Image
}

type ArtPose struct {
	AnchorX float32
	AnchorY float32
	Scale   float32
	Lift    float32
	Shear   float32
}

type ArtLimb struct {
	Hip        [2]float32
	Knee       [2]float32
	Foot       [2]float32
	UpperWidth [2]float32
	LowerWidth [2]float32
}

// ===== Constants =====

const (
	artWidth           = 320
	artHeight          = 180
	artPixelScale      = screenWidth / artWidth
	artShadeLevels     = 4
	artSpecularDot     = 0.94
	artRimDepth        = 0.6
	artFloorTop        = 150
	tachikomaArtStartX = 92
	tachikomaArtStartY = 166
	tachikomaArtEndX   = 88
	tachikomaArtEndY   = 174
	tachikomaArtStart  = 0.8
	spiderArtX         = 238
	spiderArtY         = 174
	spiderArtSlide     = 16
	spiderArtLean      = 0.1
	artRimWidth        = 0.7
)

var (
	artLight        = normalize3([3]float32{-0.55, -0.65, 0.55})
	artShadeFactors = [artShadeLevels]float32{0.36, 0.58, 0.8, 1.02}
	artOutline      = [3]float32{0.02, 0.02, 0.04}
	redRim          = [3]float32{1, 0.22, 0.12}
	blueRim         = [3]float32{0.35, 0.7, 1}
	artFloorBands   = [3][3]float32{{0.03, 0.03, 0.05}, {0.05, 0.045, 0.07}, {0.07, 0.06, 0.09}}

	tachikomaShell = Material{Base: tachikomaBlue, Rim: redRim, RimSide: 1, IsShaded: true, IsHighlight: true}
	tachikomaDark  = Material{Base: [3]float32{0.08, 0.14, 0.32}, Rim: redRim, RimSide: 1, IsShaded: true}
	tachikomaLimb  = Material{Base: tachikomaWhite, Rim: redRim, RimSide: 1, IsShaded: true}
	tachikomaFar   = Material{Base: [3]float32{0.62, 0.66, 0.74}, IsShaded: true}
	tachikomaEye   = Material{Base: tachikomaWhite, IsShaded: true, IsHighlight: true}
	flatPupil      = Material{Base: tachikomaPupil}
	flatWheel      = Material{Base: tachikomaWheel, Rim: redRim, RimSide: 1, IsShaded: true}
	flatHub        = Material{Base: tachikomaLegShade}
	spiderArmor    = Material{Base: [3]float32{0.46, 0.52, 0.44}, Rim: blueRim, RimSide: -1, IsShaded: true, IsHighlight: true}
	spiderPlate    = Material{Base: [3]float32{0.6, 0.66, 0.56}, Rim: blueRim, RimSide: -1, IsShaded: true, IsHighlight: true}
	spiderDark     = Material{Base: [3]float32{0.2, 0.23, 0.2}, Rim: blueRim, RimSide: -1, IsShaded: true}
	spiderJoint    = Material{Base: jointColor, Rim: blueRim, RimSide: -1, IsShaded: true}
	spiderFar      = Material{Base: [3]float32{0.27, 0.3, 0.26}, IsShaded: true}
	spiderEye      = Material{Base: sensorRed}
	spiderEyeCore  = Material{Base: [3]float32{1, 0.85, 0.75}}

	tachikomaRearLimbs = [2]ArtLimb{
		{Hip: [2]float32{-6, -50}, Knee: [2]float32{-46, -72}, Foot: [2]float32{-58, -12}, UpperWidth: [2]float32{3.2, 2.5}, LowerWidth: [2]float32{2.5, 1.8}},
		{Hip: [2]float32{6, -50}, Knee: [2]float32{46, -72}, Foot: [2]float32{56, -12}, UpperWidth: [2]float32{3.2, 2.5}, LowerWidth: [2]float32{2.5, 1.8}},
	}
	tachikomaFrontLimbs = [2]ArtLimb{
		{Hip: [2]float32{-17, -25}, Knee: [2]float32{-46, -36}, Foot: [2]float32{-40, -8}, UpperWidth: [2]float32{3.8, 3}, LowerWidth: [2]float32{3, 2.2}},
		{Hip: [2]float32{17, -25}, Knee: [2]float32{46, -36}, Foot: [2]float32{40, -8}, UpperWidth: [2]float32{3.8, 3}, LowerWidth: [2]float32{3, 2.2}},
	}
	spiderRearLimbs = [2]ArtLimb{
		{Hip: [2]float32{18, -116}, Knee: [2]float32{46, -158}, Foot: [2]float32{70, -6}, UpperWidth: [2]float32{7, 5.5}, LowerWidth: [2]float32{5.5, 3.5}},
		{Hip: [2]float32{-8, -118}, Knee: [2]float32{-18, -156}, Foot: [2]float32{-30, -10}, UpperWidth: [2]float32{7, 5.5}, LowerWidth: [2]float32{5.5, 3.5}},
	}
	spiderFrontLimbs = [2]ArtLimb{
		{Hip: [2]float32{-26, -100}, Knee: [2]float32{-74, -146}, Foot: [2]float32{-100, -22}, UpperWidth: [2]float32{10, 7.5}, LowerWidth: [2]float32{7.5, 4}},
		{Hip: [2]float32{30, -98}, Knee: [2]float32{64, -146}, Foot: [2]float32{74, -4}, UpperWidth: [2]float32{10, 7.5}, LowerWidth: [2]float32{7.5, 4}},
	}
)

// ===== Public API =====

func NewArtCanvas() *ArtCanvas {
	return &ArtCanvas{
		Width: artWidth, Height: artHeight,
		Final: make([]byte, artWidth*artHeight*4),
		Layer: ArtLayer{Colors: make([]byte, artWidth*artHeight*4)},
		Image: ebiten.NewImage(artWidth, artHeight),
	}
}

func (c *ArtCanvas) Render(progress, seconds float32) *ebiten.Image {
	c.paintFloor()
	approach := smoothstep(clamp(progress, 0, 1))
	spider := ArtPose{AnchorX: spiderArtX + spiderArtSlide*(1-approach), AnchorY: spiderArtY, Scale: 1, Lift: sine(seconds*3) * 3, Shear: spiderArtLean * approach}
	c.paintShadow(spider.AnchorX, spider.AnchorY, 80, 6)
	c.beginLayer()
	c.paintSpider(spider, seconds)
	c.commitLayer()
	tachikoma := ArtPose{
		AnchorX: lerp(tachikomaArtStartX, tachikomaArtEndX, approach), AnchorY: lerp(tachikomaArtStartY, tachikomaArtEndY, approach),
		Scale: lerp(tachikomaArtStart, 1, approach), Lift: sine(seconds*9) * 0.6,
	}
	c.paintShadow(tachikoma.AnchorX, tachikoma.AnchorY, 56*tachikoma.Scale, 4)
	c.beginLayer()
	c.paintTachikoma(tachikoma)
	c.commitLayer()
	c.Image.WritePixels(c.Final)
	return c.Image
}

// ===== Internal =====

func (c *ArtCanvas) paintFloor() {
	for y := range c.Height {
		band := clampInt((y-artFloorTop)/10, 0, len(artFloorBands)-1)
		tint := [2][3]float32{{0, 0, 0}, artFloorBands[band]}[boolToIndex(y >= artFloorTop)]
		for x := range c.Width {
			writeColor(c.Final[(y*c.Width+x)*4:], tint)
		}
	}
}

func (c *ArtCanvas) paintShadow(x, y, radiusX, radiusY float32) {
	for py := int(y - radiusY); py <= int(y+radiusY); py++ {
		for px := int(x - radiusX); px <= int(x+radiusX); px++ {
			dx, dy := (float32(px)+0.5-x)/radiusX, (float32(py)+0.5-y)/radiusY
			c.darkenIf(px, py, dx*dx+dy*dy <= 1)
		}
	}
}

func (c *ArtCanvas) darkenIf(x, y int, shouldDarken bool) {
	if !shouldDarken || !c.isInside(x, y) {
		return
	}
	writeColor(c.Final[(y*c.Width+x)*4:], artOutline)
}

func (c *ArtCanvas) beginLayer() {
	clear(c.Layer.Colors)
}

func (c *ArtCanvas) commitLayer() {
	for index := range c.Width * c.Height {
		c.commitPixel(index)
	}
}

func (c *ArtCanvas) commitPixel(index int) {
	x, y := index%c.Width, index/c.Width
	isOpaque := c.Layer.Colors[index*4+3] != 0
	isEdge := !isOpaque && (c.layerOpaque(x-1, y) || c.layerOpaque(x+1, y) || c.layerOpaque(x, y-1) || c.layerOpaque(x, y+1))
	if isOpaque {
		copy(c.Final[index*4:index*4+4], c.Layer.Colors[index*4:index*4+4])
		return
	}
	if isEdge {
		writeColor(c.Final[index*4:], artOutline)
	}
}

func (c *ArtCanvas) layerOpaque(x, y int) bool {
	return c.isInside(x, y) && c.Layer.Colors[(y*c.Width+x)*4+3] != 0
}

func (c *ArtCanvas) isInside(x, y int) bool {
	return x >= 0 && y >= 0 && x < c.Width && y < c.Height
}

func (c *ArtCanvas) paintTachikoma(pose ArtPose) {
	for _, limb := range tachikomaRearLimbs {
		c.paintLimb(pose, limb, tachikomaFar, tachikomaFar)
		c.paintWheel(pose, limb.Foot, 5)
	}
	c.paintEllipsoid(pose, 4, -72, 36, 26, tachikomaShell)
	c.paintEllipsoid(pose, -21, -78, 5, 4, tachikomaDark)
	c.paintEllipsoid(pose, 24, -84, 4, 3, tachikomaDark)
	c.paintCapsule(pose, 2, -52, 0, -40, 8, 8, tachikomaDark)
	c.paintEllipsoid(pose, 0, -36, 22, 15, tachikomaShell)
	c.paintEye(pose, -10, -40, 7)
	c.paintEye(pose, 10, -40, 7)
	c.paintEye(pose, 0, -27, 4)
	for _, side := range [2]float32{-1, 1} {
		c.paintCapsule(pose, 14*side, -27, 20*side, -16, 1.8, 1.6, tachikomaLimb)
		c.paintEllipsoid(pose, 21*side, -15, 2.5, 2.5, tachikomaDark)
	}
	c.paintCapsule(pose, 0, -22, 0, -18, 2.4, 2.4, tachikomaDark)
	c.paintFlatDisc(pose, 0, -17.5, 1.4, flatPupil)
	for _, limb := range tachikomaFrontLimbs {
		c.paintLimb(pose, limb, tachikomaLimb, tachikomaDark)
		c.paintWheel(pose, limb.Foot, 7.5)
	}
}

func (c *ArtCanvas) paintEye(pose ArtPose, x, y, radius float32) {
	c.paintEllipsoid(pose, x, y, radius, radius, tachikomaEye)
	c.paintFlatDisc(pose, x+radius*0.3, y, radius*0.48, flatPupil)
}

func (c *ArtCanvas) paintWheel(pose ArtPose, foot [2]float32, radius float32) {
	c.paintEllipsoid(pose, foot[0], foot[1], radius, radius, flatWheel)
	c.paintFlatDisc(pose, foot[0], foot[1], radius*0.35, flatHub)
}

func (c *ArtCanvas) paintSpider(pose ArtPose, seconds float32) {
	for _, limb := range spiderRearLimbs {
		c.paintLimb(pose, limb, spiderFar, spiderFar)
	}
	c.paintEllipsoid(pose, 5, -98, 40, 12, spiderDark)
	c.paintEllipsoid(pose, 5, -110, 52, 26, spiderArmor)
	c.paintEllipsoid(pose, 12, -123, 26, 10, spiderPlate)
	c.paintCapsule(pose, -36, -112, 30, -112, 2, 2, spiderDark)
	front := spiderFrontLimbs[0]
	front.Foot[1] -= max(0, pose.Lift)
	c.paintLimb(pose, front, spiderArmor, spiderJoint)
	c.paintLimb(pose, spiderFrontLimbs[1], spiderArmor, spiderJoint)
	c.paintEllipsoid(pose, front.Foot[0], front.Foot[1], 8, 3.5, spiderDark)
	c.paintEllipsoid(pose, spiderFrontLimbs[1].Foot[0], spiderFrontLimbs[1].Foot[1], 8, 3.5, spiderDark)
	c.paintSpiderTurret(pose, seconds)
}

func (c *ArtCanvas) paintSpiderTurret(pose ArtPose, seconds float32) {
	c.paintEllipsoid(pose, -38, -108, 21, 16, spiderArmor)
	c.paintEllipsoid(pose, -34, -116, 11, 6, spiderPlate)
	c.paintCapsule(pose, -50, -104, -98, -73, 4.5, 3.2, spiderJoint)
	c.paintEllipsoid(pose, -98, -73, 4.5, 4.5, spiderDark)
	flicker := 0.85 + 0.15*sine(seconds*14)
	c.paintFlatDisc(pose, -52, -111, 4.5*flicker, spiderEye)
	c.paintFlatDisc(pose, -52.5, -111.5, 1.5, spiderEyeCore)
}

func (c *ArtCanvas) paintLimb(pose ArtPose, limb ArtLimb, material, joint Material) {
	c.paintCapsule(pose, limb.Hip[0], limb.Hip[1], limb.Knee[0], limb.Knee[1], limb.UpperWidth[0], limb.UpperWidth[1], material)
	c.paintCapsule(pose, limb.Knee[0], limb.Knee[1], limb.Foot[0], limb.Foot[1], limb.LowerWidth[0], limb.LowerWidth[1], material)
	c.paintEllipsoid(pose, limb.Knee[0], limb.Knee[1], limb.UpperWidth[1]*1.25, limb.UpperWidth[1]*1.25, joint)
	c.paintEllipsoid(pose, limb.Hip[0], limb.Hip[1], limb.UpperWidth[0]*0.85, limb.UpperWidth[0]*0.85, joint)
}

func (p ArtPose) point(x, y float32) (float32, float32) {
	return p.AnchorX + (x+y*p.Shear)*p.Scale, p.AnchorY + y*p.Scale
}

func (c *ArtCanvas) paintEllipsoid(pose ArtPose, x, y, radiusX, radiusY float32, material Material) {
	centerX, centerY := pose.point(x, y)
	scaledX, scaledY := max(0.5, radiusX*pose.Scale), max(0.5, radiusY*pose.Scale)
	for py := int(centerY - scaledY - 1); py <= int(centerY+scaledY+1); py++ {
		for px := int(centerX - scaledX - 1); px <= int(centerX+scaledX+1); px++ {
			nx, ny := (float32(px)+0.5-centerX)/scaledX, (float32(py)+0.5-centerY)/scaledY
			c.shadePixelIf(px, py, nx, ny, material, nx*nx+ny*ny <= 1)
		}
	}
}

func (c *ArtCanvas) paintFlatDisc(pose ArtPose, x, y, radius float32, material Material) {
	c.paintEllipsoid(pose, x, y, radius, radius, material)
}

func (c *ArtCanvas) paintCapsule(pose ArtPose, fromX, fromY, toX, toY, fromRadius, toRadius float32, material Material) {
	ax, ay := pose.point(fromX, fromY)
	bx, by := pose.point(toX, toY)
	ra, rb := max(0.5, fromRadius*pose.Scale), max(0.5, toRadius*pose.Scale)
	reach := max(ra, rb) + 1
	segmentX, segmentY := bx-ax, by-ay
	lengthSquared := max(0.0001, segmentX*segmentX+segmentY*segmentY)
	for py := int(min(ay, by) - reach); py <= int(max(ay, by)+reach); py++ {
		for px := int(min(ax, bx) - reach); px <= int(max(ax, bx)+reach); px++ {
			pointX, pointY := float32(px)+0.5, float32(py)+0.5
			along := clamp(((pointX-ax)*segmentX+(pointY-ay)*segmentY)/lengthSquared, 0, 1)
			radius := ra + (rb-ra)*along
			offsetX, offsetY := (pointX-ax-segmentX*along)/radius, (pointY-ay-segmentY*along)/radius
			c.shadePixelIf(px, py, offsetX, offsetY, material, offsetX*offsetX+offsetY*offsetY <= 1)
		}
	}
}

func (c *ArtCanvas) shadePixelIf(x, y int, nx, ny float32, material Material, isCovered bool) {
	if !isCovered || !c.isInside(x, y) {
		return
	}
	nz := sqrt(max(0, 1-nx*nx-ny*ny))
	destination := c.Layer.Colors[(y*c.Width+x)*4:]
	writeColor(destination, shadeMaterial(material, nx, ny, nz))
	destination[3] = 255
}

func shadeMaterial(material Material, nx, ny, nz float32) [3]float32 {
	if !material.IsShaded {
		return material.Base
	}
	diffuse := clamp(nx*artLight[0]+ny*artLight[1]+nz*artLight[2], 0, 1)
	level := clampInt(int(diffuse*artShadeLevels), 0, artShadeLevels-1)
	shaded := scaleColor(material.Base, artShadeFactors[level])
	isRim := nx*material.RimSide > artRimWidth && nz < artRimDepth
	shaded = mixColor(shaded, material.Rim, 0.75*boolToFloat(isRim))
	isSpecular := material.IsHighlight && diffuse > artSpecularDot
	return mixColor(shaded, [3]float32{1, 1, 1}, 0.6*boolToFloat(isSpecular))
}

func scaleColor(color [3]float32, factor float32) [3]float32 {
	return [3]float32{clamp(color[0]*factor, 0, 1), clamp(color[1]*factor, 0, 1), clamp(color[2]*factor, 0, 1)}
}

func writeColor(destination []byte, color [3]float32) {
	destination[0], destination[1], destination[2], destination[3] = byte(color[0]*255), byte(color[1]*255), byte(color[2]*255), 255
}

func normalize3(vector [3]float32) [3]float32 {
	size := sqrt(vector[0]*vector[0] + vector[1]*vector[1] + vector[2]*vector[2])
	return [3]float32{vector[0] / size, vector[1] / size, vector[2] / size}
}
