package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dilgo-dev/cronk/internal/crontab"
)

type Model struct {
	jobs   []crontab.Job
	cursor int
	err    error
	width  int
	height int
}

func New() Model {
	jobs, err := crontab.Load()
	return Model{jobs: jobs, err: err}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.jobs)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "g", "home":
			m.cursor = 0
		case "G", "end":
			if len(m.jobs) > 0 {
				m.cursor = len(m.jobs) - 1
			}
		case "r":
			jobs, err := crontab.Load()
			m.jobs, m.err = jobs, err
			if m.cursor >= len(m.jobs) {
				m.cursor = max(0, len(m.jobs)-1)
			}
		}
	}
	return m, nil
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFAF00")).Padding(0, 1)
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#888888"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAF00")).Bold(true)
	disabledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Strikethrough(true)
	statusOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("#7CB342"))
	statusOff     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
)

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("cronk") + "\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render("error: "+m.err.Error()) + "\n")
		b.WriteString(helpStyle.Render("press q to quit, r to retry"))
		return b.String()
	}

	if len(m.jobs) == 0 {
		b.WriteString(helpStyle.Render("no jobs in your crontab. press a to add one (todo).") + "\n\n")
		b.WriteString(helpStyle.Render("q quit  r reload"))
		return b.String()
	}

	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-20s %-30s %-20s %s", "SCHEDULE", "COMMAND", "NEXT RUN", "STATUS")) + "\n")

	now := time.Now()
	for i, j := range m.jobs {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		schedule := truncate(j.Schedule, 20)
		command := truncate(j.Command, 30)
		next := nextRunStr(j, now)
		status := statusOK.Render("enabled")
		if j.Disabled {
			status = statusOff.Render("disabled")
		}
		line := fmt.Sprintf("%s%-20s %-30s %-20s %s", cursor, schedule, command, next, status)
		if j.Disabled {
			line = disabledStyle.Render(line)
		} else if i == m.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k move  g/G top/bottom  r reload  q quit"))
	return b.String()
}

func nextRunStr(j crontab.Job, from time.Time) string {
	if j.Reboot {
		return "on reboot"
	}
	t, ok := j.NextRun(from)
	if !ok {
		return "-"
	}
	d := time.Until(t).Round(time.Second)
	return "in " + humanDuration(d)
}

func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", h, m)
	}
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, h)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n < 4 {
		return s[:n]
	}
	return s[:n-1] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
