package application

import (
	"testing"

	"dinky/internal/application/settingstype"
	"dinky/internal/tui/menu"
	"dinky/internal/tui/statusbar"
	"dinky/internal/tui/tabbar"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sedwards2009/smidgen"
	"github.com/sedwards2009/smidgen/micro/buffer"
)

// testHarness wires up the application globals the same way Main does, but
// backed by a tcell SimulationScreen so tests never touch a real terminal.
type testHarness struct {
	t      *testing.T
	screen tcell.SimulationScreen
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()

	// Reset the process-wide application state.
	fileBuffers = nil
	fileBufferID = ""
	currentFileBuffer = nil
	findbarRecentSearchTextHistory = nil
	findbarRecentReplaceTextHistory = nil
	listDialog = nil
	inputDialog = nil
	fileDialog = nil
	settingsDialog = nil

	settings = settingstype.DefaultSettings()
	initKeyBindings()

	var ok bool
	colorscheme, ok = smidgen.LoadInternalColorscheme(settings.ColorScheme)
	if !ok {
		t.Fatalf("failed to load colorscheme %q", settings.ColorScheme)
	}

	// Do not call app.SetScreen here: a real screen makes Application.Stop()
	// try to hand the screen (and later nil) back through an internal channel,
	// which blocks once that channel is full. Tests only draw primitives
	// directly onto the simulation screen.
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("failed to initialise simulation screen: %v", err)
	}
	screen.SetSize(120, 40)
	app = tview.NewApplication()
	t.Cleanup(func() {
		screen.Fini()
	})

	modalPages = tview.NewPages()
	editorPages = tview.NewPages()

	workspaceFlex := tview.NewFlex()
	workspaceFlex.AddItem(editorPages, 0, 1, true)
	modalPages.AddPage("workspace", workspaceFlex, true, true)
	app.SetRoot(modalPages, true)

	menuBar = menu.NewMenuBar()
	menus = createMenus()
	syncMenuKeyBindings(menus, actionToKeyMapping)
	menuBar.SetMenus(menus)

	tabBarLine = tabbar.NewTabBar()
	statusBar = statusbar.NewStatusBar(app)

	t.Cleanup(func() {
		fileBuffers = nil
		fileBufferID = ""
		currentFileBuffer = nil
	})

	return &testHarness{t: t, screen: screen}
}

// open creates a new file buffer, making it the current one.
func (h *testHarness) open(contents, filename string) *FileBuffer {
	h.t.Helper()
	newFile(contents, filename)
	if currentFileBuffer == nil {
		h.t.Fatal("newFile did not set currentFileBuffer")
	}
	return currentFileBuffer
}

// text returns the full buffer contents.
func (h *testHarness) text() string {
	h.t.Helper()
	return string(currentFileBuffer.buffer.LineArray.Bytes())
}

// setSelection selects the text between two buffer locations.
func (h *testHarness) setSelection(startX, startY, endX, endY int) {
	h.t.Helper()
	cursor := currentFileBuffer.editor.Cursor()
	cursor.CurSelection[0] = buffer.Loc{X: startX, Y: startY}
	cursor.CurSelection[1] = buffer.Loc{X: endX, Y: endY}
	cursor.Loc = buffer.Loc{X: endX, Y: endY}
}

// selectAll selects the entire buffer.
func (h *testHarness) selectAll() {
	h.t.Helper()
	buf := currentFileBuffer.buffer
	h.setSelection(0, 0, buf.LineArray.End().X, buf.LineArray.End().Y)
}

// selectLine selects a whole line, including its newline when present.
func (h *testHarness) selectLine(y int) {
	h.t.Helper()
	buf := currentFileBuffer.buffer
	if y+1 < buf.LinesNum() {
		h.setSelection(0, y, 0, y+1)
		return
	}
	h.setSelection(0, y, buf.LineArray.End().X, y)
}

// menuItem finds a menu item by ID across all menus.
func (h *testHarness) menuItem(id string) *menu.MenuItem {
	for _, m := range menus {
		for _, item := range m.Items {
			if item.ID == id {
				return item
			}
		}
	}
	return nil
}

// screenText reads a row of the simulation screen.
func (h *testHarness) screenRow(y, width int) string {
	runes := make([]rune, 0, width)
	for x := 0; x < width; x++ {
		mainc, _, _, _ := h.screen.GetContent(x, y)
		runes = append(runes, mainc)
	}
	return string(runes)
}

// findBuffer returns a file buffer by ID.
func (h *testHarness) findBuffer(id string) *FileBuffer {
	return getFileBufferByID(id)
}

// drawTabBar lays out and draws the tab bar onto the simulation screen.
func (h *testHarness) drawTabBar(width int) {
	h.t.Helper()
	tabBarLine.SetRect(0, 0, width, 1)
	tabBarLine.Draw(h.screen)
}

// setCursor moves the current buffer's cursor.
func (h *testHarness) setCursor(x, y int) {
	h.t.Helper()
	currentFileBuffer.editor.Cursor().Loc = buffer.Loc{X: x, Y: y}
}

// line returns a single line of the current buffer.
func (h *testHarness) line(y int) string {
	h.t.Helper()
	return currentFileBuffer.buffer.Line(y)
}

// lines returns all lines of the current buffer.
func (h *testHarness) lines() []string {
	h.t.Helper()
	result := make([]string, currentFileBuffer.buffer.LinesNum())
	for i := range result {
		result[i] = currentFileBuffer.buffer.Line(i)
	}
	return result
}

// statusBarText renders the status bar and returns the first row of text.
func (h *testHarness) statusBarText() string {
	h.t.Helper()
	statusBar.SetRect(0, 0, 120, 1)
	statusBar.Draw(h.screen)
	return h.screenRow(0, 120)
}
