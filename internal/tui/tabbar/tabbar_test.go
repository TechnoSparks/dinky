package tabbar

import (
	"math"
	"reflect"
	"strconv"
	"testing"

	"dinky/internal/tui/stylecolor"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newTabBarScreen(t *testing.T, width, height int) tcell.SimulationScreen {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init simulation screen: %v", err)
	}
	screen.SetSize(width, height)
	t.Cleanup(screen.Fini)
	return screen
}

func assertCellStyle(t *testing.T, screen tcell.Screen, x, y int, wantFg, wantBg tcell.Color, wantBold bool) {
	t.Helper()
	_, _, style, _ := screen.GetContent(x, y)
	fg, bg, attr := style.Decompose()
	if fg != wantFg {
		t.Errorf("cell (%d,%d) foreground = %v, want %v", x, y, fg, wantFg)
	}
	if bg != wantBg {
		t.Errorf("cell (%d,%d) background = %v, want %v", x, y, bg, wantBg)
	}
	if bold := attr&tcell.AttrBold != 0; bold != wantBold {
		t.Errorf("cell (%d,%d) bold = %v, want %v", x, y, bold, wantBold)
	}
}

func assertCellRune(t *testing.T, screen tcell.Screen, x, y int, want rune) {
	t.Helper()
	got, _, _, _ := screen.GetContent(x, y)
	if got != want {
		t.Errorf("cell (%d,%d) rune = %q, want %q", x, y, got, want)
	}
}

func tabIDs(tb *TabBar) []string {
	ids := make([]string, len(tb.tabs))
	for i, tab := range tb.tabs {
		ids[i] = tab.ID
	}
	return ids
}

func fireTabBarMouse(tb *TabBar, action tview.MouseAction, x, y int, buttons tcell.ButtonMask) (bool, tview.Primitive) {
	event := tcell.NewEventMouse(x, y, buttons, tcell.ModNone)
	return tb.MouseHandler()(action, event, func(tview.Primitive) {})
}

func TestNewTabBar(t *testing.T) {
	tb := NewTabBar()
	if tb.Box == nil {
		t.Fatal("NewTabBar returned a nil Box")
	}

	fg, bg, attr := tb.BackgroundStyle.Decompose()
	if fg != stylecolor.White || bg != stylecolor.Blue {
		t.Errorf("background colors = %v/%v, want %v/%v", fg, bg, stylecolor.White, stylecolor.Blue)
	}
	if attr&tcell.AttrBold == 0 {
		t.Error("background style should be bold")
	}

	fg, bg, attr = tb.ActiveTabStyle.Decompose()
	if fg != stylecolor.White || bg != stylecolor.Black {
		t.Errorf("active tab colors = %v/%v, want %v/%v", fg, bg, stylecolor.White, stylecolor.Black)
	}
	if attr&tcell.AttrBold == 0 {
		t.Error("active tab style should be bold")
	}

	fg, bg, attr = tb.InactiveTabStyle.Decompose()
	if fg != stylecolor.White || bg != stylecolor.InactiveGrey {
		t.Errorf("inactive tab colors = %v/%v, want %v/%v", fg, bg, stylecolor.White, stylecolor.InactiveGrey)
	}
	if attr&tcell.AttrBold != 0 {
		t.Error("inactive tab style should not be bold")
	}

	if index, id := tb.Active(); index != -1 || id != "" {
		t.Errorf("Active() on empty bar = (%d, %q), want (-1, \"\")", index, id)
	}
}

func TestSetTabBackgroundColors(t *testing.T) {
	tb := NewTabBar()
	tb.SetTabBackgroundColor(tcell.ColorRed)
	tb.SetTabInactiveBackgroundColor(tcell.ColorGreen)

	if _, bg, _ := tb.ActiveTabStyle.Decompose(); bg != tcell.ColorRed {
		t.Errorf("active background = %v, want red", bg)
	}
	if _, bg, _ := tb.InactiveTabStyle.Decompose(); bg != tcell.ColorGreen {
		t.Errorf("inactive background = %v, want green", bg)
	}
}

