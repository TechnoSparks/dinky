package dialog

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sedwards2009/smidgen"
)

func newTestInputDialogOptions() InputDialogOptions {
	return InputDialogOptions{
		Title:        "Rename file",
		Message:      "Name:",
		DefaultValue: "hello.txt",
		Buttons:      []string{"OK", "Cancel"},
		Width:        60,
		Height:       10,
	}
}

func TestNewInputDialog(t *testing.T) {
	app, _ := newDialogTestApp(t, 100, 30)
	dialog := NewInputDialog(app)

	if dialog == nil {
		t.Fatal("NewInputDialog returned nil")
	}
	if dialog.Flex == nil || dialog.InputField == nil || dialog.InputLabel == nil ||
		dialog.verticalContentsFlex == nil || dialog.buttonsFlex == nil || dialog.innerFlex == nil ||
		dialog.inputFieldFlex == nil {
		t.Fatal("NewInputDialog did not initialize all internal widgets")
	}
	if dialog.app != app {
		t.Error("NewInputDialog did not store the application")
	}
}

func TestInputDialogOpenDefaultsAndAccept(t *testing.T) {
	app, screen := newDialogTestApp(t, 100, 30)
	dialog := NewInputDialog(app)
	app.SetRoot(dialog.Flex, true)

	var acceptedValues []string
	var acceptedIndices []int
	cancelCount := 0
	options := newTestInputDialogOptions()
	options.OnAccept = func(value string, index int) {
		acceptedValues = append(acceptedValues, value)
		acceptedIndices = append(acceptedIndices, index)
	}
	options.OnCancel = func() { cancelCount++ }

	dialog.Open(options)

	if got := dialog.verticalContentsFlex.GetTitle(); got != options.Title {
		t.Errorf("title = %q, want %q", got, options.Title)
	}
	if got := dialog.InputLabel.GetText(true); got != options.Message {
		t.Errorf("label = %q, want %q", got, options.Message)
	}
	if got := dialog.InputField.GetText(); got != options.DefaultValue {
		t.Errorf("default value = %q, want %q", got, options.DefaultValue)
	}
	if !dialog.InputField.HasFocus() {
		t.Error("input field should have focus after Open")
	}
	if len(dialog.Buttons) != 2 || dialog.Buttons[0].GetLabel() != "OK" || dialog.Buttons[1].GetLabel() != "Cancel" {
		t.Fatalf("unexpected buttons: %v", dialog.Buttons)
	}

	// Enter in the input field accepts with index -1.
	pressKey(dialog.InputField, tcell.KeyEnter, 0, tcell.ModNone)
	if len(acceptedValues) != 1 || acceptedValues[0] != options.DefaultValue || acceptedIndices[0] != -1 {
		t.Errorf("input field accept = %v/%v, want [%q]/[-1]", acceptedValues, acceptedIndices, options.DefaultValue)
	}

	// Buttons accept with their index.
	for i, btn := range dialog.Buttons {
		btn.SetRect(10+i*12, 5, 10, 1)
		rx, ry, rw, rh := btn.GetRect()
		x := rx + rw/2
		y := ry + rh/2
		handler := btn.MouseHandler()
		handler(tview.MouseLeftClick, tcell.NewEventMouse(x, y, tcell.Button1, tcell.ModNone), func(p tview.Primitive) {})
		_ = i
	}
	if len(acceptedValues) != 3 {
		t.Fatalf("accepted values = %v, want three calls", acceptedValues)
	}
	if acceptedIndices[1] != 0 || acceptedIndices[2] != 1 {
		t.Errorf("button indices = %v, want [0 1]", acceptedIndices[1:])
	}
	if acceptedValues[1] != options.DefaultValue || acceptedValues[2] != options.DefaultValue {
		t.Errorf("button accepted values = %v, want default value", acceptedValues[1:])
	}

	// Escape on either the field or a button cancels.
	pressKey(dialog.InputField, tcell.KeyEscape, 0, tcell.ModNone)
	pressKey(dialog.Buttons[0], tcell.KeyEscape, 0, tcell.ModNone)
	if cancelCount != 2 {
		t.Errorf("cancel calls = %d, want 2", cancelCount)
	}
	if len(acceptedValues) != 3 {
		t.Errorf("Escape unexpectedly accepted a value: %v", acceptedValues)
	}

	app.ForceDraw()
	for _, want := range []string{options.Title, options.Message, options.DefaultValue, "OK", "Cancel"} {
		if !screenContains(screen, 100, 30, want) {
			t.Errorf("drawn input dialog does not contain %q", want)
		}
	}
}

