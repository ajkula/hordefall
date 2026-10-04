package game

import (
	"errors"

	"hordefall/internal/config"
)

// ===== Types =====

type ButtonBindings struct {
	Fire    int `json:"fire"`
	AimLock int `json:"aimLock"`
	Dash    int `json:"dash"`
}

// ===== Constants =====

const bindingsFileName = "controls.json"

var remapStepNames = [3]string{"FIRE", "AIM LOCK", "DASH"}

// ===== Public API =====

func LoadButtonBindings() (ButtonBindings, error) {
	var bindings ButtonBindings
	if err := config.Load(bindingsFileName, &bindings); err != nil {
		return ButtonBindings{}, err
	}
	return bindings, validateBindings(bindings)
}

func SaveButtonBindings(bindings ButtonBindings) error {
	return config.Save(bindingsFileName, bindings)
}

func BindingsFromSteps(buttons [3]int) ButtonBindings {
	return ButtonBindings{Fire: buttons[0], AimLock: buttons[1], Dash: buttons[2]}
}

// ===== Internal =====

func validateBindings(bindings ButtonBindings) error {
	isInRange := func(button int) bool { return button >= 0 && button < maximumRawButtons }
	if !isInRange(bindings.Fire) || !isInRange(bindings.AimLock) || !isInRange(bindings.Dash) {
		return errors.New("controls.json: button index out of range")
	}
	return nil
}
