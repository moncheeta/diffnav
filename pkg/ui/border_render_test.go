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
	return renderPanesWith(t, config.DefaultConfig(), mutate)
}

func renderPanesWith(t *testing.T, cfg config.Config, mutate func(*mainModel)) []string {
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
		{"diff focused", func(m *mainModel) { m.activePanel = DiffViewerPanel }, "┮", false},
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

// With the header hidden there is no rule above the panes, so the divider is
// the only thing left to carry focus: heavy while the sidebar has it, light
// while the diff does.
func TestDividerFollowsFocus(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mutate    func(*mainModel)
		wantHeavy bool
	}{
		{"tree focused", func(m *mainModel) { m.activePanel = FileTreePanel }, true},
		{"file search list", func(m *mainModel) { m.searchingFiles = true }, true},
		{"diff focused", func(m *mainModel) { m.activePanel = DiffViewerPanel }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, hideHeader := range []bool{false, true} {
				cfg := config.DefaultConfig()
				cfg.UI.HideHeader = hideHeader
				rows := renderPanesWith(t, cfg, tc.mutate)

				heavy := 0
				for _, r := range rows {
					if strings.Contains(r, "┃") {
						heavy++
						if !strings.Contains(r, focusFg) {
							t.Fatalf("hideHeader=%v: heavy divider not in focus colour: %q",
								hideHeader, r)
						}
					}
				}
				if (heavy > 0) != tc.wantHeavy {
					t.Fatalf("hideHeader=%v: heavy divider rows = %d, want heavy=%v",
						hideHeader, heavy, tc.wantHeavy)
				}
			}
		})
	}
}

// Hiding the header must take the rule above the panes with it -- otherwise a
// stray line is left at the very top, and headerHeight (which budgets for
// both rows) leaves the layout one row over.
func TestHideHeaderRemovesTopRule(t *testing.T) {
	shown := config.DefaultConfig()
	hidden := config.DefaultConfig()
	hidden.UI.HideHeader = true

	focusTree := func(m *mainModel) { m.activePanel = FileTreePanel }
	withHeader := renderPanesWith(t, shown, focusTree)
	without := renderPanesWith(t, hidden, focusTree)

	firstShown := ansiSeq.ReplaceAllString(withHeader[0], "")
	if !strings.Contains(strings.ToUpper(firstShown), "DIFFNAV") {
		t.Fatalf("expected the header first with it shown, got %q", firstShown)
	}
	if !strings.ContainsAny(ansiSeq.ReplaceAllString(withHeader[1], ""), "━─") {
		t.Fatal("expected the rule directly under the header")
	}

	for i, r := range without {
		plain := ansiSeq.ReplaceAllString(r, "")
		if strings.Contains(strings.ToUpper(plain), "DIFFNAV") {
			t.Fatalf("row %d still shows the header: %q", i, plain)
		}
	}
	first := ansiSeq.ReplaceAllString(without[0], "")
	if strings.HasPrefix(first, "━") || strings.HasPrefix(first, "─") {
		t.Fatalf("stray rule left at the top: %q", first)
	}
	if len(without) >= len(withHeader) {
		t.Fatalf("hiding the header should free rows: %d without vs %d with",
			len(without), len(withHeader))
	}
}
