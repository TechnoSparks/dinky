package statusbar

import (
	"strings"
	"testing"

	"dinky/internal/tui/stylecolor"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newTestScreen(t *testing.T, width, height int) tcell.SimulationScreen {
	t.Helper()

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("failed to initialize simulation screen: %v", err)
	}
	screen.SetSize(width, height)
	t.Cleanup(screen.Fini)
	return screen
}

func screenRow(screen tcell.SimulationScreen, y int, width int) string {
	runes := make([]rune, 0, width)
	for x := 0; x < width; x++ {
		mainc, _, _, _ := screen.GetContent(x, y)
		if mainc == 0 {
			mainc = ' '
		}
		runes = append(runes, mainc)
	}
	return string(runes)
}

func styleAt(screen tcell.SimulationScreen, x, y int) tcell.Style {
	_, _, style, _ := screen.GetContent(x, y)
	return style
}

func assertStyleColors(t *testing.T, got tcell.Style, want tcell.Style, context string) {
	t.Helper()

	gotFG, gotBG, _ := got.Decompose()
	wantFG, wantBG, _ := want.Decompose()
	if gotFG != wantFG || gotBG != wantBG {
		t.Errorf("%s: got fg/bg %v/%v, want %v/%v", context, gotFG, gotBG, wantFG, wantBG)
	}
}

// startTestApplication runs a tview application on a SimulationScreen so that
// QueueUpdateDraw (used by the status bar's timeout timers) can be serviced.
func startTestApplication(t *testing.T) *tview.Application {
	t.Helper()

	screen := tcell.NewSimulationScreen("UTF-8")
	app := tview.NewApplication()
	app.SetScreen(screen)
	screen.SetSize(100, 10)
	app.SetRoot(tview.NewBox(), true)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = app.Run()
	}()

	// Wait for the event loop to be running before the test starts queueing.
	app.QueueUpdate(func() {})

	t.Cleanup(func() {
		app.Stop()
		<-done
	})

	return app
}

func TestNewStatusBar(t *testing.T) {
	app := tview.NewApplication()
	statusBar := NewStatusBar(app)

	if statusBar == nil {
		t.Fatal("NewStatusBar returned nil")
	}
	if statusBar.Box == nil {
		t.Fatal("NewStatusBar returned a nil Box")
	}
	if statusBar.app != app {
		t.Error("NewStatusBar did not store the application")
	}

	tests := []struct {
		name       string
		style      tcell.Style
		wantFG     tcell.Color
		wantBG     tcell.Color
		styleLabel string
	}{
		{name: "normal", style: statusBar.Style, wantFG: stylecolor.White, wantBG: stylecolor.Blue, styleLabel: "Style"},
		{name: "message", style: statusBar.MessageStyle, wantFG: stylecolor.White, wantBG: stylecolor.Green, styleLabel: "MessageStyle"},
		{name: "error", style: statusBar.ErrorStyle, wantFG: stylecolor.White, wantBG: stylecolor.Red, styleLabel: "ErrorStyle"},
		{name: "warning", style: statusBar.WarningStyle, wantFG: stylecolor.White, wantBG: stylecolor.Yellow, styleLabel: "WarningStyle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fg, bg, _ := tt.style.Decompose()
			if fg != tt.wantFG || bg != tt.wantBG {
				t.Errorf("%s fg/bg = %v/%v, want %v/%v", tt.styleLabel, fg, bg, tt.wantFG, tt.wantBG)
			}
		})
	}
}

