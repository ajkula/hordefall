package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ajkula/ajkurig/input"
	"github.com/ajkula/ajkurig/rig"
	"github.com/ajkula/ajkurig/scores"
	"github.com/ajkula/ajkurig/shell"
)

// ===== Constants =====

const (
	remapFire    = 0
	remapAimLock = 1
	remapDash    = 2
	remapPause   = 3
)

var legacyRemapOrder = []string{"fire", "aim_lock", "dash", "pause", "select", "up", "down", "left", "right"}

// ===== Public API =====

func TestLayoutKeepsTheLegacyRemapOrder(t *testing.T) {
	layout := input.NewLayout(remaps, len(remaps))
	if len(layout.Remaps) != len(legacyRemapOrder) {
		t.Fatalf("%d remaps, want %d", len(layout.Remaps), len(legacyRemapOrder))
	}
	for index, key := range legacyRemapOrder {
		if layout.Remaps[index].Key != key {
			t.Fatalf("remap %d is %q, legacy saves expect %q", index, layout.Remaps[index].Key, key)
		}
	}
}

func TestLegacyControlsFileIsMigrated(t *testing.T) {
	directory := t.TempDir()
	writeLegacySave(t, directory, input.BindingsFileName, `{"fire":2,"aimLock":3,"dash":0}`)
	app, _ := newTestAppIn(t, directory)
	bindings := app.Input.Bindings()
	gamepad := [3]int{*bindings.Binding(input.DeviceGamepad, remapFire), *bindings.Binding(input.DeviceGamepad, remapAimLock), *bindings.Binding(input.DeviceGamepad, remapDash)}
	if gamepad != [3]int{2, 3, 0} {
		t.Fatalf("migrated gamepad buttons %v", gamepad)
	}
	if *bindings.Binding(input.DeviceGamepad, remapPause) != input.DefaultBinding || *bindings.Binding(input.DeviceKeyboard, remapFire) != input.DefaultBinding {
		t.Fatalf("missing bindings did not default: %+v", bindings)
	}
}

func TestLegacyBindingArraysAreMigrated(t *testing.T) {
	directory := t.TempDir()
	writeLegacySave(t, directory, input.BindingsFileName, `{"buttons":[0,1,5,8,9,-1,-1,-1,-1],"keys":[-1,21,2,-1,-1,-1,-1,-1,-1],"fire":7}`)
	app, _ := newTestAppIn(t, directory)
	bindings := app.Input.Bindings()
	if *bindings.Binding(input.DeviceGamepad, remapFire) != 0 || *bindings.Binding(input.DeviceGamepad, remapDash) != 5 || *bindings.Binding(input.DeviceGamepad, remapPause) != 8 {
		t.Fatalf("gamepad array lost: %+v", bindings)
	}
	if *bindings.Binding(input.DeviceKeyboard, remapFire) != input.DefaultBinding || *bindings.Binding(input.DeviceKeyboard, remapAimLock) != 21 || *bindings.Binding(input.DeviceKeyboard, remapDash) != 2 {
		t.Fatalf("keyboard array lost: %+v", bindings)
	}
	restarted, _ := newTestAppIn(t, directory)
	saved := restarted.Input.Bindings()
	if *saved.Binding(input.DeviceGamepad, remapDash) != 5 {
		t.Fatal("migrated bindings were not saved in the new format")
	}
}

func TestLegacyHighScoresAreMigrated(t *testing.T) {
	directory := t.TempDir()
	writeLegacySave(t, directory, scores.TableFileName, `{"entries":[{"initials":"GRG","score":4200,"survivedSeconds":95.5,"kills":321,"level":12},{"initials":"BOB","score":900,"survivedSeconds":30,"kills":40,"level":3}]}`)
	app, _ := newTestAppIn(t, directory)
	best := app.Scores.Best()
	if len(app.Scores.Entries) != 2 || best.Initials != "GRG" || best.Score != 4200 || best.Seconds != 95.5 || best.Stats[statKills] != 321 || best.Stats[statLevel] != 12 {
		t.Fatalf("migrated table %+v", app.Scores.Entries)
	}
	restarted, _ := newTestAppIn(t, directory)
	if restarted.Scores.Best().Stats[statKills] != 321 {
		t.Fatal("migrated scores were not saved in the new format")
	}
}

