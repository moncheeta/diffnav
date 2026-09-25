package common

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ThemeAuto is the value of the `theme` setting that follows the operating
// system's light/dark appearance, choosing between `lightTheme` and
// `darkTheme` instead of naming one theme outright.
const ThemeAuto = "auto"

// LightAppearance reports whether the system is in light mode.
// DIFFNAV_APPEARANCE=light|dark forces the answer, which is also how a
// platform without appearance detection can still switch.
func LightAppearance() bool {
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

// ThemeForAppearance picks the theme matching the current appearance.
func ThemeForAppearance(lightTheme, darkTheme string) string {
	if LightAppearance() {
		return lightTheme
	}
	return darkTheme
}
