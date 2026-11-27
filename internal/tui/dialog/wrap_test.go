package dialog

import (
	"strings"
	"testing"
)

func TestClampDialogWidth(t *testing.T) {
	// ScreenWidth defaults to 0 in tests.
	cases := []struct {
		in, want int
	}{
		{0, 1},
		{1, 1},
		{20, 20},
		{75, 75},
		{100, 75},
		{200, 75},
	}
	for _, tc := range cases {
		if got := ClampDialogWidth(tc.in); got != tc.want {
			t.Errorf("ClampDialogWidth(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestWrapText(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		width  int
		expect string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"wrap at space", "hello world", 6, "hello\nworld"},
		{"preserve newline", "a\nb", 10, "a\nb"},
		{"hard break long word", "abcdefghij", 4, "abcd\nefgh\nij"},
		{"empty", "", 5, ""},
		{"width 1", "ab", 1, "a\nb"},
	}
	for _, tc := range cases {
		got := WrapText(tc.text, tc.width)
		if got != tc.expect {
			t.Errorf("%s: WrapText(%q, %d) = %q, want %q", tc.name, tc.text, tc.width, got, tc.expect)
		}
	}
	// sanity: wrapped output lines never exceed width
	for _, tc := range cases {
		for _, line := range strings.Split(WrapText(tc.text, tc.width), "\n") {
			if len([]rune(line)) > tc.width && tc.width > 0 {
				t.Errorf("%s: wrapped line %q exceeds width %d", tc.name, line, tc.width)
			}
		}
	}
}
