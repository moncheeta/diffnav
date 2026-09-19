package diffviewer

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
)

// point is a cell in the diff viewer's rendered output: a row of the pane and
// a visual column within that row.
type point struct {
	row, col int
}

func (p point) before(o point) bool {
	if p.row != o.row {
		return p.row < o.row
	}
	return p.col < o.col
}

// selection is a character range over the rendered pane. It is anchored to the
// screen rather than to the content, so scrolling clears it instead of leaving
// the highlight sitting over different text.
type selection struct {
	active bool
	anchor point
	head   point
}

// ordered returns the two ends in reading order.
func (s selection) ordered() (point, point) {
	if s.head.before(s.anchor) {
		return s.head, s.anchor
	}
	return s.anchor, s.head
}

// empty reports whether the selection covers no cells, which is what a plain
// click without a drag produces.
func (s selection) empty() bool {
	return !s.active || s.anchor == s.head
}

// span returns the half-open column range selected on the given row, clamped
// to the row's width, and whether any of it is selected at all.
//
// Both ends of the drag are inclusive, so the character the drag started on is
// selected whichever way it is dragged. Only the far end is turned into an
// exclusive bound here.
func (s selection) span(row, width int) (int, int, bool) {
	start, end := s.ordered()
	if row < start.row || row > end.row {
		return 0, 0, false
	}
	from, to := 0, width
	if row == start.row {
		from = start.col
	}
	if row == end.row {
		to = end.col + 1
	}
	from = min(max(from, 0), width)
	to = min(max(to, 0), width)
	if to <= from {
		return 0, 0, false
	}
	return from, to, true
}

// extract pulls the selected text out of the rendered lines, without styling
// and without the trailing blanks that pad each row out to the pane width.
func (s selection) extract(lines []string) string {
	if s.empty() {
		return ""
	}
	start, end := s.ordered()

	var out []string
	for row := max(start.row, 0); row <= end.row && row < len(lines); row++ {
		line := lines[row]
		from, to, ok := s.span(row, ansi.StringWidth(line))
		if !ok {
			out = append(out, "")
			continue
		}
		out = append(out, strings.TrimRight(ansi.Strip(ansi.Cut(line, from, to)), " "))
	}
	return strings.Join(out, "\n")
}

// apply draws the selection over the rendered pane. The selected span is
// re-rendered flat so the highlight reads as one solid block, the way a
// terminal's own selection does; the text around it keeps its colours.
func (s selection) apply(view string) string {
	if s.empty() {
		return view
	}
	style := lipgloss.NewStyle().
		Background(common.Colors[common.SelectBg]).
		Foreground(common.Colors[common.SelectFg])

	start, end := s.ordered()
	lines := strings.Split(view, "\n")
	for row := max(start.row, 0); row <= end.row && row < len(lines); row++ {
		line := lines[row]
		width := ansi.StringWidth(line)
		from, to, ok := s.span(row, width)
		if !ok {
			continue
		}
		lines[row] = ansi.Cut(line, 0, from) +
			style.Render(ansi.Strip(ansi.Cut(line, from, to))) +
			ansi.Cut(line, to, width)
	}
	return strings.Join(lines, "\n")
}

// wordAnchor is the word a double-click landed on. A drag starting from it
// extends the selection a whole word at a time, so the word it began on always
// stays intact.
type wordAnchor struct {
	active   bool
	row      int
	from, to int // inclusive cell range
}

// isWordRune decides what a double-click grabs. Letters, digits and
// underscores, which is the usual terminal convention: double-clicking
// `foo.bar` picks out `foo`, not the whole expression.
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// wordAt finds the word covering a visual column of plain (unstyled) text and
// returns its inclusive cell range. Columns are cells rather than rune indices,
// so wide characters stay lined up with what is on screen.
func wordAt(plain string, col int) (int, int, bool) {
	type cell struct {
		r          rune
		start, end int // end is exclusive
	}

	var cells []cell
	at := 0
	for _, r := range plain {
		w := ansi.StringWidth(string(r))
		if w == 0 {
			w = 1
		}
		cells = append(cells, cell{r: r, start: at, end: at + w})
		at += w
	}

	idx := -1
	for i, c := range cells {
		if col >= c.start && col < c.end {
			idx = i
			break
		}
	}
	if idx < 0 || !isWordRune(cells[idx].r) {
		return 0, 0, false
	}

	lo := idx
	for lo > 0 && isWordRune(cells[lo-1].r) {
		lo--
	}
	hi := idx
	for hi < len(cells)-1 && isWordRune(cells[hi+1].r) {
		hi++
	}
	return cells[lo].start, cells[hi].end - 1, true
}
