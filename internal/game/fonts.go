package game

import (
	"embed"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"hordefall/internal/i18n"
)

// ===== Types =====

type FontWeight uint8

type FontFamily [fontWeightCount]*text.GoTextFaceSource

type FontChains struct {
	families map[string]FontFamily
	weights  map[*text.GoTextFaceSource]FontWeight
}

// ===== Constants =====

const (
	WeightMedium FontWeight = iota
	WeightBold
	fontWeightCount
)

//go:embed fonts/Exo2-*.ttf fonts/NotoSans*.ttf
var fallbackFonts embed.FS

var fallbackFamilyFiles = map[string][fontWeightCount]string{
	"cyrillic": {"fonts/Exo2-Medium.ttf", "fonts/Exo2-Bold.ttf"},
	"hangul":   {"fonts/NotoSansKR-Medium.ttf", "fonts/NotoSansKR-Bold.ttf"},
	"han-jp":   {"fonts/NotoSansJP-Medium.ttf", "fonts/NotoSansJP-Bold.ttf"},
	"han-sc":   {"fonts/NotoSansSC-Medium.ttf", "fonts/NotoSansSC-Bold.ttf"},
}

var scriptFamilyOrders = map[string][]string{
	"han-jp": {"cyrillic", "han-jp", "han-sc", "hangul", "go"},
	"hangul": {"cyrillic", "hangul", "han-jp", "han-sc", "go"},
}

var defaultFamilyOrder = []string{"cyrillic", "han-sc", "han-jp", "hangul", "go"}

// ===== Public API =====

func NewFontChains(primary FontFamily) *FontChains {
	chains := &FontChains{
		families: map[string]FontFamily{"primary": primary},
		weights:  map[*text.GoTextFaceSource]FontWeight{primary[WeightMedium]: WeightMedium, primary[WeightBold]: WeightBold},
	}
	chains.families["go"] = FontFamily{loadFaceSourceOrNil(goregular.TTF), loadFaceSourceOrNil(gobold.TTF)}
	for script, files := range fallbackFamilyFiles {
		chains.families[script] = FontFamily{loadEmbeddedFace(files[WeightMedium]), loadEmbeddedFace(files[WeightBold])}
	}
	return chains
}

func (c *FontChains) Faces(source *text.GoTextFaceSource, size float64) []text.Face {
	weight := c.weights[source]
	order, isSpecific := scriptFamilyOrders[i18n.Current().Script]
	order = [2][]string{defaultFamilyOrder, order}[boolToIndex(isSpecific)]
	faces := []text.Face{&text.GoTextFace{Source: source, Size: size}}
	for _, name := range order {
		faces = appendFaceIf(faces, c.families[name][weight], size)
	}
	return faces
}

// ===== Internal =====

func (c *FontChains) familyFor(isBold bool) *text.GoTextFaceSource {
	return c.families["primary"][boolToIndex(isBold)]
}

func appendFaceIf(faces []text.Face, source *text.GoTextFaceSource, size float64) []text.Face {
	if source == nil {
		return faces
	}
	return append(faces, &text.GoTextFace{Source: source, Size: size})
}

func loadEmbeddedFace(path string) *text.GoTextFaceSource {
	data, err := fallbackFonts.ReadFile(path)
	if err != nil {
		return nil
	}
	return loadFaceSourceOrNil(data)
}
