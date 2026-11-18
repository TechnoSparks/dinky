package table2

import (
	"fmt"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestExplore(t *testing.T) {
	for _, tc := range []struct {
		text  string
		width int
	}{
		{"", 5},
		{"hello world", 5},
		{"hello world", 7},
		{"hello world", 11},
		{"abcdefgh", 3},
		{"a\nb", 10},
		{"你好世界", 4},
		{"[red]hello", 5},
		{"one two three", 6},
		{"ab cd", 2},
	} {
		fmt.Printf("WordWrap(%q,%d) = %q\n", tc.text, tc.width, WordWrap(tc.text, tc.width))
	}
	fmt.Printf("Escape([red]) = %q\n", Escape("[red]"))
	fmt.Printf("Escape([red[) = %q\n", Escape("[red["))
	fmt.Printf("Escape([red[]) = %q\n", Escape("[red[]"))
	fmt.Printf("Unescape([red[]) = %q\n", Unescape("[red[]"))
	fmt.Printf("ColorHex(red)=%q default=%q rgb=%q unset=%q\n", ColorHex(tcell.ColorRed), ColorHex(tcell.ColorDefault), ColorHex(tcell.NewRGBColor(1, 2, 3)), ColorHex(ColorUnset))
	fmt.Printf("TaggedStringWidth([red]abc)=%d\n", TaggedStringWidth("[red]abc"))
	fmt.Printf("StripTags=%q\n", StripTags([]byte("a[red]b[blue[]c"), true, false))
}
