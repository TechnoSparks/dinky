package table2

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Box drawing runes used by tview.Borders. They are spelled out here so that the
// tests assert the actual runes that end up on the screen.
const (
	runeHorizontal = '─'
	runeVertical   = '│'
	runeTopLeft    = '┌'
	runeTopRight   = '┐'
	runeLeftT      = '├'
	runeRightT     = '┤'
	runeTopT       = '┬'
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

// screenRow returns the runes of one screen row as a string.
func screenRow(screen tcell.Screen, y, width int) string {
	var b strings.Builder
	for x := 0; x < width; x++ {
		r, _, _, _ := screen.GetContent(x, y)
		b.WriteRune(r)
	}
	return b.String()
}

func styleAt(t *testing.T, screen tcell.Screen, x, y int) tcell.Style {
	t.Helper()
	_, _, style, _ := screen.GetContent(x, y)
	return style
}

func foregroundAt(t *testing.T, screen tcell.Screen, x, y int) tcell.Color {
	t.Helper()
	fg, _, _ := styleAt(t, screen, x, y).Decompose()
	return fg
}

func backgroundAt(t *testing.T, screen tcell.Screen, x, y int) tcell.Color {
	t.Helper()
	_, bg, _ := styleAt(t, screen, x, y).Decompose()
	return bg
}

func runeAt(t *testing.T, screen tcell.Screen, x, y int) rune {
	t.Helper()
	r, _, _, _ := screen.GetContent(x, y)
	return r
}

func drawTable(t *testing.T, table *Table, screen tcell.Screen) {
	t.Helper()
	x, y, width, height := table.GetRect()
	if width <= 0 || height <= 0 {
		t.Fatalf("table has an invalid rectangle: %d,%d %dx%d", x, y, width, height)
	}
	table.Draw(screen)
}

// sendKey feeds a key event to the table's input handler.
func sendKey(table *Table, key tcell.Key, r rune) {
	table.InputHandler()(tcell.NewEventKey(key, r, tcell.ModNone), func(tview.Primitive) {})
}

// sendKeyWithFocus feeds a key event to the table's input handler while
// recording any focus changes.
func sendKeyWithFocus(table *Table, key tcell.Key, r rune, focused *tview.Primitive) {
	table.InputHandler()(tcell.NewEventKey(key, r, tcell.ModNone), func(p tview.Primitive) {
		*focused = p
	})
}

func TestNewTableDefaults(t *testing.T) {
	table := NewTable()

	if got := table.GetRowCount(); got != 0 {
		t.Errorf("GetRowCount() = %d, want 0", got)
	}
	if got := table.GetColumnCount(); got != 0 {
		t.Errorf("GetColumnCount() = %d, want 0", got)
	}
	if rows, columns := table.GetSelectable(); rows || columns {
		t.Errorf("GetSelectable() = %v, %v, want false, false", rows, columns)
	}
	if row, column := table.GetSelection(); row != 0 || column != 0 {
		t.Errorf("GetSelection() = %d, %d, want 0, 0", row, column)
	}
	if row, column := table.GetOffset(); row != 0 || column != 0 {
		t.Errorf("GetOffset() = %d, %d, want 0, 0", row, column)
	}
	if got := table.GetXScroll(); got != 0 {
		t.Errorf("GetXScroll() = %d, want 0", got)
	}
	if table.borders {
		t.Error("new table must not draw borders")
	}
	if table.separator != ' ' {
		t.Errorf("default separator = %q, want space", table.separator)
	}
	if table.doubleClickDuration != StandardDoubleClick {
		t.Errorf("double click duration = %v, want %v", table.doubleClickDuration, StandardDoubleClick)
	}
	if cell := table.GetCell(0, 0); cell == nil {
		t.Fatal("GetCell() returned nil for a table without cells")
	} else if cell.Text != "" {
		t.Errorf("empty table cell text = %q, want empty", cell.Text)
	}
}

func TestTableSetAndGetCell(t *testing.T) {
	table := NewTable()

	table.SetCellSimple(0, 0, "a")
	table.SetCellSimple(1, 1, "b")
	if got := table.GetRowCount(); got != 2 {
		t.Errorf("GetRowCount() = %d, want 2", got)
	}
	if got := table.GetColumnCount(); got != 2 {
		t.Errorf("GetColumnCount() = %d, want 2", got)
	}
	if got := table.GetCell(1, 1).Text; got != "b" {
		t.Errorf("cell (1,1) text = %q, want %q", got, "b")
	}
	// Cells that were never set are returned as empty cells but are not
	// inserted into the table.
	if got := table.GetCell(0, 1).Text; got != "" {
		t.Errorf("cell (0,1) text = %q, want empty", got)
	}
	if table.GetCell(0, 1) == table.GetCell(0, 1) {
		t.Error("uninitialized cells should not be cached in the table")
	}
	if got := table.GetRowCount(); got != 2 {
		t.Errorf("reading a cell changed the row count to %d", got)
	}

	// Setting a cell far away extends the table with empty cells.
	table.SetCellSimple(4, 3, "far")
	if got := table.GetRowCount(); got != 5 {
		t.Errorf("GetRowCount() = %d, want 5", got)
	}
	if got := table.GetColumnCount(); got != 4 {
		t.Errorf("GetColumnCount() = %d, want 4", got)
	}
	if got := table.GetCell(4, 3).Text; got != "far" {
		t.Errorf("cell (4,3) text = %q, want %q", got, "far")
	}
	if got := table.GetCell(3, 2).Text; got != "" {
		t.Errorf("intermediate cell text = %q, want empty", got)
	}

	// A directly instantiated cell can be used and the same pointer is
	// returned by GetCell.
	cell := &TableCell{Text: "custom", Align: tview.AlignRight}
	table.SetCell(4, 0, cell)
	if got := table.GetCell(4, 0); got != cell {
		t.Error("GetCell() did not return the pointer that was stored")
	}
	if got := table.GetCell(4, 0).Align; got != tview.AlignRight {
		t.Errorf("cell alignment = %d, want %d", got, tview.AlignRight)
	}

	// Negative coordinates and out-of-range coordinates are safe.
	if got := table.GetCell(-1, 0); got == nil || got.Text != "" {
		t.Errorf("GetCell(-1, 0) = %#v, want empty cell", got)
	}
	if got := table.GetCell(0, -1); got == nil || got.Text != "" {
		t.Errorf("GetCell(0, -1) = %#v, want empty cell", got)
	}
	if got := table.GetCell(99, 99); got == nil || got.Text != "" {
		t.Errorf("GetCell(99, 99) = %#v, want empty cell", got)
	}
}

func TestTableClear(t *testing.T) {
	table := NewTable()
	table.SetCellSimple(0, 0, "a")
	table.SetCellSimple(3, 3, "b")
	if table.GetRowCount() == 0 || table.GetColumnCount() == 0 {
		t.Fatal("test setup failed: table is empty")
	}

	table.Clear()

	if got := table.GetRowCount(); got != 0 {
		t.Errorf("after Clear(), GetRowCount() = %d, want 0", got)
	}
	if got := table.GetColumnCount(); got != 0 {
		t.Errorf("after Clear(), GetColumnCount() = %d, want 0", got)
	}

	// The table can be filled again after clearing.
	table.SetCellSimple(0, 0, "new")
	if got := table.GetCell(0, 0).Text; got != "new" {
		t.Errorf("cell text after refill = %q, want %q", got, "new")
	}
	if got := table.GetColumnCount(); got != 1 {
		t.Errorf("GetColumnCount() after refill = %d, want 1", got)
	}
}

func TestTableRowMutation(t *testing.T) {
	newGrid := func(t *testing.T) *Table {
		t.Helper()
		table := NewTable()
		for row := 0; row < 3; row++ {
			for column := 0; column < 3; column++ {
				table.SetCellSimple(row, column, string(rune('a'+row))+string(rune('0'+column)))
			}
		}
		return table
	}

	rowTexts := func(table *Table, rows, columns int) [][]string {
		result := make([][]string, rows)
		for row := 0; row < rows; row++ {
			result[row] = make([]string, columns)
			for column := 0; column < columns; column++ {
				result[row][column] = table.GetCell(row, column).Text
			}
		}
		return result
	}

	t.Run("insert row in the middle", func(t *testing.T) {
		table := newGrid(t)
		table.InsertRow(1)
		if got := table.GetRowCount(); got != 4 {
			t.Fatalf("GetRowCount() = %d, want 4", got)
		}
		got := rowTexts(table, 4, 3)
		want := [][]string{{"a0", "a1", "a2"}, {"", "", ""}, {"b0", "b1", "b2"}, {"c0", "c1", "c2"}}
		for row := range want {
			for column := range want[row] {
				if got[row][column] != want[row][column] {
					t.Errorf("cell (%d,%d) = %q, want %q", row, column, got[row][column], want[row][column])
				}
			}
		}
	})

	t.Run("insert row at the beginning", func(t *testing.T) {
		table := newGrid(t)
		table.InsertRow(0)
		if got := table.GetCell(0, 0).Text; got != "" {
			t.Errorf("new first row cell = %q, want empty", got)
		}
		if got := table.GetCell(1, 0).Text; got != "a0" {
			t.Errorf("shifted first row cell = %q, want %q", got, "a0")
		}
	})

	t.Run("insert row out of range is ignored", func(t *testing.T) {
		table := newGrid(t)
		table.InsertRow(3)
		table.InsertRow(100)
		if got := table.GetRowCount(); got != 3 {
			t.Errorf("GetRowCount() = %d, want 3", got)
		}
		if got := table.GetCell(0, 0).Text; got != "a0" {
			t.Errorf("cell (0,0) = %q, want %q", got, "a0")
		}
	})

	t.Run("remove row", func(t *testing.T) {
		table := newGrid(t)
		table.RemoveRow(1)
		if got := table.GetRowCount(); got != 2 {
			t.Fatalf("GetRowCount() = %d, want 2", got)
		}
		if got := table.GetCell(0, 0).Text; got != "a0" {
			t.Errorf("cell (0,0) = %q, want %q", got, "a0")
		}
		if got := table.GetCell(1, 0).Text; got != "c0" {
			t.Errorf("cell (1,0) = %q, want %q", got, "c0")
		}
	})

	t.Run("remove row out of range is ignored", func(t *testing.T) {
		table := newGrid(t)
		table.RemoveRow(-1)
		table.RemoveRow(3)
		if got := table.GetRowCount(); got != 3 {
			t.Errorf("GetRowCount() = %d, want 3", got)
		}
	})
}

func TestTableColumnMutation(t *testing.T) {
	newGrid := func(t *testing.T) *Table {
		t.Helper()
		table := NewTable()
		for row := 0; row < 2; row++ {
			for column := 0; column < 4; column++ {
				table.SetCellSimple(row, column, string(rune('a'+row))+string(rune('0'+column)))
			}
		}
		return table
	}

	t.Run("insert column shifts cells right", func(t *testing.T) {
		table := newGrid(t)
		table.InsertColumn(1)
		if got := table.GetCell(0, 0).Text; got != "a0" {
			t.Errorf("cell (0,0) = %q, want %q", got, "a0")
		}
		if got := table.GetCell(0, 1).Text; got != "" {
			t.Errorf("inserted cell (0,1) = %q, want empty", got)
		}
		if got := table.GetCell(0, 2).Text; got != "a1" {
			t.Errorf("shifted cell (0,2) = %q, want %q", got, "a1")
		}
		if got := table.GetCell(1, 2).Text; got != "b1" {
			t.Errorf("shifted cell (1,2) = %q, want %q", got, "b1")
		}
	})

	t.Run("insert column beyond a short row leaves it unchanged", func(t *testing.T) {
		table := NewTable()
		table.SetCellSimple(0, 0, "a0")
		table.SetCellSimple(0, 1, "a1")
		table.SetCellSimple(1, 0, "b0")
		table.InsertColumn(1)
		if got := table.GetCell(0, 1).Text; got != "" {
			t.Errorf("inserted cell = %q, want empty", got)
		}
		if got := table.GetCell(1, 0).Text; got != "b0" {
			t.Errorf("short row cell = %q, want %q", got, "b0")
		}
	})

	t.Run("remove column shifts cells left", func(t *testing.T) {
		table := newGrid(t)
		table.RemoveColumn(1)
		if got := table.GetColumnCount(); got != 3 {
			t.Fatalf("GetColumnCount() = %d, want 3", got)
		}
		if got := table.GetCell(0, 0).Text; got != "a0" {
			t.Errorf("cell (0,0) = %q, want %q", got, "a0")
		}
		if got := table.GetCell(0, 1).Text; got != "a2" {
			t.Errorf("cell (0,1) = %q, want %q", got, "a2")
		}
		if got := table.GetCell(1, 2).Text; got != "b3" {
			t.Errorf("cell (1,2) = %q, want %q", got, "b3")
		}
	})

	t.Run("remove column out of range is ignored", func(t *testing.T) {
		table := newGrid(t)
		table.RemoveColumn(-1)
		table.RemoveColumn(99)
		if got := table.GetColumnCount(); got != 4 {
			t.Errorf("GetColumnCount() = %d, want 4", got)
		}
		if got := table.GetCell(0, 0).Text; got != "a0" {
			t.Errorf("cell (0,0) = %q, want %q", got, "a0")
		}
	})
}

func TestTableCellAccessors(t *testing.T) {
	cell := NewTableCell("[red]abc")
	if cell.Text != "[red]abc" {
		t.Errorf("Text = %q", cell.Text)
	}
	if cell.Align != tview.AlignLeft {
		t.Errorf("Align = %d, want AlignLeft", cell.Align)
	}
	if !cell.Transparent {
		t.Error("new cell should be transparent")
	}
	if cell.width != 3 {
		t.Errorf("width of tagged text = %d, want 3", cell.width)
	}

	cell.SetText("abcdef")
	if cell.width != 6 {
		t.Errorf("width after SetText = %d, want 6", cell.width)
	}
	cell.SetMaxWidth(3)
	if cell.width != 3 {
		t.Errorf("width after SetMaxWidth = %d, want 3", cell.width)
	}
	cell.SetText("ab")
	if cell.width != 2 {
		t.Errorf("width of short text with MaxWidth = %d, want 2", cell.width)
	}
	cell.SetMaxWidth(0)
	if cell.width != 2 {
		t.Errorf("width after resetting MaxWidth = %d, want 2", cell.width)
	}

	cell.SetAlign(tview.AlignCenter)
	if cell.Align != tview.AlignCenter {
		t.Errorf("Align = %d, want AlignCenter", cell.Align)
	}
	cell.SetExpansion(3)
	if cell.Expansion != 3 {
		t.Errorf("Expansion = %d, want 3", cell.Expansion)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("SetExpansion(-1) did not panic")
			}
		}()
		cell.SetExpansion(-1)
	}()

	cell.SetReference("payload")
	if got := cell.GetReference(); got != "payload" {
		t.Errorf("GetReference() = %v, want payload", got)
	}
	cell.SetSelectable(false)
	if !cell.NotSelectable {
		t.Error("SetSelectable(false) did not set NotSelectable")
	}
	cell.SetSelectable(true)
	if cell.NotSelectable {
		t.Error("SetSelectable(true) did not clear NotSelectable")
	}
	cell.SetTransparency(false)
	if cell.Transparent {
		t.Error("SetTransparency(false) did not clear Transparent")
	}
	cell.SetClickedFunc(func() bool { return true })
	if cell.Clicked == nil {
		t.Error("SetClickedFunc() did not store the handler")
	}
	cell.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorBlue))
	if got, _, _ := cell.SelectedStyle.Decompose(); got != tcell.ColorDefault {
		t.Errorf("unexpected selected style: %v", cell.SelectedStyle)
	}

	x, y, width := cell.GetLastPosition()
	if x != 0 || y != 0 || width != 2 {
		t.Errorf("GetLastPosition() = %d, %d, %d, want 0, 0, 2", x, y, width)
	}
}

