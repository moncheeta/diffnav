package diffviewer

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/robinovitch61/viewport/viewport"
	"github.com/robinovitch61/viewport/viewport/item"
)

func testLines(n int) []diffLine {
	lines := make([]diffLine, n)
	for i := range lines {
		lines[i] = diffLine{item: item.NewItem(fmt.Sprintf("line %03d", i))}
	}
	return lines
}

// libraryPageStep is how far the viewport's own paging moves: a full screen of
// content. PageStep is measured against it so the arithmetic in contentLines
// stays honest if the library changes how it accounts for header/footer.
func libraryPageStep(t *testing.T, height int, header []string) int {
	t.Helper()
	km := viewport.DefaultKeyMap()
	vp := viewport.New(80, height, viewport.WithKeyMap[diffLine](km))
	vp.SetObjects(testLines(500))
	if len(header) > 0 {
		vp.SetHeader(header)
	}
	before, _ := vp.GetTopItemIdxAndLineOffset()
	vp, _ = vp.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	after, _ := vp.GetTopItemIdxAndLineOffset()
	return after - before
}

func TestPageStepLeavesOverlap(t *testing.T) {
	for _, height := range []int{10, 20, 40} {
		for _, withHeader := range []bool{false, true} {
			name := fmt.Sprintf("h=%d header=%v", height, withHeader)
			t.Run(name, func(t *testing.T) {
				m := New(false)
				m.SetSize(80+scrollbarWidth, height)
				m.fvp.SetObjects(testLines(500))

				var header []string
				if withHeader {
					m.dir = &cachedNode{path: "some/dir"}
					header = strings.Split(m.headerView(), "\n")
					m.fvp.SetHeader(header)
				}

				want := libraryPageStep(t, height, header) - pageOverlap
				if got := m.PageStep(); got != want {
					t.Fatalf("PageStep() = %d, want %d (library pages %d)",
						got, want, want+pageOverlap)
				}
			})
		}
	}
}

// A page must never scroll further than a screen, and never stall.
func TestPageStepBounds(t *testing.T) {
	for _, height := range []int{1, 2, 3, 5, 10, 60} {
		m := New(false)
		m.SetSize(80+scrollbarWidth, height)
		m.fvp.SetObjects(testLines(500))

		step := m.PageStep()
		if step < 1 {
			t.Fatalf("height=%d: PageStep() = %d, must advance at least a line", height, step)
		}
		if visible := m.contentLines(); visible > 0 && step > visible {
			t.Fatalf("height=%d: PageStep() = %d scrolls past the %d visible lines",
				height, step, visible)
		}
	}
}
