package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// ===== Types =====

type ButtonBindings struct {
	Fire    int `json:"fire"`
	AimLock int `json:"aimLock"`
	Dash    int `json:"dash"`
}

// ===== Constants =====

const (
	configDirectoryName = "hordefall"
	bindingsFileName    = "controls.json"
)

var remapStepNames = [3]string{"FIRE", "AIM LOCK", "DASH"}

// ===== Public API =====

func LoadButtonBindings() (ButtonBindings, error) {
	path, err := configFilePath(bindingsFileName)
	if err != nil {
		return ButtonBindings{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return ButtonBindings{}, err
	}
	var bindings ButtonBindings
	if err := json.Unmarshal(content, &bindings); err != nil {
		return ButtonBindings{}, err
	}
	return bindings, validateBindings(bindings)
}

func SaveButtonBindings(bindings ButtonBindings) error {
	path, err := configFilePath(bindingsFileName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(bindings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func BindingsFromSteps(buttons [3]int) ButtonBindings {
	return ButtonBindings{Fire: buttons[0], AimLock: buttons[1], Dash: buttons[2]}
}

// ===== Internal =====

func configFilePath(fileName string) (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, configDirectoryName, fileName), nil
}

func validateBindings(bindings ButtonBindings) error {
	isInRange := func(button int) bool { return button >= 0 && button < maximumRawButtons }
	if !isInRange(bindings.Fire) || !isInRange(bindings.AimLock) || !isInRange(bindings.Dash) {
		return errors.New("controls.json: button index out of range")
	}
	return nil
}
