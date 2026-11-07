package game

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/ajkula/ajkurig/i18n"
	"github.com/ajkula/ajkurig/ui"
)

var placeholderPattern = regexp.MustCompile(`%[-0-9.]*[sdf]`)

var translationCallPattern = regexp.MustCompile(`(?:i18n\.T|i18n\.F|i18n\.TOr)\("([a-z_.]+)"`)

func TestEveryKeyUsedByTheGameExistsInEnglish(t *testing.T) {
	if err := loadGameLanguages(); err != nil {
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
	for _, weather := range weatherDefinitions[1:] {
		keys["weather."+weather.Key] = true
	}
	for _, remap := range remaps {
		keys["action."+remap.Key] = true
	}
	for _, topic := range chatterTopics {
		for line := 1; line <= topic.LineCount; line++ {
			keys["tsukumo."+topic.Key+"."+strconv.Itoa(line)] = true
		}
	}
	for key := range keys {
		isPrefix := strings.HasSuffix(key, ".")
		if !isPrefix && i18n.T(key) == key {
			t.Errorf("missing English text for %s", key)
		}
	}
}

func TestEveryLanguageMatchesEnglish(t *testing.T) {
	if err := loadGameLanguages(); err != nil {
		t.Fatal(err)
	}
	languages := i18n.Languages()
	if len(languages) < 9 || languages[0].Code != i18n.FallbackCode {
		t.Fatalf("languages %v", languages)
	}
	english := languages[0].Strings
	for _, language := range languages {
		for key, text := range language.Strings {
			reference, exists := english[key]
			if !exists {
				t.Errorf("%s: unknown key %s", language.Code, key)
				continue
			}
			if !slices.Equal(placeholderPattern.FindAllString(reference, -1), placeholderPattern.FindAllString(text, -1)) {
				t.Errorf("%s: %s placeholders differ from English", language.Code, key)
			}
		}
	}
}

func TestEveryCharacterOfEveryLanguageHasAGlyph(t *testing.T) {
	if err := loadGameLanguages(); err != nil {
		t.Fatal(err)
	}
	for _, missing := range ui.FindMissingGlyphs(i18n.Languages(), ui.DefaultFontFiles()) {
		t.Errorf("no glyph for %s", missing)
	}
}
