package main

import (
	"fmt"
	"math/bits"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// ===== Types =====

type Action uint16

type Controls struct {
	MoveX          float32
	MoveY          float32
	Held           Action
	JustPressed    Action
	RawJustPressed int
}

type KeyBinding struct {
	Key    ebiten.Key
	Action Action
}

type GamepadBinding struct {
	Button ebiten.StandardGamepadButton
	Action Action
}

type RawBinding struct {
	Button ebiten.GamepadButton
	Action Action
}

type InputReader struct {
	previous       Action
	rawPrevious    uint64
	gamepadIDs     []ebiten.GamepadID
	customBindings []RawBinding
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
)

const (
	analogDeadzone         = 0.25
	analogDirectionMinimum = 0.5
	rawHorizontalAxis      = 0
	rawVerticalAxis        = 1
	maximumRawButtons      = 64
)

const actionFireConfirm = ActionFire | ActionConfirm

var keyboardBindings = []KeyBinding{
	{ebiten.KeyW, ActionUp}, {ebiten.KeyArrowUp, ActionUp},
	{ebiten.KeyS, ActionDown}, {ebiten.KeyArrowDown, ActionDown},
	{ebiten.KeyA, ActionLeft}, {ebiten.KeyArrowLeft, ActionLeft},
	{ebiten.KeyD, ActionRight}, {ebiten.KeyArrowRight, ActionRight},
	{ebiten.KeyJ, actionFireConfirm}, {ebiten.KeyZ, actionFireConfirm},
	{ebiten.KeyK, ActionAimLock}, {ebiten.KeyX, ActionAimLock},
	{ebiten.KeySpace, ActionDash}, {ebiten.KeyL, ActionDash}, {ebiten.KeyC, ActionDash},
	{ebiten.KeyEnter, ActionConfirm | ActionStart},
	{ebiten.KeyBackspace, ActionSelect},
	{ebiten.KeyEscape, ActionPause}, {ebiten.KeyP, ActionPause},
	{ebiten.KeyF2, ActionSelect},
	{ebiten.KeyF1, ActionDebug},
}

var gamepadSystemBindings = []GamepadBinding{
	{ebiten.StandardGamepadButtonLeftTop, ActionUp},
	{ebiten.StandardGamepadButtonLeftBottom, ActionDown},
	{ebiten.StandardGamepadButtonLeftLeft, ActionLeft},
	{ebiten.StandardGamepadButtonLeftRight, ActionRight},
	{ebiten.StandardGamepadButtonCenterRight, ActionPause | ActionConfirm | ActionStart},
	{ebiten.StandardGamepadButtonCenterLeft, ActionSelect},
}

var gamepadDefaultActionBindings = []GamepadBinding{
	{ebiten.StandardGamepadButtonRightBottom, actionFireConfirm},
	{ebiten.StandardGamepadButtonRightRight, ActionAimLock},
	{ebiten.StandardGamepadButtonRightLeft, ActionDash},
	{ebiten.StandardGamepadButtonRightTop, ActionDash},
}

var actionNames = []string{"Up", "Down", "Left", "Right", "Fire", "AimLock", "Dash", "Confirm", "Pause", "Select", "Debug", "Start"}

// ===== Public API =====

func (r *InputReader) Read() Controls {
	r.gamepadIDs = ebiten.AppendGamepadIDs(r.gamepadIDs[:0])
	analogX, analogY := r.readAnalog()
	rawPressed := r.readRawPressed()
	held := readKeyboard() | r.readGamepadButtons(rawPressed) | directionsFromAnalog(analogX, analogY)
	digitalX := boolToFloat(held&ActionRight != 0) - boolToFloat(held&ActionLeft != 0)
	digitalY := boolToFloat(held&ActionDown != 0) - boolToFloat(held&ActionUp != 0)
	analogWeight := boolToFloat(length(analogX, analogY) > analogDeadzone)
	moveX := digitalX + (analogX-digitalX)*analogWeight
	moveY := digitalY + (analogY-digitalY)*analogWeight
	scale := 1 / max(1, length(moveX, moveY))
	controls := Controls{
		MoveX: moveX * scale, MoveY: moveY * scale,
		Held: held, JustPressed: held &^ r.previous,
		RawJustPressed: lowestButton(rawPressed &^ r.rawPrevious),
	}
	r.previous, r.rawPrevious = held, rawPressed
	return controls
}

func (r *InputReader) UseCustomBindings(bindings ButtonBindings) {
	r.customBindings = []RawBinding{
		{ebiten.GamepadButton(bindings.Fire), actionFireConfirm},
		{ebiten.GamepadButton(bindings.AimLock), ActionAimLock},
		{ebiten.GamepadButton(bindings.Dash), ActionDash},
	}
}

func (r *InputReader) DescribeDevices(controls Controls) []string {
	lines := []string{"INPUT DEBUG  (F1 / Select to close)", "Actions held: " + describeActions(controls.Held)}
	for _, id := range r.gamepadIDs {
		lines = append(lines, describeGamepad(id)...)
	}
	return appendIf(lines, "No gamepad detected", len(r.gamepadIDs) == 0)
}

// ===== Internal =====

func readKeyboard() Action {
	held := Action(0)
	for _, binding := range keyboardBindings {
		held |= binding.Action * Action(boolToIndex(ebiten.IsKeyPressed(binding.Key)))
	}
	return held
}

func (r *InputReader) readGamepadButtons(rawPressed uint64) Action {
	held := Action(0)
	for _, id := range r.gamepadIDs {
		held |= readStandardButtons(id, gamepadSystemBindings)
		held |= readStandardButtons(id, gamepadDefaultActionBindings) * Action(boolToIndex(r.customBindings == nil))
	}
	for _, binding := range r.customBindings {
		held |= binding.Action * Action(rawPressed>>uint(binding.Button)&1)
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
