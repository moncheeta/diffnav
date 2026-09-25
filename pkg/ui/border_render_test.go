package ui

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"
	"testing"

	"github.com/dlvhdr/diffnav/pkg/config"
	zone "github.com/lrstanley/bubblezone/v2"
)

var ansiSeq = regexp.MustCompile("\x1b\\[[0-9;]*m")

// renderWith renders the whole app and returns its lines.
func renderWith(t *testing.T, cfg config.Config, mutate func(*mainModel)) []string {
	t.Helper()
	zone.NewGlobal()

	m := New(ModelOpts{}, cfg)
	m.width, m.height = 120, 30
	m.diffViewer.SetSize(120, 25)
	if mutate != nil {
		mutate(&m)
	}
	return strings.Split(m.View().Content, "\n")
}

// ansiFg is the escape a foreground colour renders as. Derived from the tint
// rather than written out: the chrome follows the theme, so a fixed hex only
// holds for whichever theme the machine happened to be on.
func ansiFg(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("38;2;%d;%d;%d", r>>8, g>>8, b>>8)
}

func paneColors(t *testing.T) (focus, muted string) {
	t.Helper()
	m := New(ModelOpts{}, config.DefaultConfig())
	return ansiFg(m.styles.Tint.Blue), ansiFg(m.styles.Tint.Black)
}

// The rule above the panes is heavy on the focused side and light on the other,
// joined to the divider by a junction glyph, so the panes read apart without
// depending on colour alone.
func TestFocusedPaneGetsHeavyRule(t *testing.T) {
	focus, muted := paneColors(t)

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
			rows := renderWith(t, config.DefaultConfig(), tc.mutate)

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

			plain := ansiSeq.ReplaceAllString(sep, "")
			left, right, ok := strings.Cut(plain, tc.wantJunction)
			if !ok {
				t.Fatal("junction not found after stripping ansi")
			}

			if got := strings.Contains(left, "━"); got != tc.leftHeavy {
				t.Errorf("left rule heavy = %v, want %v (%q)", got, tc.leftHeavy, left)
			}
			if got := strings.Contains(right, "━"); got == tc.leftHeavy {
				t.Errorf("right rule heavy = %v, want %v (%q)", got, !tc.leftHeavy, right)
			}
			if strings.HasPrefix(sep, "\x1b["+focus+"m") != tc.leftHeavy {
				t.Errorf("focus colour on the wrong side: %q", sep[:40])
			}
			if !strings.Contains(sep, muted) {
				t.Error("expected the unfocused half to be muted")
			}
		})
	}
}

// The divider carries focus as well: it is the only pane chrome left once the
// header, and with it the rule, is hidden.
func TestDividerFollowsFocus(t *testing.T) {
	focus, _ := paneColors(t)

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

				heavy := 0
				for _, r := range renderWith(t, cfg, tc.mutate) {
					if strings.Contains(r, "┃") {
						heavy++
						if !strings.Contains(r, focus) {
							t.Fatalf("hideHeader=%v: heavy divider not in the focus colour: %q",
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
