package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ===== Types =====

type WormSprites struct {
	Segments [wormSpriteAngles]*ebiten.Image
	Heads    [wormSpriteAngles]*ebiten.Image
}

type WormMaterial uint8

type wormPixelShape func(along, across float32) (WormMaterial, float32)

// ===== Constants =====

const (
	MaterialNone WormMaterial = iota
	MaterialMetal
	MaterialPlate
	MaterialBright
	MaterialVent
	MaterialCore
	MaterialSensor
	MaterialOutline
	materialCount
)

const (
	wormSpriteAngles      = 32
	wormSpritePixel       = 2
	wormDepthSteps        = 8
	wormSegmentCanvas     = 30
	wormHeadCanvas        = 60
	wormSegmentArtLength  = 13
	wormSegmentArtRadius  = 11
	wormHeadArtRadius     = 15
	wormShadeLevels       = 6
	wormPlateShadeDrop    = 2
	wormBrightShadeRaise  = 2
	wormLightLength       = 1.05
	wormDitherAmplitude   = 0.5
	wormHeadBodyBack      = -11
	wormHeadBodyFront     = 6
	wormHeadSnoutFront    = 14
	wormHeadSnoutRadius   = 10
	wormMandibleRoot      = 10
	wormMandibleTip       = 24
	wormEmitterCenter     = 15
	wormEmitterRadius     = 3
	wormSensorOffset      = 9
	wormSegmentRingInset  = 2.5
	wormSegmentRivetInset = 4.5
)

var wormMetalShades = [wormShadeLevels][4]uint8{
	{30, 34, 48, 255}, {54, 62, 84, 255}, {86, 98, 126, 255},
	{128, 142, 170, 255}, {184, 196, 218, 255}, {240, 246, 255, 255},
}

var wormFlatColors = [materialCount][4]uint8{
	MaterialVent:    {70, 215, 255, 255},
	MaterialCore:    {225, 255, 255, 255},
	MaterialSensor:  {255, 48, 30, 255},
	MaterialOutline: {8, 8, 14, 255},
}

var wormShadeOffsets = [materialCount]int{MaterialPlate: -wormPlateShadeDrop, MaterialBright: wormBrightShadeRaise}

var wormIsShaded = [materialCount]bool{MaterialMetal: true, MaterialPlate: true, MaterialBright: true}

var wormBayer = [4]float32{0, 0.5, 0.75, 0.25}

var wormLight = [3]float32{-0.45 / wormLightLength, -0.55 / wormLightLength, 0.7 / wormLightLength}

// ===== Public API =====

func NewWormSprites() *WormSprites {
	sprites := &WormSprites{}
	for angle := range wormSpriteAngles {
		heading := float32(angle) * 2 * math.Pi / wormSpriteAngles
		sprites.Segments[angle] = rasterizeWormSprite(wormSegmentCanvas, heading, wormSegmentShape)
		sprites.Heads[angle] = rasterizeWormSprite(wormHeadCanvas, heading, wormHeadShape)
	}
	return sprites
}

func wormSpriteAngleIndex(axisX, axisY float32) int {
	turns := atan2(axisY, axisX) / (2 * math.Pi)
	return (int(math.Round(float64(turns*wormSpriteAngles))) + wormSpriteAngles) % wormSpriteAngles
}

func wormDepthScale(height float32) float32 {
	return float32(math.Round(float64(wormPerspective(height)*wormDepthSteps))) / wormDepthSteps
}

// ===== Internal =====

func rasterizeWormSprite(canvas int, heading float32, shape wormPixelShape) *ebiten.Image {
	materials := make([]WormMaterial, canvas*canvas)
	normals := make([]float32, canvas*canvas)
	cosineHeading, sineHeading := cosine(heading), sine(heading)
	center := float32(canvas) / 2
	for index := range materials {
		offsetX, offsetY := float32(index%canvas)+0.5-center, float32(index/canvas)+0.5-center
		along := offsetX*cosineHeading + offsetY*sineHeading
		across := -offsetX*sineHeading + offsetY*cosineHeading
		materials[index], normals[index] = shape(along, across)
	}
	outlineWormSprite(materials, canvas)
	pixels := make([]byte, canvas*canvas*4)
	for index, material := range materials {
		color := wormPixelColor(material, normals[index], -sineHeading, cosineHeading, index%canvas, index/canvas)
		copy(pixels[index*4:], color[:])
	}
	image := ebiten.NewImage(canvas, canvas)
	image.WritePixels(pixels)
	return image
}

