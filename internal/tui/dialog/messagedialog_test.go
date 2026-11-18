package dialog

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestNewMessageDialog(t *testing.T) {
	app, _ := newDialogTestApp(t, 100, 30)
	dialog := NewMessageDialog(app)

	if dialog == nil {
		t.Fatal("NewMessageDialog returned nil")
	}
	if dialog.Flex == nil || dialog.messageView == nil || dialog.verticalContentsFlex == nil ||
		dialog.buttonsFlex == nil || dialog.innerFlex == nil {
		t.Fatal("NewMessageDialog did not initialize all internal widgets")
	}
	if dialog.app != app {
		t.Error("NewMessageDialog did not store the application")
	}
	if got := dialog.Flex.GetItemCount(); got != 3 {
		t.Errorf("top-level flex item count = %d, want 3", got)
	}
}

func TestMessageDialogOpenAndButtonCallbacks(t *testing.T) {
	app, screen := newDialogTestApp(t, 100, 30)
	dialog := NewMessageDialog(app)
	app.SetRoot(dialog.Flex, true)

	var clickedButtons []string
	var clickedIndices []int
	closeCount := 0
	dialog.OnButtonClick = func(button string, index int) {
		clickedButtons = append(clickedButtons, button)
		clickedIndices = append(clickedIndices, index)
	}
	dialog.OnClose = func() { closeCount++ }

	dialog.Open("Dialog title", "The message body", []string{"OK", "Cancel"}, 60, 10)

	if got := dialog.verticalContentsFlex.GetTitle(); got != "Dialog title" {
		t.Errorf("title = %q, want Dialog title", got)
	}
	if got := dialog.messageView.GetText(true); got != "The message body" {
		t.Errorf("message = %q, want The message body", got)
	}
	if len(dialog.Buttons) != 2 {
		t.Fatalf("button count = %d, want 2", len(dialog.Buttons))
	}
	if dialog.Buttons[0].GetLabel() != "OK" || dialog.Buttons[1].GetLabel() != "Cancel" {
		t.Errorf("button labels = %q, %q", dialog.Buttons[0].GetLabel(), dialog.Buttons[1].GetLabel())
	}

	pressKey(dialog.Buttons[0], tcell.KeyEnter, 0, tcell.ModNone)
	pressKey(dialog.Buttons[1], tcell.KeyEnter, 0, tcell.ModNone)
	if len(clickedButtons) != 2 || clickedButtons[0] != "OK" || clickedButtons[1] != "Cancel" {
		t.Errorf("clicked buttons = %v, want [OK Cancel]", clickedButtons)
	}
	if len(clickedIndices) != 2 || clickedIndices[0] != 0 || clickedIndices[1] != 1 {
		t.Errorf("clicked indices = %v, want [0 1]", clickedIndices)
	}

	// Escape is handled by the input filter and invokes OnClose without
	// selecting the focused button.
	pressKey(dialog.Buttons[0], tcell.KeyEscape, 0, tcell.ModNone)
	if closeCount != 1 {
		t.Errorf("OnClose calls = %d, want 1", closeCount)
	}
	if len(clickedButtons) != 2 {
		t.Errorf("Escape unexpectedly selected a button: %v", clickedButtons)
	}

	app.ForceDraw()
	for _, want := range []string{"Dialog title", "The message body", "OK", "Cancel"} {
		if !screenContains(screen, 100, 30, want) {
			t.Errorf("drawn dialog does not contain %q", want)
		}
	}
}

func TestMessageDialogFocus(t *testing.T) {
	app, _ := newDialogTestApp(t, 100, 30)
	dialog := NewMessageDialog(app)
	app.SetRoot(dialog.Flex, true)
	dialog.Open("Title", "Message", []string{"One", "Two", "Three"}, 60, 10)

	dialog.Focus(func(p tview.Primitive) { app.SetFocus(p) })
	if !dialog.Buttons[0].HasFocus() {
		t.Error("Focus should delegate to the first button")
	}

	dialog.FocusButton(2)
	if !dialog.Buttons[2].HasFocus() {
		t.Error("FocusButton(2) should focus the third button")
	}
	if dialog.Buttons[0].HasFocus() {
		t.Error("previous button should lose focus")
	}

	// Left/right and Tab navigate between buttons through the input filter.
	app.SetFocus(dialog.Buttons[1])
	pressKey(dialog.Buttons[1], tcell.KeyLeft, 0, tcell.ModNone)
	if !dialog.Buttons[0].HasFocus() {
		t.Error("Left should move focus to the previous button")
	}

	pressKey(dialog.Buttons[0], tcell.KeyRight, 0, tcell.ModNone)
	if !dialog.Buttons[1].HasFocus() {
		t.Error("Right should move focus to the next button")
	}

	pressKey(dialog.Buttons[1], tcell.KeyTab, 0, tcell.ModNone)
	if !dialog.Buttons[2].HasFocus() {
		t.Error("Tab on the middle button should move to the last button")
	}
	pressKey(dialog.Buttons[2], tcell.KeyTab, 0, tcell.ModNone)
	if !dialog.Buttons[0].HasFocus() {
		t.Error("Tab on the last button should wrap to the first")
	}

	app.SetFocus(dialog.Buttons[0])
	pressKey(dialog.Buttons[0], tcell.KeyLeft, 0, tcell.ModNone)
	if !dialog.Buttons[0].HasFocus() {
		t.Error("Left on the first button should keep focus there")
	}
	app.SetFocus(dialog.Buttons[2])
	pressKey(dialog.Buttons[2], tcell.KeyRight, 0, tcell.ModNone)
	if !dialog.Buttons[2].HasFocus() {
		t.Error("Right on the last button should keep focus there")
	}
}

func TestMessageDialogFocusWithoutButtons(t *testing.T) {
	app, _ := newDialogTestApp(t, 100, 30)
	dialog := NewMessageDialog(app)

	delegated := false
	dialog.Focus(func(p tview.Primitive) { delegated = true })
	if delegated {
		t.Error("Focus should not delegate when there are no buttons")
	}
}

func TestMessageDialogMouseHandlerConsumesEvents(t *testing.T) {
	app, _ := newDialogTestApp(t, 100, 30)
	dialog := NewMessageDialog(app)
	app.SetRoot(dialog.Flex, true)
	dialog.Open("Title", "Message", []string{"OK"}, 60, 10)

	consumed, capture := dialog.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(1, 1, tcell.Button1, tcell.ModNone),
		func(p tview.Primitive) {},
	)
	if !consumed || capture != nil {
		t.Errorf("MouseHandler returned consumed=%v capture=%v, want true, nil", consumed, capture)
	}
}
