package game

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ajkula/ajkurig/colorx"
	"github.com/ajkula/ajkurig/mathx"
)

// ===== Types =====

type RollBonus uint8

type logoPen struct {
	target  *ebiten.Image
	centerX float32
	centerY float32
	radius  float32
	path    *vector.Path
}

type logoPictogram func(pen *logoPen, tint colorx.RGB)

// ===== Constants =====

const (
	RollHeal RollBonus = iota
	RollNuke
	rollBonusCount
)

const (
	fullTurn              = 2 * 3.14159265
	logoPictogramLift     = 0.55
	logoEllipseSegments   = 40
	shotDirectionAngle    = 0.1
	shotElementY          = -0.33
	shotElementScale      = 0.47
	shotSalvoSize         = 3
	shotSalvoSpacing      = 0.31
	shotSalvoStart        = 0.32
	shotPerspectiveGrowth = 0.25
	zoneGroundY           = 0.44
	zoneGroundRadiusX     = 0.92
	zoneGroundRadiusY     = zoneGroundRadiusX / 2
	zoneGroundRimWidth    = 0.07
	zonePictogramScale    = 0.8
	zonePictogramLift     = 0.46
	rollTrackWidth        = 0.14
	rollPictogramScale    = 0.82
	rollTimerStartAngle   = -fullTurn / 4
	logoBadgeOutlineWidth = 1.5
	logoBadgeTextScale    = 0.54
	logoBadgeArrowWidth   = 0.85
	logoBadgeArrowGap     = 0.12
)

var (
	logoBackground  = colorx.RGB{0.08, 0.09, 0.12}
	logoWhite       = colorx.RGB{1, 1, 1}
	logoGold        = colorx.RGB{1, 0.8, 0.25}
	logoHealColor   = colorx.RGB{0.4, 1, 0.5}
	logoNukeColor   = colorx.RGB{1, 0.85, 0.2}
	cannonDark      = colorx.RGB{0.16, 0.17, 0.2}
	cannonMetal     = colorx.RGB{0.42, 0.45, 0.52}
	cannonHighlight = colorx.RGB{0.72, 0.76, 0.84}
)

var cannonPlacements = [2][3]float32{{-0.24, 0.24, 0.85}, {-0.32, 0.54, 1}}

var badgeArrowShape = [7][2]float32{{0.5, 0}, {1, 0.5}, {0.68, 0.5}, {0.68, 1}, {0.32, 1}, {0.32, 0.5}, {0, 0.5}}

var badgeOutlineOffsets = [8][2]float64{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}

var weaponLogoLayouts = [weaponKindCount]func(pen *logoPen, kind WeaponKind){
	WeaponPulseShot:     drawShotLogo,
	WeaponEmberBolt:     drawShotLogo,
	WeaponFrostShard:    drawShotLogo,
	WeaponVoltBolt:      drawShotLogo,
	WeaponFrostNova:     drawZoneLogo,
	WeaponArcLightning:  drawZoneLogo,
	WeaponOilFlask:      drawZoneLogo,
	WeaponDownpour:      drawZoneLogo,
	WeaponOrbitBlades:   drawZoneLogo,
	WeaponSeismicHammer: drawZoneLogo,
	WeaponStaticMines:   drawZoneLogo,
}

var zonePictograms = [weaponKindCount]logoPictogram{
	WeaponFrostNova:     drawFrostNovaPictogram,
	WeaponArcLightning:  drawArcLightningPictogram,
	WeaponOilFlask:      drawOilFlaskPictogram,
	WeaponDownpour:      drawDownpourPictogram,
	WeaponOrbitBlades:   drawOrbitBladesPictogram,
	WeaponSeismicHammer: drawSeismicHammerPictogram,
	WeaponStaticMines:   drawStaticMinesPictogram,
}

var elementPictograms = [elementCount]logoPictogram{
	ElementNone:  drawPulsePictogram,
	ElementFire:  drawFlamePictogram,
	ElementFrost: drawFrostNovaPictogram,
	ElementShock: drawBoltPictogram,
}

