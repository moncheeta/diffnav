package diffviewer

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/robinovitch61/viewport/viewport"
	"github.com/robinovitch61/viewport/viewport/item"
)

const testViewportHeight = 20

func newPagingViewport(t *testing.T) *viewport.Model[diffLine] {
	t.Helper()
	vp := viewport.New(80, testViewportHeight, viewport.WithKeyMap[diffLine](ViewportKeyMap))
	lines := make([]diffLine, 200)
	for i := range lines {
		lines[i] = diffLine{item: item.NewItem(fmt.Sprintf("line %03d", i))}
	}
	vp.SetObjects(lines)
	return vp
}

func pressKey(t *testing.T, vp *viewport.Model[diffLine], code rune, mod tea.KeyMod) int {
	t.Helper()
	vp, _ = vp.Update(tea.KeyPressMsg{Code: code, Mod: mod})
	top, _ := vp.GetTopItemIdxAndLineOffset()
	return top
}

// PageDown/PageUp are supported by the viewport but were left unbound, so
// pgdn/pgup did nothing. Guard the bindings and their relation to ctrl+d.
func TestPageKeysScrollAFullPage(t *testing.T) {
	vp := newPagingViewport(t)
	if top, _ := vp.GetTopItemIdxAndLineOffset(); top != 0 {
		t.Fatalf("want start at top, got %d", top)
	}

	pageDown := pressKey(t, vp, tea.KeyPgDown, 0)
	if pageDown == 0 {
		t.Fatal("pgdown did not scroll: binding never reached the viewport")
	}

	if pageUp := pressKey(t, vp, tea.KeyPgUp, 0); pageUp != 0 {
		t.Fatalf("pgup did not return to the top, got %d", pageUp)
	}

	halfPageDown := pressKey(t, newPagingViewport(t), 'd', tea.ModCtrl)
	if pageDown <= halfPageDown {
		t.Fatalf("want pgdown (%d) to scroll further than ctrl+d (%d)", pageDown, halfPageDown)
	}
}