func TestInputDialogFieldKeyFilter(t *testing.T) {
	app, _ := newDialogTestApp(t, 100, 30)
	dialog := NewInputDialog(app)
	app.SetRoot(dialog.Flex, true)

	filterCalls := 0
	options := newTestInputDialogOptions()
	options.DefaultValue = "abc"
	options.OnAccept = func(string, int) {}
	options.FieldKeyFilter = func(event *tcell.EventKey) bool {
		filterCalls++
		return event.Rune() != 'x'
	}
	dialog.Open(options)

	before := dialog.InputField.GetText()
	pressKey(dialog.InputField, tcell.KeyRune, 'x', tcell.ModNone)
	if filterCalls != 1 {
		t.Errorf("field key filter calls = %d, want 1", filterCalls)
	}
	if got := dialog.InputField.GetText(); got != before {
		t.Errorf("rejected key changed field text from %q to %q", before, got)
	}

	allowed := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
	if got := dialog.inputFilter(allowed); got != allowed {
		t.Error("inputFilter should pass through keys permitted by FieldKeyFilter")
	}
	rejected := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	if got := dialog.inputFilter(rejected); got != nil {
		t.Error("inputFilter should swallow keys rejected by FieldKeyFilter")
	}
	if filterCalls != 3 {
		t.Errorf("field key filter calls = %d, want 3", filterCalls)
	}

	// With no filter configured, events pass through.
	dialog.options.FieldKeyFilter = nil
	event := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	if got := dialog.inputFilter(event); got != event {
		t.Error("inputFilter should pass events through when no FieldKeyFilter is set")
	}
}

func TestInputDialogFocusNavigation(t *testing.T) {
	app, _ := newDialogTestApp(t, 100, 30)
	dialog := NewInputDialog(app)
	app.SetRoot(dialog.Flex, true)
	options := newTestInputDialogOptions()
	options.OnAccept = func(string, int) {}
	dialog.Open(options)

	if !dialog.InputField.HasFocus() {
		t.Fatal("input field should start focused")
	}

	pressKey(dialog.InputField, tcell.KeyTab, 0, tcell.ModNone)
	if !dialog.Buttons[0].HasFocus() {
		t.Error("Tab from the input field should focus the first button")
	}

	pressKey(dialog.Buttons[0], tcell.KeyTab, 0, tcell.ModNone)
	if !dialog.Buttons[1].HasFocus() {
		t.Error("Tab from the first button should focus the second button")
	}

	pressKey(dialog.Buttons[1], tcell.KeyTab, 0, tcell.ModNone)
	if !dialog.InputField.HasFocus() {
		t.Error("Tab from the last button should focus the input field")
	}

	pressKey(dialog.InputField, tcell.KeyTab, 0, tcell.ModShift)
	if !dialog.Buttons[1].HasFocus() {
		t.Error("Shift+Tab from the input field should focus the previous widget")
	}

	app.SetFocus(dialog.Buttons[1])
	pressKey(dialog.Buttons[1], tcell.KeyLeft, 0, tcell.ModNone)
	if !dialog.Buttons[0].HasFocus() {
		t.Error("Left should move focus to the previous button")
	}

	pressKey(dialog.Buttons[0], tcell.KeyRight, 0, tcell.ModNone)
	if !dialog.Buttons[1].HasFocus() {
		t.Error("Right should move focus to the next button")
	}

	// Focus delegates to the input field.
	app.SetFocus(dialog.Buttons[0])
	dialog.Focus(func(p tview.Primitive) { app.SetFocus(p) })
	if !dialog.InputField.HasFocus() {
		t.Error("Focus should delegate to the input field")
	}

	app.SetFocus(dialog.Buttons[1])
	pressKey(dialog.Buttons[1], tcell.KeyRight, 0, tcell.ModNone)
	if !dialog.Buttons[1].HasFocus() {
		t.Error("Right on the last button should keep focus there")
	}
}

func TestInputDialogMouseHandlerConsumesEvents(t *testing.T) {
	app, _ := newDialogTestApp(t, 100, 30)
	dialog := NewInputDialog(app)
	app.SetRoot(dialog.Flex, true)
	options := newTestInputDialogOptions()
	options.OnAccept = func(string, int) {}
	dialog.Open(options)

	consumed, capture := dialog.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(1, 1, tcell.Button1, tcell.ModNone),
		func(p tview.Primitive) {},
	)
	if !consumed || capture != nil {
		t.Errorf("MouseHandler returned consumed=%v capture=%v, want true, nil", consumed, capture)
	}
}

func TestInputDialogSetSmidgenKeybindings(t *testing.T) {
	app, _ := newDialogTestApp(t, 100, 30)
	dialog := NewInputDialog(app)

	// Must not panic; the keybindings are passed to the embedded smidgen view.
	dialog.SetSmidgenKeybindings(smidgen.Keybindings{})
}
