package settingstype

import "github.com/sedwards2009/smidgen/micro/clipboard"

// ClipboardModes lists the clipboard modes in the order the Settings dialog shows them.
var ClipboardModes = []string{clipboard.ModeTerminal, clipboard.ModeExternal, clipboard.ModeInternal}

type Settings struct {
	ShowWhitespace         bool    `json:"showWhitespace"`
	ShowLineNumbers        bool    `json:"showLineNumbers"`
	ShowMatchBracket       bool    `json:"showMatchBracket"`
	SoftWrap               bool    `json:"softWrap"`
	TabSize                int     `json:"tabSize"`
	TabCharacter           string  `json:"tabCharacter"` // "tab" or "space"
	ColorScheme            string  `json:"colorScheme"`
	ShowTrailingWhitespace bool    `json:"showTrailingWhitespace"`
	VerticalRuler          float64 `json:"verticalRuler"` // 0.0 means off
	EnsureFinalNewline     bool    `json:"ensureFinalNewline"`
	Clipboard              string  `json:"clipboard"` // one of ClipboardModes
}

func DefaultSettings() Settings {
	return Settings{
		ShowWhitespace:         false,
		ShowLineNumbers:        true,
		ShowMatchBracket:       true,
		SoftWrap:               false,
		TabSize:                4,
		TabCharacter:           "space",
		ColorScheme:            "default",
		ShowTrailingWhitespace: true,
		VerticalRuler:          0.0,
		EnsureFinalNewline:     true,
		Clipboard:              clipboard.ModeTerminal,
	}
}
