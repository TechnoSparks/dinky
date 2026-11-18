package dialog

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newDialogTestApp(t *testing.T, width, height int) (*tview.Application, tcell.SimulationScreen) {
	t.Helper()

	screen := tcell.NewSimulationScreen("UTF-8")
	app := tview.NewApplication()
	app.SetScreen(screen)
	screen.SetSize(width, height)
	t.Cleanup(screen.Fini)

	return app, screen
}

func screenRow(screen tcell.SimulationScreen, y int, width int) string {
	runes := make([]rune, 0, width)
	for x := 0; x < width; x++ {
		mainc, _, _, _ := screen.GetContent(x, y)
		if mainc == 0 {
			mainc = ' '
		}
		runes = append(runes, mainc)
	}
	return string(runes)
}

func screenContains(screen tcell.SimulationScreen, width, height int, want string) bool {
	for y := 0; y < height; y++ {
		if strings.Contains(screenRow(screen, y, width), want) {
			return true
		}
	}
	return false
}

func pressKey(primitive tview.Primitive, key tcell.Key, r rune, mod tcell.ModMask) {
	primitive.InputHandler()(tcell.NewEventKey(key, r, mod), func(p tview.Primitive) {})
}