func TestComputeTabWidth(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  int
	}{
		{name: "empty", title: "", want: 7},
		{name: "single ASCII", title: "a", want: 8},
		{name: "several ASCII", title: "abc", want: 10},
		{name: "spaces count", title: "a b", want: 10},
		{name: "wide runes", title: "日本", want: 11},
		{name: "mixed", title: "a日b", want: 11},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := computeTabWidth(Tab{Title: tc.title}); got != tc.want {
				t.Errorf("computeTabWidth(%q) = %d, want %d", tc.title, got, tc.want)
			}
		})
	}
}

func TestAddTab(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("one", "1")
	tb.AddTab("two", "2")

	if len(tb.tabs) != 2 {
		t.Fatalf("tab count = %d, want 2", len(tb.tabs))
	}
	if tb.tabs[0].Title != "one" || tb.tabs[0].ID != "1" {
		t.Errorf("first tab = %+v, want title=one id=1", tb.tabs[0])
	}
	if tb.tabs[1].Title != "two" || tb.tabs[1].ID != "2" {
		t.Errorf("second tab = %+v, want title=two id=2", tb.tabs[1])
	}
	for i, tab := range tb.tabs {
		if want := computeTabWidth(tab); tab.width != want {
			t.Errorf("tab %d width = %d, want %d", i, tab.width, want)
		}
		if tab.placed {
			t.Errorf("tab %d should not be placed before Draw/ensurePlaced", i)
		}
	}
}

func TestRemoveTab(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.AddTab("ccc", "c")

	tb.RemoveTab("b")
	if got, want := tabIDs(tb), []string{"a", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tabs after removing b = %v, want %v", got, want)
	}

	// Removing an unknown ID is a no-op.
	tb.RemoveTab("missing")
	if got, want := tabIDs(tb), []string{"a", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tabs after removing unknown id = %v, want %v", got, want)
	}

	// Removing the active tab leaves active pointing at the same index, which
	// now holds the following tab.
	tb.active = 0
	tb.RemoveTab("a")
	if got, want := tabIDs(tb), []string{"c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tabs after removing active tab = %v, want %v", got, want)
	}
	if index, id := tb.Active(); index != 0 || id != "c" {
		t.Errorf("Active() = (%d, %q), want (0, \"c\")", index, id)
	}

	// Removing a tab snaps render positions because startAnim has no redraw
	// hook in tests.
	tb2 := NewTabBar()
	tb2.AddTab("aaa", "a")
	tb2.AddTab("bbb", "b")
	tb2.AddTab("ccc", "c")
	tb2.ensurePlaced()
	tb2.tabs[0].renderX = 99
	tb2.tabs[1].renderX = -4
	tb2.RemoveTab("c")
	wantX := []float64{0, 10}
	for i, want := range wantX {
		if tb2.tabs[i].renderX != want {
			t.Errorf("tab %d renderX after RemoveTab = %v, want %v", i, tb2.tabs[i].renderX, want)
		}
	}
}

func TestSetTabTitle(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.ensurePlaced()

	tb.SetTabTitle("a", "longer title")
	if tb.tabs[0].Title != "longer title" {
		t.Errorf("title = %q, want %q", tb.tabs[0].Title, "longer title")
	}
	if want := computeTabWidth(tb.tabs[0]); tb.tabs[0].width != want {
		t.Errorf("width = %d, want %d", tb.tabs[0].width, want)
	}

	// A same-width rename must not trigger the slide animation, so the render
	// position is left untouched.
	tb.tabs[0].renderX = 5
	tb.tabs[0].placed = true
	tb.SetTabTitle("a", "another name")
	if tb.tabs[0].renderX != 5 {
		t.Errorf("renderX after same-width rename = %v, want 5", tb.tabs[0].renderX)
	}

	tb.SetTabTitle("missing", "nope")
	if tb.tabs[0].Title != "another name" {
		t.Errorf("unknown id changed a title: %q", tb.tabs[0].Title)
	}
}

func TestActiveAndSetActive(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.AddTab("ccc", "c")

	if index, id := tb.Active(); index != 0 || id != "a" {
		t.Fatalf("initial Active() = (%d, %q), want (0, \"a\")", index, id)
	}

	tb.SetActive("b")
	if index, id := tb.Active(); index != 1 || id != "b" {
		t.Errorf("Active() after SetActive(b) = (%d, %q), want (1, \"b\")", index, id)
	}

	tb.SetActive("missing")
	if index, id := tb.Active(); index != 1 || id != "b" {
		t.Errorf("Active() after unknown SetActive = (%d, %q), want (1, \"b\")", index, id)
	}
}

