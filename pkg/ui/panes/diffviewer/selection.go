package diffviewer

import (
	"strings"

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
		to = end.col
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
