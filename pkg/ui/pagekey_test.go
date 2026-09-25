package ui

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/dlvhdr/diffnav/pkg/config"
	zone "github.com/lrstanley/bubblezone/v2"
)

// loadedModel is a mainModel with one file's diff actually rendered, so key
// handling can be driven against real content.
func loadedModel(t *testing.T) mainModel {
	t.Helper()
	zone.NewGlobal()

	f, err := os.Open("panes/filetree/testdata/gh_dash_pr.diff")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	files, _, err := gitdiff.Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	m := New(ModelOpts{}, config.DefaultConfig())
	m.width, m.height = 120, 30
	m.files = files
	m.fileTree = m.fileTree.SetFiles(files)
	m.diffViewer.SetSize(120, 25)

	dv, cmd := m.diffViewer.SetFilePatch(files[0])
	m.diffViewer = dv
	if cmd == nil {
		t.Fatal("SetFilePatch returned no command")
	}
	m.diffViewer, _ = m.diffViewer.Update(cmd()) // runs delta
	return m
}

// The page keys are wired through mainModel's key switch rather than the
// viewport's keymap, so drive them the way the program does. Testing PageStep's
// arithmetic alone would not notice the key never reaching it.
func TestPageKeysScrollThroughUpdate(t *testing.T) {
	m := loadedModel(t)
	if m.diffViewer.TopItemIdx() != 0 {
		t.Fatalf("expected to start at the top, got %d", m.diffViewer.TopItemIdx())
	}
	step := m.diffViewer.PageStep()
	if step < 2 {
		t.Fatalf("PageStep() = %d, too small to tell a page from a line", step)
	}

	down, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	afterDown := down.(mainModel)
	got := afterDown.diffViewer.TopItemIdx()
	if got == 0 {
		t.Fatal("pgdown did not scroll: the key never reached the handler")
	}
	if got != step {
		t.Errorf("pgdown scrolled %d lines, want PageStep() = %d", got, step)
	}

	up, _ := afterDown.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	if back := up.(mainModel).diffViewer.TopItemIdx(); back != 0 {
		t.Errorf("pgup returned to %d, want back to the top", back)
	}
}

// A page must move further than a single line, which is what the arrow keys do.
func TestPageKeyMovesFurtherThanArrow(t *testing.T) {
	// A fresh model per key: the diff viewer holds a pointer to the viewport,
	// so two updates branched off one model would share its scroll position.
	page, _ := loadedModel(t).Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	line, _ := loadedModel(t).Update(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})

	pageTop := page.(mainModel).diffViewer.TopItemIdx()
	lineTop := line.(mainModel).diffViewer.TopItemIdx()
	if pageTop <= lineTop {
		t.Fatalf("pgdown moved to %d, ctrl+e to %d: a page must move further", pageTop, lineTop)
	}
}
