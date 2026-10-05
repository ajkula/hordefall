package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ===== Types =====

type PixelGlyph [pixelGlyphRows]string

type PixelBatch struct {
	vertices []ebiten.Vertex
	indices  []uint32
}

// ===== Constants =====

const (
	pixelGlyphRows    = 7
	pixelGlyphColumns = 5
	pixelGlyphAdvance = pixelGlyphColumns + 1
	pixelShadowAlpha  = 0.85
)

var pixelGlyphs = map[rune]PixelGlyph{
	'0': {" ### ", "#   #", "#  ##", "# # #", "##  #", "#   #", " ### "},
	'1': {"  #  ", " ##  ", "  #  ", "  #  ", "  #  ", "  #  ", " ### "},
	'2': {" ### ", "#   #", "    #", "   # ", "  #  ", " #   ", "#####"},
	'3': {"#####", "   # ", "  #  ", "   # ", "    #", "#   #", " ### "},
	'4': {"   # ", "  ## ", " # # ", "#  # ", "#####", "   # ", "   # "},
	'5': {"#####", "#    ", "#### ", "    #", "    #", "#   #", " ### "},
	'6': {"  ## ", " #   ", "#    ", "#### ", "#   #", "#   #", " ### "},
	'7': {"#####", "    #", "   # ", "  #  ", " #   ", " #   ", " #   "},
	'8': {" ### ", "#   #", "#   #", " ### ", "#   #", "#   #", " ### "},
	'9': {" ### ", "#   #", "#   #", " ####", "    #", "   # ", " ##  "},
	'A': {" ### ", "#   #", "#   #", "#####", "#   #", "#   #", "#   #"},
	'B': {"#### ", "#   #", "#   #", "#### ", "#   #", "#   #", "#### "},
	'C': {" ### ", "#   #", "#    ", "#    ", "#    ", "#   #", " ### "},
	'D': {"#### ", "#   #", "#   #", "#   #", "#   #", "#   #", "#### "},
	'E': {"#####", "#    ", "#    ", "#### ", "#    ", "#    ", "#####"},
	'F': {"#####", "#    ", "#    ", "#### ", "#    ", "#    ", "#    "},
	'G': {" ### ", "#   #", "#    ", "# ###", "#   #", "#   #", " ####"},
	'H': {"#   #", "#   #", "#   #", "#####", "#   #", "#   #", "#   #"},
	'I': {" ### ", "  #  ", "  #  ", "  #  ", "  #  ", "  #  ", " ### "},
	'J': {"  ###", "   # ", "   # ", "   # ", "   # ", "#  # ", " ##  "},
	'K': {"#   #", "#  # ", "# #  ", "##   ", "# #  ", "#  # ", "#   #"},
	'L': {"#    ", "#    ", "#    ", "#    ", "#    ", "#    ", "#####"},
	'M': {"#   #", "## ##", "# # #", "# # #", "#   #", "#   #", "#   #"},
	'N': {"#   #", "#   #", "##  #", "# # #", "#  ##", "#   #", "#   #"},
	'O': {" ### ", "#   #", "#   #", "#   #", "#   #", "#   #", " ### "},
	'P': {"#### ", "#   #", "#   #", "#### ", "#    ", "#    ", "#    "},
	'Q': {" ### ", "#   #", "#   #", "#   #", "# # #", "#  # ", " ## #"},
	'R': {"#### ", "#   #", "#   #", "#### ", "# #  ", "#  # ", "#   #"},
	'S': {" ####", "#    ", "#    ", " ### ", "    #", "    #", "#### "},
	'T': {"#####", "  #  ", "  #  ", "  #  ", "  #  ", "  #  ", "  #  "},
	'U': {"#   #", "#   #", "#   #", "#   #", "#   #", "#   #", " ### "},
	'V': {"#   #", "#   #", "#   #", "#   #", "#   #", " # # ", "  #  "},
	'W': {"#   #", "#   #", "#   #", "# # #", "# # #", "# # #", " # # "},
	'X': {"#   #", "#   #", " # # ", "  #  ", " # # ", "#   #", "#   #"},
	'Y': {"#   #", "#   #", " # # ", "  #  ", "  #  ", "  #  ", "  #  "},
	'Z': {"#####", "    #", "   # ", "  #  ", " #   ", "#    ", "#####"},
	',': {"     ", "     ", "     ", "     ", "     ", "  #  ", " #   "},
	':': {"     ", "  #  ", "  #  ", "     ", "  #  ", "  #  ", "     "},
	'.': {"     ", "     ", "     ", "     ", "     ", " ##  ", " ##  "},
	'-': {"     ", "     ", "     ", " ### ", "     ", "     ", "     "},
	'!': {"  #  ", "  #  ", "  #  ", "  #  ", "  #  ", "     ", "  #  "},
}

var pixelAlignShift = map[text.Align]float32{text.AlignStart: 0, text.AlignCenter: 0.5, text.AlignEnd: 1}

// ===== Public API =====

func PixelTextWidth(message string, pixel float32) float32 {
	count := len([]rune(message))
	return max(0, float32(count*pixelGlyphAdvance-1)*pixel)
}

func (b *PixelBatch) DrawText(screen *ebiten.Image, message string, x, y, pixel float32, tint [3]float32, alpha float32, align text.Align) {
	left := x - PixelTextWidth(message, pixel)*pixelAlignShift[align]
	b.queueText(message, left+pixel, y+pixel, pixel, outlineColor, alpha*pixelShadowAlpha)
	b.queueText(message, left, y, pixel, tint, alpha)
	b.flush(screen)
}

// ===== Internal =====

func (b *PixelBatch) queueText(message string, left, top, pixel float32, tint [3]float32, alpha float32) {
	for index, character := range []rune(message) {
		glyph := pixelGlyphs[character]
		b.queueGlyph(&glyph, left+float32(index*pixelGlyphAdvance)*pixel, top, pixel, tint, alpha)
	}
}

func (b *PixelBatch) queueGlyph(glyph *PixelGlyph, left, top, pixel float32, tint [3]float32, alpha float32) {
	for row, line := range glyph {
		for column := range len(line) {
			b.queuePixelIf(left+float32(column)*pixel, top+float32(row)*pixel, pixel, tint, alpha, line[column] == '#')
		}
	}
}

func (b *PixelBatch) queuePixelIf(x, y, pixel float32, tint [3]float32, alpha float32, isLit bool) {
	if !isLit {
		return
	}
	base := uint32(len(b.vertices))
	for corner := range 4 {
		b.vertices = append(b.vertices, ebiten.Vertex{
			DstX: (x + float32(corner&1)*pixel) * renderScale, DstY: (y + float32(corner>>1)*pixel) * renderScale,
			SrcX: 1.5, SrcY: 1.5,
			ColorR: tint[0] * alpha, ColorG: tint[1] * alpha, ColorB: tint[2] * alpha, ColorA: alpha,
		})
	}
	b.indices = append(b.indices, base, base+1, base+2, base+1, base+3, base+2)
}

func (b *PixelBatch) flush(screen *ebiten.Image) {
	solidPixel = ensureSolidPixel(solidPixel)
	screen.DrawTriangles32(b.vertices, b.indices, solidPixel, &ebiten.DrawTrianglesOptions{})
	b.vertices, b.indices = b.vertices[:0], b.indices[:0]
}
