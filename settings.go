package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ===== Types =====

type Settings struct {
	IsMusicOn bool            `json:"musicOn"`
	Tracks    map[string]bool `json:"tracks"`
}

// ===== Constants =====

const settingsFileName = "settings.json"

// ===== Public API =====

func DefaultSettings() Settings {
	return Settings{IsMusicOn: true, Tracks: map[string]bool{}}
}

func LoadSettings() (Settings, error) {
	settings := DefaultSettings()
	path, err := configFilePath(settingsFileName)
	if err != nil {
		return settings, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return settings, err
	}
	err = json.Unmarshal(content, &settings)
	settings.Tracks = ensureTracks(settings.Tracks)
	return settings, err
}

func SaveSettings(settings Settings) error {
	path, err := configFilePath(settingsFileName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

// ===== Internal =====

func ensureTracks(tracks map[string]bool) map[string]bool {
	if tracks == nil {
		return map[string]bool{}
	}
	return tracks
}

func (g *Game) loadSettings() {
	settings, err := LoadSettings()
	g.settings = settings
	if err != nil {
		g.settings = DefaultSettings()
	}
	g.audio.SetMusicOn(g.settings.IsMusicOn)
	g.buildPlaylistMenu()
}

func (g *Game) toggleMusic() {
	g.settings.IsMusicOn = !g.settings.IsMusicOn
	g.audio.SetMusicOn(g.settings.IsMusicOn)
	g.saveSettings()
}

func (g *Game) saveSettings() {
	g.settingsMessage = ""
	if err := SaveSettings(g.settings); err != nil {
		g.settingsMessage = "Settings not saved: " + err.Error()
	}
}