func TestSetActiveScrollsTabIntoView(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 30, 1)
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.AddTab("ccc", "c")
	tb.AddTab("ddd", "d")

	tb.SetActive("d")
	// Four 10-column tabs overflow the 30-column bar. The inner usable width
	// for SetActive is width-2 (the two overflow arrows): 30+10-28 = 12.
	if tb.hscroll != 12 {
		t.Errorf("hscroll after activating last tab = %d, want 12", tb.hscroll)
	}
	if index, id := tb.Active(); index != 3 || id != "d" {
		t.Errorf("Active() = (%d, %q), want (3, \"d\")", index, id)
	}

	tb.SetActive("a")
	if tb.hscroll != 0 {
		t.Errorf("hscroll after activating first tab = %d, want 0", tb.hscroll)
	}
}

func TestTotalBarWidthAndOverflow(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 20, 1)
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")

	if got := tb.totalBarWidth(); got != 20 {
		t.Errorf("totalBarWidth() = %d, want 20", got)
	}
	if tb.isOverflow() {
		t.Error("bar with exactly fitting tabs should not overflow")
	}

	tb.AddTab("ccc", "c")
	if got := tb.totalBarWidth(); got != 30 {
		t.Errorf("totalBarWidth() = %d, want 30", got)
	}
	if !tb.isOverflow() {
		t.Error("bar with 30 columns of tabs in 20 columns should overflow")
	}
}

func TestDrawBasics(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 30, 1)
	tb.AddTab("one", "1")
	tb.AddTab("two", "2")
	screen := newTabBarScreen(t, 30, 1)

	tb.Draw(screen)

	// Bar background.
	for _, x := range []int{8, 9, 25} {
		assertCellRune(t, screen, x, 0, ' ')
		assertCellStyle(t, screen, x, 0, stylecolor.White, stylecolor.Blue, true)
	}

	// Active tab (index 0) starts at column 0: corner, padding, title, gap,
	// close glyph, corner.
	wantActive := map[int]rune{0: '◢', 1: ' ', 2: 'o', 3: 'n', 4: 'e', 5: ' ', 6: '✕', 7: '◣'}
	for x, want := range wantActive {
		assertCellRune(t, screen, x, 0, want)
		if x == 0 || x == 7 {
			// Corners use the tab background as foreground and the bar
			// background as background.
			assertCellStyle(t, screen, x, 0, stylecolor.Black, stylecolor.Blue, false)
		} else {
			assertCellStyle(t, screen, x, 0, stylecolor.White, stylecolor.Black, true)
		}
	}

	// Inactive tab (index 1) starts at column 10.
	assertCellRune(t, screen, 10, 0, '◢')
	assertCellStyle(t, screen, 10, 0, stylecolor.InactiveGrey, stylecolor.Blue, false)
	assertCellRune(t, screen, 12, 0, 't')
	assertCellStyle(t, screen, 12, 0, stylecolor.White, stylecolor.InactiveGrey, false)
	assertCellRune(t, screen, 16, 0, '✕')
	assertCellStyle(t, screen, 16, 0, stylecolor.White, stylecolor.InactiveGrey, false)
}

func TestDrawEmptyBar(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 12, 1)
	screen := newTabBarScreen(t, 12, 1)
	tb.Draw(screen)

	for x := 0; x < 12; x++ {
		assertCellRune(t, screen, x, 0, ' ')
		assertCellStyle(t, screen, x, 0, stylecolor.White, stylecolor.Blue, true)
	}
}

