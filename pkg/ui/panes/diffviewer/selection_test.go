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
		{"across rows", sel(0, 19, 2, 8), "vi.fn(\n  async (args: {\n    data:"},
		{"dragged backwards is the same", sel(2, 8, 0, 19), "vi.fn(\n  async (args: {\n    data:"},
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

// The character the drag starts on is part of the selection no matter which
// way it is dragged. Dragging left used to drop it, because the anchor was the
// larger column and the range excluded its far end.
func TestAnchorCharIsAlwaysSelected(t *testing.T) {
	lines := []string{"abcdef"}

	for _, tc := range []struct {
		name string
		s    selection
		want string
	}{
		{"dragged right from c", sel(0, 2, 0, 4), "cde"},
		{"dragged left from e", sel(0, 4, 0, 2), "cde"},
		{"dragged right one cell", sel(0, 2, 0, 3), "cd"},
		{"dragged left one cell", sel(0, 3, 0, 2), "cd"},
		{"dragged to the last cell", sel(0, 3, 0, 5), "def"},
		{"dragged from the last cell", sel(0, 5, 0, 3), "def"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.extract(lines); got != tc.want {
				t.Fatalf("extract() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWordAt(t *testing.T) {
	for _, tc := range []struct {
		name      string
		plain     string
		col       int
		wantFrom  int
		wantTo    int
		wantFound bool
	}{
		{"middle of a word", "const createMany = 1", 9, 6, 15, true},
		{"first cell of a word", "const createMany = 1", 6, 6, 15, true},
		{"last cell of a word", "const createMany = 1", 15, 6, 15, true},
		{"on a space finds nothing", "const createMany = 1", 5, 0, 0, false},
		{"on punctuation finds nothing", "const createMany = 1", 17, 0, 0, false},
		{"stops at a dot", "foo.bar", 1, 0, 2, true},
		{"word after a dot", "foo.bar", 5, 4, 6, true},
		{"underscores are part of a word", "snake_case_name", 7, 0, 14, true},
		{"digits are part of a word", "abc123", 4, 0, 5, true},
		{"at the start of the line", "word rest", 0, 0, 3, true},
		{"at the end of the line", "rest word", 8, 5, 8, true},
		{"past the end finds nothing", "short", 99, 0, 0, false},
		{"negative column finds nothing", "short", -1, 0, 0, false},
		{"empty line finds nothing", "", 0, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			from, to, ok := wordAt(tc.plain, tc.col)
			if ok != tc.wantFound {
				t.Fatalf("wordAt(%q, %d) found = %v, want %v", tc.plain, tc.col, ok, tc.wantFound)
			}
			if !ok {
				return
			}
			if from != tc.wantFrom || to != tc.wantTo {
				t.Fatalf("wordAt(%q, %d) = [%d,%d], want [%d,%d] (got %q)",
					tc.plain, tc.col, from, to, tc.wantFrom, tc.wantTo,
					tc.plain[from:to+1])
			}
		})
	}
}

// Columns are cells, not rune indices, so a wide character ahead of a word
// must not shift where that word is found.
func TestWordAtWithWideChars(t *testing.T) {
	const line = "日本語 foo" // each CJK rune is two cells: 0-1, 2-3, 4-5, space 6, foo 7-9

	from, to, ok := wordAt(line, 8)
	if !ok || from != 7 || to != 9 {
		t.Fatalf("wordAt(col 8) = [%d,%d] ok=%v, want [7,9]", from, to, ok)
	}

	from, to, ok = wordAt(line, 3) // the second half of 本
	if !ok || from != 0 || to != 5 {
		t.Fatalf("wordAt(col 3) = [%d,%d] ok=%v, want [0,5]", from, to, ok)
	}
}

// Double-click through the real renderer: the word under the cell, and nothing
// either side of it.
func TestSelectWordAtFromRenderedPane(t *testing.T) {
	m := New(false)
	m.SetSize(40+scrollbarWidth, 10)
	m.fvp.SetObjects([]diffLine{
		{item: item.NewItem("alpha bravo charlie")},
	})

	rows := strings.Split(m.baseView(), "\n")
	row, colOfBravo := -1, -1
	for i, r := range rows {
		if idx := strings.Index(ansi.Strip(r), "bravo"); idx >= 0 {
			row, colOfBravo = i, idx
			break
		}
	}
	if row < 0 {
		t.Fatalf("fixture not rendered:\n%s", ansi.Strip(m.baseView()))
	}

	if !m.SelectWordAt(row, colOfBravo+2) {
		t.Fatal("SelectWordAt found no word in the middle of \"bravo\"")
	}
	if got := m.SelectedText(); got != "bravo" {
		t.Fatalf("SelectedText() = %q, want %q", got, "bravo")
	}

	// The space between words is not part of any word.
	if m.SelectWordAt(row, colOfBravo-1) {
		t.Fatalf("SelectWordAt on a space selected %q", m.SelectedText())
	}
}

// Double-click then drag: the word the drag began on stays whole, and the far
// end snaps out to cover whole words too.
func TestExtendSelectByWord(t *testing.T) {
	newPane := func(t *testing.T, lines ...string) *Model {
		t.Helper()
		m := New(false)
		m.SetSize(60+scrollbarWidth, 12)
		objs := make([]diffLine, len(lines))
		for i, l := range lines {
			objs[i] = diffLine{item: item.NewItem(l)}
		}
		m.fvp.SetObjects(objs)
		return &m
	}

	// Locate a rendered row and the column a substring starts at.
	locate := func(t *testing.T, m *Model, want string) (int, int) {
		t.Helper()
		for i, r := range strings.Split(m.baseView(), "\n") {
			if idx := strings.Index(ansi.Strip(r), want); idx >= 0 {
				return i, idx
			}
		}
		t.Fatalf("%q not rendered:\n%s", want, ansi.Strip(m.baseView()))
		return -1, -1
	}

	t.Run("dragging forward takes whole words", func(t *testing.T) {
		m := newPane(t, "alpha bravo charlie delta")
		row, bravo := locate(t, m, "bravo")
		_, charlie := locate(t, m, "charlie")

		if !m.SelectWordAt(row, bravo+2) {
			t.Fatal("no word under the double-click")
		}
		// Drag into the middle of charlie: both words come whole.
		m.ExtendSelectByWord(row, charlie+3)
		if got := m.SelectedText(); got != "bravo charlie" {
			t.Fatalf("SelectedText() = %q, want %q", got, "bravo charlie")
		}
	})

	t.Run("dragging backward keeps the anchor word whole", func(t *testing.T) {
		m := newPane(t, "alpha bravo charlie delta")
		row, charlie := locate(t, m, "charlie")
		_, alpha := locate(t, m, "alpha")

		if !m.SelectWordAt(row, charlie+3) {
			t.Fatal("no word under the double-click")
		}
		m.ExtendSelectByWord(row, alpha+2)
		if got := m.SelectedText(); got != "alpha bravo charlie" {
			t.Fatalf("SelectedText() = %q, want %q", got, "alpha bravo charlie")
		}
	})

	t.Run("dragging across rows", func(t *testing.T) {
		m := newPane(t, "alpha bravo", "charlie delta")
		rowA, bravo := locate(t, m, "bravo")
		rowB, charlie := locate(t, m, "charlie")

		if !m.SelectWordAt(rowA, bravo+1) {
			t.Fatal("no word under the double-click")
		}
		m.ExtendSelectByWord(rowB, charlie+2)
		if got := m.SelectedText(); got != "bravo\ncharlie" {
			t.Fatalf("SelectedText() = %q, want %q", got, "bravo\ncharlie")
		}
	})

	t.Run("dragging onto a gap stops there, trailing space trimmed", func(t *testing.T) {
		m := newPane(t, "alpha bravo charlie")
		row, bravo := locate(t, m, "bravo")
		_, charlie := locate(t, m, "charlie")

		if !m.SelectWordAt(row, bravo+2) {
			t.Fatal("no word under the double-click")
		}
		m.ExtendSelectByWord(row, charlie-1) // the space before charlie
		// The space is in range but trimmed along with the row's padding.
		if got := m.SelectedText(); got != "bravo" {
			t.Fatalf("SelectedText() = %q, want %q", got, "bravo")
		}
	})

	t.Run("without a double-click it stays cell-wise", func(t *testing.T) {
		m := newPane(t, "alpha bravo charlie")
		row, bravo := locate(t, m, "bravo")

		m.BeginSelect(row, bravo) // plain drag, no word anchor
		m.ExtendSelectByWord(row, bravo+2)
		if got := m.SelectedText(); got != "bra" {
			t.Fatalf("SelectedText() = %q, want %q", got, "bra")
		}
	})
}