func TestTableCellColorsAndStyles(t *testing.T) {
	t.Run("legacy fields for a zero cell", func(t *testing.T) {
		cell := &TableCell{}
		cell.SetTextColor(tcell.ColorRed)
		if cell.Color != tcell.ColorRed {
			t.Errorf("Color = %v, want red", cell.Color)
		}
		cell.SetBackgroundColor(tcell.ColorBlue)
		if cell.BackgroundColor != tcell.ColorBlue {
			t.Errorf("BackgroundColor = %v, want blue", cell.BackgroundColor)
		}
		if cell.Transparent {
			t.Error("SetBackgroundColor() must disable transparency")
		}
		cell.SetAttributes(tcell.AttrBold)
		if cell.Attributes != tcell.AttrBold {
			t.Errorf("Attributes = %v, want bold", cell.Attributes)
		}
	})

	t.Run("style fields for a new cell", func(t *testing.T) {
		cell := NewTableCell("x")
		cell.SetTextColor(tcell.ColorRed)
		fg, _, _ := cell.Style.Decompose()
		if fg != tcell.ColorRed {
			t.Errorf("foreground = %v, want red", fg)
		}
		cell.SetBackgroundColor(tcell.ColorBlue)
		_, bg, _ := cell.Style.Decompose()
		if bg != tcell.ColorBlue {
			t.Errorf("background = %v, want blue", bg)
		}
		if cell.Transparent {
			t.Error("SetBackgroundColor() must disable transparency")
		}
		cell.SetAttributes(tcell.AttrUnderline)
		_, _, attrs := cell.Style.Decompose()
		if attrs&tcell.AttrUnderline == 0 {
			t.Error("SetAttributes() did not set underline")
		}

		want := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorGreen)
		cell.SetStyle(want)
		if cell.Style != want {
			t.Errorf("Style = %v, want %v", cell.Style, want)
		}
	})
}