func TestDrawOverflowIndicators(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 20, 1)
	for _, title := range []string{"aaa", "bbb", "ccc", "ddd", "eee"} {
		tb.AddTab(title, title)
	}
	screen := newTabBarScreen(t, 20, 1)

	tb.Draw(screen)

	// The two rightmost columns are always the overflow arrows.
	assertCellRune(t, screen, 18, 0, '⯇')
	assertCellRune(t, screen, 19, 0, '⯈')
	assertCellStyle(t, screen, 18, 0, stylecolor.White, stylecolor.Blue, true)
	assertCellStyle(t, screen, 19, 0, stylecolor.White, stylecolor.Blue, true)

	// The first two tabs are still visible at hscroll 0.
	assertCellRune(t, screen, 0, 0, '◢')
	assertCellRune(t, screen, 2, 0, 'a')
	assertCellRune(t, screen, 10, 0, '◢')
	assertCellRune(t, screen, 12, 0, 'b')

	// Scrolling by the maximum amount reveals the last tabs while keeping the
	// overflow indicators on top.
	tb.hscroll = 30
	tb.Draw(screen)
	assertCellRune(t, screen, 0, 0, '◢')
	assertCellRune(t, screen, 2, 0, 'd')
	assertCellRune(t, screen, 10, 0, '◢')
	assertCellRune(t, screen, 12, 0, 'e')
	assertCellRune(t, screen, 18, 0, '⯇')
	assertCellRune(t, screen, 19, 0, '⯈')
}

func TestDrawScrolledActiveTab(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 30, 1)
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.AddTab("ccc", "c")
	tb.AddTab("ddd", "d")
	tb.SetActive("d")
	screen := newTabBarScreen(t, 30, 1)

	tb.Draw(screen)

	// hscroll is 12, so the active fourth tab (layout x 30) starts at 18.
	assertCellRune(t, screen, 18, 0, '◢')
	assertCellStyle(t, screen, 18, 0, stylecolor.Black, stylecolor.Blue, false)
	assertCellRune(t, screen, 20, 0, 'd')
	assertCellStyle(t, screen, 20, 0, stylecolor.White, stylecolor.Black, true)
	assertCellRune(t, screen, 24, 0, '✕')
	assertCellRune(t, screen, 28, 0, '⯇')
	assertCellRune(t, screen, 29, 0, '⯈')
}

func TestDrawActiveTabOnTop(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 30, 1)
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.active = 1
	tb.ensurePlaced()
	// Force both tabs onto the same layout position to exercise draw order.
	tb.tabs[0].renderX = 0
	tb.tabs[1].renderX = 0
	screen := newTabBarScreen(t, 30, 1)

	tb.Draw(screen)

	assertCellRune(t, screen, 2, 0, 'b')
	assertCellStyle(t, screen, 2, 0, stylecolor.White, stylecolor.Black, true)
	assertCellRune(t, screen, 0, 0, '◢')
	assertCellStyle(t, screen, 0, 0, stylecolor.Black, stylecolor.Blue, false)
}

func TestMouseClickActivatesTab(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 30, 1)
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.AddTab("ccc", "c")

	var activeCalls [][2]any
	tb.OnActive = func(id string, index int) {
		activeCalls = append(activeCalls, [2]any{id, index})
	}

	consumed, capture := fireTabBarMouse(tb, tview.MouseLeftDown, 12, 0, tcell.Button1)
	if !consumed {
		t.Error("left click on a tab was not consumed")
	}
	if capture != tview.Primitive(tb) {
		t.Errorf("capture after left down = %v, want the tab bar", capture)
	}
	if tb.active != 1 {
		t.Errorf("active index = %d, want 1", tb.active)
	}
	if !tb.dragging {
		t.Error("dragging should start on a non-close left click")
	}
	if want := [][2]any{{"b", 1}}; !reflect.DeepEqual(activeCalls, want) {
		t.Errorf("OnActive calls = %v, want %v", activeCalls, want)
	}

	consumed, capture = fireTabBarMouse(tb, tview.MouseLeftUp, 12, 0, tcell.ButtonNone)
	if !consumed || capture != nil {
		t.Errorf("left up = (consumed=%v, capture=%v), want (true, nil)", consumed, capture)
	}
	if tb.dragging {
		t.Error("dragging should stop on left up")
	}

	// Clicking a further tab reports its own index.
	fireTabBarMouse(tb, tview.MouseLeftDown, 25, 0, tcell.Button1)
	fireTabBarMouse(tb, tview.MouseLeftUp, 25, 0, tcell.ButtonNone)
	if want := [][2]any{{"b", 1}, {"c", 2}}; !reflect.DeepEqual(activeCalls, want) {
		t.Errorf("OnActive calls = %v, want %v", activeCalls, want)
	}
}

