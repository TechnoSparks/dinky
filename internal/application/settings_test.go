package application

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"dinky/internal/application/settingstype"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sedwards2009/smidgen"
	"github.com/sedwards2009/smidgen/micro/clipboard"
)

func TestCleanClipboard(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{clipboard.ModeTerminal, clipboard.ModeTerminal},
		{clipboard.ModeExternal, clipboard.ModeExternal},
		{clipboard.ModeInternal, clipboard.ModeInternal},
		{"", clipboard.ModeTerminal},
		{"banana", clipboard.ModeTerminal},
	}

	for _, tt := range tests {
		if got := cleanClipboard(tt.in); got != tt.want {
			t.Errorf("cleanClipboard(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCleanTabSize(t *testing.T) {
	tests := []struct {
		in   int
		want int
	}{
		{2, 2},
		{4, 4},
		{8, 8},
		{16, 16},
		{0, 4},
		{-1, 4},
		{3, 4},
		{100, 4},
	}

	for _, tt := range tests {
		if got := cleanTabSize(tt.in); got != tt.want {
			t.Errorf("cleanTabSize(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestCleanTabCharacter(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"tab", "tab"},
		{"space", "space"},
		{"", "space"},
		{"spaces", "space"},
		{"TAB", "space"},
	}

	for _, tt := range tests {
		if got := cleanTabCharacter(tt.in); got != tt.want {
			t.Errorf("cleanTabCharacter(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCleanColorSchemeName(t *testing.T) {
	schemes := smidgen.ListColorschemes()
	if len(schemes) == 0 {
		t.Fatal("no internal colorschemes available")
	}

	if got := cleanColorSchemeName(""); got != "default" {
		t.Errorf("cleanColorSchemeName(%q) = %q, want %q", "", got, "default")
	}
	if got := cleanColorSchemeName("this-colorscheme-does-not-exist"); got != "default" {
		t.Errorf("unknown colorscheme was not replaced by default, got %q", got)
	}
	if got := cleanColorSchemeName(schemes[0]); got != schemes[0] {
		t.Errorf("cleanColorSchemeName(%q) = %q, want unchanged", schemes[0], got)
	}
}

func TestCleanColorSchemeNameKeepsUserColorscheme(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mine.micro"), []byte(`color-link default "#112233,#445566"`), 0644); err != nil {
		t.Fatal(err)
	}
	smidgen.SetUserColorschemeDir(dir)
	t.Cleanup(func() {
		smidgen.SetUserColorschemeDir("")
	})

	if got := cleanColorSchemeName("mine"); got != "mine" {
		t.Errorf("cleanColorSchemeName(%q) = %q, want unchanged", "mine", got)
	}
}

func TestUserColorschemesDirPathUsesConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	want := filepath.Join(configDir, "dinky", "colorschemes")
	if got := userColorschemesDirPath(); got != want {
		t.Errorf("userColorschemesDirPath() = %q, want %q", got, want)
	}
}

func TestDefaultSettingsEnsureFinalNewline(t *testing.T) {
	if !settingstype.DefaultSettings().EnsureFinalNewline {
		t.Error("DefaultSettings().EnsureFinalNewline should be true")
	}
}

func TestDefaultSettingsCopyOnSelectOff(t *testing.T) {
	if settingstype.DefaultSettings().CopyOnSelect {
		t.Error("DefaultSettings().CopyOnSelect should be false")
	}
}

func TestLoadSettingsMissingFileReturnsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")
	got := loadSettings(path)
	if want := settingstype.DefaultSettings(); got != want {
		t.Errorf("loadSettings(missing) = %+v, want defaults %+v", got, want)
	}
}

func TestLoadSettingsInvalidJSONReturnsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}

	got := loadSettings(path)
	if want := settingstype.DefaultSettings(); got != want {
		t.Errorf("loadSettings(invalid) = %+v, want defaults %+v", got, want)
	}
}

func TestLoadSettingsCleansInvalidValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	data := `{
		"showWhitespace": true,
		"showLineNumbers": false,
		"showMatchBracket": false,
		"softWrap": true,
		"tabSize": 7,
		"tabCharacter": "banana",
		"colorScheme": "no-such-scheme",
		"showTrailingWhitespace": false,
		"verticalRuler": 33.5,
		"clipboard": "banana"
	}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	got := loadSettings(path)

	if got.TabSize != 4 {
		t.Errorf("TabSize = %d, want cleaned value 4", got.TabSize)
	}
	if got.TabCharacter != "space" {
		t.Errorf("TabCharacter = %q, want cleaned value %q", got.TabCharacter, "space")
	}
	if got.ColorScheme != "default" {
		t.Errorf("ColorScheme = %q, want cleaned value %q", got.ColorScheme, "default")
	}
	if got.Clipboard != clipboard.ModeTerminal {
		t.Errorf("Clipboard = %q, want cleaned value %q", got.Clipboard, clipboard.ModeTerminal)
	}

	// Values that need no cleaning must be preserved.
	if !got.ShowWhitespace || got.ShowLineNumbers || got.ShowMatchBracket || !got.SoftWrap || got.ShowTrailingWhitespace {
		t.Errorf("boolean settings were not preserved: %+v", got)
	}
	if got.VerticalRuler != 33.5 {
		t.Errorf("VerticalRuler = %v, want 33.5", got.VerticalRuler)
	}
}

func TestLoadSettingsKeepsDefaultsForMissingFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"softWrap": true}`), 0644); err != nil {
		t.Fatal(err)
	}

	got := loadSettings(path)
	want := settingstype.DefaultSettings()
	want.SoftWrap = true
	if got != want {
		t.Errorf("loadSettings(partial) = %+v, want %+v", got, want)
	}
}

