package game

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ajkula/ajkurig/input"
)

// ===== Constants =====

var (
	ActionFire    = input.GameAction(0)
	ActionAimLock = input.GameAction(1)
	ActionDash    = input.GameAction(2)
)

var remaps = []input.RemapDefinition{
	{
		Key: "fire", Primary: ActionFire, KeyboardActions: ActionFire | input.ActionConfirm, GamepadActions: ActionFire | input.ActionConfirm,
		DefaultKeys: []ebiten.Key{ebiten.KeyJ, ebiten.KeyZ}, DefaultButtons: []ebiten.StandardGamepadButton{ebiten.StandardGamepadButtonRightBottom}, DefaultGamepadLabel: "A",
	},
	{
		Key: "aim_lock", Primary: ActionAimLock, KeyboardActions: ActionAimLock | input.ActionBack, GamepadActions: ActionAimLock | input.ActionBack,
		DefaultKeys: []ebiten.Key{ebiten.KeyK, ebiten.KeyX}, DefaultButtons: []ebiten.StandardGamepadButton{ebiten.StandardGamepadButtonRightRight}, DefaultGamepadLabel: "B",
	},
	{
		Key: "dash", Primary: ActionDash, KeyboardActions: ActionDash, GamepadActions: ActionDash,
		DefaultKeys:    []ebiten.Key{ebiten.KeySpace, ebiten.KeyL, ebiten.KeyC},
		DefaultButtons: []ebiten.StandardGamepadButton{ebiten.StandardGamepadButtonRightLeft, ebiten.StandardGamepadButtonRightTop}, DefaultGamepadLabel: "X",
	},
}

var mouseBindings = []input.MouseBinding{
	{Button: ebiten.MouseButtonLeft, Action: ActionFire},
	{Button: ebiten.MouseButtonRight, Action: ActionDash},
}
