package ui

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/dlvhdr/diffnav/pkg/config"
	zone "github.com/lrstanley/bubblezone/v2"
)

const (
	focusFg = "38;2;32;94;166"   // Flexoki blue-600
	mutedFg = "38;2;206;205;195" // Flexoki base-200
)

var ansiSeq = regexp.MustCompile("\x1b\\[[0-9;]*m")

func renderPanes(t *testing.T, mutate func(*mainModel)) []string {
	t.Helper()
	zone.NewGlobal()

	f, err := os.Open("panes/filetree/testdata/multiple_files.diff")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	defer f.Close()
	files, _, err := gitdiff.Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	m := New("", cfg)
	m.width, m.height = 120, 30
	m.files = files
	m.fileTree = m.fileTree.SetFiles(files)
	m.fileTree.SetSize(cfg.UI.FileTreeWidth, 20)
	m.diffViewer.SetSize(120-cfg.UI.FileTreeWidth, 20)
	mutate(&m)

	return strings.Split(m.View().Content, "\n")
}

// The separator row is the rule above the panes: heavy on the focused side,
// joined to the divider by a junction glyph.
func TestFocusedPaneGetsHeavyRule(t *testing.T) {
	for _, tc := range []struct {
		name         string
		mutate       func(*mainModel)
		wantJunction string
		leftHeavy    bool
	}{
		{"tree focused", func(m *mainModel) { m.activePanel = FileTreePanel }, "┱", true},
		{"file search list", func(m *mainModel) { m.searchingFiles = true }, "┱", true},
		{"diff focused", func(m *mainModel) { m.activePanel = DiffViewerPanel }, "┲", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := renderPanes(t, tc.mutate)
			var sep string
			for _, r := range rows {
				if strings.Contains(ansiSeq.ReplaceAllString(r, ""), tc.wantJunction) {
					sep = r
					break
				}
			}
			if sep == "" {
				t.Fatalf("no separator row containing junction %q", tc.wantJunction)
			}

			left, _, ok := strings.Cut(ansiSeq.ReplaceAllString(sep, ""), tc.wantJunction)
			if !ok {
				t.Fatal("junction not found after stripping ansi")
			}
			right := ansiSeq.ReplaceAllString(sep, "")[len(left)+len(tc.wantJunction):]

			if gotLeftHeavy := strings.Contains(left, "━"); gotLeftHeavy != tc.leftHeavy {
				t.Errorf("left rule heavy = %v, want %v (%q)", gotLeftHeavy, tc.leftHeavy, left)
			}
			if gotRightHeavy := strings.Contains(right, "━"); gotRightHeavy == tc.leftHeavy {
				t.Errorf("right rule heavy = %v, want %v (%q)", gotRightHeavy, !tc.leftHeavy, right)
			}

			// The heavy half must be drawn in the focus colour, the other muted.
			wantFocusFirst := tc.leftHeavy
			if strings.HasPrefix(sep, "\x1b["+focusFg+"m") != wantFocusFirst {
				t.Errorf("focus colour on wrong side: %q", sep[:40])
			}
			if !strings.Contains(sep, mutedFg) {
				t.Error("expected the unfocused half to be muted")
			}
		})
	}
}

// The divider is the shared edge of both panes, so it stays active regardless
// of which one has focus -- otherwise the focused pane's outline is left open.
func TestSharedDividerAlwaysActive(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*mainModel)
	}{
		{"tree focused", func(m *mainModel) { m.activePanel = FileTreePanel }},
		{"file search list", func(m *mainModel) { m.searchingFiles = true }},
		{"diff focused", func(m *mainModel) { m.activePanel = DiffViewerPanel }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := renderPanes(t, tc.mutate)

			found := 0
			for _, r := range rows {
				if !strings.Contains(r, "┃") {
					continue
				}
				found++
				if !strings.Contains(r, focusFg) {
					t.Fatalf("divider row not in focus colour: %q", r)
				}
			}
			if found == 0 {
				t.Fatal("no heavy divider rendered")
			}
		})
	}
}
