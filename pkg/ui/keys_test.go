package ui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// home/end are aliases of g/G, so they inherit the panel-aware behaviour:
// first/last file in the tree, top/bottom of the diff otherwise.
func TestTopBottomBindings(t *testing.T) {
	for _, tc := range []struct {
		name    string
		code    rune
		binding key.Binding
	}{
		{"g binds top", 'g', keys.Top},
		{"home binds top", tea.KeyHome, keys.Top},
		{"G binds bottom", 'G', keys.Bottom},
		{"end binds bottom", tea.KeyEnd, keys.Bottom},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := tea.KeyPressMsg{Code: tc.code}
			if !key.Matches(msg, tc.binding) {
				t.Fatalf("key %q does not match binding %v", msg.String(), tc.binding.Keys())
			}
		})
	}

	// home/end must not also trigger the opposite end.
	if key.Matches(tea.KeyPressMsg{Code: tea.KeyHome}, keys.Bottom) {
		t.Fatal("home should not match Bottom")
	}
	if key.Matches(tea.KeyPressMsg{Code: tea.KeyEnd}, keys.Top) {
		t.Fatal("end should not match Top")
	}
}
