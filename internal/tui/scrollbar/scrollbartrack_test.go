package scrollbar

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newScrollbarScreen(t *testing.T, width, height int) tcell.SimulationScreen {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init simulation screen: %v", err)
	}
	screen.SetSize(width, height)
	t.Cleanup(screen.Fini)
	return screen
}

func assertScrollbarCell(t *testing.T, screen tcell.Screen, x, y int, wantRune rune, wantFg, wantBg tcell.Color, wantReverse bool) {
	t.Helper()
	got, _, style, _ := screen.GetContent(x, y)
	if got != wantRune {
		t.Errorf("cell (%d,%d) rune = %q, want %q", x, y, got, wantRune)
	}
	fg, bg, attr := style.Decompose()
	if fg != wantFg {
		t.Errorf("cell (%d,%d) foreground = %v, want %v", x, y, fg, wantFg)
	}
	if bg != wantBg {
		t.Errorf("cell (%d,%d) background = %v, want %v", x, y, bg, wantBg)
	}
	if reverse := attr&tcell.AttrReverse != 0; reverse != wantReverse {
		t.Errorf("cell (%d,%d) reverse = %v, want %v", x, y, reverse, wantReverse)
	}
}

func TestScrollbarTrackDefaults(t *testing.T) {
	track := NewScrollbarTrack()
	if track.Box == nil {
		t.Fatal("NewScrollbarTrack returned a nil Box")
	}
	if track.Position() != 0 {
		t.Errorf("Position() = %d, want 0", track.Position())
	}
	if track.ThumbSize() != 10 {
		t.Errorf("ThumbSize() = %d, want 10", track.ThumbSize())
	}
	if track.Max() != 100 {
		t.Errorf("Max() = %d, want 100", track.Max())
	}
	if track.Width() != 1 {
		t.Errorf("Width() = %d, want 1", track.Width())
	}
	if track.TrackColor() != tcell.ColorDarkGray {
		t.Errorf("TrackColor() = %v, want dark gray", track.TrackColor())
	}
	if track.ThumbColor() != tcell.ColorWhite {
		t.Errorf("ThumbColor() = %v, want white", track.ThumbColor())
	}
	if track.isHorizontal || track.thin || track.dragging {
		t.Errorf("unexpected initial flags: horizontal=%v thin=%v dragging=%v", track.isHorizontal, track.thin, track.dragging)
	}
}

func TestSetPositionClamping(t *testing.T) {
	track := NewScrollbarTrack()
	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "negative clamps to zero", in: -100, want: 0},
		{name: "minus one clamps to zero", in: -1, want: 0},
		{name: "zero", in: 0, want: 0},
		{name: "middle", in: 50, want: 50},
		{name: "maximum", in: 90, want: 90},
		{name: "over maximum clamps", in: 91, want: 90},
		{name: "far over maximum clamps", in: 1000, want: 90},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := track.SetPosition(tc.in); got != tc.want {
				t.Errorf("SetPosition(%d) = %d, want %d", tc.in, got, tc.want)
			}
			if got := track.Position(); got != tc.want {
				t.Errorf("Position() = %d, want %d", got, tc.want)
			}
		})
	}

	// The clamp depends on the thumb size.
	track.SetThumbSize(20)
	if got := track.SetPosition(1000); got != 80 {
		t.Errorf("SetPosition(1000) with thumb 20 = %d, want 80", got)
	}
}

func TestSetThumbSizeMaxWidth(t *testing.T) {
	track := NewScrollbarTrack()

	for _, tc := range []struct{ in, want int }{
		{in: -10, want: 1},
		{in: 0, want: 1},
		{in: 1, want: 1},
		{in: 7, want: 7},
		{in: 100, want: 100},
	} {
		track.SetThumbSize(tc.in)
		if got := track.ThumbSize(); got != tc.want {
			t.Errorf("SetThumbSize(%d) -> ThumbSize() = %d, want %d", tc.in, got, tc.want)
		}
	}

	track.SetMax(0)
	if got := track.Max(); got != 100 {
		t.Errorf("SetMax(0) changed max to %d, want it ignored", got)
	}
	track.SetMax(-5)
	if got := track.Max(); got != 100 {
		t.Errorf("SetMax(-5) changed max to %d, want it ignored", got)
	}
	track.SetMax(42)
	if got := track.Max(); got != 42 {
		t.Errorf("SetMax(42) -> Max() = %d, want 42", got)
	}

	track.SetWidth(0)
	if got := track.Width(); got != 1 {
		t.Errorf("SetWidth(0) changed width to %d, want it ignored", got)
	}
	track.SetWidth(-2)
	if got := track.Width(); got != 1 {
		t.Errorf("SetWidth(-2) changed width to %d, want it ignored", got)
	}
	track.SetWidth(3)
	if got := track.Width(); got != 3 {
		t.Errorf("SetWidth(3) -> Width() = %d, want 3", got)
	}
}

