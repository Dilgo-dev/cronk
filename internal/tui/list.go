package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dilgo-dev/cronk/internal/crontab"
)

type view int

const (
	viewList view = iota
	viewForm
	viewDetail
)

type Model struct {
	view      view
	jobs      []crontab.Job
	cursor    int
	err       error
	width     int
	height    int
	form      formModel
	detail    detailModel
	searching bool
	search    textinput.Model
	query     string
}

func New() Model {
	jobs, err := crontab.Load()
	si := textinput.New()
	si.Prompt = "/"
	si.CharLimit = 100
	si.Width = 40
	return Model{jobs: jobs, err: err, search: si}
}

func (m Model) filtered() []int {
	if m.query == "" {
		out := make([]int, len(m.jobs))
		for i := range m.jobs {
			out[i] = i
		}
		return out
	}
	q := strings.ToLower(m.query)
	var out []int
	for i, j := range m.jobs {
		if strings.Contains(strings.ToLower(j.Schedule), q) || strings.Contains(strings.ToLower(j.Command), q) {
			out = append(out, i)
		}
	}
	return out
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) selected() (crontab.Job, bool) {
	idx := m.filtered()
	if len(idx) == 0 || m.cursor >= len(idx) {
		return crontab.Job{}, false
	}
	return m.jobs[idx[m.cursor]], true
}

func toggleJob(j crontab.Job) error {
	raw, err := crontab.LoadRaw()
	if err != nil {
		return err
	}
	var newLine string
	if j.Disabled {
		trimmed := strings.TrimLeft(j.Raw, " \t")
		trimmed = strings.TrimPrefix(trimmed, "#")
		newLine = strings.TrimLeft(trimmed, " \t")
	} else {
		newLine = "# " + j.Raw
	}
	updated := crontab.ReplaceLine(raw, j.Raw, newLine)
	return crontab.Write(updated)
}

func (m *Model) reload() {
	jobs, err := crontab.Load()
	m.jobs, m.err = jobs, err
	if m.cursor >= len(m.jobs) {
		m.cursor = max(0, len(m.jobs)-1)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	}

	if m.view == viewForm {
		var cmd tea.Cmd
		var res formResult
		m.form, cmd, res = m.form.Update(msg)
		if res.cancel {
			m.view = viewList
		}
		if res.saved {
			m.view = viewList
			m.reload()
		}
		return m, cmd
	}

	if m.view == viewDetail {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "esc", "q":
				m.view = viewList
			}
		}
		return m, nil
	}

	if m.searching {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "esc":
				m.searching = false
				m.search.SetValue("")
				m.query = ""
				m.cursor = 0
				return m, nil
			case "enter":
				m.searching = false
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		m.query = m.search.Value()
		m.cursor = 0
		return m, cmd
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			n := len(m.filtered())
			if m.cursor < n-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "g", "home":
			m.cursor = 0
		case "G", "end":
			n := len(m.filtered())
			if n > 0 {
				m.cursor = n - 1
			}
		case "r":
			m.reload()
		case "/":
			m.searching = true
			m.search.Focus()
			return m, nil
		case "esc":
			if m.query != "" {
				m.query = ""
				m.search.SetValue("")
				m.cursor = 0
			}
		case " ":
			if j, ok := m.selected(); ok {
				if err := toggleJob(j); err != nil {
					m.err = err
				} else {
					m.reload()
				}
			}
		case "enter":
			if j, ok := m.selected(); ok {
				m.detail = detailModel{job: j}
				m.view = viewDetail
			}
		case "a":
			m.form = newForm(nil)
			m.view = viewForm
		case "e":
			if j, ok := m.selected(); ok {
				m.form = newForm(&j)
				m.view = viewForm
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
	if m.view == viewForm {
		return m.form.View()
	}
	if m.view == viewDetail {
		return m.detail.View()
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("cronk") + "\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render("error: "+m.err.Error()) + "\n")
		b.WriteString(helpStyle.Render("press q to quit, r to retry"))
		return b.String()
	}

	if len(m.jobs) == 0 {
		b.WriteString(helpStyle.Render("no jobs in your crontab. press a to add one.") + "\n\n")
		b.WriteString(helpStyle.Render("a add  q quit  r reload"))
		return b.String()
	}

	if m.searching || m.query != "" {
		b.WriteString(m.search.View() + "\n\n")
	}

	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-20s %-30s %-20s %s", "SCHEDULE", "COMMAND", "NEXT RUN", "STATUS")) + "\n")

	now := time.Now()
	idxs := m.filtered()
	if len(idxs) == 0 {
		b.WriteString(helpStyle.Render("  no match") + "\n")
	}
	for i, idx := range idxs {
		j := m.jobs[idx]
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
	b.WriteString(helpStyle.Render("j/k move  enter detail  a add  e edit  space toggle  / search  r reload  q quit"))
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