func TestLegacyBestScoreBecomesTheFirstEntry(t *testing.T) {
	directory := t.TempDir()
	writeLegacySave(t, directory, legacyHighScoreFileName, `{"score":7000,"survivedSeconds":120,"kills":500,"level":15}`)
	app, _ := newTestAppIn(t, directory)
	best := app.Scores.Best()
	if len(app.Scores.Entries) != 1 || best.Initials != legacyInitials || best.Score != 7000 || best.Seconds != 120 || best.Stats[statLevel] != 15 {
		t.Fatalf("migrated best score %+v", app.Scores.Entries)
	}
}

func TestAssigningATakenKeySwaps(t *testing.T) {
	layout := input.NewLayout(remaps, len(remaps))
	bindings := input.DefaultBindings(layout)
	bindings.Assign(layout, input.DeviceKeyboard, remapFire, int(ebiten.KeyK))
	if *bindings.Binding(input.DeviceKeyboard, remapFire) != int(ebiten.KeyK) || *bindings.Binding(input.DeviceKeyboard, remapAimLock) != int(ebiten.KeyJ) {
		t.Fatalf("keyboard swap failed: fire %d aim %d", *bindings.Binding(input.DeviceKeyboard, remapFire), *bindings.Binding(input.DeviceKeyboard, remapAimLock))
	}
	bindings.Assign(layout, input.DeviceGamepad, remapFire, 4)
	bindings.Assign(layout, input.DeviceGamepad, remapDash, 4)
	if *bindings.Binding(input.DeviceGamepad, remapDash) != 4 || *bindings.Binding(input.DeviceGamepad, remapFire) != input.DefaultBinding {
		t.Fatalf("gamepad swap failed: %+v", bindings)
	}
	labels := [3]string{bindings.Label(layout, input.DeviceKeyboard, remapFire), bindings.Label(layout, input.DeviceGamepad, remapDash), bindings.Label(layout, input.DeviceGamepad, remapFire)}
	if labels != [3]string{"K", "BUTTON 4", "A"} {
		t.Fatalf("labels %q", labels)
	}
}

func TestRemapFlowAsksEssentialsThenListsAndSaves(t *testing.T) {
	app, _ := newHomeApp(t)
	controls := openControls(app, input.DeviceGamepad)
	if controls.Step != 0 || len(controls.Menu.Options) != len(app.Input.Layout().RemapsFor(input.DeviceGamepad))+2 {
		t.Fatalf("step %d, %d rows", controls.Step, len(controls.Menu.Options))
	}
	for step, button := range []int{2, 3, 0} {
		pressButton(app, 0, button)
		if controls.Step != step+1 {
			t.Fatalf("step %d not captured", step)
		}
	}
	reloaded, err := input.LoadBindings(app.Store, app.Input.Layout())
	if err != nil || *reloaded.Binding(input.DeviceGamepad, remapFire) != 2 || *reloaded.Binding(input.DeviceGamepad, remapDash) != 0 {
		t.Fatalf("saved %+v (err %v)", reloaded, err)
	}
	controls = openControls(app, input.DeviceGamepad)
	if controls.Step != len(remaps) {
		t.Fatalf("customized pad asked the essentials again")
	}
}

func TestRemapRejectsReservedAndDefaultTakenInputs(t *testing.T) {
	app, _ := newHomeApp(t)
	if !app.Input.Layout().IsReservedKey(int(ebiten.KeyEnter)) {
		t.Fatal("enter was accepted")
	}
	controls := openControls(app, input.DeviceGamepad)
	pressButton(app, input.ActionPause|input.ActionConfirm|input.ActionStart, 9)
	if controls.Step != 0 || controls.Message == "" {
		t.Fatal("the default start button was stolen")
	}
	controls.Step = len(remaps)
	controls.Menu.Options[remapPause].Activate()
	pressButton(app, input.ActionPause|input.ActionConfirm|input.ActionStart, 9)
	applied := app.Input.Bindings()
	if controls.ListenSeconds != 0 || *applied.Binding(input.DeviceGamepad, remapPause) != 9 {
		t.Fatal("pause could not take its own button")
	}
	controls.Menu.Options[len(controls.Menu.Options)-2].Activate()
	reset := app.Input.Bindings()
	if reset.IsCustomized(input.DeviceGamepad) {
		t.Fatal("reset kept custom buttons")
	}
}

// ===== Internal =====

func writeLegacySave(tb testing.TB, directory, fileName, content string) {
	folder := filepath.Join(directory, saveFolder)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, fileName), []byte(content), 0o644); err != nil {
		tb.Fatal(err)
	}
}

func openControls(app *rig.App, device input.Device) *shell.ControlsScreen {
	app.Switch(shell.ScreenControls)
	controls := app.Screen(shell.ScreenControls).(*shell.ControlsScreen)
	controls.UseDevice(device)
	waitForMenuLock(app)
	return controls
}