func TestSelectAndSelectionChanged(t *testing.T) {
	table := NewTable()
	table.SetCellSimple(0, 0, "a")
	table.SetCellSimple(1, 1, "b")

	var calls [][2]int
	table.SetSelectionChangedFunc(func(row, column int) {
		calls = append(calls, [2]int{row, column})
	})

	table.Select(1, 2)
	if row, column := table.GetSelection(); row != 1 || column != 2 {
		t.Errorf("GetSelection() = %d, %d, want 1, 2", row, column)
	}
	if len(calls) != 1 || calls[0] != [2]int{1, 2} {
		t.Errorf("selection changed calls = %v, want [[1 2]]", calls)
	}

	// Select() fires the handler even if the selection does not change and
	// even if nothing is selectable.
	table.Select(1, 2)
	if len(calls) != 2 {
		t.Errorf("selection changed calls = %d, want 2", len(calls))
	}
	if !table.clampToSelection {
		t.Error("Select() did not request selection clamping")
	}
}

func TestSetSelectable(t *testing.T) {
	table := NewTable()
	table.SetSelectable(true, true)
	if rows, columns := table.GetSelectable(); !rows || !columns {
		t.Errorf("GetSelectable() = %v, %v, want true, true", rows, columns)
	}
	table.SetSelectable(false, true)
	if rows, columns := table.GetSelectable(); rows || !columns {
		t.Errorf("GetSelectable() = %v, %v, want false, true", rows, columns)
	}
}

func TestSelectionClampingOnDraw(t *testing.T) {
	newTable := func(t *testing.T) *Table {
		t.Helper()
		table := NewTable()
		for row := 0; row < 3; row++ {
			for column := 0; column < 3; column++ {
				table.SetCellSimple(row, column, "x")
			}
		}
		table.SetSelectable(true, true)
		return table
	}

	t.Run("negative selection is clamped", func(t *testing.T) {
		table := newTable(t)
		table.SetRect(0, 0, 10, 3)
		screen := newTestScreen(t, 10, 3)
		table.Select(-5, -5)
		table.Draw(screen)
		if row, column := table.GetSelection(); row != 0 || column != 0 {
			t.Errorf("GetSelection() = %d, %d, want 0, 0", row, column)
		}
	})

	t.Run("not selectable cells are skipped", func(t *testing.T) {
		table := newTable(t)
		table.SetRect(0, 0, 10, 3)
		screen := newTestScreen(t, 10, 3)
		table.GetCell(0, 0).SetSelectable(false)
		table.GetCell(0, 1).SetSelectable(false)
		table.Select(0, 0)
		table.Draw(screen)
		if row, column := table.GetSelection(); row != 0 || column != 2 {
			t.Errorf("GetSelection() = %d, %d, want 0, 2", row, column)
		}
	})

	t.Run("selection skips to the next row", func(t *testing.T) {
		table := newTable(t)
		table.SetRect(0, 0, 10, 3)
		screen := newTestScreen(t, 10, 3)
		for column := 0; column < 3; column++ {
			table.GetCell(0, column).SetSelectable(false)
		}
		table.GetCell(1, 0).SetSelectable(true)
		table.Select(0, 0)
		table.Draw(screen)
		if row, column := table.GetSelection(); row != 1 || column != 0 {
			t.Errorf("GetSelection() = %d, %d, want 1, 0", row, column)
		}
	})
}

func TestOffsetsAndXScroll(t *testing.T) {
	table := NewTable()
	for row := 0; row < 5; row++ {
		table.SetCellSimple(row, 0, "x")
	}

	table.SetOffset(3, 2)
	if row, column := table.GetOffset(); row != 3 || column != 2 {
		t.Errorf("GetOffset() = %d, %d, want 3, 2", row, column)
	}

	table.SetXScroll(7)
	if got := table.GetXScroll(); got != 7 {
		t.Errorf("GetXScroll() = %d, want 7", got)
	}
	if row, column := table.GetOffset(); row != 3 || column != -1 {
		t.Errorf("GetOffset() after SetXScroll() = %d, %d, want 3, -1", row, column)
	}

	table.SetFixed(1, 1)
	if row, column := table.GetOffset(); row != 3 || column != 0 {
		t.Errorf("GetOffset() after SetFixed() = %d, %d, want 3, 0", row, column)
	}
}

func TestScrollToBeginningAndEnd(t *testing.T) {
	newTable := func(t *testing.T) *Table {
		t.Helper()
		table := NewTable()
		for row := 0; row < 10; row++ {
			table.SetCellSimple(row, 0, "row")
		}
		return table
	}

	t.Run("beginning resets offsets", func(t *testing.T) {
		table := newTable(t)
		table.SetOffset(4, 2)
		table.ScrollToBeginning()
		if row, column := table.GetOffset(); row != 0 || column != 0 {
			t.Errorf("GetOffset() = %d, %d, want 0, 0", row, column)
		}
		if got := table.GetXScroll(); got != 0 {
			t.Errorf("GetXScroll() = %d, want 0", got)
		}
	})

	t.Run("beginning keeps x scroll mode", func(t *testing.T) {
		table := newTable(t)
		table.SetXScroll(3)
		table.ScrollToBeginning()
		if got := table.GetXScroll(); got != 0 {
			t.Errorf("GetXScroll() = %d, want 0", got)
		}
		if row, column := table.GetOffset(); row != 0 || column != -1 {
			t.Errorf("GetOffset() = %d, %d, want 0, -1", row, column)
		}
	})

	t.Run("end is clamped during draw", func(t *testing.T) {
		table := newTable(t)
		table.SetRect(0, 0, 6, 3)
		screen := newTestScreen(t, 6, 3)
		table.ScrollToEnd()
		if row, column := table.GetOffset(); row != 10 || column != 0 {
			t.Errorf("GetOffset() before draw = %d, %d, want 10, 0", row, column)
		}
		table.Draw(screen)
		if row, column := table.GetOffset(); row != 7 || column != 0 {
			t.Errorf("GetOffset() after draw = %d, %d, want 7, 0", row, column)
		}
	})

	t.Run("end follows new data", func(t *testing.T) {
		table := newTable(t)
		table.SetRect(0, 0, 6, 3)
		screen := newTestScreen(t, 6, 3)
		table.ScrollToEnd()
		table.Draw(screen)
		table.SetCellSimple(10, 0, "new")
		table.Draw(screen)
		if row, column := table.GetOffset(); row != 8 || column != 0 {
			t.Errorf("GetOffset() = %d, %d, want 8, 0", row, column)
		}
	})
}