var rollPictograms = [rollBonusCount]logoPictogram{
	RollHeal: drawHealPictogram,
	RollNuke: drawNukePictogram,
}

var rollColors = [rollBonusCount]colorx.RGB{RollHeal: logoHealColor, RollNuke: logoNukeColor}

// ===== Public API =====

func DrawWeaponLogo(target *ebiten.Image, centerX, centerY, radius float32, kind WeaponKind) {
	weaponLogoLayouts[kind](newLogoPen(target, centerX, centerY, radius), kind)
}

func DrawRollLogo(target *ebiten.Image, centerX, centerY, radius float32, bonus RollBonus, remaining float32) {
	pen := newLogoPen(target, centerX, centerY, radius)
	tint := rollColors[bonus]
	pen.circle(0, 0, 1)
	pen.fill(logoBackground, 1)
	pen.annulusSector(1-rollTrackWidth, 1, fullTurn)
	pen.fill(tint, 0.18)
	pen.annulusSector(1-rollTrackWidth, 1, remaining*fullTurn)
	pen.fill(tint, 1)
	rollPictograms[bonus](pen.within(0, 0, rollPictogramScale), tint)
}

func DrawLogoBadge(target *ebiten.Image, centerX, centerY, radius float32, offer UpgradeOffer, source *text.GoTextFaceSource) {
	label, tint, hasArrow := LogoBadge(offer)
	if label == "" {
		return
	}
	face := &text.GoTextFace{Source: source, Size: float64(radius * logoBadgeTextScale)}
	metrics := face.Metrics()
	capHeight := float32(metrics.CapHeight)
	right := centerX + radius - logoBadgeOutlineWidth
	capTop := centerY - radius + logoBadgeOutlineWidth
	lineTop := float64(capTop) - metrics.HLineGap/2 - metrics.HAscent + metrics.CapHeight
	options := &text.DrawOptions{}
	options.PrimaryAlign = text.AlignEnd
	for _, offset := range badgeOutlineOffsets {
		drawBadgeText(target, label, face, options, float64(right)+offset[0]*logoBadgeOutlineWidth, lineTop+offset[1]*logoBadgeOutlineWidth, logoBackground)
	}
	drawBadgeText(target, label, face, options, float64(right), lineTop, tint)
	if !hasArrow {
		return
	}
	labelWidth, _ := text.Measure(label, face, 0)
	arrowRight := right - float32(labelWidth) - logoBadgeArrowGap*capHeight
	drawBadgeArrow(target, arrowRight-logoBadgeArrowWidth*capHeight, capTop, capHeight, tint)
}

func LogoBadge(offer UpgradeOffer) (label string, tint colorx.RGB, hasArrow bool) {
	isNewWeapon := offer.NextLevel == 1 && !offer.IsEvolution
	isMaximum := offer.NextLevel >= weaponTable[offer.Weapon].MaximumLevel
	labels := [2]string{strconv.Itoa(offer.NextLevel), "MAX"}
	tints := [2]colorx.RGB{logoWhite, logoGold}
	return mathx.Pick(isNewWeapon, labels[boolToIndex(isMaximum)], ""), tints[boolToIndex(offer.IsEvolution)], !isMaximum
}

// ===== Internal =====

func drawBadgeText(target *ebiten.Image, label string, face *text.GoTextFace, options *text.DrawOptions, x, y float64, tint colorx.RGB) {
	options.GeoM.Reset()
	options.GeoM.Translate(x, y)
	options.ColorScale.Reset()
	options.ColorScale.ScaleWithColor(colorx.NRGBA(tint, 1))
	text.Draw(target, label, face, options)
}

func drawBadgeArrow(target *ebiten.Image, left, top, height float32, tint colorx.RGB) {
	pen := newLogoPen(target, left, top, 1)
	points := make([][2]float32, len(badgeArrowShape))
	for index, point := range badgeArrowShape {
		points[index] = [2]float32{point[0] * logoBadgeArrowWidth * height, point[1] * height}
	}
	pen.polygon(points...)
	pen.stroke(logoBackground, 1, 2*logoBadgeOutlineWidth)
	pen.polygon(points...)
	pen.fill(tint, 1)
}

