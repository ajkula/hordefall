package game

import (
	"errors"
	"io/fs"

	"github.com/ajkula/ajkurig/input"
	"github.com/ajkula/ajkurig/mathx"
	"github.com/ajkula/ajkurig/scores"
	"github.com/ajkula/ajkurig/store"
)

// ===== Types =====

type legacyControls struct {
	Buttons []int `json:"buttons"`
	Keys    []int `json:"keys"`
	Fire    *int  `json:"fire"`
	AimLock *int  `json:"aimLock"`
	Dash    *int  `json:"dash"`
}

type legacyScoreEntry struct {
	Initials        string   `json:"initials"`
	Score           int      `json:"score"`
	SurvivedSeconds *float32 `json:"survivedSeconds"`
	Kills           int      `json:"kills"`
	Level           int      `json:"level"`
}

type legacyScoreTable struct {
	Entries []legacyScoreEntry `json:"entries"`
}

// ===== Constants =====

const (
	legacyHighScoreFileName = "highscore.json"
	legacyInitials          = "---"
	statKills               = "kills"
	statLevel               = "level"
)

// ===== Public API =====

func MigrateLegacySaves(saves store.Store, layout *input.Layout) error {
	return errors.Join(migrateLegacyControls(saves, layout), migrateLegacyScores(saves))
}

// ===== Internal =====

func migrateLegacyControls(saves store.Store, layout *input.Layout) error {
	var legacy legacyControls
	if err := saves.Load(input.BindingsFileName, &legacy); err != nil || !legacy.isLegacy() {
		return nil
	}
	bindings := input.DefaultBindings(layout)
	copyLegacyBindings(&bindings, input.DeviceKeyboard, legacy.Keys, len(layout.Remaps))
	copyLegacyBindings(&bindings, input.DeviceGamepad, legacy.Buttons, len(layout.Remaps))
	for remap, button := range [3]*int{legacy.Fire, legacy.AimLock, legacy.Dash} {
		isMigrated := button != nil && *bindings.Binding(input.DeviceGamepad, remap) == input.DefaultBinding
		mathx.SetIf(bindings.Binding(input.DeviceGamepad, remap), valueOr(button, input.DefaultBinding), isMigrated)
	}
	return input.SaveBindings(saves, layout, bindings)
}

func (l *legacyControls) isLegacy() bool {
	return len(l.Buttons) > 0 || len(l.Keys) > 0 || l.Fire != nil || l.AimLock != nil || l.Dash != nil
}

func copyLegacyBindings(bindings *input.Bindings, device input.Device, legacy []int, remapCount int) {
	for remap := range min(len(legacy), remapCount) {
		*bindings.Binding(device, remap) = legacy[remap]
	}
}

func migrateLegacyScores(saves store.Store) error {
	var legacy legacyScoreTable
	err := saves.Load(scores.TableFileName, &legacy)
	if errors.Is(err, fs.ErrNotExist) {
		return migrateLegacyBestScore(saves)
	}
	if err != nil || !legacy.isLegacy() {
		return nil
	}
	table := scores.NewTable(scores.DefaultTableSize)
	for _, entry := range legacy.Entries {
		table.Entries = append(table.Entries, entry.toEntry())
	}
	return scores.Save(saves, table)
}

func (t *legacyScoreTable) isLegacy() bool {
	for _, entry := range t.Entries {
		if entry.SurvivedSeconds != nil {
			return true
		}
	}
	return false
}

func migrateLegacyBestScore(saves store.Store) error {
	var legacy legacyScoreEntry
	if err := saves.Load(legacyHighScoreFileName, &legacy); err != nil {
		return nil
	}
	legacy.Initials = legacyInitials
	table := scores.NewTable(scores.DefaultTableSize)
	table.Insert(legacy.toEntry())
	return scores.Save(saves, table)
}

func (e legacyScoreEntry) toEntry() scores.Entry {
	return scores.Entry{
		Initials: e.Initials, Score: e.Score, Seconds: valueOr(e.SurvivedSeconds, 0),
		Stats: map[string]int{statKills: e.Kills, statLevel: e.Level},
	}
}

func valueOr[T any](pointer *T, fallback T) T {
	if pointer == nil {
		return fallback
	}
	return *pointer
}