func TestScrollbarTrackColors(t *testing.T) {
	track := NewScrollbarTrack()

	track.SetTrackColor(tcell.ColorRed)
	if got := track.TrackColor(); got != tcell.ColorRed {
		t.Errorf("TrackColor() = %v, want red", got)
	}
	track.SetThumbColor(tcell.ColorGreen)
	if got := track.ThumbColor(); got != tcell.ColorGreen {
		t.Errorf("ThumbColor() = %v, want green", got)
	}
}

func TestDrawVerticalTrackAndThumb(t *testing.T) {
	track := NewScrollbarTrack()
	track.SetRect(0, 0, 1, 10)
	screen := newScrollbarScreen(t, 1, 10)

	track.Draw(screen)

	// Position 0 with a 10% thumb fills only the first cell with the thumb
	// style; the rest is the dark track.
	assertScrollbarCell(t, screen, 0, 0, ' ', tcell.ColorDarkGray, tcell.ColorWhite, false)
	for y := 1; y < 10; y++ {
		assertScrollbarCell(t, screen, 0, y, ' ', tcell.ColorDefault, tcell.ColorDarkGray, false)
	}
}

func TestDrawVerticalHalfCells(t *testing.T) {
	track := NewScrollbarTrack()
	track.SetRect(0, 0, 1, 10)
	track.SetPosition(5)
	screen := newScrollbarScreen(t, 1, 10)

	track.Draw(screen)

	// A half-cell start and a one-cell thumb produce the two half-block runes.
	assertScrollbarCell(t, screen, 0, 0, '▄', tcell.ColorDarkGray, tcell.ColorWhite, true)
	assertScrollbarCell(t, screen, 0, 1, '▀', tcell.ColorDarkGray, tcell.ColorWhite, true)
	assertScrollbarCell(t, screen, 0, 2, ' ', tcell.ColorDefault, tcell.ColorDarkGray, false)
	assertScrollbarCell(t, screen, 0, 9, ' ', tcell.ColorDefault, tcell.ColorDarkGray, false)
}

func TestDrawVerticalBottomThumb(t *testing.T) {
	track := NewScrollbarTrack()
	track.SetRect(0, 0, 1, 10)
	track.SetPosition(90)
	screen := newScrollbarScreen(t, 1, 10)

	track.Draw(screen)

	// At the bottom the thumb is extended so it does not leave a rounding gap.
	assertScrollbarCell(t, screen, 0, 9, ' ', tcell.ColorDarkGray, tcell.ColorWhite, false)
	assertScrollbarCell(t, screen, 0, 0, ' ', tcell.ColorDefault, tcell.ColorDarkGray, false)
}

func TestDrawVerticalOversizedThumb(t *testing.T) {
	track := NewScrollbarTrack()
	track.SetRect(0, 0, 1, 10)
	track.SetPosition(5)
	track.SetThumbSize(200)
	screen := newScrollbarScreen(t, 1, 10)

	track.Draw(screen)

	// A thumb larger than max is clamped to max, covering the whole track.
	for y := 0; y < 10; y++ {
		assertScrollbarCell(t, screen, 0, y, ' ', tcell.ColorDarkGray, tcell.ColorWhite, false)
	}
}

func TestDrawHorizontalTrackAndThumb(t *testing.T) {
	track := NewScrollbarTrack()
	track.SetRect(0, 0, 10, 1)
	track.SetHorizontal(true)
	screen := newScrollbarScreen(t, 10, 1)

	track.Draw(screen)

	assertScrollbarCell(t, screen, 0, 0, ' ', tcell.ColorDarkGray, tcell.ColorWhite, false)
	for x := 1; x < 10; x++ {
		assertScrollbarCell(t, screen, x, 0, ' ', tcell.ColorDefault, tcell.ColorDarkGray, false)
	}
}