func TestMouseCloseButton(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 30, 1)
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")

	var activeCalls []string
	var closeCalls []string
	tb.OnActive = func(id string, index int) {
		activeCalls = append(activeCalls, id)
	}
	tb.OnTabCloseClick = func(id string, index int) {
		closeCalls = append(closeCalls, id)
	}

	// The close glyph for the second tab is at layout x 10+10-4 = 16.
	consumed, capture := fireTabBarMouse(tb, tview.MouseLeftDown, 16, 0, tcell.Button1)
	if !consumed || capture != nil {
		t.Errorf("close click = (consumed=%v, capture=%v), want (true, nil)", consumed, capture)
	}
	if tb.dragging {
		t.Error("close click must not start a drag")
	}
	if tb.active != 1 {
		t.Errorf("active index = %d, want 1", tb.active)
	}
	if want := []string{"b"}; !reflect.DeepEqual(activeCalls, want) {
		t.Errorf("OnActive calls = %v, want %v", activeCalls, want)
	}
	if want := []string{"b"}; !reflect.DeepEqual(closeCalls, want) {
		t.Errorf("OnTabCloseClick calls = %v, want %v", closeCalls, want)
	}
}

func TestMouseClickBackgroundAndOutside(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 40, 1)
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")

	called := false
	tb.OnActive = func(string, int) { called = true }

	// Right of the last tab but inside the bar.
	consumed, capture := fireTabBarMouse(tb, tview.MouseLeftDown, 35, 0, tcell.Button1)
	if !consumed || capture != nil {
		t.Errorf("background click = (consumed=%v, capture=%v), want (true, nil)", consumed, capture)
	}
	if tb.active != 0 || called || tb.dragging {
		t.Errorf("background click changed state: active=%d called=%v dragging=%v", tb.active, called, tb.dragging)
	}

	// Outside the bar's row.
	consumed, capture = fireTabBarMouse(tb, tview.MouseLeftDown, 5, 1, tcell.Button1)
	if consumed || capture != nil {
		t.Errorf("click outside row = (consumed=%v, capture=%v), want (false, nil)", consumed, capture)
	}
	if called {
		t.Error("OnActive called for a click outside the bar")
	}
}

func TestMouseOverflowArrowsAndScroll(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 20, 1)
	for _, title := range []string{"aaa", "bbb", "ccc", "ddd", "eee"} {
		tb.AddTab(title, title)
	}

	// Mouse wheel scrolls by tabScrollStep and clamps at the maximum.
	fireTabBarMouse(tb, tview.MouseScrollDown, 0, 0, tcell.ButtonNone)
	if tb.hscroll != 20 {
		t.Errorf("hscroll after one wheel down = %d, want 20", tb.hscroll)
	}
	fireTabBarMouse(tb, tview.MouseScrollDown, 0, 0, tcell.ButtonNone)
	if tb.hscroll != 30 {
		t.Errorf("hscroll after two wheel downs = %d, want 30", tb.hscroll)
	}
	fireTabBarMouse(tb, tview.MouseScrollDown, 0, 0, tcell.ButtonNone)
	if tb.hscroll != 30 {
		t.Errorf("hscroll should clamp at 30, got %d", tb.hscroll)
	}
	fireTabBarMouse(tb, tview.MouseScrollUp, 0, 0, tcell.ButtonNone)
	if tb.hscroll != 10 {
		t.Errorf("hscroll after wheel up = %d, want 10", tb.hscroll)
	}
	fireTabBarMouse(tb, tview.MouseScrollUp, 0, 0, tcell.ButtonNone)
	fireTabBarMouse(tb, tview.MouseScrollUp, 0, 0, tcell.ButtonNone)
	if tb.hscroll != 0 {
		t.Errorf("hscroll should clamp at 0, got %d", tb.hscroll)
	}

	// The left arrow scrolls back toward the start.
	tb.hscroll = 20
	consumed, capture := fireTabBarMouse(tb, tview.MouseLeftDown, 18, 0, tcell.Button1)
	if !consumed || capture != nil {
		t.Errorf("left arrow click = (consumed=%v, capture=%v), want (true, nil)", consumed, capture)
	}
	if tb.hscroll != 0 {
		t.Errorf("hscroll after left arrow = %d, want 0", tb.hscroll)
	}

	// The right arrow scrolls forward.
	consumed, capture = fireTabBarMouse(tb, tview.MouseLeftDown, 19, 0, tcell.Button1)
	if !consumed || capture != nil {
		t.Errorf("right arrow click = (consumed=%v, capture=%v), want (true, nil)", consumed, capture)
	}
	if tb.hscroll != 20 {
		t.Errorf("hscroll after right arrow = %d, want 20", tb.hscroll)
	}
}

