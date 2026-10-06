package game

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/image/font/sfnt"

	"hordefall/internal/i18n"
)

var translationCallPattern = regexp.MustCompile(`(?:i18n\.T|i18n\.F|i18n\.TOr|translatedLabel)\("([a-z_.]+)"`)

func TestEveryKeyUsedByTheGameExistsInEnglish(t *testing.T) {
	if err := i18n.Load(); err != nil {
		t.Fatal(err)
	}
	i18n.Use(i18n.FallbackCode)
	keys := map[string]bool{"menu.music_off": true, "menu.music_on": true, "remap.verb_key": true, "remap.verb_button": true,
		"device.keyboard": true, "device.gamepad": true, "demo.siege": true, "demo.arsenal": true, "remap.custom_loaded": true, "remap.default_loaded": true}
	files, _ := filepath.Glob("*.go")
	for _, file := range files {
		source, _ := os.ReadFile(file)
		for _, match := range translationCallPattern.FindAllStringSubmatch(string(source), -1) {
			keys[match[1]] = true
		}
	}
	for _, weapon := range weaponTable {
		keys["weapon."+weapon.Key], keys["weapon."+weapon.Key+".description"] = true, true
	}
	for _, passive := range passiveTable {
		keys["passive."+passive.Key], keys["passive."+passive.Key+".description"] = true, true
	}
	for _, weather := range weatherTable[1:] {
		keys["weather."+weather.Key] = true
	}
	for _, action := range remapTable {
		keys["action."+action.Key] = true
	}
	for key := range keys {
		isPrefix := strings.HasSuffix(key, ".")
		if !isPrefix && i18n.T(key) == key {
			t.Errorf("missing English text for %s", key)
		}
	}
}

func TestEveryCharacterOfEveryLanguageHasAGlyph(t *testing.T) {
	if err := i18n.Load(); err != nil {
		t.Fatal(err)
	}
	fonts := []*sfnt.Font{mustParseFont(t, rajdhaniMedium)}
	for _, files := range fallbackFamilyFiles {
		data, err := fallbackFonts.ReadFile(files[WeightMedium])
		if err != nil {
			t.Fatal(err)
		}
		fonts = append(fonts, mustParseFont(t, data))
	}
	var buffer sfnt.Buffer
	for _, language := range i18n.Languages() {
		text := language.Name
		for _, value := range language.Strings {
			text += value
		}
		for _, character := range text {
			if unicode.IsSpace(character) || isCombiningMark(character) || hasGlyph(fonts, &buffer, character) {
				continue
			}
			t.Errorf("%s: no glyph for %q (U+%04X)", language.Code, character, character)
		}
	}
}

func mustParseFont(t *testing.T, data []byte) *sfnt.Font {
	font, err := sfnt.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return font
}

func hasGlyph(fonts []*sfnt.Font, buffer *sfnt.Buffer, character rune) bool {
	for _, font := range fonts {
		index, err := font.GlyphIndex(buffer, character)
		if err == nil && index != 0 {
			return true
		}
	}
	return false
}

func isCombiningMark(character rune) bool {
	return unicode.Is(unicode.Mn, character) || unicode.Is(unicode.Mc, character)
}
