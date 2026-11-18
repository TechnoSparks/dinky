package utility

import (
	"strings"
	"testing"
)

func TestWordWrapEmpty(t *testing.T) {
	if got := WordWrap("", 10); len(got) != 1 || got[0] != "" {
		t.Errorf("WordWrap(%q, 10) = %v, want [\"\"]", "", got)
	}
}

func TestWordWrapMaxWidthNonPositive(t *testing.T) {
	for _, w := range []int{0, -1, -100} {
		if got := WordWrap("hello world", w); got != nil {
			t.Errorf("WordWrap with maxWidth=%d returned %v, want nil", w, got)
		}
	}
}

func TestWordWrapShortText(t *testing.T) {
	got := WordWrap("hello world", 20)
	if len(got) != 1 || got[0] != "hello world" {
		t.Errorf("got %v, want [hello world]", got)
	}
}

func TestWordWrapWrapsAtSpaces(t *testing.T) {
	got := WordWrap("hello world foo bar", 11)
	want := []string{"hello", "world foo", "bar"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWordWrapForcesBreakForLongWord(t *testing.T) {
	got := WordWrap("supercalifragilisticexpialidocious", 10)
	if len(got) < 2 {
		t.Fatalf("expected wrapping, got %v", got)
	}
	for _, line := range got {
		if len([]rune(line)) > 10 {
			t.Errorf("line %q exceeds width 10", line)
		}
	}
}

func TestWordWrapPreservesMandatoryBreaks(t *testing.T) {
	got := WordWrap("line one\nline two\r\nline three", 40)
	want := []string{"line one", "line two", "line three"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWordWrapTrimsLines(t *testing.T) {
	got := WordWrap("  hello   world  ", 40)
	if len(got) != 1 || got[0] != "  hello   world  " {
		t.Errorf("got %v, want [  hello   world  ]", got)
	}
}

func TestWordWrapUnicode(t *testing.T) {
	got := WordWrap("café résumé naïve", 8)
	if len(got) < 2 {
		t.Fatalf("expected wrapping, got %v", got)
	}
	for _, line := range got {
		if strings.TrimSpace(line) == "" {
			t.Errorf("blank wrapped line")
		}
	}
}

func TestWordWrapCJK(t *testing.T) {
	got := WordWrap("日本語テスト", 4)
	// Each CJK rune is width 2, so width 4 fits two runes per line.
	if len(got) != 3 {
		t.Fatalf("got %v, want 3 lines", got)
	}
}

func TestWrapParagraph(t *testing.T) {
	got := WrapParagraph([]string{"hello", "world", "foo"}, 11)
	want := []string{"hello", "world foo"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWrapParagraphNormalizesWhitespace(t *testing.T) {
	got := WrapParagraph([]string{"  hello   world  ", "foo"}, 40)
	if len(got) != 1 || got[0] != "hello world foo" {
		t.Errorf("got %v, want [hello world foo]", got)
	}
}

func TestWrapParagraphEmpty(t *testing.T) {
	got := WrapParagraph(nil, 10)
	if len(got) != 1 || got[0] != "" {
		t.Errorf("WrapParagraph(nil, 10) = %v, want [\"\"]", got)
	}
}
