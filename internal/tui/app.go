package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xjjjjjj/breathe/internal/config"
	"github.com/0xjjjjjj/breathe/internal/history"
	"github.com/0xjjjjjj/breathe/internal/scanner"
)

type View int

const (
	ViewScan View = iota
	ViewJunk
)

type Model struct {
	cfg         *config.Config
	tree        *scanner.Tree
	matcher     *scanner.Matcher
	scanning    bool
	scanPath    string
	currentPath string // Currently viewed directory
	spinner     spinner.Model
	cursor      int
	offset      int // Viewport scroll offset
	selected    map[string]bool
	view        View
	width       int
	height      int
	fileCount   int
	err         error
	results     chan scanner.ScanResult // Channel for receiving scan results
	lastPath    string                  // Last file/dir scanned (for progress display)
	db          *history.DB             // History database for tracking deletions
	statusMsg   string                  // Status message to show user
}

type scanResultMsg scanner.ScanResult
type scanDoneMsg struct{}
type pollResultsMsg struct{}

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

	// Initialize tree and results channel upfront
	tree := scanner.NewTree(scanPath)
	results := make(chan scanner.ScanResult, 1000)

	// Open history database (errors logged but not fatal - TUI can work without history)
	db, err := history.Open(config.DataPath())
	if err != nil {
		// Continue without history - delete will still work, just won't be logged
		db = nil
	}

	// Start scanner in background immediately
	go scanner.Scan(scanPath, results)

	return Model{
		cfg:         cfg,
		scanPath:    scanPath,
		currentPath: scanPath,
		spinner:     s,
		selected:    make(map[string]bool),
		matcher:     scanner.NewMatcher(cfg.JunkPatterns),
		view:        ViewScan,
		tree:        tree,
		results:     results,
		scanning:    true,
		db:          db,
	}
}

// deleteItem moves an item to trash and removes it from the tree
func (m *Model) deleteItem(path string) {
	cleaner := scanner.NewCleaner(m.db, true) // true = use trash
	if err := cleaner.Delete(path); err != nil {
		m.statusMsg = fmt.Sprintf("Error: %v", err)
		return
	}

	// Remove from tree
	m.tree.Remove(path)
	m.statusMsg = fmt.Sprintf("Trashed: %s", filepath.Base(path))
}

func (m Model) Init() tea.Cmd {
	// Start spinner and polling for scan results
	return tea.Batch(m.spinner.Tick, pollResults(m.results))
}

// pollResults creates a command that reads from the results channel
func pollResults(results chan scanner.ScanResult) tea.Cmd {
	return func() tea.Msg {
		r, ok := <-results
		if !ok {
			return scanDoneMsg{}
		}
		return scanResultMsg(r)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		children := m.tree.Children(m.currentPath)
		maxItems := m.visibleItems()
		m.statusMsg = "" // Clear status on any keypress

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(children)-1 {
				m.cursor++
				// Scroll down if cursor goes past visible area
				if m.cursor >= m.offset+maxItems {
					m.offset = m.cursor - maxItems + 1
				}
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				// Scroll up if cursor goes above visible area
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}
		case "enter", "l", "right":
			// Drill into directory
			if m.cursor < len(children) && children[m.cursor].IsDir {
				m.currentPath = children[m.cursor].Path
				m.cursor = 0
				m.offset = 0
			}
		case "h", "left", "backspace":
			// Go up to parent directory (but not above scan root)
			if m.currentPath != m.scanPath {
				m.currentPath = filepath.Dir(m.currentPath)
				m.cursor = 0
				m.offset = 0
			}
		case "tab":
			if m.view == ViewScan {
				m.view = ViewJunk
			} else {
				m.view = ViewScan
			}
		case " ":
			// Toggle selection
			if m.cursor < len(children) {
				path := children[m.cursor].Path
				if m.selected[path] {
					delete(m.selected, path)
				} else {
					m.selected[path] = true
				}
			}
		case "d":
			// Delete selected items (or current item if none selected)
			if len(m.selected) > 0 {
				for path := range m.selected {
					m.deleteItem(path)
				}
				m.selected = make(map[string]bool)
			} else if m.cursor < len(children) {
				m.deleteItem(children[m.cursor].Path)
			}
			// Reset cursor if it's now out of bounds
			newChildren := m.tree.Children(m.currentPath)
			if m.cursor >= len(newChildren) && m.cursor > 0 {
				m.cursor = len(newChildren) - 1
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case scanResultMsg:
		if m.tree != nil && scanner.ScanResult(msg).Err == nil {
			entry := scanner.ScanResult(msg).Entry
			m.tree.AddEntry(entry)
			m.fileCount++
			m.lastPath = entry.Path
		}
		// Continue polling for more results
		return m, pollResults(m.results)

	case scanDoneMsg:
		m.scanning = false
		return m, nil
	}

	return m, nil
}

// visibleItems returns how many items fit in the viewport
func (m Model) visibleItems() int {
	// Reserve lines for header (2-3), total line, and footer
	available := m.height - 6
	if m.scanning {
		available-- // Extra line for "scanning" path
	}
	if available < 5 {
		available = 5
	}
	return available
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
		// Show current path being scanned (truncated)
		if m.lastPath != "" {
			rel, _ := filepath.Rel(m.scanPath, m.lastPath)
			if len(rel) > 60 {
				parts := strings.Split(rel, string(filepath.Separator))
				if len(parts) > 3 {
					rel = filepath.Join(parts[0], "...", parts[len(parts)-1])
				}
			}
			s += helpStyle.Render(fmt.Sprintf("  → %s", rel)) + "\n"
		}
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

	// Status message
	if m.statusMsg != "" {
		s += "\n" + junkStyle.Render(m.statusMsg)
	}

	// Footer
	s += "\n" + helpStyle.Render("[↑↓] Navigate  [Enter] Open dir  [h] Back  [Space] Select  [d] Delete  [Tab] Junk  [q] Quit")

	return s
}

func (m Model) renderTree() string {
	var s string
	children := m.tree.Children(m.currentPath)

	// Show breadcrumb if not at root
	if m.currentPath != m.scanPath {
		rel, _ := filepath.Rel(m.scanPath, m.currentPath)
		s += helpStyle.Render(fmt.Sprintf("📂 %s", rel)) + "\n\n"
	}

	maxItems := m.visibleItems()
	endIdx := m.offset + maxItems
	if endIdx > len(children) {
		endIdx = len(children)
	}

	// Show scroll indicator at top if needed
	if m.offset > 0 {
		s += helpStyle.Render(fmt.Sprintf("  ↑ %d more above\n", m.offset))
	}

	for i := m.offset; i < endIdx; i++ {
		child := children[i]

		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}

		// Selection marker
		selectMark := " "
		if m.selected[child.Path] {
			selectMark = "●"
		}

		icon := "📄"
		if child.IsDir {
			icon = "📁"
		}

		line := fmt.Sprintf("%s%s %s %s %s",
			prefix,
			selectMark,
			icon,
			child.Name,
			sizeStyle.Render(formatSize(child.Size)))

		if i == m.cursor {
			line = selectedStyle.Render(line)
		}

		s += line + "\n"
	}

	// Show scroll indicator at bottom if needed
	if endIdx < len(children) {
		s += helpStyle.Render(fmt.Sprintf("  ↓ %d more below\n", len(children)-endIdx))
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