func TestDrawThinHorizontal(t *testing.T) {
	track := NewScrollbarTrack()
	track.SetRect(0, 0, 10, 1)
	track.SetHorizontal(true)
	track.SetThin(true)
	track.SetBackgroundColor(tcell.ColorBlack)
	screen := newScrollbarScreen(t, 10, 1)

	track.Draw(screen)

	// The whole row uses the upper-half block; the first column is the thumb.
	assertScrollbarCell(t, screen, 0, 0, '▀', tcell.ColorWhite, tcell.ColorBlack, false)
	for x := 1; x < 10; x++ {
		assertScrollbarCell(t, screen, x, 0, '▀', tcell.ColorDarkGray, tcell.ColorBlack, false)
	}
}

func TestDrawBeforeDrawAndEmptyRect(t *testing.T) {
	track := NewScrollbarTrack()
	track.SetRect(0, 0, 0, 0)
	screen := newScrollbarScreen(t, 5, 5)
	screen.Fill('X', tcell.StyleDefault)

	calls := 0
	track.SetBeforeDrawFunc(func(tcell.Screen) { calls++ })

	track.Draw(screen)
	if calls != 1 {
		t.Errorf("beforeDraw calls = %d, want 1", calls)
	}
	if got, _, _, _ := screen.GetContent(0, 0); got != 'X' {
		t.Errorf("cell drawn despite empty rect: %q", got)
	}

	track.SetRect(0, 0, 1, 1)
	track.Draw(screen)
	if calls != 2 {
		t.Errorf("beforeDraw calls = %d, want 2", calls)
	}
	if got, _, _, _ := screen.GetContent(0, 0); got != '▀' {
		t.Errorf("cell rune = %q, want a drawn half block", got)
	}
}

func TestMouseHandlerVerticalDrag(t *testing.T) {
	track := NewScrollbarTrack()
	track.SetRect(0, 0, 1, 10)
	var changed []int
	track.SetChangedFunc(func(position int) {
		changed = append(changed, position)
	})
	handler := track.MouseHandler()
	setFocus := func(tview.Primitive) {}

	consumed, capture := handler(tview.MouseLeftDown, tcell.NewEventMouse(0, 5, tcell.Button1, tcell.ModNone), setFocus)
	if !consumed || capture != tview.Primitive(track) {
		t.Errorf("left down = (consumed=%v, capture=%v), want (true, track)", consumed, capture)
	}
	if !track.dragging {
		t.Error("dragging = false after left down, want true")
	}
	if got, want := track.Position(), 45; got != want {
		t.Errorf("position after click = %d, want %d", got, want)
	}
	if want := []int{45}; len(changed) != 1 || changed[0] != want[0] {
		t.Errorf("changed = %v, want %v", changed, want)
	}

	consumed, capture = handler(tview.MouseMove, tcell.NewEventMouse(0, 9, tcell.Button1, tcell.ModNone), setFocus)
	if !consumed || capture != tview.Primitive(track) {
		t.Errorf("move = (consumed=%v, capture=%v), want (true, track)", consumed, capture)
	}
	if got, want := track.Position(), 85; got != want {
		t.Errorf("position after move = %d, want %d", got, want)
	}
	if want := []int{45, 85}; len(changed) != 2 || changed[0] != want[0] || changed[1] != want[1] {
		t.Errorf("changed = %v, want %v", changed, want)
	}

	consumed, capture = handler(tview.MouseLeftUp, tcell.NewEventMouse(0, 9, tcell.ButtonNone, tcell.ModNone), setFocus)
	if !consumed || capture != nil {
		t.Errorf("left up = (consumed=%v, capture=%v), want (true, nil)", consumed, capture)
	}
	if track.dragging {
		t.Error("dragging = true after left up, want false")
	}
	if got := track.Position(); got != 85 {
		t.Errorf("position after release = %d, want 85", got)
	}
}

