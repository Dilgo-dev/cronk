package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dilgo-dev/cronk/internal/config"
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
	settings  config.Settings
}

func New() Model {
	settings, _ := config.Load()
	jobs, err := crontab.Load()
	si := textinput.New()
	si.Prompt = "  search "
	si.CharLimit = 100
	si.Width = 40
	si.PromptStyle = lipgloss.NewStyle().Foreground(colorPink).Bold(true)
	si.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	return Model{jobs: jobs, err: err, search: si, width: 100, settings: settings}
}

func (m Model) Init() tea.Cmd { return nil }

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

func (m Model) View() string {
	if m.view == viewForm {
		return m.form.View(m.width)
	}
	if m.view == viewDetail {
		m.detail.dateFormat = m.settings.DateFormat
		return m.detail.View(m.width)
	}
	return m.listView()
}

func (m Model) listView() string {
	var b strings.Builder

	count := len(m.jobs)
	right := fmt.Sprintf("%d jobs", count)
	if m.query != "" {
		right = fmt.Sprintf("%d / %d", len(m.filtered()), count)
	}
	b.WriteString(topBar(m.width, "crontab manager", right) + "\n")
	b.WriteString(divider(m.width) + "\n\n")

	if m.err != nil {
		b.WriteString("  " + stError.Render("✗ error") + "\n")
		b.WriteString("  " + stMuted.Render(m.err.Error()) + "\n\n")
		b.WriteString(divider(m.width) + "\n")
		b.WriteString(statusBar(m.width, [2]string{"r", "retry"}, [2]string{"q", "quit"}))
		return b.String()
	}

	if count == 0 {
		b.WriteString("  " + stSection.Render("no jobs yet") + "\n")
		b.WriteString("  " + stMuted.Render("press ") + stKey.Render(" a ") + stMuted.Render(" to add your first cron job") + "\n\n")
		b.WriteString(divider(m.width) + "\n")
		b.WriteString(statusBar(m.width, [2]string{"a", "add"}, [2]string{"r", "reload"}, [2]string{"q", "quit"}))
		return b.String()
	}

	if m.searching || m.query != "" {
		b.WriteString(m.search.View() + "\n\n")
	}

	cw := columnWidths(m.width)
	b.WriteString(renderHeader(cw) + "\n")
	b.WriteString(stDivider.Render(strings.Repeat("╌", cw.total())) + "\n")

	now := time.Now()
	idxs := m.filtered()
	if len(idxs) == 0 {
		b.WriteString("  " + stMuted.Italic(true).Render("no match") + "\n")
	}
	for i, idx := range idxs {
		j := m.jobs[idx]
		b.WriteString(renderRow(j, i == m.cursor, now, cw) + "\n")
	}

	b.WriteString("\n" + divider(m.width) + "\n")
	b.WriteString(statusBar(m.width,
		[2]string{"j/k", "move"},
		[2]string{"⏎", "detail"},
		[2]string{"a", "add"},
		[2]string{"e", "edit"},
		[2]string{"␣", "toggle"},
		[2]string{"/", "search"},
		[2]string{"q", "quit"},
	))
	return b.String()
}

type colWidths struct {
	gutter, schedule, command, next, status int
}

func (c colWidths) total() int {
	return c.gutter + c.schedule + 2 + c.command + 2 + c.next + 2 + c.status
}

func columnWidths(termWidth int) colWidths {
	termWidth = clampWidth(termWidth)
	cw := colWidths{gutter: 2, schedule: 18, next: 18, status: 10}
	used := cw.gutter + cw.schedule + 2 + cw.next + 2 + cw.status + 2
	cw.command = termWidth - used
	if cw.command < 20 {
		cw.command = 20
	}
	return cw
}

func renderHeader(cw colWidths) string {
	return strings.Repeat(" ", cw.gutter) +
		stColHeader.Render(pad("SCHEDULE", cw.schedule)) + "  " +
		stColHeader.Render(pad("COMMAND", cw.command)) + "  " +
		stColHeader.Render(pad("NEXT RUN", cw.next)) + "  " +
		stColHeader.Render(pad("STATUS", cw.status))
}

func renderRow(j crontab.Job, selected bool, now time.Time, cw colWidths) string {
	schedule := truncate(j.Schedule, cw.schedule)
	command := truncate(j.Command, cw.command)
	next := nextRunStr(j, now)
	if lipgloss.Width(next) > cw.next {
		next = truncate(next, cw.next)
	}

	var statusText string
	if j.Disabled {
		statusText = "○ off"
	} else {
		statusText = "● on"
	}
	statusText = pad(statusText, cw.status)

	gutter := "  "
	if selected {
		gutter = lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("▎ ")
	}

	body := pad(schedule, cw.schedule) + "  " +
		pad(command, cw.command) + "  " +
		pad(next, cw.next) + "  "

	if selected {
		body = stRowSelected.Render(body + statusText)
		return gutter + body
	}

	if j.Disabled {
		body = stRowDisabled.Render(body)
		statusText = stBadgeOff.Render(statusText)
	} else {
		body = stRow.Render(body)
		statusText = stBadgeOn.Render(statusText)
	}
	return gutter + body + statusText
}

func pad(s string, n int) string {
	if lipgloss.Width(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-lipgloss.Width(s))
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
	if lipgloss.Width(s) <= n {
		return s
	}
	if n < 4 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
