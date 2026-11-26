package application

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"dinky/internal/application/settingstype"

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

func TestDefaultSettingsEnsureFinalNewline(t *testing.T) {
	if !settingstype.DefaultSettings().EnsureFinalNewline {
		t.Error("DefaultSettings().EnsureFinalNewline should be true")
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
