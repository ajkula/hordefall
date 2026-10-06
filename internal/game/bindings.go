package game

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"hordefall/internal/config"
)

// ===== Types =====

type RemapAction uint8

type RemapDefinition struct {
	Key                 string
	Primary             Action
	KeyboardActions     Action
	GamepadActions      Action
	DefaultKeys         []ebiten.Key
	DefaultButtons      []ebiten.StandardGamepadButton
	DefaultGamepadLabel string
}

type ControlBindings struct {
	Buttons [remapActionCount]int `json:"buttons"`
	Keys    [remapActionCount]int `json:"keys"`
}

type legacyButtonBindings struct {
	Fire    *int `json:"fire"`
	AimLock *int `json:"aimLock"`
	Dash    *int `json:"dash"`
}

// ===== Constants =====

const (
	RemapFire RemapAction = iota
	RemapAimLock
	RemapDash
	RemapPause
	RemapSelect
	RemapUp
	RemapDown
	RemapLeft
	RemapRight
	remapActionCount
)

const (
	bindingsFileName    = "controls.json"
	defaultBinding      = -1
	essentialRemapCount = 3
)

var remapTable = [remapActionCount]RemapDefinition{
	RemapFire: {
		Key: "fire", Primary: ActionFire, KeyboardActions: actionFireConfirm, GamepadActions: actionFireConfirm,
		DefaultKeys: []ebiten.Key{ebiten.KeyJ, ebiten.KeyZ}, DefaultButtons: []ebiten.StandardGamepadButton{ebiten.StandardGamepadButtonRightBottom}, DefaultGamepadLabel: "A",
	},
	RemapAimLock: {
		Key: "aim_lock", Primary: ActionAimLock, KeyboardActions: ActionAimLock, GamepadActions: ActionAimLock,
		DefaultKeys: []ebiten.Key{ebiten.KeyK, ebiten.KeyX}, DefaultButtons: []ebiten.StandardGamepadButton{ebiten.StandardGamepadButtonRightRight}, DefaultGamepadLabel: "B",
	},
	RemapDash: {
		Key: "dash", Primary: ActionDash, KeyboardActions: ActionDash, GamepadActions: ActionDash,
		DefaultKeys:    []ebiten.Key{ebiten.KeySpace, ebiten.KeyL, ebiten.KeyC},
		DefaultButtons: []ebiten.StandardGamepadButton{ebiten.StandardGamepadButtonRightLeft, ebiten.StandardGamepadButtonRightTop}, DefaultGamepadLabel: "X",
	},
	RemapPause: {
		Key: "pause", Primary: ActionPause, KeyboardActions: ActionPause, GamepadActions: ActionPause | ActionConfirm | ActionStart,
		DefaultKeys: []ebiten.Key{ebiten.KeyEscape, ebiten.KeyP}, DefaultButtons: []ebiten.StandardGamepadButton{ebiten.StandardGamepadButtonCenterRight}, DefaultGamepadLabel: "START",
	},
	RemapSelect: {
		Key: "select", Primary: ActionSelect, KeyboardActions: ActionSelect, GamepadActions: ActionSelect,
		DefaultKeys: []ebiten.Key{ebiten.KeyBackspace, ebiten.KeyF2}, DefaultButtons: []ebiten.StandardGamepadButton{ebiten.StandardGamepadButtonCenterLeft}, DefaultGamepadLabel: "SELECT",
	},
	RemapUp:    {Key: "up", Primary: ActionUp, KeyboardActions: ActionUp, DefaultKeys: []ebiten.Key{ebiten.KeyW}},
	RemapDown:  {Key: "down", Primary: ActionDown, KeyboardActions: ActionDown, DefaultKeys: []ebiten.Key{ebiten.KeyS}},
	RemapLeft:  {Key: "left", Primary: ActionLeft, KeyboardActions: ActionLeft, DefaultKeys: []ebiten.Key{ebiten.KeyA}},
	RemapRight: {Key: "right", Primary: ActionRight, KeyboardActions: ActionRight, DefaultKeys: []ebiten.Key{ebiten.KeyD}},
}

var remapActionsByDevice = [inputDeviceCount][]RemapAction{
	DeviceKeyboard: {RemapFire, RemapAimLock, RemapDash, RemapPause, RemapSelect, RemapUp, RemapDown, RemapLeft, RemapRight},
	DeviceGamepad:  {RemapFire, RemapAimLock, RemapDash, RemapPause, RemapSelect},
}

