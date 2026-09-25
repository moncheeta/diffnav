package diffviewer

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/dlvhdr/diffnav/pkg/filenode"
	"github.com/dlvhdr/diffnav/pkg/icons"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
	"github.com/dlvhdr/diffnav/pkg/utils"
	"github.com/robinovitch61/viewport/filterableviewport"
	"github.com/robinovitch61/viewport/viewport"
	"github.com/robinovitch61/viewport/viewport/item"
)

const dirHeaderHeight = 3

// pageOverlap is how many lines stay on screen across a page jump, so there is
// something to read against. The viewport's own paging keeps none.
const pageOverlap = 2

// viewportFooterLines is the progress line the viewport draws below the content.
const viewportFooterLines = 1

type cachedNode struct {
	path      string
	files     []*gitdiff.File
	additions int64
	deletions int64
	diff      []diffLine
}

type nodeCache map[string]*cachedNode

func cacheKey(path string, sideBySide bool) string {
	key := path + ":" + common.Themes.Current().ID
	if sideBySide {
		return key + ":sbs"
	}
	return key
}

type diffLine struct {
	item item.Item
}

func (o diffLine) GetItem() item.Item {
	return o.item
}

var ViewportKeyMap = viewport.KeyMap{
	// PageDown/PageUp are deliberately left unbound: the viewport pages a
	// whole screen of content, leaving no overlap. The page keys are handled
	// a level up so they can scroll PageStep instead.
	HalfPageDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "scroll half page down"),
	),
	HalfPageUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "scroll half page up"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "prev file"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "next file"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "bottom"),
	),
	Top: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "top"),
	),
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "scroll left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "scroll right"),
	),
}

type Model struct {
	common.Common
	textSel    selection
	wordDrag   wordAnchor
	fvp        *filterableviewport.Model[diffLine]
	file       *cachedNode
	dir        *cachedNode
	cache      nodeCache
	sideBySide bool
	preamble   string
	sb         common.Scrollbar
}

// SetPreamble stores the preamble text (e.g. commit metadata from git show).
func (m *Model) SetPreamble(preamble string) {
	m.preamble = preamble
}

