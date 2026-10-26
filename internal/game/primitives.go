package game

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ajkula/ajkurig/draw"
)

// ===== Constants =====

var solidPixel *ebiten.Image

// ===== Internal =====

func ensureSolidPixel(existing *ebiten.Image) *ebiten.Image {
	if existing != nil {
		return existing
	}
	canvas := ebiten.NewImage(3, 3)
	canvas.Fill(color.White)
	return canvas.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}

func fillRect(screen *ebiten.Image, x, y, width, height float32, tint color.Color) {
	vector.FillRect(screen, x*draw.Scale, y*draw.Scale, width*draw.Scale, height*draw.Scale, tint, false)
}

func fillRoundedRect(screen *ebiten.Image, x, y, width, height, radius float32, tint color.Color) {
	if width <= 0 || height <= 0 {
		return
	}
	left, top, right, bottom := x*draw.Scale, y*draw.Scale, (x+width)*draw.Scale, (y+height)*draw.Scale
	corner := min(radius, width/2, height/2) * draw.Scale
	var path vector.Path
	path.MoveTo(left+corner, top)
	path.ArcTo(right, top, right, bottom, corner)
	path.ArcTo(right, bottom, left, bottom, corner)
	path.ArcTo(left, bottom, left, top, corner)
	path.ArcTo(left, top, right, top, corner)
	path.Close()
	options := &vector.DrawPathOptions{AntiAlias: true}
	options.ColorScale.ScaleWithColor(tint)
	vector.FillPath(screen, &path, &vector.FillOptions{}, options)
}
