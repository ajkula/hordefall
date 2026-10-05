package game

import (
	"fmt"
	"math/bits"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ===== Types =====

type Action uint16

type InputDevice uint8

type Controls struct {
	MoveX          float32
	MoveY          float32
	Held           Action
	JustPressed    Action
	RawJustPressed int
	AimX           float32
	AimY           float32
	HasAimStick    bool
}

type KeyBinding struct {
	Key    ebiten.Key
	Action Action
}

type GamepadBinding struct {
	Button ebiten.StandardGamepadButton
	Action Action
}

type InputReader struct {
	previous       Action
	rawPrevious    uint64
	gamepadIDs     []ebiten.GamepadID
	bindings       ControlBindings
	justPressedKey []ebiten.Key
	lastDevice     InputDevice
}

// ===== Constants =====

const (
	ActionUp Action = 1 << iota
	ActionDown
	ActionLeft
	ActionRight
	ActionFire
	ActionAimLock
	ActionDash
	ActionConfirm
	ActionPause
	ActionSelect
	ActionDebug
	ActionStart
	ActionMute
	ActionFullscreen
)

const (
	DeviceKeyboard InputDevice = iota
	DeviceGamepad
	inputDeviceCount
)

const (
	analogDeadzone         = 0.25
	analogDirectionMinimum = 0.5
	aimStickDeadzone       = 0.35
	rawHorizontalAxis      = 0
	rawVerticalAxis        = 1
	maximumRawButtons      = 64
)

const actionFireConfirm = ActionFire | ActionConfirm

var keyboardSystemBindings = []KeyBinding{
	{ebiten.KeyArrowUp, ActionUp}, {ebiten.KeyArrowDown, ActionDown},
	{ebiten.KeyArrowLeft, ActionLeft}, {ebiten.KeyArrowRight, ActionRight},
	{ebiten.KeyEnter, ActionConfirm | ActionStart},
	{ebiten.KeyF1, ActionDebug},
	{ebiten.KeyM, ActionMute},
	{ebiten.KeyF11, ActionFullscreen},
}

var gamepadSystemBindings = []GamepadBinding{
	{ebiten.StandardGamepadButtonLeftTop, ActionUp},
	{ebiten.StandardGamepadButtonLeftBottom, ActionDown},
	{ebiten.StandardGamepadButtonLeftLeft, ActionLeft},
	{ebiten.StandardGamepadButtonLeftRight, ActionRight},
}

var keyboardFixedLabels = map[Action]string{ActionStart: "ENTER", ActionConfirm: "ENTER"}

var actionNames = []string{"Up", "Down", "Left", "Right", "Fire", "AimLock", "Dash", "Confirm", "Pause", "Select", "Debug", "Start", "Mute"}

// ===== Public API =====

func (r *InputReader) Read() Controls {
	r.gamepadIDs = ebiten.AppendGamepadIDs(r.gamepadIDs[:0])
	analogX, analogY := r.readAnalog()
	rawPressed := r.readRawPressed()
	keyboardHeld := r.readKeyboard()
	gamepadHeld := r.readGamepadButtons(rawPressed) | directionsFromAnalog(analogX, analogY)
	held := keyboardHeld | gamepadHeld
	r.lastDevice = pickDevice(r.lastDevice, keyboardHeld != 0, gamepadHeld != 0 || rawPressed != 0)
	digitalX := boolToFloat(held&ActionRight != 0) - boolToFloat(held&ActionLeft != 0)
	digitalY := boolToFloat(held&ActionDown != 0) - boolToFloat(held&ActionUp != 0)
	analogWeight := boolToFloat(length(analogX, analogY) > analogDeadzone)
	moveX := digitalX + (analogX-digitalX)*analogWeight
	moveY := digitalY + (analogY-digitalY)*analogWeight
	scale := 1 / max(1, length(moveX, moveY))
	aimX, aimY := r.readAimStick()
	controls := Controls{
		MoveX: moveX * scale, MoveY: moveY * scale,
		Held: held, JustPressed: held &^ r.previous,
		RawJustPressed: lowestButton(rawPressed &^ r.rawPrevious),
		HasAimStick:    length(aimX, aimY) > aimStickDeadzone,
	}
	controls.AimX, controls.AimY = normalize(aimX, aimY)
	r.previous, r.rawPrevious = held, rawPressed
	return controls
}

func (r *InputReader) LastDevice() InputDevice {
	return r.lastDevice
}

func (r *InputReader) ButtonLabel(action Action) string {
	label := [inputDeviceCount]map[Action]string{keyboardFixedLabels, nil}[r.lastDevice][action]
	for _, remapAction := range slices.Backward(remapActionsByDevice[r.lastDevice]) {
		definition := &remapTable[remapAction]
		deviceActions := [inputDeviceCount]Action{definition.KeyboardActions, definition.GamepadActions}[r.lastDevice]
		label = [2]string{label, r.bindings.Label(r.lastDevice, remapAction)}[boolToIndex(deviceActions&action != 0)]
	}
	return label
}

func (r *InputReader) UseBindings(bindings ControlBindings) {
	r.bindings = bindings
}

func (r *InputReader) Bindings() ControlBindings {
	return r.bindings
}

func (r *InputReader) JustPressedKey() int {
	r.justPressedKey = inpututil.AppendJustPressedKeys(r.justPressedKey[:0])
	if len(r.justPressedKey) == 0 {
		return -1
	}
	return int(r.justPressedKey[0])
}

func (r *InputReader) DescribeDevices(controls Controls) []string {
	lines := []string{"INPUT DEBUG  (F1 / Select to close)", "Actions held: " + describeActions(controls.Held)}
	for _, id := range r.gamepadIDs {
		lines = append(lines, describeGamepad(id)...)
	}
	return appendIf(lines, "No gamepad detected", len(r.gamepadIDs) == 0)
}

// ===== Internal =====

func pickDevice(current InputDevice, isKeyboardUsed, isGamepadUsed bool) InputDevice {
	afterKeyboard := [2]InputDevice{current, DeviceKeyboard}[boolToIndex(isKeyboardUsed)]
	return [2]InputDevice{afterKeyboard, DeviceGamepad}[boolToIndex(isGamepadUsed)]
}

func (r *InputReader) readKeyboard() Action {
	held := Action(0)
	for _, binding := range keyboardSystemBindings {
		held |= binding.Action * Action(boolToIndex(ebiten.IsKeyPressed(binding.Key)))
	}
	for _, action := range remapActionsByDevice[DeviceKeyboard] {
		held |= remapTable[action].KeyboardActions * Action(boolToIndex(r.isRemappedKeyPressed(action)))
	}
	return held
}

func (r *InputReader) isRemappedKeyPressed(action RemapAction) bool {
	binding := r.bindings.Keys[action]
	keys := remapTable[action].DefaultKeys
	isPressed := binding != defaultBinding && ebiten.IsKeyPressed(ebiten.Key(binding))
	for _, key := range keys {
		isPressed = isPressed || (binding == defaultBinding && ebiten.IsKeyPressed(key))
	}
	return isPressed
}

func (r *InputReader) readGamepadButtons(rawPressed uint64) Action {
	held := Action(0)
	for _, id := range r.gamepadIDs {
		held |= readStandardButtons(id, gamepadSystemBindings)
		held |= r.readDefaultGamepadActions(id)
	}
	for _, action := range remapActionsByDevice[DeviceGamepad] {
		binding := r.bindings.Buttons[action]
		isPressed := binding != defaultBinding && rawPressed>>uint(max(0, binding))&1 != 0
		held |= remapTable[action].GamepadActions * Action(boolToIndex(isPressed))
	}
	return held
}

func (r *InputReader) readDefaultGamepadActions(id ebiten.GamepadID) Action {
	held := Action(0)
	for _, action := range remapActionsByDevice[DeviceGamepad] {
		isDefault := r.bindings.Buttons[action] == defaultBinding
		for _, button := range remapTable[action].DefaultButtons {
			isPressed := isDefault && ebiten.IsStandardGamepadButtonPressed(id, button)
			held |= remapTable[action].GamepadActions * Action(boolToIndex(isPressed))
		}
	}
	return held
}

func readStandardButtons(id ebiten.GamepadID, bindings []GamepadBinding) Action {
	held := Action(0)
	for _, binding := range bindings {
		held |= binding.Action * Action(boolToIndex(ebiten.IsStandardGamepadButtonPressed(id, binding.Button)))
	}
	return held
}

func (r *InputReader) readRawPressed() uint64 {
	pressed := uint64(0)
	for _, id := range r.gamepadIDs {
		pressed |= readRawButtons(id)
	}
	return pressed
}

func readRawButtons(id ebiten.GamepadID) uint64 {
	pressed := uint64(0)
	for button := range ebiten.GamepadButton(min(ebiten.GamepadButtonCount(id), maximumRawButtons)) {
		pressed |= uint64(boolToIndex(ebiten.IsGamepadButtonPressed(id, button))) << uint(button)
	}
	return pressed
}

func lowestButton(mask uint64) int {
	if mask == 0 {
		return -1
	}
	return bits.TrailingZeros64(mask)
}

func (r *InputReader) readAnalog() (float32, float32) {
	strongestX, strongestY := float32(0), float32(0)
	for _, id := range r.gamepadIDs {
		x, y := readGamepadStick(id)
		isStronger := length(x, y) > length(strongestX, strongestY)
		strongestX += (x - strongestX) * boolToFloat(isStronger)
		strongestY += (y - strongestY) * boolToFloat(isStronger)
	}
	return strongestX, strongestY
}

func (r *InputReader) readAimStick() (float32, float32) {
	strongestX, strongestY := float32(0), float32(0)
	for _, id := range r.gamepadIDs {
		x, y := readRightStick(id)
		isStronger := length(x, y) > length(strongestX, strongestY)
		strongestX += (x - strongestX) * boolToFloat(isStronger)
		strongestY += (y - strongestY) * boolToFloat(isStronger)
	}
	return strongestX, strongestY
}

func readRightStick(id ebiten.GamepadID) (float32, float32) {
	isStandard := boolToFloat(ebiten.IsStandardGamepadLayoutAvailable(id))
	x := float32(ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisRightStickHorizontal)) * isStandard
	y := float32(ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisRightStickVertical)) * isStandard
	return x, y
}