func New(sideBySide bool, styles *common.Styles) Model {
	vp := viewport.New(
		0,
		0,
		viewport.WithKeyMap[diffLine](ViewportKeyMap),
		viewport.WithStyles[diffLine](
			viewport.Styles{
				SelectionPrefix: lipgloss.NewStyle().Foreground(styles.Tint.Blue).Render("▐"),
				SelectedItemStyle: lipgloss.NewStyle().
					Background(styles.Colors.SelectionBg).
					Foreground(styles.Tint.BrightWhite),
				FooterStyle: lipgloss.NewStyle().
					BorderForeground(styles.Tint.BrightBlack).
					Background(styles.Tint.BrightBlack).
					Border(lipgloss.Border{Left: "", Right: ""}, false, true, false, true).
					Foreground(styles.Tint.White).
					Italic(true),
			},
		),
	)

	sb := common.Scrollbar{
		Styles: common.ScrollbarStyles{
			Thumb: lipgloss.NewStyle().Foreground(styles.Tint.Blue),
			Track: lipgloss.NewStyle().Foreground(styles.Tint.Black),
		},
	}

	filterableViewportKeyMap := filterableviewport.DefaultKeyMap()
	filterableViewportKeyMap.CancelFilterKey = key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc/ctrl+c", "cancel filter"),
	)

	filterableViewportStyles := filterableviewport.DefaultStyles()
	filterableViewportStyles.Filter.Focused.TextInput.Text = lipgloss.NewStyle().
		Foreground(styles.Tint.BrightWhite)
	filterableViewportStyles.Filter.Unfocused.TextInput.Text = lipgloss.NewStyle().
		Foreground(styles.Tint.White)
	filterableViewportStyles.Filter.Empty = lipgloss.NewStyle().Foreground(styles.Tint.White)
	filterableViewportStyles.MatchesCount.Matches = lipgloss.NewStyle().
		Foreground(styles.Tint.BrightYellow)
	filterableViewportStyles.Match.Focused = lipgloss.NewStyle().
		Background(lipgloss.Darken(styles.Tint.Yellow, 0.2)).
		Foreground(lipgloss.Lighten(styles.Tint.Yellow, 0.5))
	filterableViewportStyles.Match.Unfocused = lipgloss.NewStyle().
		Background(lipgloss.Lighten(styles.Colors.SelectionBg, 0.1)).
		Foreground(lipgloss.Lighten(styles.Tint.BrightWhite, 0.5))

	return Model{
		Common: common.Common{
			Styles: styles,
		},
		sb: sb,
		fvp: filterableviewport.New(
			vp,
			filterableviewport.WithKeyMap[diffLine](filterableViewportKeyMap),
			filterableviewport.WithStyles[diffLine](filterableViewportStyles),
			filterableviewport.WithPrefixText[diffLine](
				lipgloss.NewStyle().Bold(true).Render("Filter:"),
			),
			filterableviewport.WithFilterModes[diffLine](
				[]filterableviewport.FilterMode{
					filterableviewport.ExactFilterMode(
						key.NewBinding(
							key.WithKeys("/"),
							key.WithHelp("/", "exact fitler mode"),
						),
					), filterableviewport.RegexFilterMode(
						key.NewBinding(
							key.WithKeys("ctrl+r"),
							key.WithHelp("ctrl+r", "regex fitler mode"),
						),
					),
					filterableviewport.FuzzyFilterMode(
						key.NewBinding(
							key.WithKeys("ctrl+f"),
							key.WithHelp("ctrl+f", "fuzzy fitler mode"),
						),
					),
					filterableviewport.CaseInsensitiveFilterMode(key.NewBinding(
						key.WithKeys("ctrl+s"),
						key.WithHelp("ctrl+s", "case insensitive filter"),
					)),
				},
			),
			filterableviewport.WithPlaceholderText[diffLine]("type to search…"),
			filterableviewport.WithItemDescriptor[diffLine]("lines"),
			filterableviewport.WithEmptyText[diffLine](
				"Search… "+lipgloss.NewStyle().
					Faint(true).
					Render("(/ exact ⋅ ⌃+s insensitive ⋅ ⌃+r regex ⋅ ⌃+f fuzzy)"),
			),
			filterableviewport.WithFilterLinePosition[diffLine](filterableviewport.FilterLineTop),
			filterableviewport.WithMatchingItemsOnly[diffLine](false),
			filterableviewport.WithCanToggleMatchingItemsOnly[diffLine](true),
			filterableviewport.WithVerticalPad[diffLine](8),
			filterableviewport.WithHorizontalPad[diffLine](8),
		),
		sideBySide: sideBySide,
		cache:      map[string]*cachedNode{},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	log.Debug("diffviewer", "msg", fmt.Sprintf("%T", msg))
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case diffContentMsg:
		if _, ok := m.cache[msg.cacheKey]; ok {
			m.cache[msg.cacheKey].diff = msg.lines
		}
		m.fvp.SetObjects(msg.lines)
	}

	fvp, fvpCmd := m.fvp.Update(msg)
	cmds = append(cmds, fvpCmd)
	m.fvp = fvp

	return m, tea.Batch(cmds...)
}

const scrollbarWidth = 2 // 1 scrollbar character + 1 padding

func (m Model) View() string {
	return m.textSel.apply(m.baseView(), m.selectionStyle())
}

// selectionStyle is how dragged text is highlighted: the accent as a ground
// with the tint's own background as the text, so it reads as a solid block in
// either appearance.
func (m Model) selectionStyle() lipgloss.Style {
	t := common.Themes.Current()
	return lipgloss.NewStyle().Background(t.Blue).Foreground(t.Bg)
}

