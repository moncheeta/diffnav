package diffviewer

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/robinovitch61/viewport/viewport/item"
)

func sel(aRow, aCol, hRow, hCol int) selection {
	return selection{
		active: true,
		anchor: point{row: aRow, col: aCol},
		head:   point{row: hRow, col: hCol},
	}
}

func TestExtract(t *testing.T) {
	lines := []string{
		"const createMany = vi.fn(",
		"  async (args: {",
		"    data: { id: string; }",
	}

	for _, tc := range []struct {
		name string
		s    selection
		want string
	}{
		{"within one row", sel(0, 6, 0, 16), "createMany"},
		{"whole first row", sel(0, 0, 0, 25), "const createMany = vi.fn("},
		{"across rows", sel(0, 19, 2, 8), "vi.fn(\n  async (args: {\n    data"},
		{"dragged backwards is the same", sel(2, 8, 0, 19), "vi.fn(\n  async (args: {\n    data"},
		{"click with no drag selects nothing", sel(1, 4, 1, 4), ""},
		{"zero-width row yields a blank line", sel(0, 19, 1, 0), "vi.fn(\n"},
		{"past the end of a row clamps", sel(1, 2, 1, 500), "async (args: {"},
		{"past the last row clamps", sel(2, 0, 9, 0), "    data: { id: string; }"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.extract(lines); got != tc.want {
				t.Fatalf("extract() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractStripsStylingAndPadding(t *testing.T) {
	// Styled like delta renders it, padded out to the pane width.
	lines := []string{
		"\x1b[38;2;175;48;41mconst\x1b[0m \x1b[38;2;16;15;15mx\x1b[0m = \x1b[38;2;94;64;157m1\x1b[0m        ",
	}
	got := sel(0, 0, 0, ansi.StringWidth(lines[0])).extract(lines)

	if strings.Contains(got, "\x1b") {
		t.Fatalf("copied text still carries escape codes: %q", got)
	}
	if got != "const x = 1" {
		t.Fatalf("extract() = %q, want %q (padding must be trimmed)", got, "const x = 1")
	}
}

func TestApplyKeepsWidthAndSurroundingColour(t *testing.T) {
	line := "\x1b[31mhello\x1b[0m world \x1b[32mtail\x1b[0m"
	before := ansi.StringWidth(line)

	out := sel(0, 6, 0, 11).apply(line)
	if got := ansi.StringWidth(out); got != before {
		t.Fatalf("apply() changed the row width: %d -> %d", before, got)
	}
	if plain := ansi.Strip(out); plain != ansi.Strip(line) {
		t.Fatalf("apply() changed the text: %q -> %q", ansi.Strip(line), plain)
	}
	// The green tail sits after the selection and must keep its colour.
	if !strings.Contains(out, "\x1b[32m") {
		t.Fatalf("colour after the selection was lost: %q", out)
	}
}

func TestApplyIsANoopWithoutASelection(t *testing.T) {
	line := "\x1b[31mhello\x1b[0m"
	for _, s := range []selection{{}, sel(0, 3, 0, 3)} {
		if got := s.apply(line); got != line {
			t.Fatalf("apply() with empty selection changed the view: %q", got)
		}
	}
}

// Drive the real renderer: select a region of an actual pane and check what
// lands on the clipboard.
func TestSelectedTextFromRenderedPane(t *testing.T) {
	m := New(false)
	m.SetSize(40+scrollbarWidth, 10)
	m.fvp.SetObjects([]diffLine{
		{item: item.NewItem("alpha bravo charlie")},
		{item: item.NewItem("delta echo foxtrot")},
		{item: item.NewItem("golf hotel india")},
	})

	rows := strings.Split(m.baseView(), "\n")
	find := func(want string) int {
		t.Helper()
		for i, r := range rows {
			if strings.Contains(ansi.Strip(r), want) {
				return i
			}
		}
		t.Fatalf("%q not found in rendered pane:\n%s", want, ansi.Strip(m.baseView()))
		return -1
	}

	alpha, golf := find("alpha"), find("golf")

	m.BeginSelect(alpha, 6)
	m.ExtendSelect(golf, 4)
	got := m.SelectedText()

	want := "bravo charlie\ndelta echo foxtrot\ngolf"
	if got != want {
		t.Fatalf("SelectedText() = %q, want %q", got, want)
	}

	if !m.HasSelection() {
		t.Fatal("HasSelection() = false after a drag")
	}
	m.ClearSelection()
	if m.HasSelection() || m.SelectedText() != "" {
		t.Fatal("ClearSelection() left a selection behind")
	}
}
