package common

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"charm.land/lipgloss/v2"
)

type Key int

// Available colors.
const (
	Selected Key = iota
	DarkerSelected
	SelectedFg
	MutedFg
	BorderFocus
	BorderMuted
)

// Chrome colors are Flexoki (https://stephango.com/flexoki), picked to match
// the terminal. DIFFNAV_APPEARANCE=light|dark forces a variant; with it unset
// we ask macOS, so this works in any shell without extra setup.
func lightAppearance() bool {
	switch strings.ToLower(os.Getenv("DIFFNAV_APPEARANCE")) {
	case "light":
		return true
	case "dark":
		return false
	}
	return detectLightAppearance()
}

// `defaults read -g AppleInterfaceStyle` prints "Dark" in dark mode and exits
// non-zero in light mode, where the key is simply absent.
func detectLightAppearance() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
	if err != nil {
		return true
	}
	return !strings.Contains(strings.ToLower(string(out)), "dark")
}

var Colors = chromeColors()

func chromeColors() map[Key]color.RGBA {
	if lightAppearance() {
		return map[Key]color.RGBA{
			Selected:       {R: 0xE6, G: 0xE4, B: 0xD9, A: 0xFF}, // base-100
			DarkerSelected: {R: 0xF2, G: 0xF0, B: 0xE5, A: 0xFF}, // base-50
			SelectedFg:     {R: 0x10, G: 0x0F, B: 0x0F, A: 0xFF}, // black
			MutedFg:        {R: 0x6F, G: 0x6E, B: 0x69, A: 0xFF}, // base-600
			BorderFocus:    {R: 0x20, G: 0x5E, B: 0xA6, A: 0xFF}, // blue-600
			BorderMuted:    {R: 0xCE, G: 0xCD, B: 0xC3, A: 0xFF}, // base-200
		}
	}
	return map[Key]color.RGBA{
		Selected:       {R: 0x28, G: 0x27, B: 0x26, A: 0xFF}, // base-900
		DarkerSelected: {R: 0x1C, G: 0x1B, B: 0x1A, A: 0xFF}, // base-950
		SelectedFg:     {R: 0xCE, G: 0xCD, B: 0xC3, A: 0xFF}, // base-200
		MutedFg:        {R: 0x87, G: 0x85, B: 0x80, A: 0xFF}, // base-500
		BorderFocus:    {R: 0x43, G: 0x85, B: 0xBE, A: 0xFF}, // blue-400
		BorderMuted:    {R: 0x40, G: 0x3E, B: 0x3C, A: 0xFF}, // base-800
	}
}

var BgStyles = map[Key]lipgloss.Style{
	Selected:       lipgloss.NewStyle().Background(Colors[Selected]),
	DarkerSelected: lipgloss.NewStyle().Background(Colors[DarkerSelected]),
}

// lipglossColorToHex converts a color.Color to hex string
func LipglossColorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}