func TestScrollMetrics(t *testing.T) {
	newTable := func(t *testing.T) *Table {
		t.Helper()
		table := NewTable()
		table.SetCellSimple(0, 0, "abc")
		table.SetCellSimple(0, 1, "de")
		table.SetCellSimple(0, 2, "fg")
		return table
	}

	tests := []struct {
		name           string
		borders        bool
		fixedColumns   int
		width          int
		wantScrollable int
		wantViewport   int
		wantMaxXOffset int
	}{
		{
			name:           "no borders, no fixed columns",
			width:          6,
			wantScrollable: 7,
			wantViewport:   6,
			wantMaxXOffset: 2,
		},
		{
			name:           "no borders, one fixed column",
			fixedColumns:   1,
			width:          6,
			wantScrollable: 4,
			wantViewport:   3,
			wantMaxXOffset: 2,
		},
		{
			name:           "borders, no fixed columns",
			borders:        true,
			width:          6,
			wantScrollable: 10,
			wantViewport:   6,
			wantMaxXOffset: 5,
		},
		{
			name:           "borders, one fixed column",
			borders:        true,
			fixedColumns:   1,
			width:          6,
			wantScrollable: 6,
			wantViewport:   2,
			wantMaxXOffset: 5,
		},
		{
			name:           "columns fit on screen",
			width:          40,
			wantScrollable: 7,
			wantViewport:   40,
			wantMaxXOffset: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			table := newTable(t)
			table.SetBorders(test.borders)
			table.SetFixed(0, test.fixedColumns)
			table.SetRect(0, 0, test.width, 3)
			if got := table.ScrollableWidth(); got != test.wantScrollable {
				t.Errorf("ScrollableWidth() = %d, want %d", got, test.wantScrollable)
			}
			if got := table.ScrollableViewportWidth(); got != test.wantViewport {
				t.Errorf("ScrollableViewportWidth() = %d, want %d", got, test.wantViewport)
			}
			if got := table.MaximumXOffset(); got != test.wantMaxXOffset {
				t.Errorf("MaximumXOffset() = %d, want %d", got, test.wantMaxXOffset)
			}
		})
	}
}

func TestGetVisibleColumnRange(t *testing.T) {
	// GetVisibleColumnRange() reads the indices recorded by the last draw.
	table := NewTable()
	table.visibleColumnIndices = []int{2, 3, 5}
	first, last := table.GetVisibleColumnRange()
	if first != 2 || last != 5 {
		t.Errorf("GetVisibleColumnRange() = %d, %d, want 2, 5", first, last)
	}

	// A freshly created table has no recorded columns, so the function panics.
	// This documents the current implementation: Draw() never populates
	// visibleColumnIndices.
	t.Run("panics before draw", func(t *testing.T) {
		fresh := NewTable()
		defer func() {
			if recover() == nil {
				t.Error("GetVisibleColumnRange() did not panic without visible columns")
			}
		}()
		fresh.GetVisibleColumnRange()
	})
}

func TestCellAt(t *testing.T) {
	t.Run("without borders", func(t *testing.T) {
		table := NewTable()
		table.SetCellSimple(0, 0, "a")
		table.SetCellSimple(0, 1, "bb")
		table.SetCellSimple(0, 2, "ccc")
		table.SetCellSimple(1, 0, "d")
		table.SetCellSimple(1, 1, "ee")
		table.SetCellSimple(1, 2, "fff")
		table.SetCellSimple(2, 0, "g")
		table.SetCellSimple(2, 1, "hh")
		table.SetCellSimple(2, 2, "iii")
		table.SetRect(0, 0, 10, 4)

		tests := []struct {
			x, y       int
			wantRow    int
			wantColumn int
		}{
			{0, 0, 0, 0},
			{1, 0, 0, 1},
			{2, 0, 0, 1},
			{3, 0, 0, 2},
			{5, 0, 0, 2},
			{6, 0, 0, -1},
			{0, 1, 1, 0},
			{0, 2, 2, 0},
			{0, 3, -1, 0},
		}
		for _, test := range tests {
			row, column := table.CellAt(test.x, test.y)
			if row != test.wantRow || column != test.wantColumn {
				t.Errorf("CellAt(%d, %d) = %d, %d, want %d, %d", test.x, test.y, row, column, test.wantRow, test.wantColumn)
			}
		}
	})

	t.Run("fixed row and row offset", func(t *testing.T) {
		table := NewTable()
		for row := 0; row < 3; row++ {
			table.SetCellSimple(row, 0, "x")
		}
		table.SetFixed(1, 0)
		table.SetOffset(1, 0)
		table.SetRect(0, 0, 10, 4)

		tests := []struct {
			y, wantRow int
		}{
			{0, 0},  // fixed row
			{1, 2},  // first scrolled row
			{2, -1}, // beyond the last row
		}
		for _, test := range tests {
			row, _ := table.CellAt(0, test.y)
			if row != test.wantRow {
				t.Errorf("CellAt(0, %d) row = %d, want %d", test.y, row, test.wantRow)
			}
		}
	})

	t.Run("with borders", func(t *testing.T) {
		table := NewTable()
		table.SetBorders(true)
		for row := 0; row < 2; row++ {
			for column := 0; column < 3; column++ {
				table.SetCellSimple(row, column, "x")
			}
		}
		table.SetRect(0, 0, 10, 6)

		tests := []struct {
			x, y       int
			wantRow    int
			wantColumn int
		}{
			{0, 0, 0, 0},
			{1, 0, 0, 0},
			{2, 0, 0, 0},
			{3, 0, 0, 1},
			{0, 1, 0, 0},
			{0, 2, 0, 0},
			{0, 3, 1, 0},
			{0, 4, 1, 0},
			{0, 5, -1, 0},
		}
		for _, test := range tests {
			row, column := table.CellAt(test.x, test.y)
			if row != test.wantRow || column != test.wantColumn {
				t.Errorf("CellAt(%d, %d) = %d, %d, want %d, %d", test.x, test.y, row, column, test.wantRow, test.wantColumn)
			}
		}
	})
}

func TestDrawWithoutBorders(t *testing.T) {
	screen := newTestScreen(t, 10, 2)
	table := NewTable()
	table.SetRect(0, 0, 10, 2)
	table.SetCellSimple(0, 0, "abc")
	table.SetCellSimple(0, 1, "de")
	table.SetCellSimple(1, 0, "fgh")
	table.SetCellSimple(1, 1, "ij")

	table.Draw(screen)

	if got, want := screenRow(screen, 0, 10), "abc de    "; got != want {
		t.Errorf("row 0 = %q, want %q", got, want)
	}
	if got, want := screenRow(screen, 1, 10), "fgh ij    "; got != want {
		t.Errorf("row 1 = %q, want %q", got, want)
	}
	if got, want := foregroundAt(t, screen, 0, 0), tview.Styles.PrimaryTextColor; got != want {
		t.Errorf("text foreground = %v, want %v", got, want)
	}
	// The space between the columns is the separator gap.
	if got := runeAt(t, screen, 3, 0); got != ' ' {
		t.Errorf("separator rune = %q, want space", got)
	}

	// MaybeSetSeparator does not influence the drawing: the gap stays a space.
	table.SetSeparator('|')
	table.Draw(screen)
	if got := runeAt(t, screen, 3, 0); got != ' ' {
		t.Errorf("separator rune after SetSeparator = %q, want space", got)
	}
}

