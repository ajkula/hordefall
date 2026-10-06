package game

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"hordefall/internal/i18n"
)

type FitRule struct {
	Prefix   string
	Size     float64
	IsBold   bool
	MaxWidth float64
	MaxLines int
}

var fitRules = []FitRule{
	{"title.", 80, true, 1200, 1},
	{"menu.", 24, true, 560, 1},
	{"gameplay.", 24, true, 560, 1},
	{"remap.reset", 24, true, 560, 1},
	{"audio.", 24, true, 560, 1},
	{"graphics.", 24, true, 560, 1},
	{"hint.", 16, false, 1240, 1},
	{"main.controls", 16, false, 1240, 1},
	{"remap.list_hint", 16, false, 1240, 1},
	{"remap.essentials_hint", 16, false, 1240, 1},
	{"remap.listen_hint", 16, false, 1240, 1},
	{"gameover.", 24, true, 1240, 1},
	{"upgrade.", 16, false, descriptionWrapWidth, 1},
}

func TestTranslationsFitTheirSpace(t *testing.T) {
	if err := i18n.Load(); err != nil {
		t.Fatal(err)
	}
	chains := NewFontChains(FontFamily{mustLoadFace(rajdhaniMedium), mustLoadFace(rajdhaniBold)})
	for _, language := range i18n.Languages() {
		i18n.Use(language.Code)
		for key, value := range language.Strings {
			checkFit(t, chains, language.Code, key, value)
		}
	}
	i18n.Use(i18n.FallbackCode)
}

func checkFit(t *testing.T, chains *FontChains, code, key, value string) {
	isCardName := (strings.HasPrefix(key, "weapon.") || strings.HasPrefix(key, "passive.")) && !strings.HasSuffix(key, ".description")
	isDescription := strings.HasSuffix(key, ".description")
	rules := fitRules
	if isCardName {
		rules = []FitRule{{key, 24, true, descriptionWrapWidth, 1}}
	}
	if isDescription {
		rules = []FitRule{{key, 20, false, descriptionWrapWidth, 2}}
	}
	for _, rule := range rules {
		if !strings.HasPrefix(key, rule.Prefix) {
			continue
		}
		face := measureFace(chains, rule)
		lines := wrapMeasured(value, face, rule.MaxWidth)
		widest := 0.0
		for _, line := range lines {
			widest = max(widest, text.Advance(line, face))
		}
		if len(lines) > rule.MaxLines || widest > rule.MaxWidth {
			t.Errorf("%s %s: %d lines, %.0f px wide (max %d lines, %.0f px): %q", code, key, len(lines), widest, rule.MaxLines, rule.MaxWidth, value)
		}
		return
	}
}

func measureFace(chains *FontChains, rule FitRule) text.Face {
	source := chains.familyFor(rule.IsBold)
	return combineFaces(chains.Faces(source, rule.Size))
}

func wrapMeasured(message string, face text.Face, maximumWidth float64) []string {
	lines := []string{}
	current := ""
	for _, token := range wrapTokens(message) {
		separator := [2]string{"", " "}[boolToIndex(token.IsSpaced && current != "")]
		if current != "" && text.Advance(current+separator+token.Text, face) > maximumWidth {
			lines = append(lines, current)
			current, separator = "", ""
		}
		current += separator + token.Text
	}
	return append(lines, current)
}