func drawShotLogo(pen *logoPen, kind WeaponKind) {
	definition := &weaponTable[kind]
	tint := liftedLogoColor(definition.Color)
	directionX, directionY := cosine(shotDirectionAngle), sine(shotDirectionAngle)
	for _, placement := range cannonPlacements {
		drawSalvo(pen, placement, directionX, directionY, tint)
		drawCannon(pen, placement, directionX, directionY, tint)
	}
	elementPictograms[definition.Element](pen.within(0, shotElementY, shotElementScale), tint)
}

func drawCannon(pen *logoPen, placement [3]float32, directionX, directionY float32, tint colorx.RGB) {
	muzzleX, muzzleY, scale := placement[0], placement[1], placement[2]
	breechX, breechY := muzzleX-0.5*scale*directionX, muzzleY-0.5*scale*directionY
	bodyX, bodyY := muzzleX-0.2*scale*directionX, muzzleY-0.2*scale*directionY
	pen.segment(breechX, breechY, bodyX, bodyY)
	pen.stroke(cannonDark, 1, 0.3*scale)
	pen.segment(breechX, breechY, bodyX, bodyY)
	pen.stroke(cannonMetal, 1, 0.22*scale)
	pen.segment(bodyX, bodyY, muzzleX, muzzleY)
	pen.stroke(cannonDark, 1, 0.15*scale)
	pen.segment(bodyX, bodyY-0.03*scale, muzzleX, muzzleY-0.03*scale)
	pen.stroke(cannonHighlight, 0.8, 0.04*scale)
	pen.circle(muzzleX+0.02*scale*directionX, muzzleY, 0.07*scale)
	pen.fill(tint, 0.9)
}

func drawSalvo(pen *logoPen, placement [3]float32, directionX, directionY float32, tint colorx.RGB) {
	muzzleX, muzzleY, scale := placement[0], placement[1], placement[2]
	for shot := range shotSalvoSize {
		distance := shotSalvoStart + shotSalvoSpacing*float32(shot)
		size := scale * (1 + shotPerspectiveGrowth*distance)
		x, y := muzzleX+distance*directionX, muzzleY+distance*directionY
		pen.segment(x-0.2*size*directionX, y-0.2*size*directionY, x, y)
		pen.stroke(tint, 0.3, 0.07*size)
		pen.segment(x-0.06*size*directionX, y-0.06*size*directionY, x+0.05*size*directionX, y+0.05*size*directionY)
		pen.stroke(tint, 1, 0.13*size)
		pen.segment(x-0.04*size*directionX, y-0.04*size*directionY, x+0.04*size*directionX, y+0.04*size*directionY)
		pen.stroke(logoWhite, 0.85, 0.05*size)
	}
}

func drawZoneLogo(pen *logoPen, kind WeaponKind) {
	tint := liftedLogoColor(weaponTable[kind].Color)
	pen.ellipse(0, zoneGroundY, zoneGroundRadiusX, zoneGroundRadiusY, vector.Clockwise)
	pen.fill(tint, 0.2)
	pen.ellipse(0, zoneGroundY, zoneGroundRadiusX, zoneGroundRadiusY, vector.Clockwise)
	pen.ellipse(0, zoneGroundY, zoneGroundRadiusX-zoneGroundRimWidth, zoneGroundRadiusY-zoneGroundRimWidth/2, vector.CounterClockwise)
	pen.fill(tint, 0.85)
	pen.ellipse(0, zoneGroundY, 0.3, 0.15, vector.Clockwise)
	pen.fill(logoBackground, 0.4)
	zonePictograms[kind](pen.within(0, zoneGroundY-zonePictogramLift, zonePictogramScale), tint)
}

func liftedLogoColor(color colorx.RGB) colorx.RGB {
	luminance := 0.3*color[0] + 0.59*color[1] + 0.11*color[2]
	return colorx.Mix(color, logoWhite, max(0, logoPictogramLift-luminance))
}