func TestDrawBorders(t *testing.T) {
	screen := newTestScreen(t, 8, 5)
	table := NewTable()
	table.SetRect(0, 0, 8, 5)
	table.SetBorders(true)
	table.SetBordersColor(tcell.ColorRed)
	table.SetCellSimple(0, 0, "ab")
	table.SetCellSimple(0, 1, "cd")

	table.Draw(screen)

	if got, want := screenRow(screen, 0, 8), "┌──┬──┐ "; got != want {
		t.Errorf("top border = %q, want %q", got, want)
	}
	if got, want := screenRow(screen, 1, 8), "│ab│cd│ "; got != want {
		t.Errorf("text row = %q, want %q", got, want)
	}
	if got, want := screenRow(screen, 2, 8), "        "; got != want {
		t.Errorf("row 2 = %q, want %q", got, want)
	}
	if got, want := foregroundAt(t, screen, 0, 0), tcell.ColorRed; got != want {
		t.Errorf("border foreground = %v, want %v", got, want)
	}
	if got, want := runeAt(t, screen, 1, 1), 'a'; got != want {
		t.Errorf("cell rune = %q, want %q", got, want)
	}
	if got, want := foregroundAt(t, screen, 1, 1), tview.Styles.PrimaryTextColor; got != want {
		t.Errorf("cell foreground = %v, want %v", got, want)
	}
}

func TestDrawBordersMultipleRows(t *testing.T) {
	screen := newTestScreen(t, 5, 5)
	table := NewTable()
	table.SetRect(0, 0, 5, 5)
	table.SetBorders(true)
	table.SetCellSimple(0, 0, "a")
	table.SetCellSimple(1, 0, "b")

	table.Draw(screen)

	wantRows := []string{
		"┌─┐  ",
		"│a│  ",
		"├─┤  ",
		"│b│  ",
		"     ",
	}
	for y, want := range wantRows {
		if got := screenRow(screen, y, 5); got != want {
			t.Errorf("row %d = %q, want %q", y, got, want)
		}
	}
}

func TestDrawAlignment(t *testing.T) {
	screen := newTestScreen(t, 6, 3)
	table := NewTable()
	table.SetRect(0, 0, 6, 3)

	centered := NewTableCell("ab")
	centered.SetAlign(tview.AlignCenter)
	right := NewTableCell("cd")
	right.SetAlign(tview.AlignRight)
	table.SetCell(0, 0, centered)
	table.SetCell(1, 0, right)
	table.SetCellSimple(2, 0, "abcdef")

	table.Draw(screen)

	wantRows := []string{"  ab  ", "    cd", "abcdef"}
	for y, want := range wantRows {
		if got := screenRow(screen, y, 6); got != want {
			t.Errorf("row %d = %q, want %q", y, got, want)
		}
	}
}

func TestDrawColorTagsAndEllipsis(t *testing.T) {
	screen := newTestScreen(t, 6, 2)
	table := NewTable()
	table.SetRect(0, 0, 6, 2)
	table.SetCellSimple(0, 0, "[red]abc")
	truncated := NewTableCell("abcdefgh")
	truncated.SetMaxWidth(4)
	table.SetCell(1, 0, truncated)

	table.Draw(screen)

	if got, want := screenRow(screen, 0, 6), "abc   "; got != want {
		t.Errorf("tagged row = %q, want %q", got, want)
	}
	if got, want := foregroundAt(t, screen, 0, 0), tcell.ColorRed; got != want {
		t.Errorf("tagged text foreground = %v, want %v", got, want)
	}
	if got, want := foregroundAt(t, screen, 1, 0), tcell.ColorRed; got != want {
		t.Errorf("tagged text foreground = %v, want %v", got, want)
	}
	if got, want := screenRow(screen, 1, 6), "abcd  "; got != want {
		t.Errorf("truncated row = %q, want %q", got, want)
	}
}

func TestDrawCellBackground(t *testing.T) {
	newTable := func(t *testing.T) *Table {
		t.Helper()
		table := NewTable()
		table.SetCellSimple(0, 0, "a")
		table.SetCellSimple(0, 1, "b")
		table.SetCellSimple(1, 0, "c")
		table.SetCellSimple(1, 1, "d")
		table.SetRect(0, 0, 6, 2)
		return table
	}

	t.Run("cell background color", func(t *testing.T) {
		table := newTable(t)
		cell := NewTableCell("a")
		cell.SetBackgroundColor(tcell.ColorGreen)
		table.SetCell(0, 0, cell)
		screen := newTestScreen(t, 6, 2)
		table.Draw(screen)
		if got, want := backgroundAt(t, screen, 0, 0), tcell.ColorGreen; got != want {
			t.Errorf("background = %v, want %v", got, want)
		}
		if got, want := backgroundAt(t, screen, 2, 1), tview.Styles.PrimitiveBackgroundColor; got != want {
			t.Errorf("neighbor background = %v, want %v", got, want)
		}
	})

	t.Run("cells selected with swapped colors", func(t *testing.T) {
		table := newTable(t)
		table.SetSelectable(true, true)
		table.Select(1, 1)
		screen := newTestScreen(t, 6, 2)
		table.Draw(screen)
		fg, bg, _ := styleAt(t, screen, 2, 1).Decompose()
		if fg != tview.Styles.PrimitiveBackgroundColor || bg != tview.Styles.PrimaryTextColor {
			t.Errorf("selected style = fg %v bg %v, want swapped colors", fg, bg)
		}
	})

	t.Run("table selected style", func(t *testing.T) {
		table := newTable(t)
		table.SetSelectable(true, true)
		table.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorMaroon).Foreground(tcell.ColorYellow))
		table.Select(1, 1)
		screen := newTestScreen(t, 6, 2)
		table.Draw(screen)
		fg, bg, _ := styleAt(t, screen, 2, 1).Decompose()
		if fg != tcell.ColorYellow || bg != tcell.ColorMaroon {
			t.Errorf("selected style = fg %v bg %v, want yellow on maroon", fg, bg)
		}
	})

	t.Run("cell selected style overrides table style", func(t *testing.T) {
		table := newTable(t)
		table.SetSelectable(true, true)
		table.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorMaroon).Foreground(tcell.ColorYellow))
		table.GetCell(1, 1).SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorGreen))
		table.Select(1, 1)
		screen := newTestScreen(t, 6, 2)
		table.Draw(screen)
		fg, bg, _ := styleAt(t, screen, 2, 1).Decompose()
		if fg != tcell.ColorGreen || bg != tcell.ColorBlue {
			t.Errorf("selected style = fg %v bg %v, want green on blue", fg, bg)
		}
	})

	t.Run("row selection highlights the whole row", func(t *testing.T) {
		table := newTable(t)
		table.SetSelectable(true, false)
		table.Select(0, 1)
		screen := newTestScreen(t, 6, 2)
		table.Draw(screen)
		for _, x := range []int{0, 1, 2, 3} {
			if got, want := backgroundAt(t, screen, x, 0), tview.Styles.PrimaryTextColor; got != want {
				t.Errorf("x=%d background = %v, want %v", x, got, want)
			}
		}
		if got, want := backgroundAt(t, screen, 0, 1), tview.Styles.PrimitiveBackgroundColor; got != want {
			t.Errorf("second row background = %v, want %v", got, want)
		}
	})

	t.Run("column selection highlights the whole column", func(t *testing.T) {
		table := newTable(t)
		table.SetSelectable(false, true)
		table.Select(0, 1)
		screen := newTestScreen(t, 6, 2)
		table.Draw(screen)
		for _, y := range []int{0, 1} {
			if got, want := backgroundAt(t, screen, 2, y), tview.Styles.PrimaryTextColor; got != want {
				t.Errorf("y=%d background = %v, want %v", y, got, want)
			}
		}
		if got, want := backgroundAt(t, screen, 0, 0), tview.Styles.PrimitiveBackgroundColor; got != want {
			t.Errorf("unselected column background = %v, want %v", got, want)
		}
	})
}

