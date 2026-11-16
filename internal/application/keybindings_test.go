package application

import (
	"testing"

	"github.com/sedwards2009/smidgen"
)

func TestInitKeyBindingsPopulatesMappings(t *testing.T) {
	initKeyBindings()

	if len(smidgenKeyToActionMapping) == 0 {
		t.Error("smidgenKeyToActionMapping is empty")
	}
	if len(smidgenSingleLineKeyToActionMapping) == 0 {
		t.Error("smidgenSingleLineKeyToActionMapping is empty")
	}
	if len(dinkyKeyToActionMapping) == 0 {
		t.Error("dinkyKeyToActionMapping is empty")
	}
	if len(actionToKeyMapping) == 0 {
		t.Error("actionToKeyMapping is empty")
	}
	if len(dinkyKeyBindings) == 0 {
		t.Error("dinkyKeyBindings is empty")
	}
}

func TestSmidgenKeyMappingsContainExpectedEntries(t *testing.T) {
	initKeyBindings()

	tests := []struct {
		key    string
		action string
	}{
		{"Up", smidgen.ActionCursorUp},
		{"Down", smidgen.ActionCursorDown},
		{"PageUp", smidgen.ActionCursorPageUp},
		{"PageDown", smidgen.ActionCursorPageDown},
		{"Insert", smidgen.ActionToggleOverwriteMode},
		{"F11", smidgen.ActionToggleBookmark},
		{"Ctrl-j", smidgen.ActionJumpToMatchingBrace},
	}

	for _, tt := range tests {
		if got, ok := smidgenKeyToActionMapping[tt.key]; !ok {
			t.Errorf("smidgenKeyToActionMapping[%q] missing", tt.key)
		} else if got != tt.action {
			t.Errorf("smidgenKeyToActionMapping[%q] = %q, want %q", tt.key, got, tt.action)
		}
	}

	// Compound actions are stored as comma/pipe separated action names.
	if got := smidgenKeyToActionMapping["Tab"]; got != smidgen.ActionIndentSelection+","+smidgen.ActionInsertTab {
		t.Errorf("Tab mapping = %q, want indent+insert tab", got)
	}
	if got := smidgenKeyToActionMapping["Esc"]; got != smidgen.ActionEscape+","+smidgen.ActionRemoveAllMultiCursors {
		t.Errorf("Esc mapping = %q, want escape+remove cursors", got)
	}

	// The single-line bindings must not include vertical movement.
	if _, ok := smidgenSingleLineKeyToActionMapping["Up"]; ok {
		t.Error("single-line bindings should not bind Up")
	}
	if got := smidgenSingleLineKeyToActionMapping["Enter"]; got != smidgen.ActionInsertNewline {
		t.Errorf("single-line Enter mapping = %q, want %q", got, smidgen.ActionInsertNewline)
	}
}

func TestDinkyActionMappingCoversEveryAction(t *testing.T) {
	// Keep this list in sync with the ACTION_* constants in actions.go.
	allActions := []string{
		ACTION_NEW,
		ACTION_CLOSE_FILE,
		ACTION_OPEN_FILE,
		ACTION_SAVE_FILE,
		ACTION_SAVE_FILE_AS,
		ACTION_OPEN_FILE_MENU,
		ACTION_OPEN_EDIT_MENU,
		ACTION_OPEN_SELECTION_MENU,
		ACTION_OPEN_GO_MENU,
		ACTION_OPEN_TRANSFORM_MENU,
		ACTION_OPEN_VIEW_MENU,
		ACTION_OPEN_HELP_MENU,
		ACTION_TOGGLE_SOFT_WRAP,
		ACTION_TOGGLE_MATCH_BRACKET,
		ACTION_SET_TAB_SIZE,
		ACTION_SET_TAB_CHARACTER,
		ACTION_SET_LINE_ENDINGS,
		ACTION_SET_SYNTAX_HIGHLIGHTING,
		ACTION_SET_VERTICAL_RULER,
		ACTION_GO_TO_LINE,
		ACTION_QUIT,
		ACTION_FIND,
		ACTION_FIND_NEXT,
		ACTION_FIND_PREVIOUS,
		ACTION_ABOUT,
		ACTION_NEXT_EDITOR,
		ACTION_PREVIOUS_EDITOR,
		ACTION_CONVERT_TAB_SPACES,
		ACTION_TOGGLE_WHITESPACE,
		ACTION_TOGGLE_TRAILING_WHITESPACE,
		ACTION_FIND_AND_REPLACE,
		ACTION_SETTINGS,
		ACTION_SUSPEND,
		ACTION_TO_UPPERCASE,
		ACTION_TO_LOWERCASE,
		ACTION_URL_ENCODE,
		ACTION_URL_DECODE,
		ACTION_SORT_LINES,
		ACTION_REVERSE_LINES,
		ACTION_FORMAT_JSON,
		ACTION_HARD_WORD_WRAP,
		ACTION_NEXT_BOOKMARK,
		ACTION_PREVIOUS_BOOKMARK,
		ACTION_FILTER_EXTERNAL_COMMAND,
	}

	if len(allActions) != len(dinkyActionMapping) {
		t.Errorf("dinkyActionMapping has %d entries, action list has %d", len(dinkyActionMapping), len(allActions))
	}
	for _, action := range allActions {
		if _, ok := dinkyActionMapping[action]; !ok {
			t.Errorf("dinkyActionMapping is missing %q", action)
		}
	}
}

