package application

import (
	"testing"
)

func TestMeasureStringDimensions(t *testing.T) {
	tests := []struct {
		text      string
		wantWidth int
		wantHght  int
	}{
		{"", 0, 0},
		{"hello", 5, 1},
		{"a\nbb\nccc", 3, 3},
		{"héllo\n日本語", 5, 2},
		{"trailing\n", 8, 2},
	}

	for _, tt := range tests {
		width, height := MeasureStringDimensions(tt.text)
		if width != tt.wantWidth || height != tt.wantHght {
			t.Errorf("MeasureStringDimensions(%q) = (%d, %d), want (%d, %d)",
				tt.text, width, height, tt.wantWidth, tt.wantHght)
		}
	}
}

func TestMessageButtonsSize(t *testing.T) {
	tests := []struct {
		buttons []string
		want    int
	}{
		{nil, 0},
		{[]string{}, 0},
		{[]string{"OK"}, 8},
		{[]string{"OK", "Cancel"}, 26},
		{[]string{"Yes", "No", "Cancel"}, 38},
	}

	for _, tt := range tests {
		if got := MessageButtonsSize(tt.buttons); got != tt.want {
			t.Errorf("MessageButtonsSize(%v) = %d, want %d", tt.buttons, got, tt.want)
		}
	}
}

func TestShowOkDialog(t *testing.T) {
	newTestHarness(t)

	var closed int
	primitive := ShowOkDialog("Title", "Something happened", func() { closed++ })
	if primitive == nil {
		t.Fatal("ShowOkDialog returned nil")
	}
	if !modalPages.HasPage(messageDialogName) {
		t.Fatal("message dialog page was not added")
	}
	if !messageDialog.HasFocus() && (len(messageDialog.Buttons) == 0 || !messageDialog.Buttons[0].HasFocus()) {
		t.Error("OK button was not focused")
	}

	messageDialog.OnButtonClick("OK", 0)

	if closed != 1 {
		t.Errorf("onClose called %d times, want 1", closed)
	}
	if modalPages.HasPage(messageDialogName) {
		t.Error("message dialog page was not removed")
	}
}

func TestShowOkDialogEscapeCallsOnClose(t *testing.T) {
	newTestHarness(t)

	var closed int
	ShowOkDialog("Title", "Message", func() { closed++ })
	messageDialog.OnClose()

	if closed != 1 {
		t.Errorf("onClose called %d times, want 1", closed)
	}
	if modalPages.HasPage(messageDialogName) {
		t.Error("message dialog page was not removed")
	}
}

func TestShowConfirmDialogConfirmAndCancel(t *testing.T) {
	newTestHarness(t)

	var confirmed, cancelled int
	ShowConfirmDialog("Are you sure?", func() { confirmed++ }, func() { cancelled++ })
	messageDialog.OnButtonClick("OK", 0)
	if confirmed != 1 || cancelled != 0 {
		t.Errorf("confirm: confirmed=%d cancelled=%d, want 1 and 0", confirmed, cancelled)
	}

	ShowConfirmDialog("Are you sure?", func() { confirmed++ }, func() { cancelled++ })
	messageDialog.OnButtonClick("Cancel", 1)
	if confirmed != 1 || cancelled != 1 {
		t.Errorf("cancel: confirmed=%d cancelled=%d, want 1 and 1", confirmed, cancelled)
	}
}

func TestShowConfirmDialogEscapeCancels(t *testing.T) {
	newTestHarness(t)

	var confirmed, cancelled int
	ShowConfirmDialog("Are you sure?", func() { confirmed++ }, func() { cancelled++ })
	messageDialog.OnClose()

	if confirmed != 0 || cancelled != 1 {
		t.Errorf("escape: confirmed=%d cancelled=%d, want 0 and 1", confirmed, cancelled)
	}
}

func TestShowMessageDialogCustomButtons(t *testing.T) {
	newTestHarness(t)

	type click struct {
		button string
		index  int
	}
	var clicks []click
	ShowMessageDialog("Pick", "Choose one", []string{"A", "B", "C"}, nil,
		func(button string, index int) {
			clicks = append(clicks, click{button, index})
		})

	if len(messageDialog.Buttons) != 3 {
		t.Fatalf("dialog has %d buttons, want 3", len(messageDialog.Buttons))
	}

	messageDialog.OnButtonClick("B", 1)
	if len(clicks) != 1 || clicks[0] != (click{"B", 1}) {
		t.Errorf("clicks = %v, want [{B 1}]", clicks)
	}
}

func TestCloseMessageDialogWithoutDialog(t *testing.T) {
	newTestHarness(t)
	// Must be a no-op rather than a nil-pointer panic.
	CloseMessageDialog()
}
