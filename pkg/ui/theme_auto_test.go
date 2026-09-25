package ui

import (
	"testing"

	"github.com/dlvhdr/diffnav/pkg/config"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
	tint "github.com/lrstanley/bubbletint/v2"
)

func autoCfg() config.Config {
	cfg := config.DefaultConfig()
	cfg.UI.Theme = common.ThemeAuto
	cfg.UI.LightTheme = tint.TintGruvboxLight.ID
	cfg.UI.DarkTheme = tint.TintGruvboxDark.ID
	return cfg
}

func TestResolveTheme(t *testing.T) {
	t.Run("auto follows a light system", func(t *testing.T) {
		t.Setenv("DIFFNAV_THEME", "")
		t.Setenv("DIFFNAV_APPEARANCE", "light")
		if got := resolveTheme(autoCfg()); got != tint.TintGruvboxLight.ID {
			t.Fatalf("resolveTheme() = %q, want %q", got, tint.TintGruvboxLight.ID)
		}
	})

	t.Run("auto follows a dark system", func(t *testing.T) {
		t.Setenv("DIFFNAV_THEME", "")
		t.Setenv("DIFFNAV_APPEARANCE", "dark")
		if got := resolveTheme(autoCfg()); got != tint.TintGruvboxDark.ID {
			t.Fatalf("resolveTheme() = %q, want %q", got, tint.TintGruvboxDark.ID)
		}
	})

	t.Run("a named theme is left alone", func(t *testing.T) {
		t.Setenv("DIFFNAV_THEME", "")
		t.Setenv("DIFFNAV_APPEARANCE", "dark")
		cfg := autoCfg()
		cfg.UI.Theme = tint.TintNord.ID
		if got := resolveTheme(cfg); got != tint.TintNord.ID {
			t.Fatalf("resolveTheme() = %q, want the configured %q", got, tint.TintNord.ID)
		}
	})

	t.Run("DIFFNAV_THEME wins over auto", func(t *testing.T) {
		t.Setenv("DIFFNAV_APPEARANCE", "light")
		t.Setenv("DIFFNAV_THEME", tint.TintDracula.ID)
		if got := resolveTheme(autoCfg()); got != tint.TintDracula.ID {
			t.Fatalf("resolveTheme() = %q, want the override %q", got, tint.TintDracula.ID)
		}
	})
}

// Only an auto theme should be re-checked as the system changes; a pinned one
// must stay put, including when pinned by the environment.
func TestFollowsAppearance(t *testing.T) {
	for _, tc := range []struct {
		name     string
		theme    string
		envTheme string
		want     bool
	}{
		{"auto", common.ThemeAuto, "", true},
		{"pinned in config", tint.TintNord.ID, "", false},
		{"auto but pinned by env", common.ThemeAuto, tint.TintNord.ID, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DIFFNAV_THEME", tc.envTheme)
			cfg := autoCfg()
			cfg.UI.Theme = tc.theme
			m := mainModel{config: cfg}
			if got := m.followsAppearance(); got != tc.want {
				t.Fatalf("followsAppearance() = %v, want %v", got, tc.want)
			}
		})
	}
}
