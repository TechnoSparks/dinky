package filelist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// buildTestTree creates a directory with a controlled set of files and
// directories. Sizes, permissions, and modification times are all fixed so the
// sorting tests are deterministic.
func buildTestTree(t *testing.T) (string, []os.DirEntry) {
	t.Helper()
	dir := t.TempDir()
	base := time.Date(2021, 5, 6, 7, 8, 9, 0, time.UTC)

	files := []struct {
		name string
		data string
		mode os.FileMode
		mod  time.Time
	}{
		{"Alpha.go", strings.Repeat("a", 2000), 0o600, base.Add(2 * time.Hour)},
		{"beta.md", strings.Repeat("b", 1500), 0o640, base.Add(5 * time.Hour)},
		{"script.sh", strings.Repeat("s", 100), 0o755, base.Add(4 * time.Hour)},
		{"zeta.txt", strings.Repeat("z", 10), 0o644, base.Add(time.Hour)},
		{".hidden", "hidden", 0o600, base},
	}
	for _, file := range files {
		path := filepath.Join(dir, file.name)
		if err := os.WriteFile(path, []byte(file.data), file.mode); err != nil {
			t.Fatalf("WriteFile(%q): %v", file.name, err)
		}
		if err := os.Chmod(path, file.mode); err != nil {
			t.Fatalf("Chmod(%q): %v", file.name, err)
		}
		if err := os.Chtimes(path, file.mod, file.mod); err != nil {
			t.Fatalf("Chtimes(%q): %v", file.name, err)
		}
	}

	subdir := filepath.Join(dir, "adir")
	if err := os.Mkdir(subdir, 0o750); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if err := os.Chmod(subdir, 0o750); err != nil {
		t.Fatalf("Chmod subdir: %v", err)
	}
	subTime := base.Add(3 * time.Hour)
	if err := os.Chtimes(subdir, subTime, subTime); err != nil {
		t.Fatalf("Chtimes subdir: %v", err)
	}

	return dir, readEntries(t, dir)
}

func newSimulationScreen(t *testing.T, width, height int) tcell.Screen {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("simulation screen Init: %v", err)
	}
	screen.SetSize(width, height)
	t.Cleanup(screen.Fini)
	return screen
}

func waitForSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for file list update")
	}
}

func TestNewFileList(t *testing.T) {
	fileList := NewFileList(tview.NewApplication())

	if got := fileList.Path(); got != "/" {
		t.Errorf("Path() = %q, want %q", got, "/")
	}
	if fileList.sortColumn != 0 {
		t.Errorf("sortColumn = %d, want 0", fileList.sortColumn)
	}
	if fileList.sortDirection != 1 {
		t.Errorf("sortDirection = %d, want 1", fileList.sortDirection)
	}
	if got := fileList.table.GetColumnCount(); got != len(fileList.columnDescriptors) {
		t.Errorf("column count = %d, want %d", got, len(fileList.columnDescriptors))
	}
	if got := fileList.table.GetRowCount(); got != 1 {
		t.Errorf("initial row count = %d, want 1 (header only)", got)
	}

	wantHeaders := []string{"Name ▼", "Size", "Modified", "Permissions", "Owner", "Group"}
	for i, want := range wantHeaders {
		cell := fileList.table.GetCell(0, i)
		if cell.Text != want {
			t.Errorf("header %d = %q, want %q", i, cell.Text, want)
		}
		if !cell.NotSelectable {
			t.Errorf("header %d is selectable, want not selectable", i)
		}
	}
}