// baseView is the pane before any selection highlight. Selection coordinates
// index into this, and copying reads from it.
func (m Model) baseView() string {
	vpView := m.fvp.View()
	itemMetrics := m.fvp.GetItemMetrics()
	scrollbar := m.sb.View(
		m.fvp.GetHeight(),
		itemMetrics.TotalItems,
		itemMetrics.FirstVisibleItemIdx,
		itemMetrics.LastVisibleItemIdx,
	)
	if scrollbar != "" {
		vpView = lipgloss.JoinHorizontal(lipgloss.Top, vpView, scrollbar)
	}
	return vpView
}

// BeginSelect starts a drag selection at a cell of the rendered pane.
func (m *Model) BeginSelect(row, col int) {
	at := point{row: row, col: col}
	m.textSel = selection{active: true, anchor: at, head: at}
	m.wordDrag = wordAnchor{}
}

// ExtendSelect moves the loose end of an in-progress drag.
func (m *Model) ExtendSelect(row, col int) {
	if !m.textSel.active {
		return
	}
	m.textSel.head = point{row: row, col: col}
}

// SelectWordAt selects the word under a cell, reporting whether there was one.
// Used for double-click.
func (m *Model) SelectWordAt(row, col int) bool {
	lines := strings.Split(m.baseView(), "\n")
	if row < 0 || row >= len(lines) {
		return false
	}
	from, to, ok := wordAt(ansi.Strip(lines[row]), col)
	if !ok {
		return false
	}
	m.textSel = selection{
		active: true,
		anchor: point{row: row, col: from},
		head:   point{row: row, col: to},
	}
	m.wordDrag = wordAnchor{active: true, row: row, from: from, to: to}
	return true
}

// ExtendSelectByWord moves the loose end of a drag that began with a
// double-click, snapping to whole words so the word it began on stays intact.
func (m *Model) ExtendSelectByWord(row, col int) {
	if !m.wordDrag.active {
		m.ExtendSelect(row, col)
		return
	}

	headFrom, headTo := col, col
	lines := strings.Split(m.baseView(), "\n")
	if row >= 0 && row < len(lines) {
		if from, to, ok := wordAt(ansi.Strip(lines[row]), col); ok {
			headFrom, headTo = from, to
		}
	}

	anchor := m.wordDrag
	// Dragging back past the anchor word flips which end is fixed.
	if (point{row: row, col: headFrom}).before(point{row: anchor.row, col: anchor.from}) {
		m.textSel = selection{
			active: true,
			anchor: point{row: anchor.row, col: anchor.to},
			head:   point{row: row, col: headFrom},
		}
		return
	}
	m.textSel = selection{
		active: true,
		anchor: point{row: anchor.row, col: anchor.from},
		head:   point{row: row, col: headTo},
	}
}

// SelectedText is the selected text, stripped of styling.
func (m Model) SelectedText() string {
	if m.textSel.empty() {
		return ""
	}
	return m.textSel.extract(strings.Split(m.baseView(), "\n"))
}

// HasSelection reports whether any text is selected.
func (m Model) HasSelection() bool {
	return !m.textSel.empty()
}

// ClearSelection drops the selection. It is anchored to the screen, so
// anything that moves the content calls this.
func (m *Model) ClearSelection() {
	m.textSel = selection{}
	m.wordDrag = wordAnchor{}
}

func (m *Model) SetSize(width, height int) tea.Cmd {
	m.Width = width
	m.Height = height
	m.fvp.SetWidth(m.contentWidth())
	m.fvp.SetHeight(m.Height)
	m.ClearCache()
	return m.diff()
}

func (m *Model) UpdateTheme() tea.Cmd {
	m.ClearCache()
	return m.diff()
}

func (m Model) contentWidth() int {
	return m.Width - scrollbarWidth
}

