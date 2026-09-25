package ui

import (
	"strings"
	"testing"

	"github.com/dlvhdr/diffnav/pkg/config"
)

// The rule above the panes exists to divide them from the header, so hiding
// the header has to take it along. Otherwise a stray line is left at the very
// top — and headerHeight budgets for both rows, so the layout also runs one
// row over.
func TestHideHeaderRemovesTopRule(t *testing.T) {
	shown := config.DefaultConfig()
	hidden := config.DefaultConfig()
	hidden.UI.HideHeader = true

	withHeader := renderWith(t, shown, nil)
	without := renderWith(t, hidden, nil)

	if len(withHeader) < 2 {
		t.Fatalf("expected a header and a rule, got %d rows", len(withHeader))
	}
	if !strings.ContainsAny(ansiSeq.ReplaceAllString(withHeader[1], ""), "━─") {
		t.Fatalf("expected the rule directly under the header, got %q",
			ansiSeq.ReplaceAllString(withHeader[1], ""))
	}

	first := ansiSeq.ReplaceAllString(without[0], "")
	if strings.HasPrefix(first, "━") || strings.HasPrefix(first, "─") {
		t.Fatalf("stray rule left at the top with the header hidden: %q", first)
	}
	if len(without) >= len(withHeader) {
		t.Fatalf("hiding the header should free a row: %d without vs %d with",
			len(without), len(withHeader))
	}
}
