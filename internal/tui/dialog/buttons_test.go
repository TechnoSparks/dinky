package dialog

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestCreateButtonsRowSingleButton(t *testing.T) {
	flex := tview.NewFlex()
	var gotButton string
	var gotIndex = -1

	buttons := CreateButtonsRow(flex, []string{"OK"}, func(button string, index int) {
		gotButton, gotIndex = button, index
	})

	if len(buttons) != 1 {
		t.Fatalf("got %d buttons, want 1", len(buttons))
	}
	if buttons[0] == nil || buttons[0].GetLabel() != "OK" {
		t.Fatalf("button label = %v, want OK", buttons[0])
	}
	if flex.GetItemCount() != 3 {
		t.Errorf("flex item count = %d, want 3 (spacer, button, spacer)", flex.GetItemCount())
	}
	if flex.GetItem(0) != nil || flex.GetItem(2) != nil {
		t.Error("single button row should have nil spacers on both sides")
	}
	if flex.GetItem(1) != tview.Primitive(buttons[0]) {
		t.Error("button should be the middle flex item")
	}

	pressKey(buttons[0], tcell.KeyEnter, 0, tcell.ModNone)
	if gotButton != "OK" || gotIndex != 0 {
		t.Errorf("callback = (%q, %d), want (OK, 0)", gotButton, gotIndex)
	}

	// A nil callback must be tolerated.
	nilCallbackFlex := tview.NewFlex()
	nilButtons := CreateButtonsRow(nilCallbackFlex, []string{"OK"}, nil)
	pressKey(nilButtons[0], tcell.KeyEnter, 0, tcell.ModNone)

	buttons[0].Focus(func(p tview.Primitive) {})
	if !buttons[0].HasFocus() {
		t.Error("button should have focus after Focus")
	}
	buttons[0].Blur()
	if buttons[0].HasFocus() {
		t.Error("button should not have focus after Blur")
	}

	app, screen := newDialogTestApp(t, 20, 1)
	_ = app
	buttons[0].Focus(func(p tview.Primitive) {})
	flex.SetRect(0, 0, 20, 1)
	flex.Draw(screen)
	if !screenContains(screen, 20, 1, "OK") {
		t.Error("drawn single-button row does not contain the button label")
	}
}

func TestCreateButtonsRowMultipleButtons(t *testing.T) {
	flex := tview.NewFlex()
	labels := []string{"Yes", "No", "Cancel"}
	var gotButtons []string
	var gotIndices []int

	buttons := CreateButtonsRow(flex, labels, func(button string, index int) {
		gotButtons = append(gotButtons, button)
		gotIndices = append(gotIndices, index)
	})

	if len(buttons) != len(labels) {
		t.Fatalf("got %d buttons, want %d", len(buttons), len(labels))
	}
	if flex.GetItemCount() != 5 {
		t.Errorf("flex item count = %d, want 5 (3 buttons and 2 spacers)", flex.GetItemCount())
	}

	for i, label := range labels {
		if buttons[i] == nil || buttons[i].GetLabel() != label {
			t.Errorf("button %d label = %v, want %q", i, buttons[i], label)
		}
		if flex.GetItem(i*2) != tview.Primitive(buttons[i]) {
			t.Errorf("button %d is not at flex item %d", i, i*2)
		}
	}

	for i, button := range buttons {
		pressKey(button, tcell.KeyEnter, 0, tcell.ModNone)
		if gotButtons[i] != labels[i] || gotIndices[i] != i {
			t.Errorf("callback %d = (%q, %d), want (%q, %d)", i, gotButtons[i], gotIndices[i], labels[i], i)
		}
	}

	app, screen := newDialogTestApp(t, 40, 1)
	_ = app
	flex.SetRect(0, 0, 40, 1)
	flex.Draw(screen)
	for _, label := range labels {
		if !screenContains(screen, 40, 1, label) {
			t.Errorf("drawn button row does not contain %q", label)
		}
	}
}

func TestCreateButtonsRowReplacesPreviousButtons(t *testing.T) {
	flex := tview.NewFlex()
	CreateButtonsRow(flex, []string{"One", "Two"}, func(string, int) {})
	buttons := CreateButtonsRow(flex, []string{"Three"}, func(string, int) {})

	if flex.GetItemCount() != 3 {
		t.Errorf("flex item count after replacement = %d, want 3", flex.GetItemCount())
	}
	if len(buttons) != 1 || buttons[0].GetLabel() != "Three" {
		t.Errorf("replacement buttons = %v", buttons)
	}
}