var deviceNameKeys = [inputDeviceCount]string{"device.keyboard", "device.gamepad"}

// ===== Public API =====

func DefaultControlBindings() ControlBindings {
	var bindings ControlBindings
	for action := range remapActionCount {
		bindings.Buttons[action], bindings.Keys[action] = defaultBinding, defaultBinding
	}
	return bindings
}

func LoadControlBindings() (ControlBindings, error) {
	bindings := DefaultControlBindings()
	if err := config.Load(bindingsFileName, &bindings); err != nil {
		return DefaultControlBindings(), err
	}
	var legacy legacyButtonBindings
	_ = config.Load(bindingsFileName, &legacy)
	bindings.applyLegacy(legacy)
	if err := bindings.validate(); err != nil {
		return DefaultControlBindings(), err
	}
	return bindings, nil
}

func SaveControlBindings(bindings ControlBindings) error {
	return config.Save(bindingsFileName, bindings)
}

func (b *ControlBindings) Binding(device InputDevice, action RemapAction) *int {
	slots := [inputDeviceCount]*[remapActionCount]int{&b.Keys, &b.Buttons}
	return &slots[device][action]
}

func (b *ControlBindings) IsCustomized(device InputDevice) bool {
	for action := range remapActionCount {
		if *b.Binding(device, action) != defaultBinding {
			return true
		}
	}
	return false
}

func (b *ControlBindings) ResetDevice(device InputDevice) {
	for action := range remapActionCount {
		*b.Binding(device, action) = defaultBinding
	}
}

func (b *ControlBindings) Assign(device InputDevice, action RemapAction, input int) {
	previous := *b.Binding(device, action)
	for _, other := range remapActionsByDevice[device] {
		isTaken := other != action && b.isInputBound(device, other, input)
		setIf(b.Binding(device, other), b.primaryInput(device, action, previous), isTaken)
	}
	*b.Binding(device, action) = input
}

func (b *ControlBindings) Label(device InputDevice, action RemapAction) string {
	binding := *b.Binding(device, action)
	labelers := [inputDeviceCount]func(definition *RemapDefinition, binding int) string{keyboardLabel, gamepadLabel}
	return labelers[device](&remapTable[action], binding)
}

// ===== Internal =====

func (b *ControlBindings) applyLegacy(legacy legacyButtonBindings) {
	legacyButtons := [essentialRemapCount]*int{legacy.Fire, legacy.AimLock, legacy.Dash}
	for action, button := range legacyButtons {
		isMigrated := button != nil && b.Buttons[action] == defaultBinding
		setIf(&b.Buttons[action], derefOr(button, defaultBinding), isMigrated)
	}
}

func (b *ControlBindings) validate() error {
	for action := range remapActionCount {
		isButtonValid := b.Buttons[action] >= defaultBinding && b.Buttons[action] < maximumRawButtons
		isKeyValid := b.Keys[action] >= defaultBinding && b.Keys[action] <= int(ebiten.KeyMax)
		if !isButtonValid || !isKeyValid {
			return errors.New("controls.json: binding out of range")
		}
	}
	return nil
}

func (b *ControlBindings) isInputBound(device InputDevice, action RemapAction, input int) bool {
	binding := *b.Binding(device, action)
	isDefaultKey := device == DeviceKeyboard && binding == defaultBinding && slices.Contains(remapTable[action].DefaultKeys, ebiten.Key(input))
	return binding == input || isDefaultKey
}

func (b *ControlBindings) primaryInput(device InputDevice, action RemapAction, binding int) int {
	isKeyboardDefault := device == DeviceKeyboard && binding == defaultBinding
	return [2]int{binding, int(remapTable[action].DefaultKeys[0])}[boolToIndex(isKeyboardDefault)]
}

func keyboardLabel(definition *RemapDefinition, binding int) string {
	key := [2]ebiten.Key{ebiten.Key(binding), definition.DefaultKeys[0]}[boolToIndex(binding == defaultBinding)]
	return keyName(key)
}

func gamepadLabel(definition *RemapDefinition, binding int) string {
	labels := [2]string{"BUTTON " + strconv.Itoa(binding), definition.DefaultGamepadLabel}
	return labels[boolToIndex(binding == defaultBinding)]
}

func keyName(key ebiten.Key) string {
	return strings.ToUpper(key.String())
}

func setIf(slot *int, value int, shouldSet bool) {
	if !shouldSet {
		return
	}
	*slot = value
}

func derefOr(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}