func readGamepadStick(id ebiten.GamepadID) (float32, float32) {
	if ebiten.IsStandardGamepadLayoutAvailable(id) {
		return float32(ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)),
			float32(ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical))
	}
	return float32(ebiten.GamepadAxisValue(id, rawHorizontalAxis)), float32(ebiten.GamepadAxisValue(id, rawVerticalAxis))
}

func directionsFromAnalog(x, y float32) Action {
	return ActionLeft*Action(boolToIndex(x < -analogDirectionMinimum)) |
		ActionRight*Action(boolToIndex(x > analogDirectionMinimum)) |
		ActionUp*Action(boolToIndex(y < -analogDirectionMinimum)) |
		ActionDown*Action(boolToIndex(y > analogDirectionMinimum))
}

func describeActions(actions Action) string {
	names := make([]string, 0, len(actionNames))
	for bit, name := range actionNames {
		names = appendIf(names, name, actions&(1<<bit) != 0)
	}
	return strings.Join(names, " ")
}

func describeGamepad(id ebiten.GamepadID) []string {
	rawPressed := make([]string, 0, 8)
	for button := range ebiten.GamepadButton(ebiten.GamepadButtonCount(id)) {
		rawPressed = appendIf(rawPressed, fmt.Sprint(int(button)), ebiten.IsGamepadButtonPressed(id, button))
	}
	standardPressed := make([]string, 0, 8)
	for button := range ebiten.StandardGamepadButtonMax + 1 {
		standardPressed = appendIf(standardPressed, fmt.Sprint(int(button)), ebiten.IsStandardGamepadButtonPressed(id, button))
	}
	axes := make([]string, 0, 4)
	for axis := range ebiten.GamepadAxisType(ebiten.GamepadAxisCount(id)) {
		axes = append(axes, fmt.Sprintf("%.2f", ebiten.GamepadAxisValue(id, axis)))
	}
	return []string{
		fmt.Sprintf("Pad %d: %s  standard=%v", id, ebiten.GamepadName(id), ebiten.IsStandardGamepadLayoutAvailable(id)),
		"  raw buttons: " + strings.Join(rawPressed, " "),
		"  standard buttons: " + strings.Join(standardPressed, " "),
		"  raw axes: " + strings.Join(axes, " "),
	}
}

func appendIf[T any](items []T, item T, shouldAppend bool) []T {
	if !shouldAppend {
		return items
	}
	return append(items, item)
}
