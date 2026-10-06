package i18n

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
)

var placeholderPattern = regexp.MustCompile(`%[-0-9.]*[sdf]`)

func TestEveryLanguageMatchesEnglish(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	english := languages[FallbackCode]
	for _, language := range Languages() {
		for key, text := range language.Strings {
			reference, exists := english.Strings[key]
			if !exists {
				t.Errorf("%s: unknown key %s", language.Code, key)
				continue
			}
			if !slices.Equal(placeholderPattern.FindAllString(reference, -1), placeholderPattern.FindAllString(text, -1)) {
				t.Errorf("%s: %s placeholders differ from English", language.Code, key)
			}
		}
	}
	if len(Languages()) < 9 || Languages()[0].Code != FallbackCode {
		t.Fatalf("languages %v", Languages())
	}
}

func TestMissingKeysFallBackToEnglishThenToTheKey(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	Use("fr")
	delete(languages["fr"].Strings, "menu.play")
	if T("menu.play") != "Play" || T("no.such.key") != "no.such.key" || TOr("no.such.key", "x") != "x" {
		t.Fatalf("fallbacks: %q %q", T("menu.play"), T("no.such.key"))
	}
	if Use("xx") || Current().Code != FallbackCode {
		t.Fatal("an unknown language did not fall back to English")
	}
}

func TestExternalFilesAddAndOverrideLanguages(t *testing.T) {
	directory := t.TempDir()
	folder := filepath.Join(directory, externalFolder)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"pt.json":     `{"code":"pt","name":"Português","script":"latin","strings":{"menu.play":"Jogar"}}`,
		"fr.json":     `{"code":"fr","name":"Français","strings":{"menu.quit":"Ciao"}}`,
		"broken.json": `{"code":`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(folder, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Load(directory); err != nil {
		t.Fatal(err)
	}
	Use("pt")
	if T("menu.play") != "Jogar" || T("menu.quit") != "Quit" {
		t.Fatalf("external language: %q %q", T("menu.play"), T("menu.quit"))
	}
	Use("fr")
	if T("menu.quit") != "Ciao" || T("menu.options") != "Options" {
		t.Fatalf("override: %q", T("menu.quit"))
	}
}

func TestNothingCrashesWithoutLanguages(t *testing.T) {
	saved, savedCurrent, savedFallback := languages, current, fallback
	defer func() { languages, current, fallback = saved, savedCurrent, savedFallback }()
	languages, current, fallback = map[string]*Language{}, nil, nil
	if T("menu.play") != "menu.play" || F("hud.level", 3) == "" || Current().Code != FallbackCode || Use("fr") {
		t.Fatal("empty catalog misbehaved")
	}
}
