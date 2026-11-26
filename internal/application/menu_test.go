package application

import (
	"testing"

	"dinky/internal/tui/menu"

	"github.com/sedwards2009/smidgen"
)

func TestCreateMenusStructure(t *testing.T) {
	menus := createMenus()

	wantTitles := []string{
		"[::u]F[::U]ile",
		"[::u]E[::U]dit",
		"[::u]S[::U]election",
		"[::u]G[::U]o",
		"[::u]T[::U]ransform",
		"[::u]V[::U]iew",
		"[::u]H[::U]elp",
	}
	wantShortcuts := []rune{'f', 'e', 's', 'g', 't', 'v', 'h'}
	wantItemCounts := []int{11, 15, 8, 11, 13, 8, 3}

	if len(menus) != len(wantTitles) {
		t.Fatalf("createMenus returned %d menus, want %d", len(menus), len(wantTitles))
	}

	seen := map[string]bool{}
	for i, m := range menus {
		if m.Title != wantTitles[i] {
			t.Errorf("menu %d title = %q, want %q", i, m.Title, wantTitles[i])
		}
		if m.Shortcut != wantShortcuts[i] {
			t.Errorf("menu %d shortcut = %q, want %q", i, m.Shortcut, wantShortcuts[i])
		}
		if len(m.Items) != wantItemCounts[i] {
			t.Errorf("menu %q has %d items, want %d", m.Title, len(m.Items), wantItemCounts[i])
		}

		for j, item := range m.Items {
			if item.Title == "" {
				if item.ID != "" {
					t.Errorf("menu %q item %d is a separator but has ID %q", m.Title, j, item.ID)
				}
				if item.Callback != nil {
					t.Errorf("menu %q item %d is a separator but has a callback", m.Title, j)
				}
				continue
			}
			if item.ID == "" {
				t.Errorf("menu %q item %d (%q) has no ID", m.Title, j, item.Title)
			}
			if item.Callback == nil {
				t.Errorf("menu %q item %d (%q) has no callback", m.Title, j, item.Title)
			}
			if seen[item.ID] {
				t.Errorf("menu item ID %q appears more than once", item.ID)
			}
			seen[item.ID] = true
		}
	}
}

func TestSyncMenuKeyBindings(t *testing.T) {
	menuList := createMenus()
	syncMenuKeyBindings(menuList, map[string]string{
		ACTION_NEW:    "Ctrl-n",
		ACTION_FIND:   "Ctrl-f",
		"NotAnAction": "Ctrl-z",
	})

	find := func(list []*menu.Menu, id string) *menu.MenuItem {
		for _, m := range list {
			for _, item := range m.Items {
				if item.ID == id {
					return item
				}
			}
		}
		return nil
	}

	if got := find(menuList, ACTION_NEW).Shortcut; got != "Ctrl-n" {
		t.Errorf("New shortcut = %q, want Ctrl-n", got)
	}
	if got := find(menuList, ACTION_FIND).Shortcut; got != "Ctrl-f" {
		t.Errorf("Find shortcut = %q, want Ctrl-f", got)
	}
	if got := find(menuList, ACTION_SAVE_FILE).Shortcut; got != "" {
		t.Errorf("Save shortcut = %q, want empty when not in mapping", got)
	}
}

func TestSyncToggleMenuItem(t *testing.T) {
	menuList := createMenus()

	syncToggleMenuItem(menuList, ACTION_TOGGLE_SOFT_WRAP, "Soft Wrap", true)
	if got := menuItemByID(menuList, ACTION_TOGGLE_SOFT_WRAP).Title; got != "\u2713 Soft Wrap" {
		t.Errorf("checked title = %q, want %q", got, "\u2713 Soft Wrap")
	}

	syncToggleMenuItem(menuList, ACTION_TOGGLE_SOFT_WRAP, "Soft Wrap", false)
	if got := menuItemByID(menuList, ACTION_TOGGLE_SOFT_WRAP).Title; got != "  Soft Wrap" {
		t.Errorf("unchecked title = %q, want %q", got, "  Soft Wrap")
	}
}

func TestSyncMenuFromBuffer(t *testing.T) {
	h := newTestHarness(t)
	h.open("hello", "example.go")
	buf := currentFileBuffer.buffer

	buf.Settings["softwrap"] = true
	buf.Settings["ruler"] = false
	buf.Settings["matchbrace"] = false
	buf.Settings["showwhitespace"] = true
	buf.Settings["hltrailingws"] = false
	buf.Settings["tabsize"] = float64(8)
	buf.Settings["tabstospaces"] = true
	buf.Settings["fileformat"] = "dos"
	buf.Settings["colorcolumn"] = float64(42)

	syncMenuFromBuffer(buf)

	tests := []struct {
		id   string
		want string
	}{
		{ACTION_TOGGLE_SOFT_WRAP, "\u2713 Soft Wrap"},
		{smidgen.ActionToggleRuler, "  Line Numbers"},
		{ACTION_TOGGLE_MATCH_BRACKET, "  Match Brackets"},
		{ACTION_TOGGLE_WHITESPACE, "\u2713 Show Whitespace"},
		{ACTION_TOGGLE_TRAILING_WHITESPACE, "  Show Trailing Whitespace"},
		{ACTION_SET_TAB_SIZE, "Tab Size (8)\u2026"},
		{ACTION_SET_TAB_CHARACTER, "Tab Character (Space)\u2026"},
		{ACTION_CONVERT_TAB_SPACES, "Convert All Tabs to Spaces"},
		{ACTION_SET_LINE_ENDINGS, "Line Endings (CRLF)\u2026"},
		{ACTION_SET_SYNTAX_HIGHLIGHTING, "Syntax (Go)\u2026"},
		{ACTION_SET_VERTICAL_RULER, "Vertical Ruler (42)\u2026"},
	}

	for _, tt := range tests {
		item := menuItemByID(menus, tt.id)
		if item == nil {
			t.Errorf("menu item %q not found", tt.id)
			continue
		}
		if item.Title != tt.want {
			t.Errorf("menu item %q title = %q, want %q", tt.id, item.Title, tt.want)
		}
	}
}

