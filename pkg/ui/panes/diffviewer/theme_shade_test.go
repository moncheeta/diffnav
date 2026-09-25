package diffviewer

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/dlvhdr/diffnav/pkg/ui/common"
	tint "github.com/lrstanley/bubbletint/v2"
)

var hexColor = regexp.MustCompile(`#([0-9a-fA-F]{6})`)

// luminance of a #rrggbb string, 0 (black) to 1 (white).
func luminance(hex string) float64 {
	v, err := strconv.ParseInt(strings.TrimPrefix(hex, "#"), 16, 64)
	if err != nil {
		return -1
	}
	r := float64((v>>16)&0xff) / 255
	g := float64((v>>8)&0xff) / 255
	b := float64(v&0xff) / 255
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func deltaArgsFor(t *testing.T, themeID string) string {
	t.Helper()
	common.RegisterSupportedTints()
	if ok := common.Themes.SetTintID(themeID); !ok {
		t.Fatalf("theme %q is not registered", themeID)
	}
	styles := common.MakeStyles()
	m := New(false, &styles)
	m.SetSize(120, 25)
	return strings.Join(m.makeDeltaArgs(false, 120, deltaOpts{}), " ")
}

// The added/removed grounds are derived from the tint's accents. Deriving them
// by darkening suits a dark tint but paints dark bands across a light one, so
// the direction has to follow the tint.
func TestDeltaGroundsFollowTheTint(t *testing.T) {
	darkArgs := deltaArgsFor(t, tint.TintGruvboxDark.ID)
	lightArgs := deltaArgsFor(t, tint.TintGruvboxLight.ID)

	if darkArgs == lightArgs {
		t.Fatal("delta args are identical for a light and a dark tint")
	}

	mean := func(args string) float64 {
		var sum float64
		var n int
		for _, m := range hexColor.FindAllString(args, -1) {
			if l := luminance(m); l >= 0 {
				sum += l
				n++
			}
		}
		if n == 0 {
			t.Fatal("no colours in the delta args")
		}
		return sum / float64(n)
	}

	darkMean, lightMean := mean(darkArgs), mean(lightArgs)
	t.Logf("mean colour luminance — dark tint %.3f, light tint %.3f", darkMean, lightMean)
	if lightMean <= darkMean {
		t.Fatalf("the light tint's diff colours (%.3f) are no brighter than the dark tint's (%.3f)",
			lightMean, darkMean)
	}
}

// The chrome selection is derived the same way and has the same hazard.
func TestSelectionFollowsTheTint(t *testing.T) {
	common.RegisterSupportedTints()

	read := func(id string) float64 {
		common.Themes.SetTintID(id)
		s := common.MakeStyles()
		return luminance(common.LipglossColorToHex(s.Colors.SelectionBg))
	}

	dark := read(tint.TintGruvboxDark.ID)
	light := read(tint.TintGruvboxLight.ID)
	t.Logf("selection background luminance — dark %.3f, light %.3f", dark, light)
	if light <= dark {
		t.Fatalf("light tint selection (%.3f) is not lighter than dark tint's (%.3f)", light, dark)
	}
}