func TestFileListLoadEntries(t *testing.T) {
	dir, entries := buildTestTree(t)
	fileList := NewFileList(tview.NewApplication())
	fileList.setEntries(entries, dir)

	if got := fileList.Path(); got != dir {
		t.Errorf("Path() = %q, want %q", got, dir)
	}
	// Five visible files/directories plus the header row.
	if got := fileList.table.GetRowCount(); got != 6 {
		t.Errorf("row count = %d, want 6", got)
	}

	rowByName := make(map[string]int)
	for row := 1; row < fileList.table.GetRowCount(); row++ {
		name := fileList.table.GetCell(row, 0).Text
		for _, entry := range entries {
			if strings.HasSuffix(name, " "+entry.Name()) {
				rowByName[entry.Name()] = row
			}
		}
	}
	for _, entry := range entries {
		if entry.Name() == ".hidden" {
			continue
		}
		if _, ok := rowByName[entry.Name()]; !ok {
			t.Errorf("entry %q missing from table", entry.Name())
		}
	}

	zetaRow, ok := rowByName["zeta.txt"]
	if !ok {
		t.Fatal("zeta.txt row not found")
	}
	if got, want := fileList.table.GetCell(zetaRow, 0).Text, emojiForFileType(entryByName(entries, "zeta.txt"))+" zeta.txt"; got != want {
		t.Errorf("name cell = %q, want %q", got, want)
	}
	if got, want := fileList.table.GetCell(zetaRow, 1).Text, formatSize(10); got != want {
		t.Errorf("size cell = %q, want %q", got, want)
	}
	if got, want := fileList.table.GetCell(zetaRow, 2).Text, "2021-05-06 16:08:09"; got != want {
		t.Errorf("modified cell = %q, want %q", got, want)
	}
	if got, want := fileList.table.GetCell(zetaRow, 3).Text, "-rw-r--r--"; got != want {
		t.Errorf("permissions cell = %q, want %q", got, want)
	}
	if got := fileList.table.GetCell(zetaRow, 4).Text; got == "" || got == "?" {
		t.Errorf("owner cell = %q, want resolved owner", got)
	}
	if got := fileList.table.GetCell(zetaRow, 5).Text; got == "" {
		t.Error("group cell is empty")
	}

	if got, want := fileList.table.GetCell(zetaRow, 1).Align, tview.AlignRight; got != want {
		t.Errorf("size cell alignment = %d, want %d", got, want)
	}
	if got, want := fileList.table.GetCell(zetaRow, 0).Align, tview.AlignLeft; got != want {
		t.Errorf("name cell alignment = %d, want %d", got, want)
	}

	if row, column := fileList.table.GetSelection(); row != 1 || column != 0 {
		t.Errorf("selection = (%d, %d), want (1, 0)", row, column)
	}
	if row, column := fileList.table.GetOffset(); row != 0 || column != 0 {
		t.Errorf("offset = (%d, %d), want (0, 0)", row, column)
	}
}

func TestFileListLoadEntriesEmpty(t *testing.T) {
	fileList := NewFileList(tview.NewApplication())
	fileList.setEntries(nil, "/empty")

	if got := fileList.Path(); got != "/empty" {
		t.Errorf("Path() = %q, want %q", got, "/empty")
	}
	if got := fileList.table.GetRowCount(); got != 1 {
		t.Errorf("row count = %d, want 1", got)
	}
}

func TestFileListFilterVisible(t *testing.T) {
	dir, entries := buildTestTree(t)
	fileList := NewFileList(tview.NewApplication())
	fileList.setEntries(entries, dir)

	visibleNames := namesOf(fileList.visibleEntries)
	for _, name := range visibleNames {
		if strings.HasPrefix(name, ".") && name != ".." {
			t.Errorf("hidden entry %q is visible", name)
		}
	}
	if len(visibleNames) != 5 {
		t.Errorf("visible entries = %v, want 5 entries", visibleNames)
	}

	parent := aliasDirEntry(entries[0], "..")
	filtered := fileList.filterVisible([]os.DirEntry{
		parent,
		fakeDirEntry{name: ".hidden"},
		fakeDirEntry{name: "visible"},
	})
	if got, want := namesOf(filtered), []string{"..", "visible"}; !equalStrings(got, want) {
		t.Errorf("filterVisible() = %v, want %v", got, want)
	}
}

func TestFileListSetShowHidden(t *testing.T) {
	dir, entries := buildTestTree(t)
	fileList := NewFileList(tview.NewApplication())
	fileList.setEntries(entries, dir)

	if got := fileList.table.GetRowCount(); got != 6 {
		t.Fatalf("row count with hidden files hidden = %d, want 6", got)
	}

	fileList.SetShowHidden(true)
	if !fileList.showHidden {
		t.Error("showHidden = false after SetShowHidden(true)")
	}
	if got := fileList.table.GetRowCount(); got != 7 {
		t.Errorf("row count with hidden files shown = %d, want 7", got)
	}
	hiddenVisible := false
	for row := 1; row < fileList.table.GetRowCount(); row++ {
		if strings.HasSuffix(fileList.table.GetCell(row, 0).Text, " .hidden") {
			hiddenVisible = true
		}
	}
	if !hiddenVisible {
		t.Error("hidden file was not shown")
	}

	// Calling it again with the same value is a no-op.
	fileList.SetShowHidden(true)
	if got := fileList.table.GetRowCount(); got != 7 {
		t.Errorf("row count after repeat = %d, want 7", got)
	}

	fileList.SetShowHidden(false)
	if got := fileList.table.GetRowCount(); got != 6 {
		t.Errorf("row count after hiding again = %d, want 6", got)
	}
}

