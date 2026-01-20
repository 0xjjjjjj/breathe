package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xjjjjjj/breathe/internal/config"
	"github.com/0xjjjjjj/breathe/internal/scanner"
)

type View int

const (
	ViewScan View = iota
	ViewJunk
)

type Model struct {
	cfg       *config.Config
	tree      *scanner.Tree
	matcher   *scanner.Matcher
	scanning  bool
	scanPath  string
	spinner   spinner.Model
	cursor    int
	expanded  map[string]bool
	selected  map[string]bool
	view      View
	width     int
	height    int
	fileCount int
	err       error
}

type scanResultMsg scanner.ScanResult
type scanDoneMsg struct{}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236"))

	sizeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	junkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

func NewModel(cfg *config.Config, scanPath string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		cfg:      cfg,
		scanPath: scanPath,
		spinner:  s,
		expanded: make(map[string]bool),
		selected: make(map[string]bool),
		matcher:  scanner.NewMatcher(cfg.JunkPatterns),
		view:     ViewScan,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startScan())
}

func (m Model) startScan() tea.Cmd {
	return func() tea.Msg {
		return nil // Signal to start scanning
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			m.cursor++
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter", "l", "right":
			// Toggle expand
		case "tab":
			if m.view == ViewScan {
				m.view = ViewJunk
			} else {
				m.view = ViewScan
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case nil:
		// Start scan
		if m.tree == nil {
			m.tree = scanner.NewTree(m.scanPath)
			m.scanning = true
			return m, m.doScan()
		}

	case scanResultMsg:
		if m.tree != nil && msg.Err == nil {
			m.tree.AddEntry(msg.Entry)
			m.fileCount++
		}
		return m, nil

	case scanDoneMsg:
		m.scanning = false
		return m, nil
	}

	return m, nil
}

func (m Model) doScan() tea.Cmd {
	return func() tea.Msg {
		results := make(chan scanner.ScanResult, 1000)
		go scanner.Scan(m.scanPath, results)

		for r := range results {
			if r.Err == nil {
				m.tree.AddEntry(r.Entry)
				m.fileCount++
			}
		}
		return scanDoneMsg{}
	}
}

func (m Model) View() string {
	if m.tree == nil {
		return "Initializing..."
	}

	var s string

	// Header
	if m.scanning {
		s += fmt.Sprintf("%s Scanning... %d files | %s\n",
			m.spinner.View(),
			m.fileCount,
			m.scanPath)
	} else {
		s += fmt.Sprintf("Scan complete: %d files | %s\n",
			m.fileCount,
			m.scanPath)
	}

	s += titleStyle.Render(fmt.Sprintf("Total: %s", formatSize(m.tree.Root().Size))) + "\n\n"

	// Tree view
	if m.view == ViewScan {
		s += m.renderTree()
	} else {
		s += m.renderJunk()
	}

	// Footer
	s += "\n" + helpStyle.Render("[↑↓] Navigate  [Enter] Expand  [Tab] Junk view  [q] Quit")

	return s
}

func (m Model) renderTree() string {
	var s string
	children := m.tree.Children(m.scanPath)

	for i, child := range children {
		if i >= 20 {
			s += fmt.Sprintf("  ... and %d more\n", len(children)-20)
			break
		}

		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}

		icon := "📄"
		if child.IsDir {
			icon = "📁"
		}

		line := fmt.Sprintf("%s%s %s %s",
			prefix,
			icon,
			child.Name,
			sizeStyle.Render(formatSize(child.Size)))

		if i == m.cursor {
			line = selectedStyle.Render(line)
		}

		s += line + "\n"
	}

	return s
}

func (m Model) renderJunk() string {
	groups := m.matcher.GroupJunk(m.tree)

	if len(groups) == 0 {
		return "No junk detected\n"
	}

	var s string
	s += junkStyle.Render(fmt.Sprintf("🗑️  Detected Junk (%d groups)\n\n", len(groups)))

	for _, g := range groups {
		safeIcon := "✓"
		if !g.Safe {
			safeIcon = "⚠"
		}
		s += fmt.Sprintf("[%s] %s (%d dirs) %s\n",
			safeIcon,
			g.Name,
			len(g.Paths),
			sizeStyle.Render(formatSize(g.Total)))
	}

	return s
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func Run(cfg *config.Config, path string) error {
	p := tea.NewProgram(NewModel(cfg, path), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