func outlineWormSprite(materials []WormMaterial, canvas int) {
	isFilled := make([]bool, len(materials))
	for index, material := range materials {
		isFilled[index] = material != MaterialNone
	}
	for index := range materials {
		x, y := index%canvas, index/canvas
		hasNeighbor := isFilledAt(isFilled, canvas, x-1, y) || isFilledAt(isFilled, canvas, x+1, y) || isFilledAt(isFilled, canvas, x, y-1) || isFilledAt(isFilled, canvas, x, y+1)
		isEdge := !isFilled[index] && hasNeighbor
		materials[index] = [2]WormMaterial{materials[index], MaterialOutline}[boolToIndex(isEdge)]
	}
}

func isFilledAt(isFilled []bool, canvas, x, y int) bool {
	isInside := x >= 0 && y >= 0 && x < canvas && y < canvas
	return isInside && isFilled[clampInt(y, 0, canvas-1)*canvas+clampInt(x, 0, canvas-1)]
}

func wormPixelColor(material WormMaterial, across, normalX, normalY float32, x, y int) [4]uint8 {
	if !wormIsShaded[material] {
		return wormFlatColors[material]
	}
	depth := sqrt(max(0, 1-across*across))
	lambert := max(0, normalX*across*wormLight[0]+normalY*across*wormLight[1]+depth*wormLight[2])
	dither := (wormBayer[(y%2)*2+x%2] - 0.5) * wormDitherAmplitude
	level := int(lambert*(wormShadeLevels-1)+0.5+dither) + wormShadeOffsets[material]
	return wormMetalShades[clampInt(level, 0, wormShadeLevels-1)]
}

func wormSegmentShape(along, across float32) (WormMaterial, float32) {
	half := float32(wormSegmentArtLength) / 2
	isInside := abs(along) <= half && abs(across) <= wormSegmentArtRadius
	if !isInside {
		return MaterialNone, 0
	}
	normal := across / wormSegmentArtRadius
	isRing := abs(abs(along)-(half-wormSegmentRingInset)) < 0.75
	isRivet := abs(abs(along)-(half-wormSegmentRivetInset)) < 0.8 && abs(across+wormSegmentArtRadius*0.55) < 0.9
	isVent := abs(along) < 2.2 && abs(across-wormSegmentArtRadius*0.45) < 1.1
	material := [2]WormMaterial{MaterialMetal, MaterialPlate}[boolToIndex(isRing)]
	material = [2]WormMaterial{material, MaterialBright}[boolToIndex(isRivet)]
	material = [2]WormMaterial{material, MaterialVent}[boolToIndex(isVent)]
	return material, normal
}

func wormHeadShape(along, across float32) (WormMaterial, float32) {
	isBody := along >= wormHeadBodyBack && along <= wormHeadBodyFront && abs(across) <= wormHeadArtRadius
	snoutRadius := wormHeadSnoutRadius - (along-wormHeadBodyFront)*0.4
	isSnout := along > wormHeadBodyFront && along <= wormHeadSnoutFront && abs(across) <= snoutRadius
	mandibleCenter := wormSensorOffset - (along-wormMandibleRoot)*0.45
	mandibleWidth := 2.4 - (along-wormMandibleRoot)*0.09
	isMandible := along >= wormMandibleRoot && along <= wormMandibleTip && abs(abs(across)-mandibleCenter) <= mandibleWidth
	emitterDistance := length(along-wormEmitterCenter, across)
	isEmitter := emitterDistance <= wormEmitterRadius
	isCore := emitterDistance <= 1.3
	isSensor := along >= -4 && along <= 2 && abs(abs(across)-wormSensorOffset) <= 1.2
	isCrest := abs(across) <= 2 && along >= wormHeadBodyBack+1 && along <= 3
	isRing := (abs(along+8) < 0.75 || abs(along-3) < 0.75) && isBody
	material := MaterialNone
	material = [2]WormMaterial{material, MaterialMetal}[boolToIndex(isBody || isSnout)]
	material = [2]WormMaterial{material, MaterialPlate}[boolToIndex(isCrest || isRing)]
	material = [2]WormMaterial{material, MaterialBright}[boolToIndex(isMandible)]
	material = [2]WormMaterial{material, MaterialSensor}[boolToIndex(isSensor)]
	material = [2]WormMaterial{material, MaterialVent}[boolToIndex(isEmitter)]
	material = [2]WormMaterial{material, MaterialCore}[boolToIndex(isCore)]
	radius := [2]float32{wormHeadArtRadius, wormHeadSnoutRadius}[boolToIndex(isSnout || isMandible)]
	return material, clamp(across/radius, -1, 1)
}
