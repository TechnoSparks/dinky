package menu

import (
	"reflect"
	"strings"
	"testing"

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

func contentAt(screen tcell.SimulationScreen, x, y int) rune {
	mainc, _, _, _ := screen.GetContent(x, y)
	return mainc
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

func keyEvent(key tcell.Key, r rune, mod tcell.ModMask) *tcell.EventKey {
	return tcell.NewEventKey(key, r, mod)
}

func TestNewMenuBarDefaults(t *testing.T) {
	menuBar := NewMenuBar()
	if menuBar == nil {
		t.Fatal("NewMenuBar returned nil")
	}
	if menuBar.Box == nil {
		t.Fatal("NewMenuBar returned a MenuBar with a nil Box")
	}
	if !reflect.DeepEqual(menuBar.selectedPath, []int{-1}) {
		t.Errorf("selectedPath = %v, want [-1]", menuBar.selectedPath)
	}

	if fg, bg, attr := menuBar.MenuBarStyle.Decompose(); fg != tcell.NewHexColor(0xf3f3f3) || bg != tcell.NewHexColor(0x007ace) || attr&tcell.AttrBold == 0 {
		t.Errorf("unexpected MenuBarStyle fg/bg/attr: %v/%v/%v", fg, bg, attr)
	}
	if _, _, attr := menuBar.MenuSelectedStyle.Decompose(); attr&tcell.AttrReverse == 0 {
		t.Error("MenuSelectedStyle should be reversed")
	}
	if menuBar.MenuStyle != menuBar.MenuBarStyle {
		t.Error("MenuStyle should match MenuBarStyle by default")
	}
}

func TestSetMenusOpenCloseAndOnClose(t *testing.T) {
	menus := []*Menu{
		{Title: "File", Items: []*MenuItem{{Title: "Open"}}},
		{Title: "Edit", Items: []*MenuItem{{Title: "Undo"}}},
	}

	menuBar := NewMenuBar()
	menuBar.SetMenus(menus)
	if !reflect.DeepEqual(menuBar.menus, menus) {
		t.Fatalf("SetMenus did not store the menus")
	}

	menuBar.Open(1)
	if !reflect.DeepEqual(menuBar.selectedPath, []int{1, 0}) {
		t.Errorf("after Open(1), selectedPath = %v, want [1 0]", menuBar.selectedPath)
	}

	var closeArgs []tview.Primitive
	menuBar.SetOnClose(func(nextFocus tview.Primitive) {
		closeArgs = append(closeArgs, nextFocus)
	})

	menuBar.Close()
	if !reflect.DeepEqual(menuBar.selectedPath, []int{-1}) {
		t.Errorf("after Close, selectedPath = %v, want [-1]", menuBar.selectedPath)
	}
	if len(closeArgs) != 1 || closeArgs[0] != nil {
		t.Errorf("OnClose calls = %v, want one call with nil", closeArgs)
	}

	menuBar.SetOnClose(func(p tview.Primitive) {
		closeArgs = append(closeArgs, p)
	})
	menuBar.Open(0)
	menuBar.Close()
	if len(closeArgs) != 2 || closeArgs[1] != nil {
		t.Errorf("replacement OnClose was not used, calls = %v", closeArgs)
	}
}

func TestMeasureWidthsAndMenuWidthInCells(t *testing.T) {
	tests := []struct {
		name             string
		items            []*MenuItem
		wantTitleWidth   int
		wantShortcutSize int
		wantMenuWidth    int
	}{
		{
			name:             "nil items",
			items:            nil,
			wantTitleWidth:   0,
			wantShortcutSize: 0,
			wantMenuWidth:    6,
		},
		{
			name:             "empty items",
			items:            []*MenuItem{},
			wantTitleWidth:   0,
			wantShortcutSize: 0,
			wantMenuWidth:    6,
		},
		{
			name: "ascii",
			items: []*MenuItem{
				{Title: "Open", Shortcut: "Ctrl+O"},
				{Title: "Close", Shortcut: ""},
			},
			wantTitleWidth:   5,
			wantShortcutSize: 6,
			wantMenuWidth:    17,
		},
		{
			name: "wide runes and separators",
			items: []*MenuItem{
				{Title: "日本", Shortcut: "Alt+Z"},
				{Title: "", Shortcut: ""},
				{Title: "x", Shortcut: "F1"},
			},
			wantTitleWidth:   4,
			wantShortcutSize: 5,
			wantMenuWidth:    15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			titleWidth, shortcutWidth := measureWidths(tt.items)
			if titleWidth != tt.wantTitleWidth || shortcutWidth != tt.wantShortcutSize {
				t.Errorf("measureWidths() = %d, %d, want %d, %d", titleWidth, shortcutWidth, tt.wantTitleWidth, tt.wantShortcutSize)
			}
			if got := menuWidthInCells(tt.items); got != tt.wantMenuWidth {
				t.Errorf("menuWidthInCells() = %d, want %d", got, tt.wantMenuWidth)
			}
		})
	}
}

func TestNextMenuItem(t *testing.T) {
	items := []*MenuItem{
		{Title: "A"},
		{Title: ""}, // separator
		{Title: "B"},
		{Title: ""}, // separator
		{Title: "C"},
	}

	tests := []struct {
		name          string
		selectedIndex int
		direction     int
		want          int
	}{
		{name: "down over separator", selectedIndex: 0, direction: 1, want: 2},
		{name: "down plain", selectedIndex: 2, direction: 1, want: 4},
		{name: "down wraps", selectedIndex: 4, direction: 1, want: 0},
		{name: "up plain", selectedIndex: 4, direction: -1, want: 2},
		{name: "up over separator", selectedIndex: 2, direction: -1, want: 0},
		{name: "up wraps", selectedIndex: 0, direction: -1, want: 4},
		{name: "down starts on separator", selectedIndex: 1, direction: 1, want: 2},
		{name: "up starts on separator", selectedIndex: 1, direction: -1, want: 0},
		{name: "down skips separator after wrap", selectedIndex: 3, direction: 1, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextMenuItem(items, tt.selectedIndex, tt.direction); got != tt.want {
				t.Errorf("nextMenuItem(items, %d, %d) = %d, want %d", tt.selectedIndex, tt.direction, got, tt.want)
			}
		})
	}
}