func TestMouseHandlerDragClampsAndReleases(t *testing.T) {
	track := NewScrollbarTrack()
	track.SetRect(0, 0, 1, 10)
	var changed []int
	track.SetChangedFunc(func(position int) { changed = append(changed, position) })
	handler := track.MouseHandler()
	setFocus := func(tview.Primitive) {}

	handler(tview.MouseLeftDown, tcell.NewEventMouse(0, 5, tcell.Button1, tcell.ModNone), setFocus)

	// Dragging above the track clamps to zero.
	handler(tview.MouseMove, tcell.NewEventMouse(0, -3, tcell.Button1, tcell.ModNone), setFocus)
	if got := track.Position(); got != 0 {
		t.Errorf("position after dragging above track = %d, want 0", got)
	}

	// Dragging below the track clamps to max-thumbSize.
	handler(tview.MouseMove, tcell.NewEventMouse(0, 200, tcell.Button1, tcell.ModNone), setFocus)
	if got := track.Position(); got != 90 {
		t.Errorf("position after dragging below track = %d, want 90", got)
	}

	// A move with no button held ends the drag and releases capture.
	consumed, capture := handler(tview.MouseMove, tcell.NewEventMouse(0, 5, tcell.ButtonNone, tcell.ModNone), setFocus)
	if !consumed || capture != nil {
		t.Errorf("buttonless move = (consumed=%v, capture=%v), want (true, nil)", consumed, capture)
	}
	if track.dragging {
		t.Error("dragging should end when no button is held")
	}
	if want := []int{45, 0, 90}; len(changed) != len(want) || changed[0] != want[0] || changed[1] != want[1] || changed[2] != want[2] {
		t.Errorf("changed = %v, want %v", changed, want)
	}
}

func TestMouseHandlerIgnores(t *testing.T) {
	track := NewScrollbarTrack()
	track.SetRect(0, 0, 1, 10)
	handler := track.MouseHandler()
	setFocus := func(tview.Primitive) {}

	tests := []struct {
		name   string
		x, y   int
		action tview.MouseAction
	}{
		{name: "right of the track", x: 2, y: 5, action: tview.MouseLeftDown},
		{name: "left of the track", x: -1, y: 5, action: tview.MouseLeftDown},
		{name: "below the track", x: 0, y: 10, action: tview.MouseLeftDown},
		{name: "above the track", x: 0, y: -1, action: tview.MouseLeftDown},
		{name: "unhandled action", x: 0, y: 5, action: tview.MouseMiddleDown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			consumed, capture := handler(tc.action, tcell.NewEventMouse(tc.x, tc.y, tcell.Button1, tcell.ModNone), setFocus)
			if consumed || capture != nil {
				t.Errorf("handler = (consumed=%v, capture=%v), want (false, nil)", consumed, capture)
			}
			if track.dragging {
				t.Error("dragging started for an ignored event")
			}
		})
	}

	// A thumb as large as the maximum makes the bar non-interactive.
	track.SetThumbSize(100)
	consumed, capture := handler(tview.MouseLeftDown, tcell.NewEventMouse(0, 5, tcell.Button1, tcell.ModNone), setFocus)
	if consumed || capture != nil {
		t.Errorf("full-thumb click = (consumed=%v, capture=%v), want (false, nil)", consumed, capture)
	}
}

func TestMouseHandlerScrollWheel(t *testing.T) {
	track := NewScrollbarTrack()
	track.SetRect(0, 0, 1, 10)
	var changed []int
	track.SetChangedFunc(func(position int) { changed = append(changed, position) })
	handler := track.MouseHandler()
	setFocus := func(tview.Primitive) {}

	consumed, capture := handler(tview.MouseScrollDown, tcell.NewEventMouse(0, 5, tcell.ButtonNone, tcell.ModNone), setFocus)
	if !consumed || capture != nil {
		t.Errorf("scroll down = (consumed=%v, capture=%v), want (true, nil)", consumed, capture)
	}
	if got, want := track.Position(), 5; got != want {
		t.Errorf("position after scroll down = %d, want %d", got, want)
	}

	handler(tview.MouseScrollDown, tcell.NewEventMouse(0, 5, tcell.ButtonNone, tcell.ModNone), setFocus)
	if got, want := track.Position(), 10; got != want {
		t.Errorf("position after second scroll down = %d, want %d", got, want)
	}

	handler(tview.MouseScrollUp, tcell.NewEventMouse(0, 5, tcell.ButtonNone, tcell.ModNone), setFocus)
	if got, want := track.Position(), 5; got != want {
		t.Errorf("position after scroll up = %d, want %d", got, want)
	}
	if want := []int{5, 10, 5}; len(changed) != len(want) || changed[0] != want[0] || changed[1] != want[1] || changed[2] != want[2] {
		t.Errorf("changed = %v, want %v", changed, want)
	}
}

