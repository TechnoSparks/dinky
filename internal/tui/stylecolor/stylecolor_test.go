package stylecolor

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestPaletteColors(t *testing.T) {
	tests := []struct {
		name string
		got  tcell.Color
		want tcell.Color
	}{
		{name: "White", got: White, want: tcell.NewHexColor(0xf3f3f3).TrueColor()},
		{name: "Blue", got: Blue, want: tcell.NewHexColor(0x007ace).TrueColor()},
		{name: "Black", got: Black, want: tcell.NewHexColor(0x000000).TrueColor()},
		{name: "LightGray", got: LightGray, want: tcell.NewHexColor(0xaaaaaa).TrueColor()},
		{name: "Green", got: Green, want: tcell.NewHexColor(0x0b835c).TrueColor()},
		{name: "DarkGray", got: DarkGray, want: tcell.NewHexColor(0x333333).TrueColor()},
		{name: "Red", got: Red, want: tcell.NewHexColor(0xa4090c).TrueColor()},
		{name: "Yellow", got: Yellow, want: tcell.NewHexColor(0xb88f00).TrueColor()},
		{name: "InactiveGrey", got: InactiveGrey, want: tcell.NewHexColor(0x404040)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.got.Valid() {
				t.Errorf("%s is not a valid color", tc.name)
			}
			if tc.got == tcell.ColorDefault {
				t.Errorf("%s is the default color", tc.name)
			}
			if tc.got != tc.want {
				t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestColorVariablesAssigned(t *testing.T) {
	tests := []struct {
		name string
		got  tcell.Color
		want tcell.Color
	}{
		{name: "ButtonLabelColor", got: ButtonLabelColor, want: Black},
		{name: "ButtonLabelFocusedColor", got: ButtonLabelFocusedColor, want: Black},
		{name: "ButtonBackgroundColor", got: ButtonBackgroundColor, want: White},
		{name: "ButtonBackgroundFocusedColor", got: ButtonBackgroundFocusedColor, want: Blue},
		{name: "ButtonBackgroundDisabledColor", got: ButtonBackgroundDisabledColor, want: White},
		{name: "ButtonLabelDisabledColor", got: ButtonLabelDisabledColor, want: LightGray},
		{name: "InputFieldLabelColor", got: InputFieldLabelColor, want: Black},
		{name: "InputFieldFieldBackgroundColor", got: InputFieldFieldBackgroundColor, want: DarkGray},
		{name: "InputFieldFieldBackgroundFocusedColor", got: InputFieldFieldBackgroundFocusedColor, want: Black},
		{name: "InputFieldFieldTextColor", got: InputFieldFieldTextColor, want: LightGray},
		{name: "InputFieldFieldTextFocusedColor", got: InputFieldFieldTextFocusedColor, want: White},
		{name: "InputFieldPlaceholderTextColor", got: InputFieldPlaceholderTextColor, want: LightGray},
		{name: "ListMainTextColor", got: ListMainTextColor, want: White},
		{name: "ListSecondaryTextColor", got: ListSecondaryTextColor, want: LightGray},
		{name: "ListShortcutColor", got: ListShortcutColor, want: LightGray},
		{name: "ListSelectedTextColor", got: ListSelectedTextColor, want: White},
		{name: "ListSelectedBackgroundColor", got: ListSelectedBackgroundColor, want: Blue},
		{name: "DropDownTextColor", got: DropDownTextColor, want: White},
		{name: "DropDownBackgroundColor", got: DropDownBackgroundColor, want: Black},
		{name: "DropDownSelectedTextColor", got: DropDownSelectedTextColor, want: White},
		{name: "DropDownSelectedBackgroundColor", got: DropDownSelectedBackgroundColor, want: Blue},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got == tcell.ColorDefault || !tc.got.Valid() {
				t.Errorf("%s was not initialised: %v", tc.name, tc.got)
			}
			if tc.got != tc.want {
				t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestCheckboxStyles(t *testing.T) {
	tests := []struct {
		name string
		got  tcell.Style
		want tcell.Style
		fg   tcell.Color
		bg   tcell.Color
	}{
		{name: "Label", got: CheckboxLabelStyle, want: blackOnGrayStyle, fg: Black, bg: LightGray},
		{name: "Unchecked", got: CheckboxUncheckedStyle, want: blackOnGrayStyle, fg: Black, bg: LightGray},
		{name: "Checked", got: CheckboxCheckedStyle, want: whiteOnGreenStyle, fg: White, bg: Green},
		{name: "Focus", got: CheckboxFocusStyle, want: whiteOnBlueStyle, fg: White, bg: Blue},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%s style = %v, want %v", tc.name, tc.got, tc.want)
			}
			fg, bg, _ := tc.got.Decompose()
			if fg != tc.fg || bg != tc.bg {
				t.Errorf("%s style colors = %v/%v, want %v/%v", tc.name, fg, bg, tc.fg, tc.bg)
			}
		})
	}
}

func TestCheckboxStrings(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "Checked", got: CheckboxCheckedString, want: "[✓]"},
		{name: "Unchecked", got: CheckboxUncheckedString, want: "[ ]"},
		{name: "CursorChecked", got: CheckboxCursorCheckedString, want: "[✓]"},
		{name: "CursorUnchecked", got: CheckboxCursorUncheckedString, want: "[ ]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}

	if CheckboxCheckedString != CheckboxCursorCheckedString {
		t.Errorf("checked strings differ: %q vs %q", CheckboxCheckedString, CheckboxCursorCheckedString)
	}
	if CheckboxUncheckedString != CheckboxCursorUncheckedString {
		t.Errorf("unchecked strings differ: %q vs %q", CheckboxUncheckedString, CheckboxCursorUncheckedString)
	}
}

func TestTviewStyleOverrides(t *testing.T) {
	tests := []struct {
		name string
		got  tcell.Color
		want tcell.Color
	}{
		{name: "TitleColor", got: tview.Styles.TitleColor, want: Black},
		{name: "BorderColor", got: tview.Styles.BorderColor, want: Black},
		{name: "PrimitiveBackgroundColor", got: tview.Styles.PrimitiveBackgroundColor, want: LightGray},
		{name: "PrimaryTextColor", got: tview.Styles.PrimaryTextColor, want: Black},
		{name: "SecondaryTextColor", got: tview.Styles.SecondaryTextColor, want: Black},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("tview.Styles.%s = %v, want %v", tc.name, tc.got, tc.want)
			}
		})
	}
}