func TestDragReordersTabs(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 40, 1)
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.AddTab("ccc", "c")

	var reorderCalls []string
	var activeCalls []string
	tb.OnReorder = func(id string, newIndex int) {
		reorderCalls = append(reorderCalls, id+":"+strconv.Itoa(newIndex))
	}
	tb.OnActive = func(id string, index int) {
		activeCalls = append(activeCalls, id)
	}

	consumed, capture := fireTabBarMouse(tb, tview.MouseLeftDown, 3, 0, tcell.Button1)
	if !consumed || capture != tview.Primitive(tb) || !tb.dragging {
		t.Fatalf("drag start = (consumed=%v, capture=%v, dragging=%v)", consumed, capture, tb.dragging)
	}
	if tb.dragIndex != 0 {
		t.Errorf("dragIndex at start = %d, want 0", tb.dragIndex)
	}

	// Cross into the middle tab's left half (layout x 5).
	fireTabBarMouse(tb, tview.MouseMove, 5, 0, tcell.Button1)
	if got, want := tabIDs(tb), []string{"b", "a", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order after first move = %v, want %v", got, want)
	}
	if tb.dragIndex != 1 || tb.active != 1 {
		t.Errorf("after first move dragIndex=%d active=%d, want 1/1", tb.dragIndex, tb.active)
	}

	// Move past the end to land in the last slot.
	fireTabBarMouse(tb, tview.MouseMove, 20, 0, tcell.Button1)
	if got, want := tabIDs(tb), []string{"b", "c", "a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order after second move = %v, want %v", got, want)
	}
	if tb.dragIndex != 2 || tb.active != 2 {
		t.Errorf("after second move dragIndex=%d active=%d, want 2/2", tb.dragIndex, tb.active)
	}

	consumed, capture = fireTabBarMouse(tb, tview.MouseLeftUp, 20, 0, tcell.ButtonNone)
	if !consumed || capture != nil {
		t.Errorf("drag release = (consumed=%v, capture=%v), want (true, nil)", consumed, capture)
	}
	if tb.dragging || tb.dragIndex != -1 {
		t.Errorf("after release dragging=%v dragIndex=%d, want false/-1", tb.dragging, tb.dragIndex)
	}

	if want := []string{"a:1", "a:2"}; !reflect.DeepEqual(reorderCalls, want) {
		t.Errorf("OnReorder calls = %v, want %v", reorderCalls, want)
	}
	if want := []string{"a"}; !reflect.DeepEqual(activeCalls, want) {
		t.Errorf("OnActive calls = %v, want %v", activeCalls, want)
	}
}

func TestDragTargetIndex(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.AddTab("ccc", "c")

	tests := []struct {
		name    string
		relX    int
		hscroll int
		want    int
	}{
		{name: "negative clamps to first", relX: -10, want: 0},
		{name: "first half of first", relX: 0, want: 0},
		{name: "just before middle of first", relX: 4, want: 0},
		{name: "middle of first goes to second", relX: 5, want: 1},
		{name: "near end of second", relX: 14, want: 1},
		{name: "middle of second goes to third", relX: 15, want: 2},
		{name: "past end clamps to last", relX: 500, want: 2},
		{name: "hscroll shifts the boundary", relX: 0, hscroll: 10, want: 1},
		{name: "hscroll past first tab", relX: 4, hscroll: 10, want: 1},
		{name: "hscroll into second tab", relX: 5, hscroll: 10, want: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tb.hscroll = tc.hscroll
			if got := tb.dragTargetIndex(tc.relX); got != tc.want {
				t.Errorf("dragTargetIndex(%d) with hscroll=%d = %d, want %d", tc.relX, tc.hscroll, got, tc.want)
			}
		})
	}

	empty := NewTabBar()
	if got := empty.dragTargetIndex(5); got != -1 {
		t.Errorf("dragTargetIndex on empty bar = %d, want -1", got)
	}
}