func TestMouseHandlerHorizontal(t *testing.T) {
	track := NewScrollbarTrack()
	track.SetRect(0, 0, 10, 1)
	track.SetHorizontal(true)
	handler := track.MouseHandler()
	setFocus := func(tview.Primitive) {}

	consumed, capture := handler(tview.MouseLeftDown, tcell.NewEventMouse(5, 0, tcell.Button1, tcell.ModNone), setFocus)
	if !consumed || capture != tview.Primitive(track) {
		t.Errorf("horizontal click = (consumed=%v, capture=%v), want (true, track)", consumed, capture)
	}
	if got, want := track.Position(), 45; got != want {
		t.Errorf("position after horizontal click = %d, want %d", got, want)
	}

	handler(tview.MouseMove, tcell.NewEventMouse(9, 0, tcell.Button1, tcell.ModNone), setFocus)
	if got, want := track.Position(), 85; got != want {
		t.Errorf("position after horizontal move = %d, want %d", got, want)
	}

	handler(tview.MouseLeftUp, tcell.NewEventMouse(9, 0, tcell.ButtonNone, tcell.ModNone), setFocus)

	// The minor axis is the row; a click on another row is ignored.
	consumed, capture = handler(tview.MouseLeftDown, tcell.NewEventMouse(5, 1, tcell.Button1, tcell.ModNone), setFocus)
	if consumed || capture != nil {
		t.Errorf("click on wrong row = (consumed=%v, capture=%v), want (false, nil)", consumed, capture)
	}
}

func TestSetPositionFromMajor(t *testing.T) {
	track := NewScrollbarTrack()
	track.SetMax(100)
	track.SetThumbSize(10)
	var changed []int
	track.SetChangedFunc(func(position int) { changed = append(changed, position) })

	tests := []struct {
		name        string
		majorAxis   int
		majorLength int
		want        int
	}{
		{name: "before start clamps to zero", majorAxis: -1, majorLength: 10, want: 0},
		{name: "start", majorAxis: 0, majorLength: 10, want: 0},
		{name: "middle", majorAxis: 5, majorLength: 10, want: 45},
		{name: "near end", majorAxis: 9, majorLength: 10, want: 85},
		{name: "end", majorAxis: 10, majorLength: 10, want: 90},
		{name: "past end clamps", majorAxis: 100, majorLength: 10, want: 90},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			track.SetPosition(0)
			track.setPositionFromMajor(tc.majorAxis, tc.majorLength)
			if got := track.Position(); got != tc.want {
				t.Errorf("setPositionFromMajor(%d,%d) position = %d, want %d", tc.majorAxis, tc.majorLength, got, tc.want)
			}
		})
	}
	if len(changed) != len(tests) {
		t.Errorf("changed callback count = %d, want %d", len(changed), len(tests))
	}

	// A non-positive major length is ignored.
	before := len(changed)
	track.setPositionFromMajor(5, 0)
	if got := track.Position(); got != 90 {
		t.Errorf("position changed for zero major length: %d", got)
	}
	if len(changed) != before {
		t.Errorf("changed callback fired for zero major length")
	}

	// A one-cell thumb has no half-thumb offset.
	track.SetThumbSize(1)
	track.setPositionFromMajor(5, 10)
	if got := track.Position(); got != 50 {
		t.Errorf("position with thumb size 1 = %d, want 50", got)
	}
}

func TestNewScrollbar(t *testing.T) {
	sb := NewScrollbar()
	if sb.Flex == nil || sb.Track == nil || sb.UpButton == nil || sb.DownButton == nil {
		t.Fatal("NewScrollbar left a field nil")
	}
	if got := sb.UpButton.GetLabel(); got != "▲" {
		t.Errorf("up button label = %q, want ▲", got)
	}
	if got := sb.DownButton.GetLabel(); got != "▼" {
		t.Errorf("down button label = %q, want ▼", got)
	}
	if sb.Track.isHorizontal {
		t.Error("new scrollbar track should be vertical")
	}
}

func TestScrollbarSetHorizontalAndThin(t *testing.T) {
	sb := NewScrollbar()

	sb.SetHorizontal(true)
	if !sb.Track.isHorizontal {
		t.Error("track not marked horizontal")
	}
	if got := sb.UpButton.GetLabel(); got != "◄" {
		t.Errorf("up button label = %q, want ◄", got)
	}
	if got := sb.DownButton.GetLabel(); got != "►" {
		t.Errorf("down button label = %q, want ►", got)
	}

	sb.SetHorizontal(false)
	if sb.Track.isHorizontal {
		t.Error("track still marked horizontal")
	}
	if got := sb.UpButton.GetLabel(); got != "▲" {
		t.Errorf("up button label = %q, want ▲", got)
	}
	if got := sb.DownButton.GetLabel(); got != "▼" {
		t.Errorf("down button label = %q, want ▼", got)
	}

	sb.SetThin(true)
	if !sb.Track.thin {
		t.Error("SetThin did not mark the track thin")
	}
	sb.SetThin(false)
	if sb.Track.thin {
		t.Error("SetThin(false) did not clear the track's thin flag")
	}
}

