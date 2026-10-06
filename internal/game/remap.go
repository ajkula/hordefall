package game

import (
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"hordefall/internal/i18n"
)

// ===== Constants =====

const (
	remapListenSeconds = 5
	remapSubtitleTop   = menuTitleTop + 96
	remapMessageTop    = menuTitleTop + 128
	remapStepTop       = 270
	remapStepSpacing   = 56
	noInput            = -1
)

var remapInputVerbKeys = [inputDeviceCount]string{"remap.verb_key", "remap.verb_button"}

// ===== Internal =====

func (g *Game) loadBindings() {
	bindings, err := LoadControlBindings()
	g.controlBindings = bindings
	g.input.UseBindings(bindings)
	messages := [2]string{"remap.custom_loaded", "remap.default_loaded"}
	g.bindingsMessage = messages[boolToIndex(err != nil)]
}

func (g *Game) openRemapFrom(returnState GameState) {
	g.remapReturnState = returnState
	g.remapDevice = g.input.LastDevice()
	g.remapStep = essentialRemapCount * boolToIndex(g.controlBindings.IsCustomized(g.remapDevice))
	g.remapListenSeconds = 0
	g.remapMessage = ""
	g.buildRemapMenu()
	g.switchState(StateRemap)
}

func (g *Game) openRemapFromOptions() {
	g.openRemapFrom(StateOptions)
}

func (g *Game) closeRemap() {
	g.switchState(g.remapReturnState)
}

func (g *Game) buildRemapMenu() {
	g.remapMenu = g.remapMenu[:0]
	for _, action := range remapActionsByDevice[g.remapDevice] {
		g.remapMenu = append(g.remapMenu, MenuOption{remapRowLabel(action), listenForRemap(action), nil})
	}
	g.remapMenu = append(g.remapMenu,
		MenuOption{translatedLabel("remap.reset"), (*Game).resetRemapDevice, nil},
		MenuOption{translatedLabel("menu.back"), (*Game).closeRemap, nil},
	)
}

func remapRowLabel(action RemapAction) func(game *Game) string {
	return func(g *Game) string {
		isListening := g.remapListenSeconds > 0 && g.remapListening == action
		countdown := int(math.Ceil(float64(g.remapListenSeconds)))
		values := [2]string{g.controlBindings.Label(g.remapDevice, action), i18n.F("remap.listening", remapVerb(g.remapDevice), countdown)}
		return actionName(action) + "    " + values[boolToIndex(isListening)]
	}
}

func listenForRemap(action RemapAction) func(game *Game) {
	return func(g *Game) {
		g.remapListening = action
		g.remapListenSeconds = remapListenSeconds
		g.remapMessage = ""
	}
}

func (g *Game) updateRemap() {
	g.updateBackdropDemo()
	input := g.readRemapInput()
	g.remapListenSeconds = max(0, g.remapListenSeconds-deltaSeconds)
	if g.menuLockSeconds > 0 {
		return
	}
	if g.remapStep < essentialRemapCount {
		g.captureEssentialStep(input)
		return
	}
	if g.remapListenSeconds > 0 {
		g.captureListenedRow(input)
		return
	}
	if g.isGoingBack() {
		g.closeRemap()
		return
	}
	g.navigateMenu(g.remapMenu)
}

func (g *Game) readRemapInput() int {
	key := g.input.JustPressedKey()
	inputs := [inputDeviceCount]int{key, g.controls.RawJustPressed}
	return inputs[g.remapDevice]
}

func (g *Game) captureEssentialStep(input int) {
	if g.controls.JustPressed&ActionPause != 0 {
		g.closeRemap()
		return
	}
	if !g.tryAssignRemap(RemapAction(g.remapStep), input) {
		return
	}
	g.remapStep++
}

func (g *Game) captureListenedRow(input int) {
	if !g.tryAssignRemap(g.remapListening, input) {
		return
	}
	g.remapListenSeconds = 0
}

func (g *Game) tryAssignRemap(action RemapAction, input int) bool {
	if input == noInput {
		return false
	}
	blocker := g.remapBlocker(action, input)
	if blocker != "" {
		g.remapMessage = blocker
		return false
	}
	g.controlBindings.Assign(g.remapDevice, action, input)
	g.applyControlBindings(i18n.T("remap.saved"))
	g.menuLockSeconds = menuLockDuration
	return true
}

