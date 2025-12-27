// Package tui provides a terminal user interface for trak using bubbletea.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/laurent/trak/internal/ops"
)

// View represents the current view in the TUI.
type View int

const (
	ViewMain View = iota
	ViewRemoteBrowser
	ViewNewTrack
)

// Model is the main bubbletea model for trak TUI.
type Model struct {
	ops            *ops.Ops
	repoName       string
	view           View
	table          table.Model
	remoteTable    table.Model
	spinner        spinner.Model
	help           help.Model
	keys           KeyMap
	tracks         []ops.TrackWithStatus
	remoteBranches []ops.RemoteBranch
	loading        bool
	notification   string
	notifyTime     time.Time
	err            error
	width          int
	height         int
	quitting       bool
	// New track input
	textInput textinput.Model
}

// KeyMap defines the keybindings for the TUI.
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Refresh key.Binding
	Browse  key.Binding
	New     key.Binding
	Delete  key.Binding
	Sync    key.Binding
	AI      key.Binding
	Back    key.Binding
	Quit    key.Binding
	Help    key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "jump to track"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Browse: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "browse remote"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new track"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete track"),
		),
		Sync: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sync track"),
		),
		AI: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "run AI"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// ShortHelp returns keybindings to show in the mini help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Refresh, k.Browse, k.Enter, k.Quit, k.Help}
}

// FullHelp returns keybindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Refresh, k.Browse, k.New},
		{k.Sync, k.AI, k.Delete},
		{k.Back, k.Quit, k.Help},
	}
}

// Styles for the TUI - clean, high-contrast color scheme.
var (
	// Soft blue for titles and primary accent
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("117")). // Light blue
			MarginBottom(1)

	// Clean header with good contrast
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")). // White
			Background(lipgloss.Color("238")). // Medium gray
			Padding(0, 1)

	// Soft highlight for selected row - blue background, white text
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")). // White
			Background(lipgloss.Color("24"))   // Deep blue

	// Light gray for normal text - good readability
	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")) // Light gray

	// Muted gray for secondary text
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")) // Medium gray

	// Soft red for errors - not too harsh
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("167")) // Soft red

	// Soft green for success/info
	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")) // Soft green

	// Muted help text
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")). // Gray
			MarginTop(1)

	// Input prompt style
	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")). // Light blue
			Bold(true)

	// Input box style
	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("117")). // Light blue border
			Padding(0, 1)
)

// New creates a new TUI model.
func New(o *ops.Ops, repoName string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("117")) // Match title color

	h := help.New()
	h.ShowAll = false

	// Setup text input for new track creation
	ti := textinput.New()
	ti.Placeholder = "feature/my-new-branch"
	ti.CharLimit = 100
	ti.Width = 40
	ti.PromptStyle = inputStyle
	ti.TextStyle = normalStyle
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))

	return Model{
		ops:       o,
		repoName:  repoName,
		view:      ViewMain,
		spinner:   s,
		help:      h,
		keys:      DefaultKeyMap(),
		loading:   true,
		width:     80,
		height:    24,
		textInput: ti,
	}
}

// Messages for async operations.

type tracksLoadedMsg struct {
	tracks []ops.TrackWithStatus
}

type remoteBranchesLoadedMsg struct {
	branches []ops.RemoteBranch
}

type operationCompleteMsg struct {
	message string
	isError bool
}

type errMsg struct {
	err error
}

// Commands

func (m Model) loadTracks() tea.Msg {
	tracks, err := m.ops.ListTracksWithStatus()
	if err != nil {
		return errMsg{err}
	}
	return tracksLoadedMsg{tracks}
}

func (m Model) loadRemoteBranches() tea.Msg {
	branches, err := m.ops.ListRemoteBranches()
	if err != nil {
		return errMsg{err}
	}
	return remoteBranchesLoadedMsg{branches}
}

func (m Model) jumpToTrack(branch string) tea.Cmd {
	return func() tea.Msg {
		err := m.ops.JumpToTrack(branch)
		if err != nil {
			return operationCompleteMsg{message: err.Error(), isError: true}
		}
		return operationCompleteMsg{message: fmt.Sprintf("Jumped to %s", branch), isError: false}
	}
}

func (m Model) syncTrack(branch string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.ops.SyncTrack(branch)
		if err != nil {
			return operationCompleteMsg{message: err.Error(), isError: true}
		}
		if result.HasConflicts {
			return operationCompleteMsg{
				message: fmt.Sprintf("Conflicts in %s - resolve manually", result.ConflictsPath),
				isError: true,
			}
		}
		msg := "Synced"
		if result.PRCreated {
			msg = fmt.Sprintf("Synced, PR #%d created", result.PRNumber)
		}
		return operationCompleteMsg{message: msg, isError: false}
	}
}