func TestScrollbarSetChangedFunc(t *testing.T) {
	sb := NewScrollbar()
	var got []int
	sb.SetChangedFunc(func(position int) { got = append(got, position) })

	if sb.changedFunc == nil {
		t.Error("scrollbar changedFunc is nil")
	}
	if sb.Track.changedFunc == nil {
		t.Error("track changedFunc is nil")
	}

	sb.Track.setPositionFromMajor(5, 10)
	if want := []int{45}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("changed callback = %v, want %v", got, want)
	}
}

func TestScrollbarButtonsChangePosition(t *testing.T) {
	sb := NewScrollbar()
	var got []int
	sb.SetChangedFunc(func(position int) { got = append(got, position) })
	enter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	setFocus := func(tview.Primitive) {}

	sb.DownButton.InputHandler()(enter, setFocus)
	if pos := sb.Track.Position(); pos != 5 {
		t.Errorf("position after down button = %d, want 5", pos)
	}
	sb.UpButton.InputHandler()(enter, setFocus)
	if pos := sb.Track.Position(); pos != 0 {
		t.Errorf("position after up button = %d, want 0", pos)
	}
	if want := []int{5, 0}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("changed callbacks = %v, want %v", got, want)
	}

	// Pressing up while already at the top reports the clamped position.
	got = nil
	sb.UpButton.InputHandler()(enter, setFocus)
	if want := []int{0}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("changed callbacks at top = %v, want %v", got, want)
	}
}

func TestScrollbarDrawVerticalAndUpdateHook(t *testing.T) {
	sb := NewScrollbar()
	sb.SetRect(0, 0, 1, 10)
	screen := newScrollbarScreen(t, 1, 10)

	calls := 0
	sb.UpdateHook = func(got *Scrollbar) {
		calls++
		if got != sb {
			t.Errorf("UpdateHook received %p, want %p", got, sb)
		}
		got.Track.SetThumbSize(5)
	}

	sb.Draw(screen)
	if calls != 1 {
		t.Errorf("UpdateHook calls = %d, want 1", calls)
	}
	if got := sb.Track.ThumbSize(); got != 5 {
		t.Errorf("UpdateHook did not update thumb size: %d", got)
	}

	// Track occupies the first eight rows; the two buttons take the last two.
	// With thumb size 5 in eight rows the thumb is drawn as a single upper
	// half-cell block in the first row.
	assertScrollbarCell(t, screen, 0, 0, '▀', tcell.ColorDarkGray, tcell.ColorWhite, true)
	assertScrollbarCell(t, screen, 0, 7, ' ', tcell.ColorDefault, tcell.ColorDarkGray, false)
	upRune, _, _, _ := screen.GetContent(0, 8)
	if upRune != '▲' {
		t.Errorf("row 8 rune = %q, want ▲", upRune)
	}
	downRune, _, _, _ := screen.GetContent(0, 9)
	if downRune != '▼' {
		t.Errorf("row 9 rune = %q, want ▼", downRune)
	}

	sb.Draw(screen)
	if calls != 2 {
		t.Errorf("UpdateHook calls after second draw = %d, want 2", calls)
	}
}

func TestScrollbarDrawHorizontal(t *testing.T) {
	sb := NewScrollbar()
	sb.SetRect(0, 0, 10, 1)
	sb.SetHorizontal(true)
	screen := newScrollbarScreen(t, 10, 1)

	sb.Draw(screen)

	// Track is the first eight columns, then the two arrow buttons.
	assertScrollbarCell(t, screen, 0, 0, ' ', tcell.ColorDarkGray, tcell.ColorWhite, false)
	assertScrollbarCell(t, screen, 7, 0, ' ', tcell.ColorDefault, tcell.ColorDarkGray, false)
	left, _, _, _ := screen.GetContent(8, 0)
	if left != '◄' {
		t.Errorf("column 8 rune = %q, want ◄", left)
	}
	right, _, _, _ := screen.GetContent(9, 0)
	if right != '►' {
		t.Errorf("column 9 rune = %q, want ►", right)
	}
}