func TestMoveTab(t *testing.T) {
	tests := []struct {
		name string
		from int
		to   int
		want []string
	}{
		{name: "same index", from: 0, to: 0, want: []string{"a", "b", "c"}},
		{name: "forward", from: 0, to: 2, want: []string{"b", "c", "a"}},
		{name: "backward", from: 2, to: 0, want: []string{"c", "a", "b"}},
		{name: "adjacent", from: 0, to: 1, want: []string{"b", "a", "c"}},
		{name: "negative from", from: -1, to: 0, want: []string{"a", "b", "c"}},
		{name: "negative to", from: 0, to: -1, want: []string{"a", "b", "c"}},
		{name: "from out of range", from: 3, to: 0, want: []string{"a", "b", "c"}},
		{name: "to out of range", from: 0, to: 3, want: []string{"a", "b", "c"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tb := NewTabBar()
			tb.AddTab("aaa", "a")
			tb.AddTab("bbb", "b")
			tb.AddTab("ccc", "c")
			tb.moveTab(tc.from, tc.to)
			if got := tabIDs(tb); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("moveTab(%d,%d) order = %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestTabIndexAtX(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 30, 1)
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.AddTab("ccc", "c")

	tests := []struct {
		name      string
		relX      int
		hscroll   int
		width     int
		wantIndex int
		wantLeft  int
		wantClose bool
	}{
		{name: "left of first tab", relX: -1, width: 30, wantIndex: -1, wantLeft: -1},
		{name: "first tab", relX: 0, width: 30, wantIndex: 0, wantLeft: 0},
		{name: "first tab close", relX: 6, width: 30, wantIndex: 0, wantLeft: 0, wantClose: true},
		{name: "first tab trailing", relX: 9, width: 30, wantIndex: 0, wantLeft: 0},
		{name: "second tab", relX: 10, width: 30, wantIndex: 1, wantLeft: 10},
		{name: "second tab close", relX: 16, width: 30, wantIndex: 1, wantLeft: 10, wantClose: true},
		{name: "third tab", relX: 20, width: 30, wantIndex: 2, wantLeft: 20},
		{name: "past the end", relX: 35, width: 40, wantIndex: -1, wantLeft: -1},
		{name: "overflow arrow area", relX: 18, width: 20, wantIndex: -1, wantLeft: -1},
		{name: "overflow close", relX: 16, width: 20, wantIndex: 1, wantLeft: 10, wantClose: true},
		{name: "overflow hidden past arrow", relX: 19, width: 20, wantIndex: -1, wantLeft: -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tb.SetRect(0, 0, tc.width, 1)
			tb.hscroll = tc.hscroll
			index, left, closeClick := tb.tabIndexAtX(tc.relX)
			if index != tc.wantIndex || left != tc.wantLeft || closeClick != tc.wantClose {
				t.Errorf("tabIndexAtX(%d) = (%d,%d,%v), want (%d,%d,%v)",
					tc.relX, index, left, closeClick, tc.wantIndex, tc.wantLeft, tc.wantClose)
			}
		})
	}
}

func TestAutoScrollTick(t *testing.T) {
	tb := NewTabBar()
	tb.SetRect(0, 0, 20, 1)
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.AddTab("ccc", "c")

	// Not dragging: no scrolling.
	tb.hscroll = 5
	tb.autoScrollTick()
	if tb.hscroll != 5 {
		t.Errorf("hscroll changed while not dragging: %d", tb.hscroll)
	}

	// Holding against the right edge scrolls by dragAutoScrollStep.
	tb.dragging = true
	tb.dragIndex = 2
	tb.hscroll = 0
	tb.dragMouseX = 18
	tb.autoScrollTick()
	if tb.hscroll != dragAutoScrollStep {
		t.Errorf("hscroll after right-edge tick = %d, want %d", tb.hscroll, dragAutoScrollStep)
	}

	// Scrolling clamps to the maximum (total width 30 - bar width 20 = 10).
	tb.hscroll = 100
	tb.dragMouseX = 0
	tb.autoScrollTick()
	if tb.hscroll != 10 {
		t.Errorf("hscroll after clamp = %d, want 10", tb.hscroll)
	}

	// A bar that does not overflow does not auto-scroll.
	tb.SetRect(0, 0, 40, 1)
	tb.hscroll = 0
	tb.dragMouseX = 39
	tb.autoScrollTick()
	if tb.hscroll != 0 {
		t.Errorf("hscroll on non-overflowing bar = %d, want 0", tb.hscroll)
	}

	// Ticks while not dragging are ignored even with overflow.
	tb.SetRect(0, 0, 20, 1)
	tb.dragging = false
	tb.hscroll = 3
	tb.autoScrollTick()
	if tb.hscroll != 3 {
		t.Errorf("hscroll changed on non-drag tick: %d", tb.hscroll)
	}
}

func TestEnsurePlaced(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.AddTab("ccc", "c")

	tb.ensurePlaced()
	for i, want := range []float64{0, 10, 20} {
		if tb.tabs[i].renderX != want {
			t.Errorf("tab %d renderX = %v, want %v", i, tb.tabs[i].renderX, want)
		}
		if !tb.tabs[i].placed {
			t.Errorf("tab %d not marked placed", i)
		}
	}
}

func TestAnimateStep(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.AddTab("ccc", "c")
	tb.ensurePlaced()

	tb.tabs[1].renderX = 0
	if !tb.animateStep() {
		t.Fatal("animateStep reported no movement while a tab was displaced")
	}
	// The second tab's target is 10; it eases 30% of the way.
	if want := 3.0; math.Abs(tb.tabs[1].renderX-want) > 1e-9 {
		t.Errorf("renderX after one step = %v, want %v", tb.tabs[1].renderX, want)
	}

	for i := 0; i < 100; i++ {
		if !tb.animateStep() {
			break
		}
	}
	for i, want := range []float64{0, 10, 20} {
		if math.Abs(tb.tabs[i].renderX-want) > 1e-9 {
			t.Errorf("tab %d settled at renderX %v, want %v", i, tb.tabs[i].renderX, want)
		}
	}
}

func TestAnimateStepSkipsDraggedTab(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.AddTab("ccc", "c")
	tb.ensurePlaced()

	tb.dragging = true
	tb.dragIndex = 1
	tb.tabs[1].renderX = 7
	tb.tabs[2].renderX = 0

	if !tb.animateStep() {
		t.Fatal("animateStep reported no movement for the following tab")
	}
	if tb.tabs[1].renderX != 7 {
		t.Errorf("dragged tab renderX = %v, want it pinned at 7", tb.tabs[1].renderX)
	}
	// The third tab targets layout x 20 (after the dragged tab's slot).
	if want := 6.0; math.Abs(tb.tabs[2].renderX-want) > 1e-9 {
		t.Errorf("following tab renderX = %v, want %v", tb.tabs[2].renderX, want)
	}
}

func TestStartAnimWithoutRedrawHookSnaps(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("aaa", "a")
	tb.AddTab("bbb", "b")
	tb.ensurePlaced()
	tb.tabs[0].renderX = 99
	tb.tabs[1].renderX = -3

	tb.startAnim()

	for i, want := range []float64{0, 10} {
		if tb.tabs[i].renderX != want {
			t.Errorf("tab %d renderX = %v, want %v", i, tb.tabs[i].renderX, want)
		}
		if !tb.tabs[i].placed {
			t.Errorf("tab %d not marked placed", i)
		}
	}
}

func TestStopAnim(t *testing.T) {
	tb := NewTabBar()
	tb.animating = true
	tb.animStop = make(chan struct{})

	tb.stopAnim()
	if tb.animating {
		t.Error("stopAnim did not clear animating")
	}
	if tb.animStop != nil {
		t.Error("stopAnim did not clear animStop")
	}

	// A second call must be a harmless no-op.
	tb.stopAnim()
}