func (g *Game) remapBlocker(action RemapAction, input int) string {
	isReservedKey := g.remapDevice == DeviceKeyboard && slices.ContainsFunc(keyboardSystemBindings, func(binding KeyBinding) bool {
		return int(binding.Key) == input
	})
	blockers := [2]string{g.defaultButtonBlocker(action), keyName(ebiten.Key(input)) + " is reserved."}
	return blockers[boolToIndex(isReservedKey)]
}

func (g *Game) defaultButtonBlocker(action RemapAction) string {
	if g.remapDevice != DeviceGamepad {
		return ""
	}
	for _, other := range remapActionsByDevice[DeviceGamepad] {
		isDefault := g.controlBindings.Buttons[other] == defaultBinding
		isPressed := g.controls.Held&remapTable[other].Primary != 0
		if other != action && isDefault && isPressed {
			return i18n.F("remap.button_taken", actionName(other))
		}
	}
	return ""
}

func (g *Game) resetRemapDevice() {
	g.controlBindings.ResetDevice(g.remapDevice)
	g.applyControlBindings(i18n.T("remap.defaults_restored"))
}

func (g *Game) applyControlBindings(message string) {
	g.input.UseBindings(g.controlBindings)
	g.remapMessage = message
	g.bindingsMessage = "remap.custom_loaded"
	if err := SaveControlBindings(g.controlBindings); err != nil {
		g.remapMessage = i18n.F("remap.save_failed", err.Error())
	}
}

func (g *Game) drawRemap(screen *ebiten.Image) {
	g.renderer.DrawWorld(g, screen)
	g.ui.DrawRemap(g, screen)
}

func (u *UI) DrawRemap(g *Game, screen *ebiten.Image) {
	dimScreen(screen, 0.75)
	u.drawText(screen, i18n.T("title.controls"), u.title, screenWidth/2, menuTitleTop, accentColor, 1, text.AlignCenter)
	u.drawText(screen, i18n.T(deviceNameKeys[g.remapDevice]), u.bold, screenWidth/2, remapSubtitleTop, textColor, 1, text.AlignCenter)
	u.drawText(screen, g.remapMessage, u.small, screenWidth/2, remapMessageTop, healthColor, 1, text.AlignCenter)
	drawers := [2]func(u *UI, g *Game, screen *ebiten.Image){(*UI).drawRemapList, (*UI).drawEssentialSteps}
	drawers[boolToIndex(g.remapStep < essentialRemapCount)](u, g, screen)
	u.drawMusicBanner(g, screen)
}

func (u *UI) drawEssentialSteps(g *Game, screen *ebiten.Image) {
	for step := range essentialRemapCount {
		action := RemapAction(step)
		name := actionName(action)
		labels := [3]string{name + ":  " + g.controlBindings.Label(g.remapDevice, action), i18n.F("remap.press_for", remapVerb(g.remapDevice), name), name}
		state := boolToIndex(step == g.remapStep) + 2*boolToIndex(step > g.remapStep)
		colors := [3][3]float32{textColor, accentColor, textColor}
		u.drawText(screen, labels[state], u.bold, screenWidth/2, remapStepTop+float32(step)*remapStepSpacing, colors[state], 1, text.AlignCenter)
	}
	u.drawText(screen, i18n.F("remap.essentials_hint", g.input.ButtonLabel(ActionPause)), u.small, screenWidth/2, menuHintTop, textColor, 1, text.AlignCenter)
}

func (u *UI) drawRemapList(g *Game, screen *ebiten.Image) {
	u.drawMenuOptions(g, screen, g.remapMenu, menuOptionSpacing)
	hints := [2]string{
		i18n.T("remap.list_hint"),
		i18n.F("remap.listen_hint", remapVerb(g.remapDevice)),
	}
	u.drawText(screen, hints[boolToIndex(g.remapListenSeconds > 0)], u.small, screenWidth/2, menuHintTop, textColor, 1, text.AlignCenter)
}