func newLogoPen(target *ebiten.Image, centerX, centerY, radius float32) *logoPen {
	return &logoPen{target: target, centerX: centerX, centerY: centerY, radius: radius, path: &vector.Path{}}
}

func (p *logoPen) within(x, y, scale float32) *logoPen {
	centerX, centerY := p.at(x, y)
	return &logoPen{target: p.target, centerX: centerX, centerY: centerY, radius: p.radius * scale, path: p.path}
}

func (p *logoPen) at(x, y float32) (float32, float32) {
	return p.centerX + x*p.radius, p.centerY + y*p.radius
}

func (p *logoPen) moveTo(x, y float32) {
	p.path.MoveTo(p.at(x, y))
}

func (p *logoPen) lineTo(x, y float32) {
	p.path.LineTo(p.at(x, y))
}

func (p *logoPen) circle(x, y, radius float32) {
	p.moveTo(x+radius, y)
	centerX, centerY := p.at(x, y)
	p.path.Arc(centerX, centerY, radius*p.radius, 0, fullTurn, vector.Clockwise)
	p.path.Close()
}

func (p *logoPen) ellipse(x, y, radiusX, radiusY float32, direction vector.Direction) {
	step := fullTurn / logoEllipseSegments * (1 - 2*float32(direction))
	p.moveTo(x+radiusX, y)
	for segment := 1; segment < logoEllipseSegments; segment++ {
		angle := step * float32(segment)
		p.lineTo(x+radiusX*cosine(angle), y+radiusY*sine(angle))
	}
	p.path.Close()
}

func (p *logoPen) annulus(x, y, innerRadius, outerRadius float32) {
	centerX, centerY := p.at(x, y)
	p.moveTo(x+outerRadius, y)
	p.path.Arc(centerX, centerY, outerRadius*p.radius, 0, fullTurn, vector.Clockwise)
	p.path.Close()
	p.moveTo(x+innerRadius, y)
	p.path.Arc(centerX, centerY, innerRadius*p.radius, fullTurn, 0, vector.CounterClockwise)
	p.path.Close()
}

func (p *logoPen) annulusSector(innerRadius, outerRadius, sweep float32) {
	startAngle := float32(rollTimerStartAngle)
	endAngle := startAngle + sweep
	centerX, centerY := p.at(0, 0)
	p.moveTo(outerRadius*cosine(startAngle), outerRadius*sine(startAngle))
	p.path.Arc(centerX, centerY, outerRadius*p.radius, startAngle, endAngle, vector.Clockwise)
	p.lineTo(innerRadius*cosine(endAngle), innerRadius*sine(endAngle))
	p.path.Arc(centerX, centerY, innerRadius*p.radius, endAngle, startAngle, vector.CounterClockwise)
	p.path.Close()
}

func (p *logoPen) arc(x, y, radius, startAngle, endAngle float32) {
	p.moveTo(x+radius*cosine(startAngle), y+radius*sine(startAngle))
	centerX, centerY := p.at(x, y)
	p.path.Arc(centerX, centerY, radius*p.radius, startAngle, endAngle, vector.Clockwise)
}

func (p *logoPen) polygon(points ...[2]float32) {
	p.polyline(points...)
	p.path.Close()
}

func (p *logoPen) polyline(points ...[2]float32) {
	p.moveTo(points[0][0], points[0][1])
	for _, point := range points[1:] {
		p.lineTo(point[0], point[1])
	}
}

func (p *logoPen) segment(fromX, fromY, toX, toY float32) {
	p.moveTo(fromX, fromY)
	p.lineTo(toX, toY)
}

func (p *logoPen) fill(tint colorx.RGB, alpha float32) {
	options := &vector.DrawPathOptions{AntiAlias: true}
	options.ColorScale.ScaleWithColor(colorx.NRGBA(tint, alpha))
	vector.FillPath(p.target, p.path, &vector.FillOptions{}, options)
	p.path.Reset()
}

