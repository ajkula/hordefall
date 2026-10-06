package game

import (
	"hordefall/internal/config"
	"hordefall/internal/i18n"
)

// ===== Types =====

type Settings struct {
	IsMusicOn         bool             `json:"musicOn"`
	Tracks            map[string]bool  `json:"tracks"`
	IsFullscreen      bool             `json:"fullscreen"`
	IsVsyncOn         bool             `json:"vsync"`
	Resolution        RenderResolution `json:"resolution"`
	IsFPSShown        bool             `json:"showFps"`
	ShakeLevel        int              `json:"screenShake"`
	EffectsLevel      int              `json:"effects"`
	BloomLevel        int              `json:"bloom"`
	IsCRTOn           bool             `json:"crt"`
	MusicVolumeStep   int              `json:"musicVolume"`
	EffectsVolumeStep int              `json:"effectsVolume"`
	Initials          string           `json:"initials"`
	IsBenchmarkHidden bool             `json:"benchmarkHidden"`
	Language          string           `json:"language"`
}

// ===== Constants =====

const settingsFileName = "settings.json"

// ===== Public API =====

func DefaultSettings() Settings {
	return Settings{
		IsMusicOn: true, Tracks: map[string]bool{},
		IsFullscreen: false, IsVsyncOn: false, Resolution: ResolutionNative, IsFPSShown: false,
		ShakeLevel: defaultShakeLevel, EffectsLevel: defaultEffectsLevel, BloomLevel: defaultBloomLevel, IsCRTOn: true,
		MusicVolumeStep: defaultMusicVolumeStep, EffectsVolumeStep: defaultEffectsVolumeStep, Initials: defaultInitials, Language: i18n.DefaultCode,
	}
}

func LoadSettings() (Settings, error) {
	settings := DefaultSettings()
	err := config.Load(settingsFileName, &settings)
	settings.Tracks = ensureTracks(settings.Tracks)
	sanitizeGraphics(&settings)
	settings.MusicVolumeStep = clampInt(settings.MusicVolumeStep, 0, volumeSteps)
	settings.EffectsVolumeStep = clampInt(settings.EffectsVolumeStep, 0, volumeSteps)
	settings.Initials = sanitizeInitials(settings.Initials)
	return settings, err
}

func SaveSettings(settings Settings) error {
	return config.Save(settingsFileName, settings)
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
	g.applyMusicVolume()
	g.applyGraphicsSettings()
	g.applyLanguage()
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
		g.settingsMessage = i18n.F("settings.save_failed", err.Error())
	}
}
