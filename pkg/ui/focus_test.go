package ui

import "testing"

// The file tree and the file search list share one border, so focus in either
// must light it up. Previously searching left both borders muted.
func TestSidebarFocused(t *testing.T) {
	for _, tc := range []struct {
		name           string
		model          mainModel
		wantSidebarLit bool
	}{
		{
			name:           "tree focused",
			model:          mainModel{activePanel: FileTreePanel},
			wantSidebarLit: true,
		},
		{
			name:           "file search list focused",
			model:          mainModel{activePanel: FileTreePanel, searchingFiles: true},
			wantSidebarLit: true,
		},
		{
			name:           "file search list focused while diff panel active",
			model:          mainModel{activePanel: DiffViewerPanel, searchingFiles: true},
			wantSidebarLit: true,
		},
		{
			name:           "diff focused",
			model:          mainModel{activePanel: DiffViewerPanel},
			wantSidebarLit: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.model.sidebarFocused(); got != tc.wantSidebarLit {
				t.Fatalf("sidebarFocused() = %v, want %v", got, tc.wantSidebarLit)
			}
		})
	}
}
