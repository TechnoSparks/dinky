package settingsdialog_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"dinky/internal/application/settingstype"
	"dinky/internal/tui/settingsdialog"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sedwards2009/smidgen/micro/clipboard"
)

const screenWidth = 80
const screenHeight = 24

func newTestDialog(t *testing.T) (*settingsdialog.SettingsDialog, tcell.SimulationScreen) {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("failed to initialise simulation screen: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(screenWidth, screenHeight)

	sd := settingsdialog.NewSettingsDialog(tview.NewApplication())
	sd.SetSettings(settingstype.DefaultSettings())
	draw(sd, screen)
	return sd, screen
}

func draw(sd *settingsdialog.SettingsDialog, screen tcell.SimulationScreen) {
	screen.Clear()
	sd.SetRect(0, 0, screenWidth, screenHeight)
	sd.Draw(screen)
	screen.Show()
}

func screenRow(screen tcell.SimulationScreen, y int) string {
	cells, width, _ := screen.GetContents()
	var sb strings.Builder
	for x := 0; x < width; x++ {
		r := ' '
		if len(cells[y*width+x].Runes) > 0 {
			r = cells[y*width+x].Runes[0]
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// findText returns the screen column and row where text starts, or -1, -1.
func findText(screen tcell.SimulationScreen, text string) (int, int) {
	for y := 0; y < screenHeight; y++ {
		row := screenRow(screen, y)
		if i := strings.Index(row, text); i >= 0 {
			return utf8.RuneCountInString(row[:i]), y
		}
	}
	return -1, -1
}

func screenContains(screen tcell.SimulationScreen, text string) bool {
	x, _ := findText(screen, text)
	return x >= 0
}

func cellStyle(screen tcell.SimulationScreen, x, y int) tcell.Style {
	_, _, style, _ := screen.GetContent(x, y)
	return style
}

func click(t *testing.T, sd *settingsdialog.SettingsDialog, screen tcell.SimulationScreen, x, y int) {
	t.Helper()
	event := tcell.NewEventMouse(x, y, tcell.Button1, tcell.ModNone)
	sd.MouseHandler()(tview.MouseLeftClick, event, func(p tview.Primitive) {})
	draw(sd, screen)
}

func clickText(t *testing.T, sd *settingsdialog.SettingsDialog, screen tcell.SimulationScreen, text string) {
	t.Helper()
	x, y := findText(screen, text)
	if x < 0 {
		t.Fatalf("%q not found on screen", text)
	}
	click(t, sd, screen, x+1, y)
}

func assertEditorTab(t *testing.T, screen tcell.SimulationScreen) {
	t.Helper()
	if !screenContains(screen, "Defaults") {
		t.Error("Editor tab not shown: \"Defaults\" missing")
	}
	if screenContains(screen, "Color Scheme:") {
		t.Error("Theme tab content shown on Editor tab")
	}
}

func assertThemeTab(t *testing.T, screen tcell.SimulationScreen) {
	t.Helper()
	if !screenContains(screen, "Color Scheme:") {
		t.Error("Theme tab not shown: \"Color Scheme:\" missing")
	}
	if screenContains(screen, "Defaults") {
		t.Error("Editor tab content shown on Theme tab")
	}
}

func TestSettingsDialogOpensOnEditorTab(t *testing.T) {
	_, screen := newTestDialog(t)
	assertEditorTab(t, screen)
	if !screenContains(screen, "Ensure Final Newline:") {
		t.Error("Editor tab is missing the Ensure Final Newline option")
	}
}

func TestSettingsDialogSwitchesTabs(t *testing.T) {
	sd, screen := newTestDialog(t)
	x, y := findText(screen, "Theme")
	inactiveStyle := cellStyle(screen, x, y)

	clickText(t, sd, screen, "Theme")
	assertThemeTab(t, screen)
	if cellStyle(screen, x, y) == inactiveStyle {
		t.Error("Theme tab is not highlighted after clicking it")
	}

	clickText(t, sd, screen, "Editor")
	assertEditorTab(t, screen)
}

func TestSettingsDialogClickOutsideTabsKeepsCurrentTab(t *testing.T) {
	sd, screen := newTestDialog(t)

	clickText(t, sd, screen, "Theme")
	x, y := findText(screen, "Theme")
	activeStyle := cellStyle(screen, x, y)

	click(t, sd, screen, x+20, y)
	assertThemeTab(t, screen)
	if got := cellStyle(screen, x, y); got != activeStyle {
		t.Error("Theme tab lost its highlight after clicking outside the tabs")
	}
}

func TestSettingsDialogSetSettingsResetsToEditorTab(t *testing.T) {
	sd, screen := newTestDialog(t)

	clickText(t, sd, screen, "Theme")
	assertThemeTab(t, screen)

	// The dialog is reused, so reopening calls SetSettings again.
	sd.SetSettings(settingstype.DefaultSettings())
	draw(sd, screen)
	assertEditorTab(t, screen)
}

func TestSettingsDialogClipboardTab(t *testing.T) {
	sd, screen := newTestDialog(t)

	clickText(t, sd, screen, "Clipboard")
	if !screenContains(screen, "Terminal (OSC 52)") {
		t.Error("Clipboard tab not shown: \"Terminal (OSC 52)\" missing")
	}
	if screenContains(screen, "Defaults") || screenContains(screen, "Color Scheme:") {
		t.Error("other tab content shown on Clipboard tab")
	}
	if !screenContains(screen, "works over SSH") {
		t.Error("description for the default Terminal mode is missing")
	}

	// Moving the selection shows the description of that mode.
	sd.ClipboardList.SetCurrentItem(1)
	draw(sd, screen)
	if !screenContains(screen, "Needs a desktop session") {
		t.Error("description did not change to the System mode")
	}
}

func TestSettingsDialogClipboardSettingRoundTrip(t *testing.T) {
	clipboardModes := []string{clipboard.ModeTerminal, clipboard.ModeExternal, clipboard.ModeInternal}
	for _, clipboardMode := range clipboardModes {
		sd, _ := newTestDialog(t)
		settings := settingstype.DefaultSettings()
		settings.Clipboard = clipboardMode
		sd.SetSettings(settings)

		var got settingstype.Settings
		sd.SetOkFunc(func(newSettings settingstype.Settings) {
			got = newSettings
		})
		sd.OkButton.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})

		if got.Clipboard != clipboardMode {
			t.Errorf("OK returned Clipboard = %q, want %q", got.Clipboard, clipboardMode)
		}
	}
}

func TestSettingsDialogCopyOnSelectRoundTrip(t *testing.T) {
	copyOnSelectValues := []bool{true, false}
	for _, copyOnSelect := range copyOnSelectValues {
		sd, screen := newTestDialog(t)
		settings := settingstype.DefaultSettings()
		settings.CopyOnSelect = copyOnSelect
		sd.SetSettings(settings)

		clickText(t, sd, screen, "Clipboard")
		if !screenContains(screen, "Copy on Select:") {
			t.Error("Clipboard tab is missing the Copy on Select option")
		}

		var got settingstype.Settings
		sd.SetOkFunc(func(newSettings settingstype.Settings) {
			got = newSettings
		})
		sd.OkButton.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})

		if got.CopyOnSelect != copyOnSelect {
			t.Errorf("OK returned CopyOnSelect = %v, want %v", got.CopyOnSelect, copyOnSelect)
		}
	}
}