func TestFileListSortEntries(t *testing.T) {
	_, entries := buildTestTree(t)
	fileList := NewFileList(tview.NewApplication())

	visible := func() []os.DirEntry {
		result := make([]os.DirEntry, 0, len(entries))
		for _, entry := range entries {
			if !strings.HasPrefix(entry.Name(), ".") {
				result = append(result, entry)
			}
		}
		return result
	}

	tests := []struct {
		name      string
		column    int
		direction int
		want      []string
	}{
		{"name ascending", 0, 1, []string{"adir", "Alpha.go", "beta.md", "script.sh", "zeta.txt"}},
		{"name descending", 0, -1, []string{"zeta.txt", "script.sh", "beta.md", "Alpha.go", "adir"}},
		{"size ascending", 1, 1, []string{"zeta.txt", "adir", "script.sh", "beta.md", "Alpha.go"}},
		{"size descending", 1, -1, []string{"Alpha.go", "beta.md", "script.sh", "adir", "zeta.txt"}},
		{"modified ascending", 2, 1, []string{"zeta.txt", "Alpha.go", "adir", "script.sh", "beta.md"}},
		{"modified descending", 2, -1, []string{"beta.md", "script.sh", "adir", "Alpha.go", "zeta.txt"}},
		{"permissions ascending", 3, 1, []string{"Alpha.go", "beta.md", "zeta.txt", "script.sh", "adir"}},
		{"permissions descending", 3, -1, []string{"adir", "script.sh", "zeta.txt", "beta.md", "Alpha.go"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sorted := visible()
			fileList.sortEntries(sorted, tc.column, tc.direction)
			if got := namesOf(sorted); !equalStrings(got, tc.want) {
				t.Errorf("sorted names = %v, want %v", got, tc.want)
			}
		})
	}

	// Owner and group sorting uses the same owner/group for all entries in the
	// temporary directory. It must not panic or lose entries.
	for _, column := range []int{4, 5} {
		sorted := visible()
		fileList.sortEntries(sorted, column, 1)
		if len(sorted) != len(visible()) {
			t.Errorf("column %d sort lost entries: %d", column, len(sorted))
		}
	}
}

func TestFileListSetSortColumn(t *testing.T) {
	dir, entries := buildTestTree(t)
	fileList := NewFileList(tview.NewApplication())
	fileList.setEntries(entries, dir)

	fileList.SetSortColumn(1, 1)
	if got := fileList.table.GetCell(0, 1).Text; got != "Size ▼" {
		t.Errorf("Size header = %q, want %q", got, "Size ▼")
	}
	if got := fileList.table.GetCell(0, 0).Text; got != "Name" {
		t.Errorf("Name header = %q, want %q", got, "Name")
	}

	fileList.SetSortColumn(1, -1)
	if got := fileList.table.GetCell(0, 1).Text; got != "Size ▲" {
		t.Errorf("Size header = %q, want %q", got, "Size ▲")
	}

	// Clicking a different header keeps the current direction.
	fileList.table.GetCell(0, 3).Clicked()
	if got := fileList.table.GetCell(0, 3).Text; got != "Permissions ▲" {
		t.Errorf("Permissions header = %q, want %q", got, "Permissions ▲")
	}
	// Clicking the active header toggles the direction.
	fileList.table.GetCell(0, 3).Clicked()
	if got := fileList.table.GetCell(0, 3).Text; got != "Permissions ▼" {
		t.Errorf("Permissions header = %q, want %q", got, "Permissions ▼")
	}

	// Clicking the currently active name column toggles back to descending.
	fileList.SetSortColumn(0, 1)
	fileList.table.GetCell(0, 0).Clicked()
	if got := fileList.table.GetCell(0, 0).Text; got != "Name ▲" {
		t.Errorf("Name header = %q, want %q", got, "Name ▲")
	}
}