func (m Model) deleteTrack(branch string) tea.Cmd {
	return func() tea.Msg {
		err := m.ops.DeleteTrack(branch, false)
		if err != nil {
			return operationCompleteMsg{message: err.Error(), isError: true}
		}
		return operationCompleteMsg{message: fmt.Sprintf("Deleted %s", branch), isError: false}
	}
}

func (m Model) runAI(branch string) tea.Cmd {
	return func() tea.Msg {
		err := m.ops.RunAI(branch)
		if err != nil {
			return operationCompleteMsg{message: err.Error(), isError: true}
		}
		return operationCompleteMsg{message: fmt.Sprintf("AI started for %s", branch), isError: false}
	}
}

func (m Model) createTrackFromRemote(branch string) tea.Cmd {
	return func() tea.Msg {
		err := m.ops.NewTrackWorktree(branch)
		if err != nil {
			return operationCompleteMsg{message: err.Error(), isError: true}
		}
		return operationCompleteMsg{message: fmt.Sprintf("Created track for %s", branch), isError: false}
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadTracks,
	)
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Clear old notifications on any key press
		if time.Since(m.notifyTime) > 3*time.Second {
			m.notification = ""
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.keys.Back):
			if m.view == ViewRemoteBrowser || m.view == ViewNewTrack {
				m.view = ViewMain
				m.textInput.Blur()
				return m, nil
			}

		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			if m.view == ViewRemoteBrowser {
				return m, m.loadRemoteBranches
			}
			return m, m.loadTracks

		case key.Matches(msg, m.keys.Browse):
			if m.view == ViewMain {
				m.view = ViewRemoteBrowser
				m.loading = true
				return m, m.loadRemoteBranches
			}

		case key.Matches(msg, m.keys.New):
			if m.view == ViewMain {
				m.view = ViewNewTrack
				m.textInput.SetValue("")
				m.textInput.Focus()
				return m, textinput.Blink
			}

		case key.Matches(msg, m.keys.Enter):
			if m.view == ViewNewTrack {
				branch := strings.TrimSpace(m.textInput.Value())
				if branch != "" {
					m.view = ViewMain
					m.textInput.Blur()
					m.loading = true
					return m, m.createTrackFromRemote(branch)
				}
				return m, nil
			} else if m.view == ViewMain && len(m.tracks) > 0 {
				idx := m.table.Cursor()
				if idx < len(m.tracks) {
					branch := m.tracks[idx].Track.Branch
					m.quitting = true
					return m, tea.Sequence(m.jumpToTrack(branch), tea.Quit)
				}
			} else if m.view == ViewRemoteBrowser && len(m.remoteBranches) > 0 {
				idx := m.remoteTable.Cursor()
				if idx < len(m.remoteBranches) {
					branch := m.remoteBranches[idx].Name
					m.loading = true
					return m, m.createTrackFromRemote(branch)
				}
			}

		case key.Matches(msg, m.keys.Sync):
			if m.view == ViewMain && len(m.tracks) > 0 {
				idx := m.table.Cursor()
				if idx < len(m.tracks) {
					branch := m.tracks[idx].Track.Branch
					m.loading = true
					return m, m.syncTrack(branch)
				}
			}

		case key.Matches(msg, m.keys.Delete):
			if m.view == ViewMain && len(m.tracks) > 0 {
				idx := m.table.Cursor()
				if idx < len(m.tracks) {
					branch := m.tracks[idx].Track.Branch
					m.loading = true
					return m, m.deleteTrack(branch)
				}
			}

		case key.Matches(msg, m.keys.AI):
			if m.view == ViewMain && len(m.tracks) > 0 {
				idx := m.table.Cursor()
				if idx < len(m.tracks) {
					branch := m.tracks[idx].Track.Branch
					m.quitting = true
					return m, tea.Sequence(m.runAI(branch), tea.Quit)
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.table = m.buildMainTable()
		m.remoteTable = m.buildRemoteTable()

	case tracksLoadedMsg:
		m.loading = false
		m.tracks = msg.tracks
		m.table = m.buildMainTable()

	case remoteBranchesLoadedMsg:
		m.loading = false
		m.remoteBranches = msg.branches
		m.remoteTable = m.buildRemoteTable()

	case operationCompleteMsg:
		m.loading = false
		m.notification = msg.message
		m.notifyTime = time.Now()
		if msg.isError {
			m.err = fmt.Errorf("%s", msg.message)
		} else {
			m.err = nil
			// Refresh tracks after successful operation
			if m.view == ViewMain {
				return m, m.loadTracks
			}
			m.view = ViewMain
			return m, m.loadTracks
		}

	case errMsg:
		m.loading = false
		m.err = msg.err
		m.notification = msg.err.Error()
		m.notifyTime = time.Now()

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update the appropriate component based on view
	switch m.view {
	case ViewMain:
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	case ViewRemoteBrowser:
		m.remoteTable, cmd = m.remoteTable.Update(msg)
		cmds = append(cmds, cmd)
	case ViewNewTrack:
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the TUI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Header
	header := titleStyle.Render(fmt.Sprintf("┌─ trak ─────────────────────────────────────────── repo: %s ─┐", m.repoName))
	b.WriteString(header)
	b.WriteString("\n")

	// Loading indicator
	if m.loading {
		b.WriteString(fmt.Sprintf("  %s Loading...\n", m.spinner.View()))
	} else {
		b.WriteString("\n")
	}

	// Main content
	switch m.view {
	case ViewMain:
		b.WriteString(m.renderMainView())
	case ViewRemoteBrowser:
		b.WriteString(m.renderRemoteBrowserView())
	case ViewNewTrack:
		b.WriteString(m.renderNewTrackView())
	}

	// Notification
	if m.notification != "" && time.Since(m.notifyTime) < 5*time.Second {
		notifStyle := infoStyle
		if m.err != nil {
			notifStyle = errorStyle
		}
		b.WriteString("\n")
		b.WriteString(notifStyle.Render("  " + m.notification))
	}

	// Help
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(m.help.View(m.keys)))

	return b.String()
}

func (m Model) renderMainView() string {
	if len(m.tracks) == 0 {
		return dimStyle.Render("  No tracks yet. Press 'n' to create one or 'b' to browse remote branches.")
	}
	return m.table.View()
}

func (m Model) renderRemoteBrowserView() string {
	var b strings.Builder
	b.WriteString(infoStyle.Render("  Remote Branches (press enter to create track, esc to go back)"))
	b.WriteString("\n\n")

	if len(m.remoteBranches) == 0 {
		b.WriteString(dimStyle.Render("  No remote branches found."))
		return b.String()
	}

	b.WriteString(m.remoteTable.View())
	return b.String()
}

func (m Model) renderNewTrackView() string {
	var b strings.Builder
	b.WriteString(inputStyle.Render("  New Track"))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("  Enter a branch name to create a new worktree track."))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Press Enter to create, Esc to cancel."))
	b.WriteString("\n\n")
	b.WriteString("  ")
	b.WriteString(inputBoxStyle.Render(m.textInput.View()))
	return b.String()
}

func (m Model) buildMainTable() table.Model {
	columns := []table.Column{
		{Title: "BRANCH", Width: 25},
		{Title: "TYPE", Width: 10},
		{Title: "GIT", Width: 8},
		{Title: "PR", Width: 6},
		{Title: "CI", Width: 4},
		{Title: "REVIEW", Width: 6},
		{Title: "AGE", Width: 8},
	}

	rows := make([]table.Row, 0, len(m.tracks))
	for _, t := range m.tracks {
		branch := truncate(t.Track.Branch, 25)
		trackType := string(t.Track.Type)

		gitStatus := t.Status.GitStatus.String()

		prStr := "—"
		if t.Status.PR != nil {
			prStr = fmt.Sprintf("#%d", t.Status.PR.Number)
		}

		ciStr := "—"
		if t.Status.CI != nil {
			ciStr = t.Status.CI.Symbol()
		}

		reviewStr := "—"
		if t.Status.Review != nil {
			reviewStr = t.Status.Review.Symbol()
		}

		age := formatAge(t.Track.CreatedAt)

		rows = append(rows, table.Row{branch, trackType, gitStatus, prStr, ciStr, reviewStr, age})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(minInt(len(rows)+1, m.height-10)),
	)

	s := table.DefaultStyles()
	s.Header = headerStyle
	s.Selected = selectedStyle
	s.Cell = normalStyle
	t.SetStyles(s)

	return t
}

func (m Model) buildRemoteTable() table.Model {
	columns := []table.Column{
		{Title: "BRANCH", Width: 30},
		{Title: "PR", Width: 8},
		{Title: "AGE", Width: 10},
	}

	rows := make([]table.Row, 0, len(m.remoteBranches))
	for _, b := range m.remoteBranches {
		branch := truncate(b.Name, 30)

		prStr := "—"
		if b.HasPR {
			prStr = fmt.Sprintf("#%d", b.PRNumber)
		}

		age := formatDuration(b.Age)

		rows = append(rows, table.Row{branch, prStr, age})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(minInt(len(rows)+1, m.height-12)),
	)

	s := table.DefaultStyles()
	s.Header = headerStyle
	s.Selected = selectedStyle
	s.Cell = normalStyle
	t.SetStyles(s)

	return t
}

// Run starts the TUI.
func Run(o *ops.Ops, repoName string) error {
	m := New(o, repoName)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Helper functions

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatAge(t time.Time) string {
	return formatDuration(time.Since(t))
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days < 7 {
		return fmt.Sprintf("%dd", days)
	}
	weeks := days / 7
	if weeks < 4 {
		return fmt.Sprintf("%dw", weeks)
	}
	months := days / 30
	return fmt.Sprintf("%dmo", months)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
