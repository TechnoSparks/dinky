package application

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dinky/internal/application/settingstype"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sedwards2009/smidgen/micro/buffer"
)

// runParseCommandLine runs parseCommandLine with the given arguments and
// captures its stdout/stderr. os.Args, stdout, stderr and enableLogging are
// always restored.
func runParseCommandLine(t *testing.T, args ...string) (ok bool, stdout, stderr string, resultArgs []string, resultLogging bool) {
	t.Helper()

	oldArgs := os.Args
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	oldLogging := enableLogging
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		enableLogging = oldLogging
	}()

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = outW
	os.Stderr = errW
	os.Args = append([]string{"dinky"}, args...)
	enableLogging = false

	ok = parseCommandLine()
	resultArgs = append([]string(nil), os.Args...)
	resultLogging = enableLogging

	// Close the write ends before reading so ReadAll sees EOF.
	_ = outW.Close()
	_ = errW.Close()

	outBytes, _ := io.ReadAll(outR)
	errBytes, _ := io.ReadAll(errR)
	return ok, string(outBytes), string(errBytes), resultArgs, resultLogging
}

func TestParseCommandLineFileArguments(t *testing.T) {
	ok, stdout, stderr, resultArgs, _ := runParseCommandLine(t, "a.txt", "b.txt")
	if !ok {
		t.Fatalf("parseCommandLine returned false, stderr=%q", stderr)
	}
	want := []string{"dinky", "a.txt", "b.txt"}
	if strings.Join(resultArgs, "|") != strings.Join(want, "|") {
		t.Errorf("os.Args = %q, want %q", os.Args, want)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("unexpected output stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestParseCommandLineEndOfOptions(t *testing.T) {
	ok, _, _, resultArgs, _ := runParseCommandLine(t, "--", "-h", "--log", "file.txt")
	if !ok {
		t.Fatal("parseCommandLine returned false")
	}
	want := []string{"dinky", "-h", "--log", "file.txt"}
	if strings.Join(resultArgs, "|") != strings.Join(want, "|") {
		t.Errorf("os.Args = %q, want %q", os.Args, want)
	}
	if enableLogging {
		t.Error("--log after -- should be treated as a filename, not an option")
	}
}

func TestParseCommandLineStdinDash(t *testing.T) {
	ok, _, _, resultArgs, _ := runParseCommandLine(t, "-", "file.txt")
	if !ok {
		t.Fatal("parseCommandLine returned false")
	}
	want := []string{"dinky", "", "file.txt"}
	if strings.Join(resultArgs, "|") != strings.Join(want, "|") {
		t.Errorf("os.Args = %q, want %q", os.Args, want)
	}
}

func TestParseCommandLineLogging(t *testing.T) {
	ok, _, _, resultArgs, logging := runParseCommandLine(t, "--log", "file.txt")
	if !ok {
		t.Fatal("parseCommandLine returned false")
	}
	if !logging {
		t.Error("--log did not enable logging")
	}
	want := []string{"dinky", "file.txt"}
	if strings.Join(resultArgs, "|") != strings.Join(want, "|") {
		t.Errorf("os.Args = %q, want %q", os.Args, want)
	}
}

func TestParseCommandLineHelp(t *testing.T) {
	ok, stdout, _, _, _ := runParseCommandLine(t, "-h")
	if ok {
		t.Error("parseCommandLine returned true for -h")
	}
	if !strings.Contains(stdout, "Usage: dinky") || !strings.Contains(stdout, "--help") {
		t.Errorf("help output is missing expected text: %q", stdout)
	}
}

func TestParseCommandLineLongHelp(t *testing.T) {
	ok, stdout, _, _, _ := runParseCommandLine(t, "--help")
	if ok {
		t.Error("parseCommandLine returned true for --help")
	}
	if !strings.Contains(stdout, "Dinky - A little text editor") {
		t.Errorf("help output is missing expected text: %q", stdout)
	}
}

func TestParseCommandLineVersion(t *testing.T) {
	ok, stdout, _, _, _ := runParseCommandLine(t, "--version")
	if ok {
		t.Error("parseCommandLine returned true for --version")
	}
	if !strings.Contains(stdout, "Version:") || !strings.Contains(stdout, "Version time:") {
		t.Errorf("version output is missing expected text: %q", stdout)
	}
}

func TestParseCommandLineUnknownOption(t *testing.T) {
	ok, _, stderr, _, _ := runParseCommandLine(t, "-z")
	if ok {
		t.Error("parseCommandLine returned true for an unknown option")
	}
	if !strings.Contains(stderr, "Unknown option '-z'") {
		t.Errorf("stderr = %q, want unknown option message", stderr)
	}
}

func TestBoolToInt(t *testing.T) {
	if got := boolToInt(true); got != 1 {
		t.Errorf("boolToInt(true) = %d, want 1", got)
	}
	if got := boolToInt(false); got != 0 {
		t.Errorf("boolToInt(false) = %d, want 0", got)
	}
}

func TestIsBufferCRLF(t *testing.T) {
	h := newTestHarness(t)
	buf := h.open("x", "").buffer

	buf.Settings["fileformat"] = "dos"
	if !isBufferCRLF(buf) {
		t.Error("dos fileformat should be treated as CRLF")
	}

	buf.Settings["fileformat"] = "unix"
	if isBufferCRLF(buf) {
		t.Error("unix fileformat should not be treated as CRLF")
	}
}

func TestGetFileBufferByID(t *testing.T) {
	h := newTestHarness(t)
	fb := h.open("x", "")

	if got := getFileBufferByID(fb.uuid); got != fb {
		t.Errorf("getFileBufferByID(%q) = %p, want %p", fb.uuid, got, fb)
	}
	if got := getFileBufferByID("not-a-real-id"); got != nil {
		t.Errorf("getFileBufferByID(unknown) = %p, want nil", got)
	}
}

func TestNewFileInitialState(t *testing.T) {
	h := newTestHarness(t)

	settings = settingstype.DefaultSettings()
	settings.ShowWhitespace = true
	settings.ShowLineNumbers = false
	settings.ShowMatchBracket = false
	settings.SoftWrap = true
	settings.TabSize = 8
	settings.TabCharacter = "tab"
	settings.ShowTrailingWhitespace = false
	settings.VerticalRuler = 30

	fb := h.open("hello\nworld", "/tmp/some-dir/example.txt")

	if len(fileBuffers) != 1 || fileBuffers[0] != fb {
		t.Fatalf("fileBuffers = %v, want just the new buffer", fileBuffers)
	}
	if currentFileBuffer != fb {
		t.Error("newFile did not make the buffer current")
	}
	if fileBufferID != fb.uuid {
		t.Errorf("fileBufferID = %q, want %q", fileBufferID, fb.uuid)
	}
	if fb.filename != "/tmp/some-dir/example.txt" {
		t.Errorf("filename = %q", fb.filename)
	}
	if fb.uuid == "" {
		t.Error("buffer uuid is empty")
	}
	if !editorPages.HasPage(fb.uuid) {
		t.Error("editor page was not added")
	}
	if index, id := tabBarLine.Active(); index != 0 || id != fb.uuid {
		t.Errorf("active tab = (%d, %q), want (0, %q)", index, id, fb.uuid)
	}

	// Settings are copied onto the smidgen buffer.
	buf := fb.buffer
	if buf.Settings["matchbrace"] != false {
		t.Error("matchbrace setting not applied")
	}
	if buf.Settings["ruler"] != false {
		t.Error("ruler setting not applied")
	}
	if buf.Settings["showwhitespace"] != true {
		t.Error("showwhitespace setting not applied")
	}
	if buf.Settings["softwrap"] != true {
		t.Error("softwrap setting not applied")
	}
	if buf.Settings["tabsize"] != float64(8) {
		t.Errorf("tabsize = %v, want 8", buf.Settings["tabsize"])
	}
	if buf.Settings["tabstospaces"] != false {
		t.Error("tabstospaces setting not applied")
	}
	if buf.Settings["hltrailingws"] != false {
		t.Error("hltrailingws setting not applied")
	}
	if buf.Settings["colorcolumn"] != float64(30) {
		t.Errorf("colorcolumn = %v, want 30", buf.Settings["colorcolumn"])
	}

	// The tab title uses the base name of the file.
	h.drawTabBar(80)
	if row := h.screenRow(0, 80); !strings.Contains(row, "example.txt") {
		t.Errorf("tab bar row = %q, want it to contain the file name", row)
	}
}

func TestNewFileUntitledTabTitle(t *testing.T) {
	h := newTestHarness(t)
	h.open("", "")

	h.drawTabBar(80)
	if row := h.screenRow(0, 80); !strings.Contains(row, "[Untitled]") {
		t.Errorf("tab bar row = %q, want [Untitled]", row)
	}
}

func TestNewFileMultipleBuffersAndSelectTab(t *testing.T) {
	h := newTestHarness(t)
	tabBarLine.SetRect(0, 0, 120, 1)
	first := h.open("first", "first.txt")
	second := h.open("second", "second.txt")

	if len(fileBuffers) != 2 {
		t.Fatalf("fileBuffers has %d entries, want 2", len(fileBuffers))
	}
	if currentFileBuffer != second {
		t.Error("second file was not made current")
	}
	if _, id := tabBarLine.Active(); id != second.uuid {
		t.Errorf("active tab id = %q, want %q", id, second.uuid)
	}

	h.drawTabBar(80)
	row := h.screenRow(0, 80)
	if !strings.Contains(row, "first.txt") || !strings.Contains(row, "second.txt") {
		t.Errorf("tab bar row = %q, want both file names", row)
	}

	selectTab(first.uuid)
	if currentFileBuffer != first {
		t.Error("selectTab did not switch the current buffer")
	}
	if fileBufferID != first.uuid {
		t.Errorf("fileBufferID = %q, want %q", fileBufferID, first.uuid)
	}
	if _, id := tabBarLine.Active(); id != first.uuid {
		t.Errorf("active tab id = %q, want %q", id, first.uuid)
	}
}

func TestFindbarOpenCloseAndSelection(t *testing.T) {
	h := newTestHarness(t)
	fb := h.open("alpha beta gamma\ndelta", "")

	if fb.panelVFlex.GetItemCount() != 1 {
		t.Fatalf("panel starts with %d items, want 1", fb.panelVFlex.GetItemCount())
	}

	fb.openFindbar()
	if !fb.isFindbarOpen {
		t.Error("findbar was not marked open")
	}
	if fb.panelVFlex.GetItemCount() != 2 {
		t.Errorf("panel has %d items after opening findbar, want 2", fb.panelVFlex.GetItemCount())
	}

	// Opening again must not add a second copy.
	fb.openFindbar()
	if fb.panelVFlex.GetItemCount() != 2 {
		t.Errorf("panel has %d items after opening twice, want 2", fb.panelVFlex.GetItemCount())
	}

	// A selection populates the search field with the first selected line.
	h.setSelection(0, 0, 5, 0) // "alpha"
	fb.openFindbar()
	if got := fb.findbar.SearchStringField.GetText(); got != "alpha" {
		t.Errorf("findbar search text = %q, want %q", got, "alpha")
	}

	fb.findbar.OnClose()
	if fb.isFindbarOpen {
		t.Error("findbar was not marked closed")
	}
	if fb.panelVFlex.GetItemCount() != 1 {
		t.Errorf("panel has %d items after closing findbar, want 1", fb.panelVFlex.GetItemCount())
	}
}

func TestContentWidth(t *testing.T) {
	h := newTestHarness(t)
	fb := h.open("abc\n12345678\nxy", "")

	if got := fb.contentWidth(); got != 8 {
		t.Errorf("contentWidth() = %d, want 8", got)
	}

	// Tab size changes invalidate the cache.
	fb.buffer.Settings["tabsize"] = float64(4)
	if got := fb.contentWidth(); got != 8 {
		t.Errorf("contentWidth() with tab size 4 = %d, want 8", got)
	}
	fb.buffer.Insert(buffer.Loc{X: 0, Y: 2}, "a\tb") // line becomes "a\tbxy"
	fb.buffer.Settings["tabsize"] = float64(4)
	// "a" (1) + tab to column 4 (3) + "bxy" (3) = 7
	if got := fb.contentWidth(); got != 8 {
		t.Errorf("contentWidth() after insert = %d, want 8 (widest line is still 12345678)", got)
	}

	fb2 := newTestHarness(t).open("a\tb", "")
	fb2.buffer.Settings["tabsize"] = float64(4)
	if got := fb2.contentWidth(); got != 5 {
		t.Errorf("contentWidth() for tabs at size 4 = %d, want 5", got)
	}
	fb2.buffer.Settings["tabsize"] = float64(8)
	if got := fb2.contentWidth(); got != 9 {
		t.Errorf("contentWidth() for tabs at size 8 = %d, want 9", got)
	}
}

func TestContentWidthUnicode(t *testing.T) {
	h := newTestHarness(t)
	fb := h.open("日本語\nabc", "")

	if got := fb.contentWidth(); got != 6 {
		t.Errorf("contentWidth() = %d, want 6 (CJK characters are double width)", got)
	}
}

func TestSyncStatusBarFromFileBuffer(t *testing.T) {
	h := newTestHarness(t)
	fb := h.open("abc\ndef", "/tmp/example.txt")
	h.setCursor(1, 1)
	fb.buffer.OverwriteMode = true
	fb.buffer.Settings["tabsize"] = float64(8)
	fb.buffer.Settings["fileformat"] = "dos"
	fb.buffer.Insert(buffer.Loc{X: 0, Y: 0}, "x")

	syncStatusBarFromFileBuffer(statusBar)

	if statusBar.Filename != "/tmp/example.txt" {
		t.Errorf("Filename = %q", statusBar.Filename)
	}
	if statusBar.Line != 2 || statusBar.Col != 2 {
		t.Errorf("cursor = Ln %d, Col %d, want Ln 2, Col 2", statusBar.Line, statusBar.Col)
	}
	if !statusBar.IsModified {
		t.Error("IsModified = false, want true")
	}
	if statusBar.TabSize != 8 {
		t.Errorf("TabSize = %d, want 8", statusBar.TabSize)
	}
	if !statusBar.IsOverwriteMode {
		t.Error("IsOverwriteMode = false, want true")
	}
	if statusBar.LineEndings != "CRLF" {
		t.Errorf("LineEndings = %q, want CRLF", statusBar.LineEndings)
	}

	fb.buffer.Settings["fileformat"] = "unix"
	fb.buffer.ClearModified()
	syncStatusBarFromFileBuffer(statusBar)
	if statusBar.IsModified {
		t.Error("IsModified = true after ClearModified")
	}
	if statusBar.LineEndings != "LF" {
		t.Errorf("LineEndings = %q, want LF", statusBar.LineEndings)
	}
}

func TestSyncStatusBarWithNoCurrentBuffer(t *testing.T) {
	newTestHarness(t)
	fileBufferID = "missing"
	// Must be a no-op rather than panicking.
	syncStatusBarFromFileBuffer(statusBar)
}

func TestEditorInputCaptureDinkyShortcut(t *testing.T) {
	h := newTestHarness(t)
	h.open("x", "")
	if len(fileBuffers) != 1 {
		t.Fatal("expected one buffer to start")
	}

	event := tcell.NewEventKey(tcell.KeyCtrlN, 0, tcell.ModCtrl)
	if got := editorInputCapture(event); got != nil {
		t.Errorf("Ctrl-N capture returned %v, want nil", got)
	}
	if len(fileBuffers) != 2 {
		t.Errorf("Ctrl-N created %d buffers, want 2", len(fileBuffers))
	}
}

func TestEditorInputCapturePassesThroughUnboundKeys(t *testing.T) {
	newTestHarness(t).open("x", "")

	event := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	if got := editorInputCapture(event); got != event {
		t.Errorf("unbound rune was not passed through")
	}
}

func TestEditorInputCaptureOpensMenu(t *testing.T) {
	newTestHarness(t).open("x", "")

	event := tcell.NewEventKey(tcell.KeyF12, 0, tcell.ModNone)
	if got := editorInputCapture(event); got != nil {
		t.Errorf("F12 capture returned %v, want nil", got)
	}
	if !menuBar.HasFocus() {
		t.Error("F12 did not move focus to the menu bar")
	}
}

func TestHandleHorizontalWheelScroll(t *testing.T) {
	h := newTestHarness(t)
	fb := h.open(strings.Repeat("x", 60), "")
	fb.editor.SetRect(0, 0, 40, 10)
	fb.editor.Draw(h.screen)
	view := fb.editor.ActionController().GetView()

	// Scrolling left at column 0 stays at 0.
	view.StartCol = 0
	if !handleHorizontalWheelScroll(tview.MouseScrollLeft, tcell.NewEventMouse(5, 5, tcell.ButtonNone, tcell.ModNone)) {
		t.Error("horizontal scroll was not handled inside the editor")
	}
	if view.StartCol != 0 {
		t.Errorf("StartCol = %d after scrolling left at 0, want 0", view.StartCol)
	}

	// Scrolling right moves by the scroll speed (2 by default).
	if !handleHorizontalWheelScroll(tview.MouseScrollRight, tcell.NewEventMouse(5, 5, tcell.ButtonNone, tcell.ModNone)) {
		t.Error("horizontal scroll right was not handled")
	}
	if view.StartCol != 2 {
		t.Errorf("StartCol = %d after scrolling right, want 2", view.StartCol)
	}

	// Scrolling left moves back.
	if !handleHorizontalWheelScroll(tview.MouseScrollLeft, tcell.NewEventMouse(5, 5, tcell.ButtonNone, tcell.ModNone)) {
		t.Error("horizontal scroll left was not handled")
	}
	if view.StartCol != 0 {
		t.Errorf("StartCol = %d after scrolling left, want 0", view.StartCol)
	}

	// The offset is clamped to the widest line minus the viewport width.
	view.StartCol = 1000
	handleHorizontalWheelScroll(tview.MouseScrollRight, tcell.NewEventMouse(5, 5, tcell.ButtonNone, tcell.ModNone))
	bufWidth := fb.editor.ActionController().BufView().Width
	if view.StartCol != fb.contentWidth()-bufWidth {
		t.Errorf("StartCol = %d, want clamped to contentWidth-viewportWidth = %d", view.StartCol, fb.contentWidth()-bufWidth)
	}

	// Outside the editor rectangle the event is not handled.
	if handleHorizontalWheelScroll(tview.MouseScrollRight, tcell.NewEventMouse(500, 500, tcell.ButtonNone, tcell.ModNone)) {
		t.Error("horizontal scroll outside the editor was handled")
	}

	// Soft wrap disables horizontal scrolling.
	fb.buffer.Settings["softwrap"] = true
	if handleHorizontalWheelScroll(tview.MouseScrollRight, tcell.NewEventMouse(5, 5, tcell.ButtonNone, tcell.ModNone)) {
		t.Error("horizontal scroll was handled with soft wrap enabled")
	}
	fb.buffer.Settings["softwrap"] = false

	// With no current buffer the event is not handled.
	currentFileBuffer = nil
	if handleHorizontalWheelScroll(tview.MouseScrollRight, tcell.NewEventMouse(5, 5, tcell.ButtonNone, tcell.ModNone)) {
		t.Error("horizontal scroll was handled with no current buffer")
	}
}

func TestGetDinkyVersionInfo(t *testing.T) {
	if got := getDinkyVersion(); got == "" {
		t.Error("getDinkyVersion returned an empty string")
	}
	if got := getDinkyVersionTime(); got == "" {
		t.Error("getDinkyVersionTime returned an empty string")
	}
}

func TestWriteFile(t *testing.T) {
	h := newTestHarness(t)
	fb := h.open("hello\nworld", "")
	path := filepath.Join(t.TempDir(), "out.txt")

	ok, message := writeFile(path, fb.buffer)
	if !ok {
		t.Fatalf("writeFile failed: %s", message)
	}
	if !strings.Contains(message, path) {
		t.Errorf("message = %q, want it to mention %q", message, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := h.text()
	if fb.buffer.Settings["eofnewline"].(bool) && len(want) > 0 && want[len(want)-1] != '\n' {
		want += "\n"
	}
	if string(data) != want {
		t.Errorf("written contents = %q, want %q", data, want)
	}
}

func TestWriteFileEnsureFinalNewline(t *testing.T) {
	h := newTestHarness(t)
	fb := h.open("hello\nworld", "")
	path := filepath.Join(t.TempDir(), "out.txt")

	// Disabled: write exactly what the buffer contains.
	fb.buffer.Settings["eofnewline"] = false
	ok, message := writeFile(path, fb.buffer)
	if !ok {
		t.Fatalf("writeFile failed: %s", message)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "hello\nworld" {
		t.Errorf("with ensure off: got %q, want %q", data, "hello\\nworld")
	}

	// Enabled: a final newline is added to both the file and the buffer.
	fb.buffer.Settings["eofnewline"] = true
	path2 := filepath.Join(t.TempDir(), "out2.txt")
	ok, message = writeFile(path2, fb.buffer)
	if !ok {
		t.Fatalf("writeFile failed: %s", message)
	}
	data, _ = os.ReadFile(path2)
	if string(data) != "hello\nworld\n" {
		t.Errorf("with ensure on: got %q, want %q", data, "hello\\nworld\\n")
	}
	if got := h.text(); got != "hello\nworld\n" {
		t.Errorf("buffer after save = %q, want trailing newline", got)
	}
}

func TestWriteFileEnsureFinalNewlineCRLF(t *testing.T) {
	h := newTestHarness(t)
	fb := h.open("hello\nworld", "")
	fb.buffer.Settings["fileformat"] = "dos"
	fb.buffer.LineArray.Endings = buffer.FFDos
	path := filepath.Join(t.TempDir(), "out.txt")

	fb.buffer.Settings["eofnewline"] = true
	ok, message := writeFile(path, fb.buffer)
	if !ok {
		t.Fatalf("writeFile failed: %s", message)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "hello\r\nworld\r\n" {
		t.Errorf("got %q, want %q", data, "hello\\r\\nworld\\r\\n")
	}
}

func TestWriteFileFailure(t *testing.T) {
	h := newTestHarness(t)
	fb := h.open("hello", "")
	path := filepath.Join(t.TempDir(), "missing-dir", "out.txt")

	ok, message := writeFile(path, fb.buffer)
	if ok {
		t.Fatal("writeFile unexpectedly succeeded")
	}
	if !strings.HasPrefix(message, "Error writing file:") {
		t.Errorf("message = %q, want an error message", message)
	}
}