func TestFileListSelectionCallbacks(t *testing.T) {
	dir, entries := buildTestTree(t)
	fileList := NewFileList(tview.NewApplication())
	fileList.setEntries(entries, dir)

	var changedPath, changedName string
	var selectedPath, selectedName string
	fileList.SetChangedFunc(func(path string, entry os.DirEntry) {
		changedPath = path
		changedName = entry.Name()
	})
	fileList.SetSelectedFunc(func(path string, entry os.DirEntry) {
		selectedPath = path
		selectedName = entry.Name()
	})

	fileList.table.Select(3, 0)
	wantChanged := fileList.visibleEntries[2]
	if changedName != wantChanged.Name() {
		t.Errorf("changed entry = %q, want %q", changedName, wantChanged.Name())
	}
	if want := filepath.Join(dir, wantChanged.Name()); changedPath != want {
		t.Errorf("changed path = %q, want %q", changedPath, want)
	}
	if selectedName != "" {
		t.Errorf("selected callback fired on selection change: %q", selectedName)
	}

	fileList.table.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
	if selectedName != wantChanged.Name() {
		t.Errorf("selected entry = %q, want %q", selectedName, wantChanged.Name())
	}
	if want := filepath.Join(dir, wantChanged.Name()); selectedPath != want {
		t.Errorf("selected path = %q, want %q", selectedPath, want)
	}

	// The header row must not produce callbacks.
	changedName = ""
	fileList.table.Select(0, 0)
	if changedName != "" {
		t.Errorf("header row fired changed callback with %q", changedName)
	}

	// Nil callbacks are safe.
	fileList.SetChangedFunc(nil)
	fileList.SetSelectedFunc(nil)
	fileList.table.Select(2, 0)
	fileList.table.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
}

func TestFileListNavigation(t *testing.T) {
	dir, entries := buildTestTree(t)
	fileList := NewFileList(tview.NewApplication())
	fileList.setEntries(entries, dir)

	screen := newSimulationScreen(t, 100, 20)
	fileList.SetRect(0, 0, 100, 20)
	fileList.Draw(screen)

	var changes int
	fileList.SetChangedFunc(func(string, os.DirEntry) { changes++ })

	press := func(key tcell.Key, r rune) {
		t.Helper()
		fileList.table.InputHandler()(tcell.NewEventKey(key, r, tcell.ModNone), func(tview.Primitive) {})
	}
	assertSelection := func(want int) {
		t.Helper()
		if row, _ := fileList.table.GetSelection(); row != want {
			t.Errorf("selection row = %d, want %d", row, want)
		}
	}

	press(tcell.KeyDown, 0)
	assertSelection(2)
	press(tcell.KeyRune, 'j')
	assertSelection(3)
	press(tcell.KeyUp, 0)
	assertSelection(2)
	press(tcell.KeyRune, 'k')
	assertSelection(1)

	lastRow := fileList.table.GetRowCount() - 1
	press(tcell.KeyEnd, 0)
	assertSelection(lastRow)
	press(tcell.KeyHome, 0)
	assertSelection(1)

	press(tcell.KeyPgDn, 0)
	assertSelection(lastRow)
	press(tcell.KeyPgUp, 0)
	assertSelection(1)

	if changes < 8 {
		t.Errorf("changed callback fired %d times, want at least 8", changes)
	}
}

func TestFileListFocus(t *testing.T) {
	fileList := NewFileList(tview.NewApplication())
	var focused tview.Primitive
	fileList.Focus(func(p tview.Primitive) { focused = p })
	if focused != fileList.table {
		t.Errorf("Focus delegated to %T, want table", focused)
	}
}

func TestFileListColors(t *testing.T) {
	dir, entries := buildTestTree(t)
	fileList := NewFileList(tview.NewApplication())
	fileList.setEntries(entries, dir)

	fileList.SetTextColor(tcell.ColorRed)
	fileList.SetBackgroundColor(tcell.ColorBlue)
	fileList.SetSelectedBackgroundColor(tcell.ColorGreen)
	fileList.SetHeaderLabelColor(tcell.ColorYellow)
	fileList.SetHeaderBackgroundColor(tcell.ColorDarkCyan)
	fileList.setEntries(entries, dir)

	if got := fileList.table.GetCell(1, 0).Style; func() bool {
		fg, _, _ := got.Decompose()
		return fg != tcell.ColorRed
	}() {
		t.Error("text color was not applied to data cells")
	}
	if got := fileList.table.GetBackgroundColor(); got != tcell.ColorBlue {
		t.Errorf("table background = %v, want blue", got)
	}
	if got := fileList.GetBackgroundColor(); got != tcell.ColorBlue {
		t.Errorf("flex background = %v, want blue", got)
	}
	headerStyle := fileList.table.GetCell(0, 0).Style
	fg, bg, _ := headerStyle.Decompose()
	if fg != tcell.ColorYellow || bg != tcell.ColorDarkCyan {
		t.Errorf("header style = (%v, %v), want yellow on dark cyan", fg, bg)
	}
	if !strings.HasSuffix(fileList.table.GetCell(0, 0).Text, "▼") {
		t.Errorf("header text lost sort indicator: %q", fileList.table.GetCell(0, 0).Text)
	}
}

