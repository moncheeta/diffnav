package ui

import (
	tea "charm.land/bubbletea/v2"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/dlvhdr/diffnav/pkg/config"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
	tint "github.com/lrstanley/bubbletint/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

var bgSeq = regexp.MustCompile(`48;2;[0-9]+;[0-9]+;[0-9]+`)

func bodyGrounds(view string) string {
	seen := map[string]bool{}
	for _, s := range bgSeq.FindAllString(view, -1) {
		seen[s] = true
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return strings.Join(out, " ")
}

// Re-theming has to reach the diff body, not just the chrome. The panes hold a
// pointer to the shared styles; when that pointer aimed at a stale copy of the
// model, the chrome re-themed on every frame while the diff kept the colours it
// was built with.
func TestLiveThemeSwitchReachesTheDiffBody(t *testing.T) {
	zone.NewGlobal()

	f, err := os.Open("panes/filetree/testdata/gh_dash_pr.diff")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	files, _, err := gitdiff.Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("DIFFNAV_THEME", "")
	t.Setenv("DIFFNAV_APPEARANCE", "dark")

	cfg := config.DefaultConfig()
	cfg.UI.Theme = common.ThemeAuto
	cfg.UI.LightTheme = tint.TintGruvboxLight.ID
	cfg.UI.DarkTheme = tint.TintGruvboxDark.ID

	m := New(ModelOpts{}, cfg)
	m.width, m.height = 120, 30
	m.files = files
	m.diffViewer.SetSize(120, 25)
	dv, cmd := m.diffViewer.SetFilePatch(files[0])
	m.diffViewer = dv
	m.diffViewer, _ = m.diffViewer.Update(cmd())

	before := bodyGrounds(m.diffViewer.View())
	if !strings.Contains(common.Themes.Current().ID, "dark") {
		t.Fatalf("expected to start dark, got %q", common.Themes.Current().ID)
	}

	// The system flips; the next appearance tick should re-theme everything.
	t.Setenv("DIFFNAV_APPEARANCE", "light")
	out, themeCmd := m.Update(appearanceTickMsg{time.Now()})
	m2 := out.(mainModel)

	if got := common.Themes.Current().ID; got != tint.TintGruvboxLight.ID {
		t.Fatalf("theme after the tick = %q, want %q", got, tint.TintGruvboxLight.ID)
	}
	if themeCmd == nil {
		t.Fatal("re-theming produced no command to re-render the diff")
	}
	// Update returns a batch; run it out and feed every message back, which is
	// what the runtime does.
	for _, msg := range drain(themeCmd) {
		m2.diffViewer, _ = m2.diffViewer.Update(msg)
	}

	after := bodyGrounds(m2.diffViewer.View())
	t.Logf("dark  body: %s", before)
	t.Logf("light body: %s", after)
	if before == after {
		t.Fatal("the diff body did not change with the theme")
	}
}

// drain runs a command, flattening any batches, and returns the messages it
// produced. Tick commands are skipped so the test does not wait on them.
func drain(cmd tea.Cmd) []tea.Msg {
	var out []tea.Msg
	if cmd == nil {
		return out
	}
	msg := cmd()
	switch m := msg.(type) {
	case tea.BatchMsg:
		for _, c := range m {
			out = append(out, drain(c)...)
		}
	case nil:
	default:
		out = append(out, msg)
	}
	return out
}

// The footer sets its own background, so it has to set a foreground too:
// leaving the text on the terminal default makes it unreadable whenever the
// theme and the terminal disagree.
func TestFooterHasReadableContrast(t *testing.T) {
	zone.NewGlobal()
	common.RegisterSupportedTints()

	fg := regexp.MustCompile(`38;2;[0-9]+;[0-9]+;[0-9]+`)
	for _, id := range []string{tint.TintGruvboxLight.ID, tint.TintGruvboxDark.ID} {
		common.Themes.SetTintID(id)
		cfg := config.DefaultConfig()
		m := New(ModelOpts{}, cfg)
		m.width, m.height = 120, 30

		footer := m.footerView()
		if !fg.MatchString(footer) {
			t.Fatalf("%s: footer sets no foreground, so its text falls back to the terminal default: %q",
				id, footer)
		}
	}
}