func TestSaveAndLoadSettingsRoundTrip(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("HOME", t.TempDir())

	want := settingstype.Settings{
		ShowWhitespace:         true,
		ShowLineNumbers:        false,
		ShowMatchBracket:       false,
		SoftWrap:               true,
		TabSize:                8,
		TabCharacter:           "tab",
		ColorScheme:            "default",
		ShowTrailingWhitespace: false,
		VerticalRuler:          12.5,
		EnsureFinalNewline:     true,
		Clipboard:              clipboard.ModeInternal,
		CopyOnSelect:           true,
	}

	if err := SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings returned error: %v", err)
	}

	settingsPath := filepath.Join(configDir, "dinky", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings file was not written to %s: %v", settingsPath, err)
	}
	var decoded settingstype.Settings
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("settings file is not valid JSON: %v", err)
	}

	got := LoadUserSettings()
	if got != want {
		t.Errorf("LoadUserSettings() = %+v, want %+v", got, want)
	}
}

func TestUserSettingsPathUsesConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	want := filepath.Join(configDir, "dinky", "settings.json")
	if got := userSettingsPath(); got != want {
		t.Errorf("userSettingsPath() = %q, want %q", got, want)
	}
}

func TestSettingsOKAppliesClipboardMode(t *testing.T) {
	newTestHarness(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	previousMode := clipboard.Mode()
	t.Cleanup(func() {
		clipboard.SetMode(previousMode)
	})
	clipboard.SetMode(clipboard.ModeTerminal)

	handleSettings()
	internalIndex := slices.Index(settingstype.ClipboardModes, clipboard.ModeInternal)
	settingsDialog.ClipboardList.SetCurrentItem(internalIndex)
	settingsDialog.OkButton.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})

	if got := clipboard.Mode(); got != clipboard.ModeInternal {
		t.Errorf("clipboard mode after OK = %q, want %q", got, clipboard.ModeInternal)
	}
	if settings.Clipboard != clipboard.ModeInternal {
		t.Errorf("settings.Clipboard = %q, want %q", settings.Clipboard, clipboard.ModeInternal)
	}
}

func TestSettingsOKAppliesCopyOnSelect(t *testing.T) {
	newTestHarness(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	previousCopyOnSelect := clipboard.CopyOnSelect()
	t.Cleanup(func() {
		clipboard.SetCopyOnSelect(previousCopyOnSelect)
	})

	copyOnSelectValues := []bool{true, false}
	for _, copyOnSelect := range copyOnSelectValues {
		clipboard.SetCopyOnSelect(!copyOnSelect)

		handleSettings()
		settingsDialog.CopyOnSelectCheckbox.SetChecked(copyOnSelect)
		settingsDialog.OkButton.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})

		if got := clipboard.CopyOnSelect(); got != copyOnSelect {
			t.Errorf("copy on select after OK = %v, want %v", got, copyOnSelect)
		}
		if settings.CopyOnSelect != copyOnSelect {
			t.Errorf("settings.CopyOnSelect = %v, want %v", settings.CopyOnSelect, copyOnSelect)
		}
	}
}