func TestDrawFixedRowsAndColumns(t *testing.T) {
	screen := newTestScreen(t, 9, 4)
	table := NewTable()
	table.SetRect(0, 0, 9, 4)
	table.SetFixed(1, 1)
	table.SetOffset(2, 1)
	for row := 0; row < 6; row++ {
		for column := 0; column < 3; column++ {
			table.SetCellSimple(row, column, string(rune('0'+row))+string(rune('0'+column)))
		}
	}

	table.Draw(screen)

	wantRows := []string{
		"0002     ", // fixed row 0, fixed column 0, scrolled column 2
		"3032     ", // scrolled to row 3
		"4042     ",
		"5052     ",
	}
	for y, want := range wantRows {
		if got := screenRow(screen, y, 9); got != want {
			t.Errorf("row %d = %q, want %q", y, got, want)
		}
	}
}

func TestDrawXScroll(t *testing.T) {
	newTable := func(t *testing.T) *Table {
		t.Helper()
		table := NewTable()
		table.SetCellSimple(0, 0, "abc")
		table.SetCellSimple(0, 1, "def")
		table.SetCellSimple(0, 2, "ghi")
		table.SetRect(0, 0, 5, 1)
		return table
	}

	t.Run("scroll position two", func(t *testing.T) {
		table := newTable(t)
		table.SetXScroll(2)
		screen := newTestScreen(t, 5, 1)
		table.Draw(screen)
		if got, want := screenRow(screen, 0, 5), "c def"; got != want {
			t.Errorf("row = %q, want %q", got, want)
		}
		if got := table.GetXScroll(); got != 2 {
			t.Errorf("GetXScroll() = %d, want 2", got)
		}
	})

	t.Run("scroll position is clamped to the maximum", func(t *testing.T) {
		table := newTable(t)
		table.SetXScroll(100)
		screen := newTestScreen(t, 5, 1)
		table.Draw(screen)
		want := table.MaximumXOffset()
		if got := table.GetXScroll(); got != want {
			t.Errorf("GetXScroll() = %d, want %d", got, want)
		}
		if got, want := screenRow(screen, 0, 5), "ef gh"; got != want {
			t.Errorf("row = %q, want %q", got, want)
		}
	})

	t.Run("negative scroll is clamped to zero", func(t *testing.T) {
		table := newTable(t)
		table.SetXScroll(-10)
		screen := newTestScreen(t, 5, 1)
		table.Draw(screen)
		if got := table.GetXScroll(); got != 0 {
			t.Errorf("GetXScroll() = %d, want 0", got)
		}
		if got, want := screenRow(screen, 0, 5), "abc d"; got != want {
			t.Errorf("row = %q, want %q", got, want)
		}
	})
}

func TestDrawShowsSelectionWhenScrolled(t *testing.T) {
	screen := newTestScreen(t, 6, 3)
	table := NewTable()
	table.SetRect(0, 0, 6, 3)
	table.SetSelectable(true, true)
	for row := 0; row < 10; row++ {
		table.SetCellSimple(row, 0, string(rune('a'+row)))
	}
	table.Select(9, 0)

	table.Draw(screen)

	if row, column := table.GetOffset(); row != 7 || column != 0 {
		t.Errorf("GetOffset() = %d, %d, want 7, 0", row, column)
	}
	if got, want := screenRow(screen, 0, 6), "h     "; got != want {
		t.Errorf("row 0 = %q, want %q", got, want)
	}
	if got, want := screenRow(screen, 1, 6), "i     "; got != want {
		t.Errorf("row 1 = %q, want %q", got, want)
	}
	if got, want := screenRow(screen, 2, 6), "j     "; got != want {
		t.Errorf("row 2 = %q, want %q", got, want)
	}
	if got, want := backgroundAt(t, screen, 0, 2), tview.Styles.PrimaryTextColor; got != want {
		t.Errorf("selection background = %v, want %v", got, want)
	}

	// Scrolling up with the selection clamps the offset back to the selection.
	table.Select(2, 0)
	table.Draw(screen)
	if row, _ := table.GetOffset(); row != 2 {
		t.Errorf("GetOffset() after selecting row 2 = %d, want 2", row)
	}
}

func TestDrawCellLastPosition(t *testing.T) {
	screen := newTestScreen(t, 5, 2)
	table := NewTable()
	table.SetRect(0, 0, 5, 2)
	table.SetCellSimple(0, 0, "abc")
	table.SetCellSimple(1, 0, "de")

	table.Draw(screen)

	if x, y, width := table.GetCell(0, 0).GetLastPosition(); x != 0 || y != 0 || width != 3 {
		t.Errorf("cell (0,0) position = %d, %d, %d, want 0, 0, 3", x, y, width)
	}
	if x, y, width := table.GetCell(1, 0).GetLastPosition(); x != 0 || y != 1 || width != 2 {
		t.Errorf("cell (1,0) position = %d, %d, %d, want 0, 1, 2", x, y, width)
	}
}

func TestInputHandlerNavigation(t *testing.T) {
	newTable := func(t *testing.T) *Table {
		t.Helper()
		table := NewTable()
		table.SetRect(0, 0, 10, 4)
		table.SetSelectable(true, true)
		for row := 0; row < 5; row++ {
			for column := 0; column < 4; column++ {
				table.SetCellSimple(row, column, "x")
			}
		}
		screen := newTestScreen(t, 10, 4)
		table.Draw(screen)
		table.Select(0, 0)
		return table
	}

	tests := []struct {
		name       string
		key        tcell.Key
		r          rune
		wantRow    int
		wantColumn int
	}{
		{"down arrow", tcell.KeyDown, 0, 1, 0},
		{"up arrow", tcell.KeyUp, 0, 0, 0},
		{"right arrow", tcell.KeyRight, 0, 0, 1},
		{"left arrow", tcell.KeyLeft, 0, 0, 0},
		{"j", tcell.KeyRune, 'j', 1, 0},
		{"k", tcell.KeyRune, 'k', 0, 0},
		{"l", tcell.KeyRune, 'l', 0, 1},
		{"h", tcell.KeyRune, 'h', 0, 0},
		{"end", tcell.KeyEnd, 0, 4, 3},
		{"home", tcell.KeyHome, 0, 0, 0},
		{"G", tcell.KeyRune, 'G', 4, 3},
		{"g", tcell.KeyRune, 'g', 0, 0},
		{"page down", tcell.KeyPgDn, 0, 4, 0},
		{"ctrl-f", tcell.KeyCtrlF, 0, 4, 0},
		{"page up", tcell.KeyPgUp, 0, 0, 0},
		{"ctrl-b", tcell.KeyCtrlB, 0, 0, 0},
		{"unhandled rune", tcell.KeyRune, 'x', 0, 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			table := newTable(t)
			sendKey(table, test.key, test.r)
			row, column := table.GetSelection()
			if row != test.wantRow || column != test.wantColumn {
				t.Errorf("GetSelection() = %d, %d, want %d, %d", row, column, test.wantRow, test.wantColumn)
			}
		})
	}
}