func TestInputHandlerNavigation(t *testing.T) {
	menuBar := NewMenuBar()
	menuBar.SetMenus([]*Menu{
		{
			Title:    "File",
			Shortcut: 'f',
			Items: []*MenuItem{
				{ID: "a", Title: "A"},
				{Title: ""},
				{ID: "b", Title: "B"},
				{ID: "c", Title: "C"},
			},
		},
		{
			Title:    "Edit",
			Shortcut: 'e',
			Items: []*MenuItem{
				{ID: "d", Title: "D"},
				{ID: "e", Title: "E"},
			},
		},
	})
	menuBar.Open(0)

	handler := menuBar.InputHandler()
	setFocus := func(p tview.Primitive) {}

	tests := []struct {
		name  string
		event *tcell.EventKey
		want  []int
	}{
		{name: "down skips separator", event: keyEvent(tcell.KeyDown, 0, tcell.ModNone), want: []int{0, 2}},
		{name: "down to last", event: keyEvent(tcell.KeyDown, 0, tcell.ModNone), want: []int{0, 3}},
		{name: "down wraps", event: keyEvent(tcell.KeyDown, 0, tcell.ModNone), want: []int{0, 0}},
		{name: "up wraps", event: keyEvent(tcell.KeyUp, 0, tcell.ModNone), want: []int{0, 3}},
		{name: "up plain", event: keyEvent(tcell.KeyUp, 0, tcell.ModNone), want: []int{0, 2}},
		{name: "up skips separator", event: keyEvent(tcell.KeyUp, 0, tcell.ModNone), want: []int{0, 0}},
		{name: "right moves menu", event: keyEvent(tcell.KeyRight, 0, tcell.ModNone), want: []int{1, 0}},
		{name: "right clamps at end", event: keyEvent(tcell.KeyRight, 0, tcell.ModNone), want: []int{1, 0}},
		{name: "left moves menu", event: keyEvent(tcell.KeyLeft, 0, tcell.ModNone), want: []int{0, 0}},
		{name: "left clamps at start", event: keyEvent(tcell.KeyLeft, 0, tcell.ModNone), want: []int{0, 0}},
		{name: "tab is ignored", event: keyEvent(tcell.KeyTab, 0, tcell.ModNone), want: []int{0, 0}},
		{name: "alt shortcut selects menu", event: keyEvent(tcell.KeyRune, 'e', tcell.ModAlt), want: []int{1, 0}},
		{name: "alt shortcut selects first menu", event: keyEvent(tcell.KeyRune, 'f', tcell.ModAlt), want: []int{0, 0}},
		{name: "escape closes", event: keyEvent(tcell.KeyEscape, 0, tcell.ModNone), want: []int{-1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler(tt.event, setFocus)
			if !reflect.DeepEqual(menuBar.selectedPath, tt.want) {
				t.Errorf("selectedPath = %v, want %v", menuBar.selectedPath, tt.want)
			}
		})
	}
}