func (m *Model) diff() tea.Cmd {
	if m.file != nil {
		file := m.file.files[0]
		sideBySide := m.sideBySide && !file.IsNew && !file.IsDelete
		key := cacheKey(m.file.path, sideBySide)
		if cached, ok := m.cache[key]; ok && len(cached.diff) != 0 {
			m.file = cached
			m.fvp.SetObjects(cached.diff)
			return nil
		}
		node := &cachedNode{
			path:      m.file.path,
			files:     m.file.files,
			additions: m.file.additions,
			deletions: m.file.deletions,
		}
		m.file = node
		m.cache[key] = node
		return m.diffFile(node, m.contentWidth())
	} else if m.dir != nil {
		key := cacheKey(m.dir.path, m.sideBySide)
		if cached, ok := m.cache[key]; ok && len(cached.diff) != 0 {
			m.dir = cached
			m.fvp.SetObjects(cached.diff)
			return nil
		}
		node := &cachedNode{
			path:      m.dir.path,
			files:     m.dir.files,
			additions: m.dir.additions,
			deletions: m.dir.deletions,
		}
		m.dir = node
		m.cache[key] = node
		preamble := ""
		if m.dir.path == "/" {
			preamble = m.preamble
		}
		return m.diffDir(node, m.contentWidth(), m.sideBySide, preamble)
	}

	return nil
}

// TopItemIdx is the index of the first visible diff line, which is how far the
// pane has been scrolled.
func (m Model) TopItemIdx() int {
	return m.fvp.GetItemMetrics().FirstVisibleItemIdx
}

// PageStep is how far the page keys scroll: a screen of content less a couple
// of lines of overlap. Never less than one line.
func (m Model) PageStep() int {
	return max(1, m.contentLines()-pageOverlap)
}

// contentLines is the number of diff lines visible at once, which is the
// viewport minus the header it was given and the footer it draws.
func (m Model) contentLines() int {
	header := 0
	if h := m.headerView(); h != "" {
		header = len(strings.Split(h, "\n"))
	}
	return m.fvp.GetHeight() - header - viewportFooterLines
}