func TestInputHandlerWrapSelection(t *testing.T) {
	newTable := func(t *testing.T, vertical, horizontal bool) *Table {
		t.Helper()
		table := NewTable()
		table.SetSelectable(true, true)
		table.SetWrapSelection(vertical, horizontal)
		for row := 0; row < 3; row++ {
			for column := 0; column < 3; column++ {
				table.SetCellSimple(row, column, "x")
			}
		}
		return table
	}

	tests := []struct {
		name        string
		vertical    bool
		horizontal  bool
		startRow    int
		startColumn int
		key         tcell.Key
		r           rune
		wantRow     int
		wantColumn  int
	}{
		{
			name: "up without wrap", startRow: 0, startColumn: 0,
			key: tcell.KeyUp, wantRow: 0, wantColumn: 0,
		},
		{
			name: "left without wrap", startRow: 0, startColumn: 0,
			key: tcell.KeyLeft, wantRow: 0, wantColumn: 0,
		},
		{
			name: "up with vertical wrap", vertical: true, startRow: 0, startColumn: 0,
			key: tcell.KeyUp, wantRow: 2, wantColumn: 0,
		},
		{
			name: "down with vertical wrap", vertical: true, startRow: 2, startColumn: 0,
			key: tcell.KeyDown, wantRow: 0, wantColumn: 0,
		},
		{
			name: "right with horizontal wrap", horizontal: true, startRow: 1, startColumn: 2,
			key: tcell.KeyRight, wantRow: 2, wantColumn: 0,
		},
		{
			name: "right with horizontal wrap at the end", horizontal: true, startRow: 2, startColumn: 2,
			key: tcell.KeyRight, wantRow: 2, wantColumn: 2,
		},
		{
			name: "right with both wraps", vertical: true, horizontal: true, startRow: 2, startColumn: 2,
			key: tcell.KeyRight, wantRow: 0, wantColumn: 0,
		},
		{
			name: "left with both wraps", vertical: true, horizontal: true, startRow: 0, startColumn: 0,
			key: tcell.KeyLeft, wantRow: 2, wantColumn: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			table := newTable(t, test.vertical, test.horizontal)
			table.Select(test.startRow, test.startColumn)
			sendKey(table, test.key, test.r)
			row, column := table.GetSelection()
			if row != test.wantRow || column != test.wantColumn {
				t.Errorf("GetSelection() = %d, %d, want %d, %d", row, column, test.wantRow, test.wantColumn)
			}
		})
	}
}

func TestInputHandlerSkipsNotSelectableCells(t *testing.T) {
	table := NewTable()
	table.SetSelectable(true, true)
	for row := 0; row < 2; row++ {
		for column := 0; column < 3; column++ {
			table.SetCellSimple(row, column, "x")
		}
	}
	table.GetCell(0, 1).SetSelectable(false)
	table.Select(0, 0)

	sendKey(table, tcell.KeyRight, 0)
	if row, column := table.GetSelection(); row != 0 || column != 2 {
		t.Errorf("GetSelection() = %d, %d, want 0, 2", row, column)
	}
}

func TestInputHandlerSelectionCallbacks(t *testing.T) {
	table := NewTable()
	table.SetSelectable(true, true)
	for row := 0; row < 3; row++ {
		for column := 0; column < 3; column++ {
			table.SetCellSimple(row, column, "x")
		}
	}
	table.Select(1, 1)

	var changed [][2]int
	table.SetSelectionChangedFunc(func(row, column int) {
		changed = append(changed, [2]int{row, column})
	})
	var selected [][2]int
	table.SetSelectedFunc(func(row, column int) {
		selected = append(selected, [2]int{row, column})
	})
	var done []tcell.Key
	table.SetDoneFunc(func(key tcell.Key) {
		done = append(done, key)
	})

	sendKey(table, tcell.KeyDown, 0)
	if len(changed) != 1 || changed[0] != [2]int{2, 1} {
		t.Errorf("selection changed = %v, want [[2 1]]", changed)
	}
	// At the last row another down key must not fire a change event.
	sendKey(table, tcell.KeyDown, 0)
	if len(changed) != 1 {
		t.Errorf("selection changed = %v, want no additional event", changed)
	}
	sendKey(table, tcell.KeyUp, 0)
	if len(changed) != 2 || changed[1] != [2]int{1, 1} {
		t.Errorf("selection changed = %v, want second event [1 1]", changed)
	}

	sendKey(table, tcell.KeyEnter, 0)
	if len(selected) != 1 || selected[0] != [2]int{1, 1} {
		t.Errorf("selected = %v, want [[1 1]]", selected)
	}
	if len(done) != 0 {
		t.Errorf("done handler called on Enter with a selection: %v", done)
	}

	sendKey(table, tcell.KeyEscape, 0)
	sendKey(table, tcell.KeyTab, 0)
	sendKey(table, tcell.KeyBacktab, 0)
	wantDone := []tcell.Key{tcell.KeyEscape, tcell.KeyTab, tcell.KeyBacktab}
	if len(done) != len(wantDone) {
		t.Fatalf("done keys = %v, want %v", done, wantDone)
	}
	for i := range wantDone {
		if done[i] != wantDone[i] {
			t.Errorf("done key %d = %v, want %v", i, done[i], wantDone[i])
		}
	}
}

func TestInputHandlerEnterWithoutSelection(t *testing.T) {
	table := NewTable()
	table.SetCellSimple(0, 0, "a")
	var done []tcell.Key
	table.SetDoneFunc(func(key tcell.Key) {
		done = append(done, key)
	})
	var selectedCalled bool
	table.SetSelectedFunc(func(row, column int) {
		selectedCalled = true
	})

	sendKey(table, tcell.KeyEnter, 0)

	if len(done) != 1 || done[0] != tcell.KeyEnter {
		t.Errorf("done keys = %v, want [Enter]", done)
	}
	if selectedCalled {
		t.Error("selected handler was called although nothing is selectable")
	}
}

func TestInputHandlerNonSelectableNavigation(t *testing.T) {
	table := NewTable()
	for row := 0; row < 5; row++ {
		for column := 0; column < 4; column++ {
			table.SetCellSimple(row, column, "x")
		}
	}
	table.SetRect(0, 0, 10, 4)
	screen := newTestScreen(t, 10, 4)
	table.Draw(screen)

	sendKey(table, tcell.KeyDown, 0)
	if row, _ := table.GetOffset(); row != 1 {
		t.Errorf("row offset after down = %d, want 1", row)
	}
	sendKey(table, tcell.KeyUp, 0)
	if row, _ := table.GetOffset(); row != 0 {
		t.Errorf("row offset after up = %d, want 0", row)
	}
	sendKey(table, tcell.KeyUp, 0)
	if row, _ := table.GetOffset(); row != -1 {
		t.Errorf("row offset after up at top = %d, want -1", row)
	}

	sendKey(table, tcell.KeyRight, 0)
	if _, column := table.GetOffset(); column != 1 {
		t.Errorf("column offset after right = %d, want 1", column)
	}
	for i := 0; i < 5; i++ {
		sendKey(table, tcell.KeyRight, 0)
	}
	if _, column := table.GetOffset(); column != 3 {
		t.Errorf("column offset after scrolling right = %d, want 3", column)
	}
	for i := 0; i < 5; i++ {
		sendKey(table, tcell.KeyLeft, 0)
	}
	if _, column := table.GetOffset(); column != 0 {
		t.Errorf("column offset after scrolling left = %d, want 0", column)
	}

	sendKey(table, tcell.KeyPgDn, 0)
	if row, _ := table.GetOffset(); row != 3 {
		t.Errorf("row offset after page down = %d, want 3", row)
	}
	sendKey(table, tcell.KeyPgUp, 0)
	if row, _ := table.GetOffset(); row != -1 {
		t.Errorf("row offset after page up = %d, want -1", row)
	}

	sendKey(table, tcell.KeyEnd, 0)
	if row, _ := table.GetOffset(); row != -1 {
		t.Errorf("row offset after end = %d, want -1", row)
	}
	table.Draw(screen)
	if row, _ := table.GetOffset(); row != 1 {
		t.Errorf("row offset after end and draw = %d, want 1", row)
	}

	sendKey(table, tcell.KeyHome, 0)
	if row, column := table.GetOffset(); row != 0 || column != 0 {
		t.Errorf("GetOffset() after home = %d, %d, want 0, 0", row, column)
	}
}