func (p *logoPen) stroke(tint colorx.RGB, alpha, width float32) {
	options := &vector.DrawPathOptions{AntiAlias: true}
	options.ColorScale.ScaleWithColor(colorx.NRGBA(tint, alpha))
	strokeOptions := &vector.StrokeOptions{Width: width * p.radius, LineCap: vector.LineCapRound, LineJoin: vector.LineJoinRound}
	vector.StrokePath(p.target, p.path, strokeOptions, options)
	p.path.Reset()
}

func drawPulsePictogram(pen *logoPen, tint colorx.RGB) {
	pen.circle(-0.3, 0, 0.14)
	pen.fill(tint, 1)
	for wave := range 3 {
		pen.arc(-0.3, 0, 0.32+0.17*float32(wave), -0.75, 0.75)
	}
	pen.stroke(tint, 1, 0.1)
}

func drawFlamePictogram(pen *logoPen, tint colorx.RGB) {
	pen.polygon([2]float32{0, -0.62}, [2]float32{0.2, -0.3}, [2]float32{0.36, -0.36}, [2]float32{0.4, 0.1}, [2]float32{0.26, 0.42}, [2]float32{0, 0.55}, [2]float32{-0.26, 0.42}, [2]float32{-0.4, 0.1}, [2]float32{-0.28, -0.22}, [2]float32{-0.14, -0.08})
	pen.fill(tint, 1)
	pen.polygon([2]float32{0, -0.12}, [2]float32{0.18, 0.18}, [2]float32{0.14, 0.4}, [2]float32{0, 0.47}, [2]float32{-0.14, 0.4}, [2]float32{-0.18, 0.18})
	pen.fill(logoWhite, 0.8)
}

func drawBoltPictogram(pen *logoPen, tint colorx.RGB) {
	pen.polygon([2]float32{0.12, -0.62}, [2]float32{-0.32, 0.06}, [2]float32{-0.04, 0.06}, [2]float32{-0.2, 0.62}, [2]float32{0.32, -0.1}, [2]float32{0.04, -0.1})
	pen.fill(tint, 1)
}

func drawFrostNovaPictogram(pen *logoPen, tint colorx.RGB) {
	for arm := range 6 {
		angle := float32(arm) * fullTurn / 6
		directionX, directionY := cosine(angle), sine(angle)
		pen.segment(0, 0, directionX*0.58, directionY*0.58)
		branchX, branchY := directionX*0.36, directionY*0.36
		pen.segment(branchX, branchY, branchX+cosine(angle+0.7)*0.18, branchY+sine(angle+0.7)*0.18)
		pen.segment(branchX, branchY, branchX+cosine(angle-0.7)*0.18, branchY+sine(angle-0.7)*0.18)
	}
	pen.stroke(tint, 1, 0.08)
}

func drawArcLightningPictogram(pen *logoPen, tint colorx.RGB) {
	drawBoltPictogram(pen, tint)
	pen.polyline([2]float32{0.12, 0.04}, [2]float32{0.36, 0.18}, [2]float32{0.3, 0.32}, [2]float32{0.52, 0.42})
	pen.polyline([2]float32{0.08, -0.36}, [2]float32{0.28, -0.44}, [2]float32{0.3, -0.32}, [2]float32{0.48, -0.38})
	pen.polyline([2]float32{-0.2, -0.04}, [2]float32{-0.38, -0.12}, [2]float32{-0.4, 0.02}, [2]float32{-0.56, -0.04})
	pen.stroke(tint, 0.8, 0.05)
}

func drawOilFlaskPictogram(pen *logoPen, tint colorx.RGB) {
	pen.circle(0, 0.18, 0.38)
	pen.fill(tint, 1)
	pen.polygon([2]float32{-0.11, -0.48}, [2]float32{0.11, -0.48}, [2]float32{0.11, -0.14}, [2]float32{-0.11, -0.14})
	pen.fill(tint, 1)
	pen.polygon([2]float32{-0.15, -0.6}, [2]float32{0.15, -0.6}, [2]float32{0.15, -0.48}, [2]float32{-0.15, -0.48})
	pen.fill(logoWhite, 0.7)
	pen.arc(0, 0.18, 0.26, 3.6, 4.4)
	pen.stroke(logoWhite, 0.6, 0.06)
}