func TestFileListScrollbars(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 40; i++ {
		name := filepath.Join(dir, "file-"+strings.Repeat("x", i%5)+string(rune('a'+i%26))+string(rune('a'+i/26)))
		if err := os.WriteFile(name, []byte(strings.Repeat("d", i)), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	entries := readEntries(t, dir)

	fileList := NewFileList(tview.NewApplication())
	fileList.setEntries(entries, dir)

	screen := newSimulationScreen(t, 80, 20)
	fileList.SetRect(0, 0, 80, 20)
	fileList.Draw(screen)

	vertical := fileList.VerticalScrollbar.Track
	if got, want := vertical.Max(), fileList.table.GetRowCount()-1; got != want {
		t.Errorf("vertical scrollbar max = %d, want %d", got, want)
	}
	if vertical.ThumbSize() < 1 {
		t.Errorf("vertical scrollbar thumb size = %d, want >= 1", vertical.ThumbSize())
	}
	if vertical.Position() != 0 {
		t.Errorf("vertical scrollbar position = %d, want 0", vertical.Position())
	}

	fileList.VerticalScrollbar.DownButton.InputHandler()(
		tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
	if row, _ := fileList.table.GetOffset(); row <= 0 {
		t.Errorf("table offset after scrolling down = %d, want > 0", row)
	}
	if got := vertical.Position(); got <= 0 {
		t.Errorf("vertical scrollbar position = %d, want > 0", got)
	}

	fileList.VerticalScrollbar.UpButton.InputHandler()(
		tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
	if row, _ := fileList.table.GetOffset(); row != 0 {
		t.Errorf("table offset after scrolling up = %d, want 0", row)
	}

	horizontal := fileList.HorizontalScrollbar.Track
	if horizontal.ThumbSize() < 1 {
		t.Errorf("horizontal scrollbar thumb size = %d, want >= 1", horizontal.ThumbSize())
	}
}

func TestFileListSetPath(t *testing.T) {
	fileList := NewFileList(tview.NewApplication())

	// Replace the request channel so the test can observe SetPath without
	// relying on the asynchronous directory lister.
	requests := make(chan string, 1)
	fileList.dirRequestsChan = requests

	fileList.SetPath("/some/directory")
	select {
	case got := <-requests:
		if got != "/some/directory" {
			t.Errorf("SetPath sent %q, want %q", got, "/some/directory")
		}
	default:
		t.Fatal("SetPath did not send a directory request")
	}

	if got := fileList.Path(); got != "/" {
		t.Errorf("Path() before the request is processed = %q, want %q", got, "/")
	}
}

func TestFileListSetPathLoadsDirectory(t *testing.T) {
	dir, _ := buildTestTree(t)

	app := tview.NewApplication()
	screen := tcell.NewSimulationScreen("UTF-8")
	app.SetScreen(screen)
	screen.SetSize(100, 30)

	fileList := NewFileList(app)
	app.SetRoot(fileList, true)

	done := make(chan error, 1)
	go func() { done <- app.Run() }()
	t.Cleanup(func() {
		app.Stop()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("app.Run() returned %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("app did not stop")
		}
	})

	// Wait until the event loop is processing queued updates.
	app.QueueUpdate(func() {})

	changed := make(chan struct{}, 4)
	fileList.SetChangedFunc(func(string, os.DirEntry) {
		select {
		case changed <- struct{}{}:
		default:
		}
	})

	fileList.SetPath(dir)
	waitForSignal(t, changed)

	if got := fileList.Path(); got != dir {
		t.Errorf("Path() = %q, want %q", got, dir)
	}
	// The asynchronous loader prepends a synthetic ".." entry.
	if got, want := fileList.table.GetRowCount(), len(fileList.visibleEntries)+1; got != want {
		t.Errorf("row count = %d, want %d", got, want)
	}
	if len(fileList.visibleEntries) == 0 || fileList.visibleEntries[0].Name() != ".." {
		t.Errorf("first visible entry = %v, want the parent alias", namesOf(fileList.visibleEntries))
	}
}
