package common

import tint "github.com/lrstanley/bubbletint/v2"

// Flexoki (https://stephango.com/flexoki) as a pair of tints, so diffnav can
// match a terminal themed with it. The registry ships no Flexoki, and chrome
// from an unrelated palette reads as a clash against the terminal's own ground.
//
// Light mode uses the 600 weights and dark the 400s, which is what Flexoki
// specifies for text on paper versus on black.
var (
	TintFlexokiLight = &tint.Tint{
		ID:          "flexoki_light",
		DisplayName: "Flexoki Light",
		Dark:        false,
		Bg:          tint.FromHex("#FFFCF0"), // paper
		Fg:          tint.FromHex("#100F0F"), // black
		SelectionBg: tint.FromHex("#E6E4D9"), // base-100
		Cursor:      tint.FromHex("#100F0F"),

		Black:  tint.FromHex("#100F0F"),
		Red:    tint.FromHex("#AF3029"),
		Green:  tint.FromHex("#66800B"),
		Yellow: tint.FromHex("#AD8301"),
		Blue:   tint.FromHex("#205EA6"),
		Purple: tint.FromHex("#5E409D"),
		Cyan:   tint.FromHex("#24837B"),
		White:  tint.FromHex("#6F6E69"), // base-600

		BrightBlack:  tint.FromHex("#B7B5AC"), // base-300
		BrightRed:    tint.FromHex("#D14D41"),
		BrightGreen:  tint.FromHex("#879A39"),
		BrightYellow: tint.FromHex("#D0A215"),
		BrightBlue:   tint.FromHex("#4385BE"),
		BrightPurple: tint.FromHex("#8B7EC8"),
		BrightCyan:   tint.FromHex("#3AA99F"),
		BrightWhite:  tint.FromHex("#100F0F"),
	}

	TintFlexokiDark = &tint.Tint{
		ID:          "flexoki_dark",
		DisplayName: "Flexoki Dark",
		Dark:        true,
		Bg:          tint.FromHex("#100F0F"), // black
		Fg:          tint.FromHex("#CECDC3"), // base-200
		SelectionBg: tint.FromHex("#282726"), // base-900
		Cursor:      tint.FromHex("#CECDC3"),

		Black:  tint.FromHex("#100F0F"),
		Red:    tint.FromHex("#AF3029"),
		Green:  tint.FromHex("#66800B"),
		Yellow: tint.FromHex("#AD8301"),
		Blue:   tint.FromHex("#205EA6"),
		Purple: tint.FromHex("#5E409D"),
		Cyan:   tint.FromHex("#24837B"),
		White:  tint.FromHex("#CECDC3"),

		BrightBlack:  tint.FromHex("#575653"), // base-700
		BrightRed:    tint.FromHex("#D14D41"),
		BrightGreen:  tint.FromHex("#879A39"),
		BrightYellow: tint.FromHex("#D0A215"),
		BrightBlue:   tint.FromHex("#4385BE"),
		BrightPurple: tint.FromHex("#8B7EC8"),
		BrightCyan:   tint.FromHex("#3AA99F"),
		BrightWhite:  tint.FromHex("#FFFCF0"),
	}
)