func TestInputHandlerEnterExecutesItem(t *testing.T) {
	nextFocus := tview.NewBox()
	var callbackIDs []string
	var closeArgs []tview.Primitive

	menuBar := NewMenuBar()
	menuBar.SetMenus([]*Menu{
		{
			Title: "File",
			Items: []*MenuItem{
				{ID: "open", Title: "Open", Callback: func(id string) tview.Primitive {
					callbackIDs = append(callbackIDs, id)
					return nextFocus
				}},
				{Title: ""},
			},
		},
	})
	menuBar.SetOnClose(func(p tview.Primitive) {
		closeArgs = append(closeArgs, p)
	})
	menuBar.Open(0)

	menuBar.InputHandler()(keyEvent(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})
	if !reflect.DeepEqual(callbackIDs, []string{"open"}) {
		t.Errorf("callback IDs = %v, want [open]", callbackIDs)
	}
	if len(closeArgs) != 1 || closeArgs[0] != nextFocus {
		t.Errorf("OnClose args = %v, want [%v]", closeArgs, nextFocus)
	}
	if !reflect.DeepEqual(menuBar.selectedPath, []int{-1}) {
		t.Errorf("selectedPath = %v, want [-1] after activation", menuBar.selectedPath)
	}

	// Enter on a separator must not execute anything or close the menu.
	separatorCalled := false
	menuBar.SetMenus([]*Menu{{
		Title: "File",
		Items: []*MenuItem{
			{ID: "open", Title: "Open"},
			{ID: "sep", Title: "", Callback: func(string) tview.Primitive { separatorCalled = true; return nil }},
		},
	}})
	menuBar.Open(0)
	menuBar.selectedPath = []int{0, 1}
	menuBar.InputHandler()(keyEvent(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})
	if separatorCalled {
		t.Error("separator callback should not be called")
	}
	if !reflect.DeepEqual(menuBar.selectedPath, []int{0, 1}) {
		t.Errorf("separator Enter changed selectedPath to %v", menuBar.selectedPath)
	}
}

func TestMouseHandlerSelectsMenusAndItems(t *testing.T) {
	screen := newTestScreen(t, 60, 20)

	menus := []*Menu{
		{Title: "File", Items: []*MenuItem{
			{ID: "open", Title: "Open", Shortcut: "Ctrl+O"},
			{ID: "close", Title: "Close", Shortcut: "Ctrl+W"},
		}},
		{Title: "Edit", Items: []*MenuItem{
			{ID: "undo", Title: "Undo", Shortcut: "Ctrl+Z"},
		}},
	}

	var callbackIDs []string
	for _, m := range menus {
		for _, item := range m.Items {
			id := item.ID
			item.Callback = func(string) tview.Primitive {
				callbackIDs = append(callbackIDs, id)
				return nil
			}
		}
	}

	menuBar := NewMenuBar()
	menuBar.SetMenus(menus)
	menuBar.SetRect(0, 0, 60, 1)
	menuBar.Draw(screen)

	if got := menus[0].charWidth; got != 4 {
		t.Fatalf("first menu charWidth = %d, want 4", got)
	}

	var focused []tview.Primitive
	setFocus := func(p tview.Primitive) { focused = append(focused, p) }
	handler := menuBar.MouseHandler()

	// Click the "Edit" title in the menu bar.
	consumed, capture := handler(tview.MouseLeftDown, tcell.NewEventMouse(8, 0, tcell.Button1, tcell.ModNone), setFocus)
	if !consumed || capture != nil {
		t.Errorf("menu bar click returned consumed=%v capture=%v", consumed, capture)
	}
	if !reflect.DeepEqual(menuBar.selectedPath, []int{1, 0}) {
		t.Errorf("after clicking Edit, selectedPath = %v, want [1 0]", menuBar.selectedPath)
	}
	if len(focused) != 1 || focused[0] != tview.Primitive(menuBar) {
		t.Errorf("setFocus calls = %v, want menu bar", focused)
	}

	// Clicking the already-selected title closes the menu.
	closeCount := 0
	menuBar.SetOnClose(func(p tview.Primitive) { closeCount++ })
	handler(tview.MouseLeftDown, tcell.NewEventMouse(8, 0, tcell.Button1, tcell.ModNone), setFocus)
	if closeCount != 1 || !reflect.DeepEqual(menuBar.selectedPath, []int{-1}) {
		t.Errorf("clicking selected menu did not close it: closes=%d path=%v", closeCount, menuBar.selectedPath)
	}

	// Open "File" and click its first item.
	menuBar.Open(0)
	consumed, _ = handler(tview.MouseLeftUp, tcell.NewEventMouse(1, 2, tcell.Button1, tcell.ModNone), setFocus)
	if !consumed {
		t.Error("item click was not consumed")
	}
	if !reflect.DeepEqual(callbackIDs, []string{"open"}) {
		t.Errorf("item callbacks = %v, want [open]", callbackIDs)
	}
	if !reflect.DeepEqual(menuBar.selectedPath, []int{-1}) {
		t.Errorf("menu did not close after item selection: %v", menuBar.selectedPath)
	}

	// A click outside of the dropdown closes it.
	menuBar.Open(0)
	handler(tview.MouseLeftUp, tcell.NewEventMouse(50, 4, tcell.Button1, tcell.ModNone), setFocus)
	if !reflect.DeepEqual(menuBar.selectedPath, []int{-1}) {
		t.Errorf("outside click did not close menu: %v", menuBar.selectedPath)
	}

	// A click below the bar with no menu open is not consumed.
	consumed, capture = handler(tview.MouseLeftDown, tcell.NewEventMouse(1, 5, tcell.Button1, tcell.ModNone), setFocus)
	if consumed || capture != nil {
		t.Errorf("click with no open menu returned consumed=%v capture=%v, want false, nil", consumed, capture)
	}
}