func TestDinkyKeyBindingsParseToActions(t *testing.T) {
	initKeyBindings()

	tests := []struct {
		keySequence string
		wantAction  string
	}{
		{"Ctrl-n", ACTION_NEW},
		{"Ctrl-w", ACTION_CLOSE_FILE},
		{"Ctrl-o", ACTION_OPEN_FILE},
		{"Ctrl-s", ACTION_SAVE_FILE},
		{"Ctrl-g", ACTION_GO_TO_LINE},
		{"Ctrl-q", ACTION_QUIT},
		{"Ctrl-f", ACTION_FIND},
		{"Ctrl-r", ACTION_FIND_AND_REPLACE},
		{"F2", ACTION_FIND_PREVIOUS},
		{"F3", ACTION_FIND_NEXT},
		{"F9", ACTION_NEXT_EDITOR},
		{"F10", ACTION_PREVIOUS_EDITOR},
		{"F12", ACTION_OPEN_FILE_MENU},
		{"Alt-f", ACTION_OPEN_FILE_MENU},
		{"Alt-e", ACTION_OPEN_EDIT_MENU},
		{"Alt-s", ACTION_OPEN_SELECTION_MENU},
		{"Alt-g", ACTION_OPEN_GO_MENU},
		{"Alt-t", ACTION_OPEN_TRANSFORM_MENU},
		{"Alt-v", ACTION_OPEN_VIEW_MENU},
		{"Alt-h", ACTION_OPEN_HELP_MENU},
		{"Alt-PgDown", ACTION_NEXT_BOOKMARK},
		{"Alt-PgUp", ACTION_PREVIOUS_BOOKMARK},
	}

	if len(dinkyKeyBindings) != len(dinkyKeyToActionMapping) {
		t.Errorf("%d key bindings parsed, want %d", len(dinkyKeyBindings), len(dinkyKeyToActionMapping))
	}

	for _, tt := range tests {
		desc, ok := smidgen.ParseKeySequence(tt.keySequence)
		if !ok {
			t.Fatalf("test key sequence %q failed to parse", tt.keySequence)
		}
		got, ok := dinkyKeyBindings[desc]
		if !ok {
			t.Errorf("dinkyKeyBindings is missing %q", tt.keySequence)
			continue
		}
		if got != tt.wantAction {
			t.Errorf("dinkyKeyBindings[%q] = %q, want %q", tt.keySequence, got, tt.wantAction)
		}
	}
}

func TestActionToKeyMappingIncludesDinkyShortcuts(t *testing.T) {
	initKeyBindings()

	tests := []struct {
		action string
		key    string
	}{
		{ACTION_NEW, "Ctrl-n"},
		{ACTION_SAVE_FILE, "Ctrl-s"},
		{ACTION_QUIT, "Ctrl-q"},
		{ACTION_FIND, "Ctrl-f"},
		{ACTION_FIND_NEXT, "F3"},
		{ACTION_FIND_PREVIOUS, "F2"},
		{ACTION_NEXT_EDITOR, "F9"},
		{ACTION_PREVIOUS_EDITOR, "F10"},
		{ACTION_FIND_AND_REPLACE, "Ctrl-r"},
	}

	for _, tt := range tests {
		if got := actionToKeyMapping[tt.action]; got != tt.key {
			t.Errorf("actionToKeyMapping[%q] = %q, want %q", tt.action, got, tt.key)
		}
	}
}

func TestSmidgenKeybindingsAreBuilt(t *testing.T) {
	initKeyBindings()

	if smidgenDefaultKeyBindings.KeyTree == nil {
		t.Error("smidgenDefaultKeyBindings has a nil key tree")
	}
	if smidgenSingleLineKeyBindings.KeyTree == nil {
		t.Error("smidgenSingleLineKeyBindings has a nil key tree")
	}
}
