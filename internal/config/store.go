package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ===== Constants =====

const (
	directoryName  = "hordefall"
	directoryMode  = 0o755
	fileMode       = 0o644
	jsonIndentUnit = "  "
)

// ===== Public API =====

func Path(fileName string) (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, directoryName, fileName), nil
}

func Load(fileName string, target any) error {
	path, err := Path(fileName)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(content, target)
}

func Save(fileName string, value any) error {
	path, err := Path(fileName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), directoryMode); err != nil {
		return err
	}
	content, err := json.MarshalIndent(value, "", jsonIndentUnit)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, fileMode)
}