func TestDrawMenuBarAndDropdown(t *testing.T) {
	screen := newTestScreen(t, 80, 20)

	menus := []*Menu{
		{
			Title: "File",
			Items: []*MenuItem{
				{ID: "open", Title: "Open", Shortcut: "Ctrl+O"},
				{Title: ""},
				{ID: "softwrap", Title: "\u2713 Soft Wrap", Shortcut: "Alt+Z"},
			},
		},
		{Title: "Edit", Items: []*MenuItem{
			{ID: "undo", Title: "Undo", Shortcut: "Ctrl+Z"},
		}},
	}

	menuBar := NewMenuBar()
	menuBar.SetMenus(menus)
	menuBar.SetRect(0, 0, 80, 1)
	menuBar.Open(0)
	menuBar.Draw(screen)
	menuBar.AfterDraw()(screen)

	bar := screenRow(screen, 0, 80)
	for _, want := range []string{"File", "Edit"} {
		if !strings.Contains(bar, want) {
			t.Errorf("menu bar row %q does not contain %q", bar, want)
		}
	}

	// The selected first menu uses the reversed menu bar style.
	fg, bg, _ := menuBar.MenuBarStyle.Decompose()
	selectedStyle := menuBar.MenuBarStyle.Foreground(bg).Background(fg)
	assertStyleColors(t, styleAt(screen, 1, 0), selectedStyle, "selected menu bar title")
	assertStyleColors(t, styleAt(screen, 8, 0), menuBar.MenuBarStyle, "unselected menu bar title")

	itemWidth := menuWidthInCells(menus[0].Items)
	if got := contentAt(screen, 0, 1); got != '┌' {
		t.Errorf("top-left border = %q, want ┌", got)
	}
	if got := contentAt(screen, itemWidth-1, 1); got != '┐' {
		t.Errorf("top-right border = %q, want ┐", got)
	}
	if got := contentAt(screen, 0, 5); got != '└' {
		t.Errorf("bottom-left border = %q, want └", got)
	}
	if got := contentAt(screen, itemWidth-1, 5); got != '┘' {
		t.Errorf("bottom-right border = %q, want ┘", got)
	}
	if got := contentAt(screen, 0, 3); got != '├' {
		t.Errorf("separator left = %q, want ├", got)
	}
	if got := contentAt(screen, itemWidth-1, 3); got != '┤' {
		t.Errorf("separator right = %q, want ┤", got)
	}

	openRow := screenRow(screen, 2, 80)
	if !strings.Contains(openRow, "Open") || !strings.Contains(openRow, "Ctrl+O") {
		t.Errorf("first item row %q does not contain title and shortcut", openRow)
	}
	checkedRow := screenRow(screen, 4, 80)
	if !strings.Contains(checkedRow, "\u2713 Soft Wrap") || !strings.Contains(checkedRow, "Alt+Z") {
		t.Errorf("checked item row %q does not contain title and shortcut", checkedRow)
	}

	// The selected item is highlighted, the other item is not.
	assertStyleColors(t, styleAt(screen, 2, 2), menuBar.MenuSelectedStyle, "selected menu item")
	assertStyleColors(t, styleAt(screen, 2, 4), menuBar.MenuStyle, "unselected menu item")
}

func TestAfterDrawDoesNothingWhenClosed(t *testing.T) {
	screen := newTestScreen(t, 40, 10)

	menuBar := NewMenuBar()
	menuBar.SetMenus([]*Menu{{Title: "File", Items: []*MenuItem{{Title: "Open"}}}})
	menuBar.SetRect(0, 0, 40, 1)
	screen.SetContent(0, 1, 'X', nil, tcell.StyleDefault)
	menuBar.AfterDraw()(screen)

	if got := contentAt(screen, 0, 1); got != 'X' {
		t.Errorf("AfterDraw drew %q while the menu bar was closed", got)
	}
}