func TestShowMessageWarningErrorPrecedenceAndStyles(t *testing.T) {
	app := startTestApplication(t)
	statusBar := NewStatusBar(app)
	statusBar.SetRect(0, 0, 100, 1)
	screen := newTestScreen(t, 100, 1)

	drawAndRead := func() string {
		t.Helper()
		statusBar.Draw(screen)
		return screenRow(screen, 0, 100)
	}

	statusBar.ShowMessage("saved")
	if statusBar.message != "saved" {
		t.Errorf("ShowMessage did not store the message: %q", statusBar.message)
	}
	row := drawAndRead()
	if !strings.Contains(row, "saved") {
		t.Errorf("message row %q does not contain message text", row)
	}
	assertStyleColors(t, styleAt(screen, 0, 0), statusBar.MessageStyle, "message style")

	statusBar.ShowWarning("careful")
	if statusBar.warningMessage != "careful" {
		t.Errorf("ShowWarning did not store the warning: %q", statusBar.warningMessage)
	}
	row = drawAndRead()
	if !strings.Contains(row, "careful") {
		t.Errorf("warning row %q does not contain warning text", row)
	}
	if strings.Contains(row, "saved") {
		t.Errorf("warning row %q still contains the message", row)
	}
	assertStyleColors(t, styleAt(screen, 0, 0), statusBar.WarningStyle, "warning style")

	statusBar.ShowError("boom")
	if statusBar.errorMessage != "boom" {
		t.Errorf("ShowError did not store the error: %q", statusBar.errorMessage)
	}
	row = drawAndRead()
	if !strings.Contains(row, "boom") {
		t.Errorf("error row %q does not contain error text", row)
	}
	if strings.Contains(row, "careful") || strings.Contains(row, "saved") {
		t.Errorf("error row %q still contains lower-priority text", row)
	}
	assertStyleColors(t, styleAt(screen, 0, 0), statusBar.ErrorStyle, "error style")

	// Without an error the warning wins over the message.
	statusBar.errorMessage = ""
	row = drawAndRead()
	if !strings.Contains(row, "careful") || strings.Contains(row, "saved") {
		t.Errorf("warning should take precedence over message, row = %q", row)
	}
	assertStyleColors(t, styleAt(screen, 0, 0), statusBar.WarningStyle, "warning style after clearing error")

	// Without warning or error the message is shown.
	statusBar.warningMessage = ""
	row = drawAndRead()
	if !strings.Contains(row, "saved") {
		t.Errorf("message should be shown when warning is cleared, row = %q", row)
	}
	assertStyleColors(t, styleAt(screen, 0, 0), statusBar.MessageStyle, "message style after clearing warning")

	// Clearing the message returns to the normal style.
	statusBar.message = ""
	statusBar.Filename = "file.txt"
	row = drawAndRead()
	if !strings.Contains(row, "file.txt") {
		t.Errorf("normal row %q does not contain filename", row)
	}
	assertStyleColors(t, styleAt(screen, 0, 0), statusBar.Style, "normal style")
}

func TestDrawOutput(t *testing.T) {
	tests := []struct {
		name         string
		statusBar    StatusBar
		wantContains []string
		wantMissing  []string
	}{
		{
			name: "plain LF file",
			statusBar: StatusBar{
				Filename:    "notes.txt",
				Line:        1,
				Col:         0,
				TabSize:     4,
				LineEndings: "LF",
			},
			wantContains: []string{"notes.txt", "    Ln 1, Col 0", "Tab Size: 4", "LF", "F12: Menu"},
			wantMissing:  []string{"M notes.txt", "OVR"},
		},
		{
			name: "modified overwrite CRLF file",
			statusBar: StatusBar{
				Filename:        "todo.md",
				IsModified:      true,
				Line:            12,
				Col:             7,
				IsOverwriteMode: true,
				TabSize:         8,
				LineEndings:     "CRLF",
			},
			wantContains: []string{"M todo.md", "OVR Ln 12, Col 7", "Tab Size: 8", "CRLF", "F12: Menu"},
			wantMissing:  []string{"    Ln"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := newTestScreen(t, 100, 1)
			statusBar := tt.statusBar
			statusBar.Box = tview.NewBox()
			statusBar.app = tview.NewApplication()
			statusBar.SetRect(0, 0, 100, 1)

			statusBar.Draw(screen)
			row := screenRow(screen, 0, 100)

			for _, want := range tt.wantContains {
				if !strings.Contains(row, want) {
					t.Errorf("row %q does not contain %q", row, want)
				}
			}
			for _, unwanted := range tt.wantMissing {
				if strings.Contains(row, unwanted) {
					t.Errorf("row %q unexpectedly contains %q", row, unwanted)
				}
			}
		})
	}
}

func TestUpdateHookIsCalledBeforeDraw(t *testing.T) {
	screen := newTestScreen(t, 100, 1)
	statusBar := NewStatusBar(tview.NewApplication())
	statusBar.SetRect(0, 0, 100, 1)

	hookCalls := 0
	statusBar.UpdateHook = func(sb *StatusBar) {
		hookCalls++
		sb.Filename = "from-hook.txt"
		sb.Line = 42
		sb.Col = 3
	}

	statusBar.Draw(screen)
	if hookCalls != 1 {
		t.Errorf("UpdateHook calls = %d, want 1", hookCalls)
	}
	row := screenRow(screen, 0, 100)
	if !strings.Contains(row, "from-hook.txt") || !strings.Contains(row, "Ln 42, Col 3") {
		t.Errorf("draw did not use values set by UpdateHook: %q", row)
	}

	statusBar.Draw(screen)
	if hookCalls != 2 {
		t.Errorf("UpdateHook calls after second draw = %d, want 2", hookCalls)
	}
}