func TestInputHandlerEmptyTable(t *testing.T) {
	table := NewTable()
	table.SetSelectable(true, true)

	sendKey(table, tcell.KeyDown, 0)
	sendKey(table, tcell.KeyRight, 0)
	sendKey(table, tcell.KeyEnd, 0)

	if row, column := table.GetSelection(); row != 0 || column != 0 {
		t.Errorf("GetSelection() = %d, %d, want 0, 0", row, column)
	}
	if row, column := table.GetOffset(); row != 0 || column != 0 {
		t.Errorf("GetOffset() = %d, %d, want 0, 0", row, column)
	}
}

func TestMouseHandlerClick(t *testing.T) {
	screen := newTestScreen(t, 10, 3)
	table := NewTable()
	table.SetRect(0, 0, 10, 3)
	table.SetSelectable(true, true)
	for row := 0; row < 3; row++ {
		for column := 0; column < 3; column++ {
			table.SetCellSimple(row, column, "x")
		}
	}
	table.Draw(screen)

	var focused tview.Primitive
	handler := table.MouseHandler()

	event := tcell.NewEventMouse(2, 1, tcell.Button1, tcell.ModNone)
	consumed, capture := handler(tview.MouseLeftDown, event, func(p tview.Primitive) {
		focused = p
	})

	if !consumed {
		t.Error("left mouse down was not consumed")
	}
	if capture != nil {
		t.Errorf("capture = %v, want nil", capture)
	}
	if focused != table {
		t.Errorf("focused = %v, want the table", focused)
	}
	if row, column := table.GetSelection(); row != 1 || column != 2 {
		t.Errorf("GetSelection() = %d, %d, want 1, 2", row, column)
	}
}

func TestMouseHandlerOutsideTable(t *testing.T) {
	screen := newTestScreen(t, 10, 3)
	table := NewTable()
	table.SetRect(0, 0, 5, 2)
	table.GetCell(0, 0)
	table.Draw(screen)

	var focused bool
	handler := table.MouseHandler()
	consumed, _ := handler(tview.MouseLeftDown, tcell.NewEventMouse(8, 2, tcell.Button1, tcell.ModNone), func(tview.Primitive) {
		focused = true
	})

	if consumed {
		t.Error("click outside the table was consumed")
	}
	if focused {
		t.Error("click outside the table changed the focus")
	}
}

func TestMouseHandlerDoubleClick(t *testing.T) {
	screen := newTestScreen(t, 10, 3)
	table := NewTable()
	table.SetRect(0, 0, 10, 3)
	table.SetSelectable(true, true)
	for row := 0; row < 3; row++ {
		for column := 0; column < 3; column++ {
			table.SetCellSimple(row, column, "x")
		}
	}
	table.Draw(screen)
	// Make the double click window deterministic (large enough that two
	// immediate clicks always count as a double click).
	table.doubleClickDuration = time.Hour

	var doubleClicked [][2]int
	table.SetDoubleClickFunc(func(row, column int) {
		doubleClicked = append(doubleClicked, [2]int{row, column})
	})

	handler := table.MouseHandler()
	event := tcell.NewEventMouse(2, 1, tcell.Button1, tcell.ModNone)
	handler(tview.MouseLeftDown, event, func(tview.Primitive) {})
	if len(doubleClicked) != 0 {
		t.Fatalf("first click triggered a double click: %v", doubleClicked)
	}
	handler(tview.MouseLeftDown, event, func(tview.Primitive) {})
	if len(doubleClicked) != 1 || doubleClicked[0] != [2]int{1, 2} {
		t.Errorf("double click calls = %v, want [[1 2]]", doubleClicked)
	}
}

func TestMouseHandlerClickedFunc(t *testing.T) {
	screen := newTestScreen(t, 10, 3)
	table := NewTable()
	table.SetRect(0, 0, 10, 3)
	table.SetSelectable(true, true)
	for row := 0; row < 2; row++ {
		for column := 0; column < 2; column++ {
			table.SetCellSimple(row, column, "x")
		}
	}
	table.Draw(screen)
	table.Select(0, 0)

	clicked := 0
	table.GetCell(0, 1).SetClickedFunc(func() bool {
		clicked++
		return true // Do not select the cell.
	})

	handler := table.MouseHandler()
	handler(tview.MouseLeftDown, tcell.NewEventMouse(1, 0, tcell.Button1, tcell.ModNone), func(tview.Primitive) {})

	if clicked != 1 {
		t.Errorf("clicked handler calls = %d, want 1", clicked)
	}
	if row, column := table.GetSelection(); row != 0 || column != 0 {
		t.Errorf("GetSelection() = %d, %d, want unchanged 0, 0", row, column)
	}

	// A handler which returns false does not prevent selection.
	table.GetCell(0, 1).SetClickedFunc(func() bool {
		clicked++
		return false
	})
	handler(tview.MouseLeftDown, tcell.NewEventMouse(1, 0, tcell.Button1, tcell.ModNone), func(tview.Primitive) {})
	if clicked != 2 {
		t.Errorf("clicked handler calls = %d, want 2", clicked)
	}
	if row, column := table.GetSelection(); row != 0 || column != 1 {
		t.Errorf("GetSelection() = %d, %d, want 0, 1", row, column)
	}
}

func TestMouseHandlerRightClickIsIgnored(t *testing.T) {
	screen := newTestScreen(t, 10, 3)
	table := NewTable()
	table.SetRect(0, 0, 10, 3)
	table.SetSelectable(true, true)
	table.SetCellSimple(0, 0, "x")
	table.SetCellSimple(0, 1, "x")
	table.SetCellSimple(1, 0, "x")
	table.SetCellSimple(1, 1, "x")
	table.Draw(screen)

	handler := table.MouseHandler()
	consumed, capture := handler(tview.MouseRightDown, tcell.NewEventMouse(2, 0, tcell.Button2, tcell.ModNone), func(tview.Primitive) {})

	if consumed {
		t.Error("right click was consumed")
	}
	if capture != nil {
		t.Errorf("capture = %v, want nil", capture)
	}
	if row, column := table.GetSelection(); row != 0 || column != 0 {
		t.Errorf("GetSelection() = %d, %d, want unchanged 0, 0", row, column)
	}
}

func TestMouseHandlerScroll(t *testing.T) {
	screen := newTestScreen(t, 10, 3)
	table := NewTable()
	table.SetRect(0, 0, 10, 3)
	for row := 0; row < 5; row++ {
		table.SetCellSimple(row, 0, "x")
	}
	table.Draw(screen)

	handler := table.MouseHandler()
	consumed, _ := handler(tview.MouseScrollDown, tcell.NewEventMouse(0, 0, tcell.ButtonNone, tcell.ModNone), func(tview.Primitive) {})
	if !consumed {
		t.Error("scroll down was not consumed")
	}
	if row, _ := table.GetOffset(); row != 1 {
		t.Errorf("row offset after scroll down = %d, want 1", row)
	}

	consumed, _ = handler(tview.MouseScrollUp, tcell.NewEventMouse(0, 0, tcell.ButtonNone, tcell.ModNone), func(tview.Primitive) {})
	if !consumed {
		t.Error("scroll up was not consumed")
	}
	if row, _ := table.GetOffset(); row != 0 {
		t.Errorf("row offset after scroll up = %d, want 0", row)
	}
}
