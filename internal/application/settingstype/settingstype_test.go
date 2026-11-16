package settingstype

import (
	"reflect"
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	got := DefaultSettings()
	if got.TabSize != 4 {
		t.Errorf("TabSize = %d, want 4", got.TabSize)
	}
	if got.TabCharacter != "space" {
		t.Errorf("TabCharacter = %q, want space", got.TabCharacter)
	}
	if got.ColorScheme != "default" {
		t.Errorf("ColorScheme = %q, want default", got.ColorScheme)
	}
	if got.VerticalRuler != 0 {
		t.Errorf("VerticalRuler = %v, want 0", got.VerticalRuler)
	}
}

func TestSettingsZeroValue(t *testing.T) {
	var s Settings
	if !reflect.DeepEqual(s, Settings{}) {
		t.Errorf("zero value mismatch")
	}
}