func drawDownpourPictogram(pen *logoPen, tint colorx.RGB) {
	pen.circle(-0.26, -0.12, 0.22)
	pen.circle(0.04, -0.24, 0.28)
	pen.circle(0.3, -0.1, 0.2)
	pen.polygon([2]float32{-0.48, -0.12}, [2]float32{0.5, -0.12}, [2]float32{0.5, 0.06}, [2]float32{-0.48, 0.06})
	pen.fill(tint, 1)
	for drop := range 3 {
		x := -0.26 + 0.26*float32(drop)
		pen.segment(x, 0.22, x-0.1, 0.52)
	}
	pen.stroke(tint, 1, 0.08)
}

func drawOrbitBladesPictogram(pen *logoPen, tint colorx.RGB) {
	pen.annulus(0, 0, 0.43, 0.47)
	pen.fill(tint, 0.35)
	pen.circle(0, 0, 0.12)
	pen.fill(tint, 1)
	for blade := range 3 {
		angle := float32(blade) * fullTurn / 3
		pen.polygon(
			[2]float32{0.3 * cosine(angle-0.45), 0.3 * sine(angle-0.45)},
			[2]float32{0.64 * cosine(angle+0.1), 0.64 * sine(angle+0.1)},
			[2]float32{0.46 * cosine(angle+0.35), 0.46 * sine(angle+0.35)},
		)
	}
	pen.fill(tint, 1)
}

func drawSeismicHammerPictogram(pen *logoPen, tint colorx.RGB) {
	pen.segment(0.08, 0.06, 0.66, -0.22)
	pen.stroke(tint, 0.8, 0.11)
	pen.polygon([2]float32{-0.17, -0.28}, [2]float32{0.17, -0.28}, [2]float32{0.17, 0.46}, [2]float32{-0.17, 0.46})
	pen.fill(tint, 1)
	pen.polygon([2]float32{-0.17, 0.36}, [2]float32{0.17, 0.36}, [2]float32{0.17, 0.46}, [2]float32{-0.17, 0.46})
	pen.fill(logoWhite, 0.45)
	pen.polyline([2]float32{-0.36, 0.58}, [2]float32{-0.48, 0.52}, [2]float32{-0.58, 0.6}, [2]float32{-0.7, 0.54})
	pen.polyline([2]float32{0.4, 0.58}, [2]float32{0.52, 0.52}, [2]float32{0.62, 0.6}, [2]float32{0.74, 0.54})
	pen.stroke(tint, 0.75, 0.05)
	pen.segment(-0.42, 0.22, -0.56, 0.12)
	pen.segment(0.46, 0.22, 0.6, 0.12)
	pen.stroke(tint, 0.6, 0.05)
}

func drawStaticMinesPictogram(pen *logoPen, tint colorx.RGB) {
	for spike := range 8 {
		angle := float32(spike) * fullTurn / 8
		pen.segment(cosine(angle)*0.28, sine(angle)*0.28, cosine(angle)*0.5, sine(angle)*0.5)
	}
	pen.stroke(tint, 1, 0.08)
	pen.circle(0, 0, 0.32)
	pen.fill(tint, 1)
	pen.circle(0, 0, 0.12)
	pen.fill(logoWhite, 0.9)
}

func drawHealPictogram(pen *logoPen, tint colorx.RGB) {
	pen.segment(0, -0.42, 0, 0.42)
	pen.segment(-0.42, 0, 0.42, 0)
	pen.stroke(tint, 1, 0.26)
}

func drawNukePictogram(pen *logoPen, tint colorx.RGB) {
	pen.circle(0, 0, 0.11)
	pen.fill(tint, 1)
	for blade := range 3 {
		center := -fullTurn/4 + float32(blade)*fullTurn/3
		pen.arc(0, 0, 0.58, center-0.52, center+0.52)
		centerX, centerY := pen.at(0, 0)
		pen.path.Arc(centerX, centerY, 0.2*pen.radius, center+0.52, center-0.52, vector.CounterClockwise)
		pen.path.Close()
	}
	pen.fill(tint, 1)
}
