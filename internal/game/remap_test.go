package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestLegacyControlsFileIsMigrated(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("APPDATA", appData)
	directory := filepath.Join(appData, "hordefall")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, bindingsFileName), []byte(`{"fire":2,"aimLock":3,"dash":0}`), 0o644); err != nil {
		t.Fatal(err)
	}
	bindings, err := LoadControlBindings()
	if err != nil || bindings.Buttons[RemapFire] != 2 || bindings.Buttons[RemapAimLock] != 3 || bindings.Buttons[RemapDash] != 0 {
		t.Fatalf("migrated %+v (err %v)", bindings, err)
	}
	if bindings.Buttons[RemapPause] != defaultBinding || bindings.Keys[RemapFire] != defaultBinding {
		t.Fatalf("missing bindings did not default: %+v", bindings)
	}
}

func TestAssigningATakenKeySwaps(t *testing.T) {
	bindings := DefaultControlBindings()
	bindings.Assign(DeviceKeyboard, RemapFire, int(ebiten.KeyK))
	if bindings.Keys[RemapFire] != int(ebiten.KeyK) || bindings.Keys[RemapAimLock] != int(ebiten.KeyJ) {
		t.Fatalf("keyboard swap failed: fire %d aim %d", bindings.Keys[RemapFire], bindings.Keys[RemapAimLock])
	}
	bindings.Assign(DeviceGamepad, RemapFire, 4)
	bindings.Assign(DeviceGamepad, RemapDash, 4)
	if bindings.Buttons[RemapDash] != 4 || bindings.Buttons[RemapFire] != defaultBinding {
		t.Fatalf("gamepad swap failed: %+v", bindings.Buttons)
	}
	if bindings.Label(DeviceKeyboard, RemapFire) != "K" || bindings.Label(DeviceGamepad, RemapDash) != "BUTTON 4" || bindings.Label(DeviceGamepad, RemapFire) != "A" {
		t.Fatalf("labels %q %q %q", bindings.Label(DeviceKeyboard, RemapFire), bindings.Label(DeviceGamepad, RemapDash), bindings.Label(DeviceGamepad, RemapFire))
	}
}

func TestRemapFlowAsksEssentialsThenListsAndSaves(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.controlBindings = DefaultControlBindings()
	game.input.lastDevice = DeviceGamepad
	game.openRemapFrom(StateOptions)
	if game.remapStep != 0 || len(game.remapMenu) != len(remapActionsByDevice[DeviceGamepad])+2 {
		t.Fatalf("step %d, %d rows", game.remapStep, len(game.remapMenu))
	}
	for step, button := range []int{2, 3, 0} {
		game.menuLockSeconds = 0
		game.controls = Controls{RawJustPressed: button}
		game.captureEssentialStep(button)
		if game.remapStep != step+1 {
			t.Fatalf("step %d not captured", step)
		}
	}
	reloaded, err := LoadControlBindings()
	if err != nil || reloaded.Buttons[RemapFire] != 2 || reloaded.Buttons[RemapDash] != 0 {
		t.Fatalf("saved %+v (err %v)", reloaded, err)
	}
	game.openRemapFrom(StateOptions)
	if game.remapStep != essentialRemapCount {
		t.Fatalf("customized pad asked the essentials again")
	}
}

func TestRemapRejectsReservedAndDefaultTakenInputs(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	game := newHeadlessGame()
	game.controlBindings = DefaultControlBindings()
	game.remapDevice = DeviceKeyboard
	if game.tryAssignRemap(RemapFire, int(ebiten.KeyEnter)) {
		t.Fatal("enter was accepted")
	}
	game.remapDevice = DeviceGamepad
	game.controls = Controls{Held: ActionPause | ActionConfirm | ActionStart}
	if game.tryAssignRemap(RemapFire, 9) || game.remapMessage == "" {
		t.Fatal("the default start button was stolen")
	}
	game.controls = Controls{Held: ActionPause | ActionConfirm | ActionStart}
	if !game.tryAssignRemap(RemapPause, 9) {
		t.Fatal("pause could not take its own button")
	}
	game.resetRemapDevice()
	if game.controlBindings.IsCustomized(DeviceGamepad) {
		t.Fatal("reset kept custom buttons")
	}
}