func (m Model) headerView() string {
	if m.dir != nil {
		return m.dirHeaderView()
	}

	if m.file == nil || len(m.file.files) != 1 {
		return ""
	}
	name := m.file.path
	base := lipgloss.NewStyle()

	fileIcon := icons.GetIcon(name, false)
	prefix := base.Render(fileIcon) + base.Render(" ")
	name = utils.TruncateString(name, m.contentWidth()-lipgloss.Width(prefix))
	top := prefix + base.Bold(true).Render(name)

	bottom := filenode.ViewFileDiffStats(m.file.files[0], base)

	return base.
		Width(m.contentWidth()).
		Height(dirHeaderHeight - 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(m.Styles.Tint.Black).
		Render(lipgloss.JoinVertical(lipgloss.Left, top, bottom))
}

func (m Model) dirHeaderView() string {
	base := lipgloss.NewStyle().Foreground(m.Styles.Tint.Blue)
	prefix := base.Render(" ")
	name := utils.TruncateString(m.dir.path, m.contentWidth()-lipgloss.Width(prefix))

	top := prefix + base.Bold(true).Render(name)
	bottom := filenode.ViewDiffStats(m.dir.additions, m.dir.deletions, base)
	return base.
		Width(m.contentWidth()).
		Height(dirHeaderHeight - 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(m.Styles.Tint.Black).
		Render(lipgloss.JoinVertical(lipgloss.Left, top, bottom))
}

func (m Model) SetFilePatch(file *gitdiff.File) (Model, tea.Cmd) {
	m.dir = nil

	fname := filenode.GetFileName(file)
	sideBySide := m.sideBySide && !file.IsNew && !file.IsDelete
	key := cacheKey(fname, sideBySide)
	if cached, ok := m.cache[key]; ok {
		m.file = cached
		m.updateHeader()
		m.fvp.SetObjects(cached.diff)
		return m, nil
	}

	files := make([]*gitdiff.File, 1)
	files[0] = file
	additions, deletions := filenode.DiffStats(file)
	m.file = &cachedNode{
		path:      fname,
		files:     files,
		additions: additions,
		deletions: deletions,
	}
	m.cache[key] = m.file

	m.updateHeader()
	return m, m.diffFile(m.file, m.contentWidth())
}

func (m Model) SetDirPatch(dirPath string, files []*gitdiff.File) (Model, tea.Cmd) {
	m.file = nil

	key := cacheKey(dirPath, m.sideBySide)
	if cached, ok := m.cache[key]; ok {
		m.dir = cached
		m.updateHeader()
		m.fvp.SetObjects(cached.diff)
		return m, nil
	}

	var added, deleted int64
	for _, file := range files {
		na, nd := filenode.DiffStats(file)
		added += na
		deleted += nd
	}
	m.dir = &cachedNode{
		path:      dirPath,
		files:     files,
		additions: added,
		deletions: deleted,
	}
	m.cache[key] = m.dir
	preamble := ""
	if dirPath == "/" {
		preamble = m.preamble
	}
	m.updateHeader()
	return m, m.diffDir(m.dir, m.contentWidth(), m.sideBySide, preamble)
}

func (m *Model) updateHeader() {
	m.fvp.SetHeader(strings.Split(m.headerView(), "\n"))
}

// SetSideBySide updates the diff view mode and re-renders.
func (m *Model) SetSideBySide(sideBySide bool) tea.Cmd {
	m.sideBySide = sideBySide
	return m.diff()
}

// ScrollUp scrolls the viewport up by the given number of lines.
func (m *Model) ScrollUp(lines int) {
	m.fvp.ScrollUp(lines)
}

// ScrollDown scrolls the viewport down by the given number of lines.
func (m *Model) ScrollDown(lines int) {
	m.fvp.ScrollDown(lines)
}

// ScrollBottom scrolls the viewport to the bottom.
func (m *Model) ScrollBottom() {
	m.fvp.GoToBottom()
}

// ScrollTop scrolls the viewport to its top.
func (m *Model) ScrollTop() {
	m.fvp.GoToTop()
}

// ScrollLeft moves the viewport to the left by the given number of columns.
func (m *Model) ScrollLeft(cols int) {
	m.fvp.ScrollLeft(cols)
}

// ScrollRight scrolls the viewport one column away from column 0.
func (m *Model) ScrollRight(cols int) {
	m.fvp.ScrollRight(cols)
}

func (m *Model) diffFile(node *cachedNode, width int) tea.Cmd {
	if width == 0 || node == nil || len(node.files) != 1 {
		return nil
	}

	file := node.files[0]
	sideBySide := m.sideBySide && !file.IsNew && !file.IsDelete
	key := cacheKey(node.path, sideBySide)
	return func() tea.Msg {
		args := m.makeDeltaArgs(
			sideBySide,
			width,
			deltaOpts{IsNew: file.IsNew, IsDelete: file.IsDelete},
		)
		log.Info("executing delta", "cmd", fmt.Sprintf(`delta %s`, strings.Join(args, " ")))
		deltac := exec.Command("delta", args...)
		deltac.Env = os.Environ()
		deltac.Stdin = strings.NewReader(file.String() + "\n")
		out, err := deltac.Output()
		if err != nil {
			return common.ErrMsg{Err: err}
		}

		return diffContentMsg{cacheKey: key, lines: stringToDiffLines(string(out))}
	}
}

func (m *Model) diffDir(dir *cachedNode, width int, sideBySide bool, preamble string) tea.Cmd {
	if width == 0 || dir == nil {
		return nil
	}
	key := cacheKey(dir.path, sideBySide)
	return func() tea.Msg {
		args := m.makeDeltaArgs(sideBySide, width, deltaOpts{})
		log.Info("executing delta", "cmd", fmt.Sprintf(`delta %s`, strings.Join(args, " ")))
		deltac := exec.Command("delta", args...)
		deltac.Env = os.Environ()
		strs := strings.Builder{}
		for _, file := range dir.files {
			strs.WriteString(file.String())
		}
		deltac.Stdin = strings.NewReader(strs.String() + "\n")
		out, err := deltac.Output()
		if err != nil {
			if stdErr, ok := err.(*exec.ExitError); ok {
				return common.ErrMsg{Err: err, Message: string(stdErr.Stderr)}
			}
			log.Warn("delta error", "err", err)
			return common.ErrMsg{Err: err}
		}

		text := string(out)
		if preamble != "" {
			text = m.renderPreamble(preamble) + "\n" + text
		}
		return diffContentMsg{cacheKey: key, lines: stringToDiffLines(text)}
	}
}

func (m Model) renderPreamble(preamble string) string {
	preamble = strings.TrimSpace(preamble)
	if preamble == "" {
		return ""
	}

	dim := lipgloss.NewStyle().Foreground(m.Styles.Tint.Black)
	yellow := lipgloss.NewStyle().Foreground(m.Styles.Tint.Yellow)

	var out []string
	for line := range strings.SplitSeq(preamble, "\n") {
		switch {
		case strings.HasPrefix(line, "commit "):
			out = append(
				out,
				dim.Render("commit ")+yellow.Render(strings.TrimPrefix(line, "commit ")),
			)
		case strings.HasPrefix(line, "Author:"),
			strings.HasPrefix(line, "AuthorDate:"),
			strings.HasPrefix(line, "Date:"),
			strings.HasPrefix(line, "Commit:"),
			strings.HasPrefix(line, "CommitDate:"),
			strings.HasPrefix(line, "Merge:"):
			out = append(out, dim.Render(line))
		default:
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n")
}

type diffContentMsg struct {
	cacheKey string
	lines    []diffLine
}

func (m *Model) ClearCache() {
	m.cache = make(nodeCache)
}

func (m *Model) RootDiffStats() (int64, int64) {
	if item, ok := m.cache[cacheKey("/", m.sideBySide)]; ok {
		return item.additions, item.deletions
	}

	return 0, 0
}

func (m *Model) Searching() bool {
	return m.fvp.FilterFocused()
}

func (m *Model) SetFiltering() {
	var mode filterableviewport.FilterModeName
	if curr := m.fvp.GetActiveFilterMode(); curr != nil {
		mode = curr.Name
	} else {
		mode = filterableviewport.FilterExact
	}
	m.fvp.SetFilter(m.fvp.GetFilterText(), mode)
}

func (m *Model) Filtering() bool {
	return m.fvp.GetActiveFilterMode() != nil
}

func (m *Model) SelectionEnabled() bool {
	return m.fvp.GetSelectionEnabled()
}

func (m *Model) SetSelectionEnabled(val bool) {
	m.fvp.SetSelectionEnabled(val)
}

func stringToDiffLines(val string) []diffLine {
	lines := strings.Split(val, "\n")
	objects := make([]diffLine, len(lines))
	for i, line := range lines {
		objects[i] = diffLine{item: item.NewItem(line)}
	}
	return objects
}

type deltaOpts struct {
	IsNew    bool
	IsDelete bool
}

func (m *Model) makeDeltaArgs(sideBySide bool, width int, opts deltaOpts) []string {
	theme := common.Themes.Current()
	sSelection := lipgloss.NewStyle().Background(m.Styles.Colors.SelectionBg)
	selectionColor := common.LipglossColorToHex(m.Styles.Colors.SelectionBg)
	// The added/removed grounds are the accent blended towards the tint's own
	// background, so they read as a tint of the page in either appearance.
	shade := func(c color.Color, t float64) string {
		return common.LipglossColorToHex(common.TowardBackground(c, theme.Bg, t))
	}

	white := common.LipglossColorToHex(theme.White)
	green := shade(theme.Green, 0.8)
	brightGreen := shade(theme.BrightGreen, 0.5)
	plusLine := common.LipglossColorToHex(theme.BrightGreen)
	minusLine := common.LipglossColorToHex(theme.BrightRed)
	red := shade(theme.Red, 0.8)
	brightRed := shade(theme.BrightRed, 0.5)
	args := []string{
		"--paging=never",
		"--line-numbers",
		"--tabs=2",
		"--wrap-left-symbol= ",
		"--wrap-right-symbol= ",
		"--wrap-right-prefix-symbol= ",
		"--hunk-label=  󰡏 ",
		fmt.Sprintf(
			"--syntax-theme=%s",
			common.SupportedThemeToDeltaSyntax[theme.ID],
		),
		fmt.Sprintf("--file-modified-label=%s",
			utils.RemoveReset(sSelection.Foreground(m.Styles.Tint.BrightYellow).Render(" "))),
		fmt.Sprintf("--file-removed-label=%s",
			utils.RemoveReset(sSelection.Foreground(m.Styles.Tint.BrightRed).Render(" "))),
		fmt.Sprintf("--file-added-label=%s",
			utils.RemoveReset(sSelection.Foreground(m.Styles.Tint.BrightGreen).Render(" "))),
		fmt.Sprintf("--file-renamed-label=%s",
			utils.RemoveReset(sSelection.Foreground(m.Styles.Tint.Blue).Render("renamed:"))),
		fmt.Sprintf("--file-style='\"%s\" bold \"%s\"'", selectionColor, selectionColor),
		fmt.Sprintf("--file-decoration-style='\"%s\" box %s'", selectionColor, selectionColor),
		fmt.Sprintf("-w=%d", width),
		"--max-line-length=0",
		"--wrap-max-lines=unlimited",
		fmt.Sprintf(
			"--line-numbers-right-style='\"%s\" dim'",
			common.LipglossColorToHex(m.Styles.Colors.FaintBlue()),
		),
		fmt.Sprintf("--line-numbers-zero-style='\"%s\" dim'", white),
		fmt.Sprintf("--plus-style='syntax \"%s\"'", green),
		fmt.Sprintf("--plus-emph-style='syntax \"%s\"'", brightGreen),
		fmt.Sprintf("--minus-style='syntax \"%s\"'", red),
		fmt.Sprintf("--minus-emph-style='syntax \"%s\"'", brightRed),
		fmt.Sprintf(
			"--hunk-header-line-number-style='%s'",
			common.LipglossColorToHex(m.Styles.Colors.FaintBlue()),
		),
		fmt.Sprintf(
			"--hunk-header-style='%s'",
			white,
		),
		fmt.Sprintf("--hunk-header-file-style='\"%s\" dim'", white),
		fmt.Sprintf(
			"--hunk-header-decoration-style='\"%s\" ol ul'",
			common.LipglossColorToHex(m.Styles.Colors.FaintBlue()),
		),
	}

	if opts.IsNew {
		args = append(
			args,
			fmt.Sprintf("--line-numbers-plus-style='\"%s\" \"%s\"'", plusLine, green),
			"--line-numbers-minus-style=",
			"--line-numbers-left-format=",
			"--line-numbers-right-format={np:>4} ",
		)
	} else if opts.IsDelete {
		args = append(
			args,
			"--line-numbers-plus-style=",
			fmt.Sprintf("--line-numbers-minus-style=\"%s\" \"%s\"", minusLine, red),
			"--line-numbers-left-format={nm:>4} ",
			"--line-numbers-right-format=",
		)
	} else {
		args = append(
			args,
			fmt.Sprintf("--line-numbers-plus-style='\"%s\" \"%s\"'", plusLine, green),
			fmt.Sprintf("--line-numbers-minus-style='\"%s\" \"%s\"", minusLine, red),
			"--line-numbers-left-format={nm:>4} ",
			"--line-numbers-right-format={np:>4} ",
		)
	}

	if sideBySide {
		args = append(args, "--side-by-side")
	}
	if theme.Dark {
		args = append(args, "--dark")
	} else {
		args = append(args, "--light")
	}

	return args
}
