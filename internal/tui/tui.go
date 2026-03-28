package tui

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/saurabh/logr/internal/filter"
	"github.com/saurabh/logr/internal/parser"
	"github.com/saurabh/logr/internal/render"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color("12")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	countStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	dividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	searchLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")).
				Background(lipgloss.Color("236")).
				Bold(true).
				Padding(0, 1)

	activeFilterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10")).
				Background(lipgloss.Color("236")).
				Padding(0, 1)
)

// Model is the bubbletea model for the logr TUI.
type Model struct {
	entries     []parser.LogEntry
	engine      *filter.Engine
	opts        render.Options
	viewport    viewport.Model
	searchInput textinput.Model
	searching   bool
	search      string
	ready       bool
	width       int
	height      int
}

// New creates a TUI model pre-loaded with entries.
func New(entries []parser.LogEntry, cfg filter.Config, opts render.Options) *Model {
	// Force colors on in TUI mode — bubbletea handles the terminal.
	color.NoColor = false

	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 120
	ti.Width = 40

	return &Model{
		entries:     entries,
		engine:      filter.New(cfg),
		opts:        opts,
		searchInput: ti,
	}
}

// Run launches the TUI and blocks until the user quits.
func Run(entries []parser.LogEntry, cfg filter.Config, opts render.Options) error {
	m := New(entries, cfg, opts)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpHeight := m.height - 3 // header + divider + status bar
		if vpHeight < 1 {
			vpHeight = 1
		}
		if !m.ready {
			m.viewport = viewport.New(msg.Width, vpHeight)
			m.viewport.SetContent(m.renderEntries())
			m.viewport.GotoBottom()
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = vpHeight
		}

	case tea.KeyMsg:
		if m.searching {
			switch msg.Type {
			case tea.KeyEsc:
				m.searching = false
				m.search = ""
				m.searchInput.SetValue("")
				m.searchInput.Blur()
				m.viewport.SetContent(m.renderEntries())
				m.viewport.GotoBottom()
			case tea.KeyEnter:
				m.searching = false
				m.search = m.searchInput.Value()
				m.searchInput.Blur()
				m.viewport.SetContent(m.renderEntries())
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				cmds = append(cmds, cmd)
				// Live update as user types.
				m.search = m.searchInput.Value()
				m.viewport.SetContent(m.renderEntries())
			}
			return m, tea.Batch(cmds...)
		}

		// Normal (scroll) mode.
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "/":
			m.searching = true
			m.searchInput.Focus()
			return m, textinput.Blink
		case "esc":
			if m.search != "" {
				m.search = ""
				m.searchInput.SetValue("")
				m.viewport.SetContent(m.renderEntries())
				m.viewport.GotoBottom()
			}
		case "g":
			m.viewport.GotoTop()
		case "G":
			m.viewport.GotoBottom()
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if !m.ready {
		return "Loading…"
	}

	filtered := m.filteredEntries()
	total := len(m.entries)
	shown := len(filtered)

	// ── Header bar ─────────────────────────────────────────────────────────
	title := headerStyle.Render("logr")
	counts := countStyle.Render(fmt.Sprintf("%d / %d entries", shown, total))

	var filterInfo string
	if m.search != "" {
		filterInfo = activeFilterStyle.Render(fmt.Sprintf("search: %q", m.search))
	}

	headerRight := lipgloss.JoinHorizontal(lipgloss.Top, counts, filterInfo)
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(headerRight)
	if gap < 0 {
		gap = 0
	}
	padding := lipgloss.NewStyle().Background(lipgloss.Color("236")).Render(strings.Repeat(" ", gap))
	header := lipgloss.JoinHorizontal(lipgloss.Top, title, padding, headerRight)

	// ── Divider ─────────────────────────────────────────────────────────────
	divider := dividerStyle.Render(strings.Repeat("─", m.width))

	// ── Viewport ────────────────────────────────────────────────────────────
	vp := m.viewport.View()

	// ── Status / search bar ─────────────────────────────────────────────────
	var statusBar string
	if m.searching {
		label := searchLabelStyle.Render("/")
		input := m.searchInput.View()
		hint := statusStyle.Render("  enter:apply  esc:clear")
		statusBar = lipgloss.JoinHorizontal(lipgloss.Top, label, input, hint)
	} else {
		pct := 100.0
		if total > 0 {
			pct = m.viewport.ScrollPercent() * 100
		}
		statusBar = statusStyle.Render(fmt.Sprintf(
			"↑↓/jk:scroll  g/G:top/bot  /:search  esc:clear  q:quit  [%d%%]",
			int(pct),
		))
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, divider, vp, statusBar)
}

// filteredEntries returns entries that pass the engine AND the live search term.
func (m *Model) filteredEntries() []parser.LogEntry {
	var out []parser.LogEntry
	search := strings.ToLower(m.search)
	for _, e := range m.entries {
		if !m.engine.Pass(e) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(string(e.Raw)), search) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// renderEntries serialises all filtered entries into a single ANSI string
// for the viewport.
func (m *Model) renderEntries() string {
	entries := m.filteredEntries()
	if len(entries) == 0 {
		return statusStyle.Render("  no entries match current filters")
	}
	var buf bytes.Buffer
	for _, e := range entries {
		render.Render(e, &buf, m.opts)
	}
	return strings.TrimRight(buf.String(), "\n")
}
