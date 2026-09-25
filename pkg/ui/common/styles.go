package common

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
	tint "github.com/lrstanley/bubbletint/v2"
)

// lipglossColorToHex converts a color.Color to hex string
func LipglossColorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

type Styles struct {
	Tint   *tint.Tint
	Colors Colors
}

type Colors struct {
	SelectionFg       color.Color
	SelectionBg       color.Color
	DarkerSelectionBg color.Color
	FaintBlue         func() color.Color
}

// TowardBackground blends a colour towards a background by fraction t, where
// 0 is the colour untouched and 1 is the background.
//
// Deriving chrome by darkening assumed a dark theme; lightening instead — the
// obvious mirror — washes colours out to near-white, because it moves towards
// white rather than towards the ground they will sit on. Blending towards the
// background keeps the hue on both, and needs no case for either.
func TowardBackground(c, bg color.Color, t float64) color.Color {
	const steps = 101
	i := int(t * float64(steps-1))
	if i < 0 {
		i = 0
	}
	if i > steps-1 {
		i = steps - 1
	}
	return lipgloss.Blend1D(steps, c, bg)[i]
}

func MakeStyles() Styles {
	t := Themes.Current()
	if t.ID == tint.TintTokyoNightStorm.ID {
		t.BrightGreen = tint.FromHex("#9ece6a")
	}

	selectionFg := TowardBackground(t.Blue, t.Bg, 0.3)
	selectionBg := TowardBackground(t.Blue, t.Bg, 0.6)
	colors := Colors{
		SelectionFg:       selectionFg,
		SelectionBg:       selectionBg,
		DarkerSelectionBg: TowardBackground(selectionBg, t.Bg, 0.3),
		FaintBlue: func() color.Color {
			return TowardBackground(t.Blue, t.Bg, 0.8)
		},
	}

	return Styles{
		Tint:   t,
		Colors: colors,
	}
}

var SupportedThemeToDeltaSyntax = map[string]string{
	// Custom bat themes; build them with `bat cache --build`.
	TintFlexokiLight.ID:               "flexoki-light",
	TintFlexokiDark.ID:                "flexoki-dark",
	tint.TintCatppuccinFrappe.ID:      "Catppuccin Frappe",
	tint.TintCatppuccinMacchiato.ID:   "Catppuccin Macchiato",
	tint.TintCatppuccinMocha.ID:       "Catppuccin Mocha",
	tint.TintCatppuccinLatte.ID:       "Catppuccin Latte",
	tint.TintNeon.ID:                  "DarkNeon",
	tint.TintDracula.ID:               "Dracula",
	tint.TintMonokaiPro.ID:            "Monokai Extended",
	tint.TintNord.ID:                  "Nord",
	tint.TintOneHalfDark.ID:           "OneHalfDark",
	tint.TintGruvboxLight.ID:          "gruvbox-light",
	tint.TintGruvboxDark.ID:           "gruvbox-dark",
	tint.TintTokyoNight.ID:            "tokyonight_night",
	tint.TintZenburn.ID:               "zenburn",
	tint.TintGithub.ID:                "GitHub",
	tint.TintDimmedMonokai.ID:         "Monokai Extended Light",
	tint.TintOneHalfLight.ID:          "OneHalfLight",
	tint.TintBuiltinSolarizedLight.ID: "Solarized (light)",
	tint.TintBuiltinSolarizedDark.ID:  "Solarized (dark)",
}

var Themes *tint.Registry

func RegisterSupportedTints() {
	Themes = tint.NewRegistry(
		TintFlexokiDark,
		TintFlexokiLight,
		tint.TintTokyoNight,
		tint.TintCatppuccinFrappe,
		tint.TintCatppuccinMacchiato,
		tint.TintCatppuccinMocha,
		tint.TintCatppuccinLatte,
		tint.TintNeon,
		tint.TintDracula,
		tint.TintMonokaiPro,
		tint.TintNord,
		tint.TintOneHalfDark,
		tint.TintGruvboxLight,
		tint.TintGruvboxDark,
		tint.TintTokyoNight,
		tint.TintZenburn,
		tint.TintGithub,
		tint.TintDimmedMonokai,
		tint.TintOneHalfLight,
		tint.TintBuiltinSolarizedLight,
		tint.TintBuiltinSolarizedDark,
	)
}
