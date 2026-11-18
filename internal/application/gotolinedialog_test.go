package application

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/sedwards2009/smidgen/micro/buffer"
)

const goToLineSample = "first line\nsecond line\nthird line\nfourth line"

func TestParseAndGoToLine(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantX     int
		wantY     int
		wantError bool
	}{
		{"single line number", "3", 0, 2, false},
		{"line and column", "2:3", 2, 1, false},
		{"column clamps to line length", "2:999", len("second line"), 1, false},
		{"negative column clamps to zero", "2:-8", 0, 1, false},
		{"last line", "4", 0, 3, false},
		{"empty input is ignored", "", 1, 1, false},
		{"zero is out of range", "0", 1, 1, true},
		{"too large is out of range", "99", 1, 1, true},
		{"non numeric line", "abc", 1, 1, true},
		{"bad column", "2:abc", 1, 1, true},
		{"missing line before colon", ":3", 1, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHarness(t)
			h.open(goToLineSample, "")
			// Start somewhere other than (1,1) so ignored/error paths are visible.
			h.setCursor(1, 1)

			parseAndGoToLine(tt.input)

			cursor := currentFileBuffer.editor.Cursor()
			if cursor.X != tt.wantX || cursor.Y != tt.wantY {
				t.Errorf("input %q moved cursor to (%d,%d), want (%d,%d)",
					tt.input, cursor.X, cursor.Y, tt.wantX, tt.wantY)
			}
			statusText := h.statusBarText()
			if tt.wantError {
				if !strings.Contains(statusText, "Invalid") && !strings.Contains(statusText, "out of range") {
					t.Errorf("input %q did not show an error, status bar = %q", tt.input, statusText)
				}
			} else if tt.input != "" && !strings.Contains(statusText, "Jumped to line") {
				t.Errorf("input %q did not show the jump message, status bar = %q", tt.input, statusText)
			}
		})
	}
}

func TestParseAndGoToLineLocationsUseZeroBasedBuffer(t *testing.T) {
	h := newTestHarness(t)
	h.open(goToLineSample, "")
	h.setCursor(0, 0)

	parseAndGoToLine("2:7")

	cursor := currentFileBuffer.editor.Cursor()
	if cursor.Loc != (buffer.Loc{X: 6, Y: 1}) {
		t.Errorf("cursor loc = %+v, want {6 1}", cursor.Loc)
	}
	if !strings.Contains(h.statusBarText(), "Jumped to line 2") {
		t.Errorf("status bar = %q, want it to contain %q", h.statusBarText(), "Jumped to line 2")
	}
}

func TestHandleGoToLineOpensInputDialog(t *testing.T) {
	newTestHarness(t).open(goToLineSample, "")

	primitive := handleGoToLine()
	if primitive == nil {
		t.Fatal("handleGoToLine returned nil")
	}
	if !modalPages.HasPage(inputDialogName) {
		t.Fatal("input dialog page was not added")
	}
	if inputDialog.InputField.GetText() != "" {
		t.Errorf("input field text = %q, want empty", inputDialog.InputField.GetText())
	}

	// Cancel removes the page.
	CloseInputDialog()
	if modalPages.HasPage(inputDialogName) {
		t.Error("input dialog page was not removed on close")
	}
}

func TestNumericInputFilter(t *testing.T) {
	tests := []struct {
		event *tcell.EventKey
		want  bool
	}{
		{tcell.NewEventKey(tcell.KeyRune, '0', tcell.ModNone), true},
		{tcell.NewEventKey(tcell.KeyRune, '9', tcell.ModNone), true},
		{tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone), false},
		{tcell.NewEventKey(tcell.KeyRune, ':', tcell.ModNone), false},
		{tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone), true},
		{tcell.NewEventKey(tcell.KeyDelete, 0, tcell.ModNone), true},
		{tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), true},
		{tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone), true},
		{tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone), true},
		{tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone), true},
		{tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), false},
	}

	for _, tt := range tests {
		got := numericInputFilter(tt.event)
		if got != tt.want {
			t.Errorf("numericInputFilter(key=%v rune=%q) = %v, want %v",
				tt.event.Key(), tt.event.Rune(), got, tt.want)
		}
	}
}
