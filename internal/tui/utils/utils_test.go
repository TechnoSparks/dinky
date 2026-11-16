package utils

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func newScreen(t *testing.T, w, h int) tcell.SimulationScreen {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	screen.SetSize(w, h)
	t.Cleanup(func() { screen.Fini() })
	return screen
}

func TestDrawText(t *testing.T) {
	screen := newScreen(t, 20, 5)
	style := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlue)
	DrawText(screen, 2, 1, "hi", style)

	for i, r := range "hi" {
		mainc, combc, gotStyle, _ := screen.GetContent(2+i, 1)
		if mainc != r || len(combc) != 0 {
			t.Errorf("cell %d = %q, want %q", i, mainc, r)
		}
		if gotStyle != style {
			t.Errorf("cell %d style mismatch", i)
		}
	}
	// Untouched cells remain spaces.
	mainc, _, _, _ := screen.GetContent(0, 1)
	if mainc != ' ' {
		t.Errorf("untouched cell = %q, want space", mainc)
	}
}

func TestDrawSpace(t *testing.T) {
	screen := newScreen(t, 20, 5)
	style := tcell.StyleDefault
	DrawSpace(screen, 3, 2, 4, style)
	for i := 0; i < 4; i++ {
		mainc, _, _, _ := screen.GetContent(3+i, 2)
		if mainc != ' ' {
			t.Errorf("cell %d = %q, want space", i, mainc)
		}
	}
}

func TestDrawHorizontalLine(t *testing.T) {
	screen := newScreen(t, 20, 5)
	style := tcell.StyleDefault
	DrawHorizontalLine(screen, 1, 3, 5, style, '-')
	for i := 0; i < 5; i++ {
		mainc, _, _, _ := screen.GetContent(1+i, 3)
		if mainc != '-' {
			t.Errorf("cell %d = %q, want -", i, mainc)
		}
	}
}

func TestDrawCappedHorizontalLine(t *testing.T) {
	screen := newScreen(t, 20, 5)
	bs := tcell.StyleDefault.Foreground(tcell.ColorRed)
	ms := tcell.StyleDefault.Foreground(tcell.ColorGreen)
	DrawCappedHorizontalLine(screen, 0, 0, 5, bs, ms, '[', '=', ']')

	cases := []struct {
		x    int
		rune rune
	}{
		{0, '['},
		{1, '='},
		{2, '='},
		{3, '='},
		{4, ']'},
	}
	for _, c := range cases {
		mainc, _, _, _ := screen.GetContent(c.x, 0)
		if mainc != c.rune {
			t.Errorf("cell %d = %q, want %q", c.x, mainc, c.rune)
		}
	}
}

func TestDimCell(t *testing.T) {
	screen := newScreen(t, 20, 5)
	style := tcell.StyleDefault.Foreground(tcell.NewRGBColor(200, 100, 50)).Background(tcell.NewRGBColor(100, 200, 50))
	screen.SetContent(4, 2, 'X', nil, style)

	DimCell(screen, 4, 2)

	mainc, _, gotStyle, _ := screen.GetContent(4, 2)
	if mainc != 'X' {
		t.Errorf("rune changed to %q", mainc)
	}
	fg, bg, _ := gotStyle.Decompose()
	fgR, fgG, fgB := fg.TrueColor().RGB()
	bgR, bgG, bgB := bg.TrueColor().RGB()
	if fgR != 100 || fgG != 50 || fgB != 25 {
		t.Errorf("dimmed fg = (%d,%d,%d), want (100,50,25)", fgR, fgG, fgB)
	}
	if bgR != 50 || bgG != 100 || bgB != 25 {
		t.Errorf("dimmed bg = (%d,%d,%d), want (50,100,25)", bgR, bgG, bgB)
	}
}

func TestDrawDimHorizontalLine(t *testing.T) {
	screen := newScreen(t, 20, 5)
	style := tcell.StyleDefault.Foreground(tcell.NewRGBColor(200, 100, 50))
	for i := 0; i < 3; i++ {
		screen.SetContent(i, 0, 'a', nil, style)
	}
	DrawDimHorizontalLine(screen, 0, 0, 3)
	for i := 0; i < 3; i++ {
		mainc, _, gotStyle, _ := screen.GetContent(i, 0)
		if mainc != 'a' {
			t.Errorf("rune %d changed to %q", i, mainc)
		}
		fg, _, _ := gotStyle.Decompose()
		r, _, _ := fg.TrueColor().RGB()
		if r != 100 {
			t.Errorf("cell %d not dimmed: r=%d", i, r)
		}
	}
}

func TestDrawDimVerticalLine(t *testing.T) {
	screen := newScreen(t, 20, 5)
	style := tcell.StyleDefault.Foreground(tcell.NewRGBColor(200, 100, 50))
	for i := 0; i < 3; i++ {
		screen.SetContent(0, i, 'b', nil, style)
	}
	DrawDimVerticalLine(screen, 0, 0, 3)
	for i := 0; i < 3; i++ {
		mainc, _, gotStyle, _ := screen.GetContent(0, i)
		if mainc != 'b' {
			t.Errorf("rune %d changed to %q", i, mainc)
		}
		fg, _, _ := gotStyle.Decompose()
		r, _, _ := fg.TrueColor().RGB()
		if r != 100 {
			t.Errorf("cell %d not dimmed: r=%d", i, r)
		}
	}
}
