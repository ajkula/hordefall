package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// ===== Types =====

type PostProcessor struct {
	brightPass *ebiten.Shader
	blur       *ebiten.Shader
	crt        *ebiten.Shader
	scene      *ebiten.Image
	bloom      *ebiten.Image
	bloomSwap  *ebiten.Image
	isActive   bool
}

type PostSettings struct {
	BloomIntensity float32
	IsCRTOn        bool
}

// ===== Constants =====

const (
	bloomDownscale      = 4
	bloomThreshold      = 0.42
	bloomBlurPasses     = 3
	crtCurvature        = 0.035
	crtScanlineDepth    = 0.3
	crtMaskStrength     = 0.1
	crtAberrationPixels = 1.1
	crtVignette         = 0.28
	crtBrightness       = 1.12
	crtLinesPerScreen   = 360
)

var bloomLevels = [graphicsLevelCount]GraphicsLevel{{"Off", 0}, {"Soft", 1}, {"Strong", 2}}

const brightPassSource = `//kage:unit pixels
package main

var Threshold float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	sample := imageSrc0At(srcPos)
	luma := dot(sample.rgb, vec3(0.299, 0.587, 0.114))
	weight := clamp((luma-Threshold)/(1-Threshold), 0, 1)
	return vec4(sample.rgb*weight, weight)
}
`

const blurSource = `//kage:unit pixels
package main

var Direction vec2

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	sum := imageSrc0At(srcPos) * 0.227027
	sum += (imageSrc0At(srcPos+Direction) + imageSrc0At(srcPos-Direction)) * 0.1945946
	sum += (imageSrc0At(srcPos+Direction*2) + imageSrc0At(srcPos-Direction*2)) * 0.1216216
	sum += (imageSrc0At(srcPos+Direction*3) + imageSrc0At(srcPos-Direction*3)) * 0.054054
	sum += (imageSrc0At(srcPos+Direction*4) + imageSrc0At(srcPos-Direction*4)) * 0.016216
	return sum
}
`

const crtSource = `//kage:unit pixels
package main

var Curvature float
var ScanlinePeriod float
var ScanlineDepth float
var MaskStrength float
var MaskWidth float
var Aberration float
var Vignette float
var Brightness float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	origin := imageSrc0Origin()
	size := imageSrc0Size()
	centered := (srcPos-origin)/size*2 - 1
	centered *= 1 + Curvature*dot(centered, centered)
	inside := step(abs(centered.x), 1) * step(abs(centered.y), 1)
	warped := (centered*0.5+0.5)*size + origin
	shift := vec2(centered.x*Aberration, 0)
	rgb := vec3(imageSrc0At(warped+shift).r, imageSrc0At(warped).g, imageSrc0At(warped-shift).b)
	scanline := 0.5 + 0.5*cos(dstPos.y/ScanlinePeriod*6.2831853)
	rgb *= 1 - ScanlineDepth*(1-scanline)
	column := mod(floor(dstPos.x/MaskWidth), 3)
	mask := vec3(step(column, 0.5), step(abs(column-1), 0.5), step(1.5, column))
	rgb *= 1 - MaskStrength + MaskStrength*1.6*mask
	edge := centered * centered
	rgb *= 1 - Vignette*(edge.x+edge.y)*0.5
	return vec4(rgb*Brightness*inside, 1)
}
`

// ===== Public API =====

func NewPostProcessor() *PostProcessor {
	return &PostProcessor{
		brightPass: mustCompileShader(brightPassSource),
		blur:       mustCompileShader(blurSource),
		crt:        mustCompileShader(crtSource),
	}
}

func (p *PostProcessor) Begin(screen *ebiten.Image, settings PostSettings) *ebiten.Image {
	p.isActive = settings.BloomIntensity > 0 || settings.IsCRTOn
	if !p.isActive {
		return screen
	}
	p.ensureTargets(screen.Bounds().Dx(), screen.Bounds().Dy())
	p.scene.Clear()
	return p.scene
}

func (p *PostProcessor) Finish(screen *ebiten.Image, settings PostSettings) {
	if !p.isActive {
		return
	}
	finishers := [2]func(p *PostProcessor, screen *ebiten.Image){(*PostProcessor).copyScene, (*PostProcessor).applyCRT}
	finishers[boolToIndex(settings.IsCRTOn)](p, screen)
}

func (p *PostProcessor) ApplyBloom(intensity float32) {
	if !p.isActive || intensity <= 0 {
		return
	}
	downscale := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	downscale.GeoM.Scale(1.0/bloomDownscale, 1.0/bloomDownscale)
	p.bloomSwap.Clear()
	p.bloomSwap.DrawImage(p.scene, downscale)
	p.drawShaderPass(p.bloom, p.bloomSwap, p.brightPass, map[string]any{"Threshold": float32(bloomThreshold)})
	for range bloomBlurPasses {
		p.drawShaderPass(p.bloomSwap, p.bloom, p.blur, map[string]any{"Direction": []float32{1, 0}})
		p.drawShaderPass(p.bloom, p.bloomSwap, p.blur, map[string]any{"Direction": []float32{0, 1}})
	}
	upscale := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
	upscale.GeoM.Scale(bloomDownscale, bloomDownscale)
	upscale.ColorScale.Scale(intensity, intensity, intensity, intensity)
	p.scene.DrawImage(p.bloom, upscale)
}

// ===== Internal =====

func mustCompileShader(source string) *ebiten.Shader {
	shader, err := ebiten.NewShader([]byte(source))
	if err != nil {
		panic(err)
	}
	return shader
}

func (p *PostProcessor) ensureTargets(width, height int) {
	isSized := p.scene != nil && p.scene.Bounds().Dx() == width && p.scene.Bounds().Dy() == height
	if isSized {
		return
	}
	p.disposeTargets()
	bloomWidth, bloomHeight := max(1, width/bloomDownscale), max(1, height/bloomDownscale)
	p.scene = ebiten.NewImage(width, height)
	p.bloom = ebiten.NewImage(bloomWidth, bloomHeight)
	p.bloomSwap = ebiten.NewImage(bloomWidth, bloomHeight)
}

func (p *PostProcessor) disposeTargets() {
	if p.scene == nil {
		return
	}
	p.scene.Deallocate()
	p.bloom.Deallocate()
	p.bloomSwap.Deallocate()
}

func (p *PostProcessor) drawShaderPass(destination, source *ebiten.Image, shader *ebiten.Shader, uniforms map[string]any) {
	options := &ebiten.DrawRectShaderOptions{Uniforms: uniforms, Blend: ebiten.BlendCopy}
	options.Images[0] = source
	bounds := source.Bounds()
	destination.DrawRectShader(bounds.Dx(), bounds.Dy(), shader, options)
}

func (p *PostProcessor) copyScene(screen *ebiten.Image) {
	screen.DrawImage(p.scene, nil)
}

func (p *PostProcessor) applyCRT(screen *ebiten.Image) {
	p.drawShaderPass(screen, p.scene, p.crt, map[string]any{
		"Curvature":      float32(crtCurvature),
		"ScanlinePeriod": float32(p.scene.Bounds().Dy()) / crtLinesPerScreen,
		"ScanlineDepth":  float32(crtScanlineDepth),
		"MaskStrength":   float32(crtMaskStrength),
		"MaskWidth":      max(1, renderScale),
		"Aberration":     crtAberrationPixels * renderScale,
		"Vignette":       float32(crtVignette),
		"Brightness":     float32(crtBrightness),
	})
}