func TestSyncLineEndings(t *testing.T) {
	menuList := createMenus()
	buf := newTestHarness(t).open("x", "").buffer

	buf.Settings["fileformat"] = "unix"
	syncLineEndings(menuList, buf)
	if got := menuItemByID(menuList, ACTION_SET_LINE_ENDINGS).Title; got != "Line Endings (LF)\u2026" {
		t.Errorf("unix line endings menu = %q", got)
	}

	buf.Settings["fileformat"] = "dos"
	syncLineEndings(menuList, buf)
	if got := menuItemByID(menuList, ACTION_SET_LINE_ENDINGS).Title; got != "Line Endings (CRLF)\u2026" {
		t.Errorf("dos line endings menu = %q", got)
	}
}

func TestSyncTabCharacter(t *testing.T) {
	menuList := createMenus()
	buf := newTestHarness(t).open("x", "").buffer

	buf.Settings["tabstospaces"] = false
	syncTabCharacter(menuList, buf)
	if got := menuItemByID(menuList, ACTION_SET_TAB_CHARACTER).Title; got != "Tab Character (Tab)\u2026" {
		t.Errorf("tab character menu = %q", got)
	}
	if got := menuItemByID(menuList, ACTION_CONVERT_TAB_SPACES).Title; got != "Convert All Spaces to Tabs" {
		t.Errorf("convert menu = %q", got)
	}

	buf.Settings["tabstospaces"] = true
	syncTabCharacter(menuList, buf)
	if got := menuItemByID(menuList, ACTION_SET_TAB_CHARACTER).Title; got != "Tab Character (Space)\u2026" {
		t.Errorf("space character menu = %q", got)
	}
	if got := menuItemByID(menuList, ACTION_CONVERT_TAB_SPACES).Title; got != "Convert All Tabs to Spaces" {
		t.Errorf("convert menu = %q", got)
	}

	// A nil buffer resets the tab character label without touching convert.
	menuItemByID(menuList, ACTION_SET_TAB_CHARACTER).Title = "stale"
	syncTabCharacter(menuList, nil)
	if got := menuItemByID(menuList, ACTION_SET_TAB_CHARACTER).Title; got != "Tab Character\u2026" {
		t.Errorf("nil buffer tab character menu = %q", got)
	}
}

func TestSyncTabSize(t *testing.T) {
	menuList := createMenus()

	for _, size := range []int{2, 4, 8} {
		syncTabSize(menuList, size)
		want := "Tab Size (" + string(rune('0'+size)) + ")\u2026"
		if got := menuItemByID(menuList, ACTION_SET_TAB_SIZE).Title; got != want {
			t.Errorf("syncTabSize(%d) title = %q, want %q", size, got, want)
		}
	}
}

func TestSyncSyntaxHighlighting(t *testing.T) {
	menuList := createMenus()

	// Nil buffer falls back to the generic label.
	syncSyntaxHighlighting(menuList, nil)
	if got := menuItemByID(menuList, ACTION_SET_SYNTAX_HIGHLIGHTING).Title; got != "Syntax\u2026" {
		t.Errorf("nil buffer syntax menu = %q", got)
	}

	buf := newTestHarness(t).open("x", "notes.unknownextension").buffer
	buf.Settings["filetype"] = "Unknown"
	syncSyntaxHighlighting(menuList, buf)
	if got := menuItemByID(menuList, ACTION_SET_SYNTAX_HIGHLIGHTING).Title; got != "Syntax (Auto)\u2026" {
		t.Errorf("auto syntax menu = %q", got)
	}

	buf.Settings["filetype"] = "go"
	syncSyntaxHighlighting(menuList, buf)
	if got := menuItemByID(menuList, ACTION_SET_SYNTAX_HIGHLIGHTING).Title; got != "Syntax (Go)\u2026" {
		t.Errorf("explicit syntax menu = %q", got)
	}
}

func TestSyncVerticalRuler(t *testing.T) {
	menuList := createMenus()

	syncVerticalRuler(menuList, 0)
	if got := menuItemByID(menuList, ACTION_SET_VERTICAL_RULER).Title; got != "Vertical Ruler (Off)\u2026" {
		t.Errorf("off ruler menu = %q", got)
	}

	syncVerticalRuler(menuList, 80)
	if got := menuItemByID(menuList, ACTION_SET_VERTICAL_RULER).Title; got != "Vertical Ruler (80)\u2026" {
		t.Errorf("on ruler menu = %q", got)
	}
}

func menuItemByID(menuList []*menu.Menu, id string) *menu.MenuItem {
	for _, m := range menuList {
		for _, item := range m.Items {
			if item.ID == id {
				return item
			}
		}
	}
	return nil
}