func TestLoadEditorColorSchemeAppliesUserColorscheme(t *testing.T) {
	h := newTestHarness(t)
	dir := t.TempDir()
	text := `color-link default "#112233,#445566"`
	if err := os.WriteFile(filepath.Join(dir, "mine.micro"), []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	smidgen.SetUserColorschemeDir(dir)
	t.Cleanup(func() {
		smidgen.SetUserColorschemeDir("")
	})
	fb := h.open("hello", "")
	wantForeground := tcell.GetColor("#112233")
	wantBackground := tcell.GetColor("#445566")

	loadEditorColorScheme("mine")

	_, editorBackground, _ := colorscheme.GetColor("default").Decompose()
	if editorBackground != wantBackground {
		t.Errorf("editor background = %v, want %v", editorBackground, wantBackground)
	}
	tabForeground, tabBackground, _ := tabBarLine.ActiveTabStyle.Decompose()
	if tabForeground != wantForeground || tabBackground != wantBackground {
		t.Errorf("active tab = %v on %v, want %v on %v", tabForeground, tabBackground, wantForeground, wantBackground)
	}
	if got := fb.hScrollbar.Track.GetBackgroundColor(); got != wantBackground {
		t.Errorf("scrollbar background = %v, want %v", got, wantBackground)
	}
}

func TestLoadEditorColorSchemeUnknownNameFallsBackToMonokai(t *testing.T) {
	newTestHarness(t)
	monokai, _ := smidgen.LoadInternalColorscheme("monokai")
	_, wantBackground, _ := monokai.GetColor("default").Decompose()

	loadEditorColorScheme("no-such-scheme")

	if _, got, _ := colorscheme.GetColor("default").Decompose(); got != wantBackground {
		t.Errorf("editor background = %v, want monokai's %v", got, wantBackground)
	}
}

// useUserColorschemes points smidgen at a temporary user colour scheme folder holding the given files.
func useUserColorschemes(t *testing.T, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(dir, name+".micro"), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	smidgen.SetUserColorschemeDir(dir)
	t.Cleanup(func() {
		smidgen.SetUserColorschemeDir("")
	})
}

func TestColorschemeWarning(t *testing.T) {
	tests := []struct {
		problems []string
		want     string
	}{
		{problems: nil, want: ""},
		{problems: []string{}, want: ""},
		{problems: []string{"line 7: bad"}, want: "mine.micro: line 7: bad"},
		{problems: []string{"line 7: bad", "line 9: bad"}, want: "mine.micro: line 7: bad (and 1 more)"},
		{problems: []string{"line 7: bad", "line 9: bad", "line 12: bad"}, want: "mine.micro: line 7: bad (and 2 more)"},
	}
	for _, tt := range tests {
		if got := colorschemeWarning("mine", tt.problems); got != tt.want {
			t.Errorf("colorschemeWarning(%q) = %q, want %q", tt.problems, got, tt.want)
		}
	}
}

func TestLoadEditorColorSchemeReturnsWarningForBrokenFile(t *testing.T) {
	newTestHarness(t)
	useUserColorschemes(t, map[string]string{
		"broken": "color-link default \"#112233,#445566\"\noops\nalso oops\n",
		"fine":   "color-link default \"#112233,#445566\"\n",
	})

	tests := []struct {
		name string
		want string
	}{
		{name: "broken", want: "broken.micro: line 2: not a valid color-link statement (and 1 more)"},
		{name: "fine", want: ""},
		{name: "monokai", want: ""},
		{name: "no-such-scheme", want: ""},
	}
	for _, tt := range tests {
		if got := loadEditorColorScheme(tt.name); got != tt.want {
			t.Errorf("loadEditorColorScheme(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}

	// The valid lines of the broken file still apply.
	loadEditorColorScheme("broken")
	if _, bg, _ := colorscheme.GetColor("default").Decompose(); bg != tcell.GetColor("#445566") {
		t.Errorf("broken file background = %v, want #445566", bg)
	}
}

func TestSettingsOKShowsColorschemeWarning(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	useUserColorschemes(t, map[string]string{
		"broken": "oops\n",
		"fine":   "color-link default \"#112233,#445566\"\n",
	})

	tests := []struct {
		scheme      string
		wantWarning bool
	}{
		{scheme: "broken", wantWarning: true},
		{scheme: "fine", wantWarning: false},
	}
	for _, tt := range tests {
		t.Run(tt.scheme, func(t *testing.T) {
			h := newTestHarness(t)
			settings.ColorScheme = tt.scheme

			handleSettings()
			settingsDialog.OkButton.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})

			got := strings.Contains(h.statusBarText(), "broken.micro: line 1: not a valid color-link statement")
			if got != tt.wantWarning {
				t.Errorf("warning shown = %v, want %v", got, tt.wantWarning)
			}
		})
	}
}
