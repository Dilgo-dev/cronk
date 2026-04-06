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

type formField struct {
	label string
	kind  crontab.FieldKind
	input textinput.Model
}

type formModel struct {
	fields    []formField
	command   textinput.Model
	focus     int
	editing   *crontab.Job
	saveError string
}

func newForm(edit *crontab.Job) formModel {
	labels := []string{"minute", "hour", "day of month", "month", "day of week"}
	kinds := []crontab.FieldKind{crontab.FieldMinute, crontab.FieldHour, crontab.FieldDom, crontab.FieldMonth, crontab.FieldDow}
	defaults := []string{"*", "*", "*", "*", "*"}

	if edit != nil && !strings.HasPrefix(strings.TrimSpace(edit.Schedule), "@") {
		parts := strings.Fields(edit.Schedule)
		if len(parts) == 5 {
			defaults = parts
		}
	}

	fields := make([]formField, 5)
	for i := range fields {
		ti := textinput.New()
		ti.CharLimit = 40
		ti.Width = 14
		ti.SetValue(defaults[i])
		ti.Prompt = "│ "
		ti.PromptStyle = lipgloss.NewStyle().Foreground(colorDim)
		ti.TextStyle = lipgloss.NewStyle().Foreground(colorText)
		ti.Cursor.Style = lipgloss.NewStyle().Foreground(colorPink)
		fields[i] = formField{label: labels[i], kind: kinds[i], input: ti}
	}
	fields[0].input.Focus()
	fields[0].input.PromptStyle = lipgloss.NewStyle().Foreground(colorPink).Bold(true)

	cmd := textinput.New()
	cmd.CharLimit = 500
	cmd.Width = 60
	cmd.Placeholder = "echo hello"
	cmd.Prompt = "│ "
	cmd.PromptStyle = lipgloss.NewStyle().Foreground(colorDim)
	cmd.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	cmd.Cursor.Style = lipgloss.NewStyle().Foreground(colorPink)
	if edit != nil {
		cmd.SetValue(edit.Command)
	}

	return formModel{fields: fields, command: cmd, editing: edit}
}

func (f *formModel) scheduleString() string {
	parts := make([]string, 5)
	for i, fl := range f.fields {
		parts[i] = strings.TrimSpace(fl.input.Value())
	}
	return strings.Join(parts, " ")
}

func (f *formModel) applyPreset(p string) {
	presets := map[string][5]string{
		"@hourly":  {"0", "*", "*", "*", "*"},
		"@daily":   {"0", "0", "*", "*", "*"},
		"@weekly":  {"0", "0", "*", "*", "0"},
		"@monthly": {"0", "0", "1", "*", "*"},
	}
	if v, ok := presets[p]; ok {
		for i := range f.fields {
			f.fields[i].input.SetValue(v[i])
		}
	}
}

func (f *formModel) focusNext(delta int) {
	total := len(f.fields) + 1
	dimPrompt := lipgloss.NewStyle().Foreground(colorDim)
	activePrompt := lipgloss.NewStyle().Foreground(colorPink).Bold(true)
	for i := range f.fields {
		f.fields[i].input.Blur()
		f.fields[i].input.PromptStyle = dimPrompt
	}
	f.command.Blur()
	f.command.PromptStyle = dimPrompt
	f.focus = (f.focus + delta + total) % total
	if f.focus < len(f.fields) {
		f.fields[f.focus].input.Focus()
		f.fields[f.focus].input.PromptStyle = activePrompt
	} else {
		f.command.Focus()
		f.command.PromptStyle = activePrompt
	}
}

func (f formModel) Update(msg tea.Msg) (formModel, tea.Cmd, formResult) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return f, nil, formResult{cancel: true}
		case "tab", "down":
			f.focusNext(1)
			return f, nil, formResult{}
		case "shift+tab", "up":
			f.focusNext(-1)
			return f, nil, formResult{}
		case "ctrl+s":
			if err := f.save(); err != nil {
				f.saveError = err.Error()
				return f, nil, formResult{}
			}
			return f, nil, formResult{saved: true}
		case "ctrl+h":
			f.applyPreset("@hourly")
			return f, nil, formResult{}
		case "ctrl+d":
			f.applyPreset("@daily")
			return f, nil, formResult{}
		case "ctrl+w":
			f.applyPreset("@weekly")
			return f, nil, formResult{}
		case "ctrl+t":
			f.applyPreset("@monthly")
			return f, nil, formResult{}
		}
	}
	if f.focus < len(f.fields) {
		f.fields[f.focus].input, cmd = f.fields[f.focus].input.Update(msg)
	} else {
		f.command, cmd = f.command.Update(msg)
	}
	return f, cmd, formResult{}
}

type formResult struct {
	cancel bool
	saved  bool
}

func (f *formModel) save() error {
	schedule := f.scheduleString()
	command := strings.TrimSpace(f.command.Value())
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}
	if err := crontab.ValidateSchedule(schedule); err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}
	raw, err := crontab.LoadRaw()
	if err != nil {
		return err
	}
	newLine := schedule + " " + command
	var updated string
	if f.editing != nil && f.editing.Raw != "" {
		updated = crontab.ReplaceLine(raw, f.editing.Raw, newLine)
	} else {
		updated = crontab.AppendJob(raw, newLine)
	}
	return crontab.Write(updated)
}

func (f formModel) View(width int) string {
	var b strings.Builder
	title := "add job"
	if f.editing != nil {
		title = "edit job"
	}
	b.WriteString(topBar(width, title, "") + "\n")
	b.WriteString(divider(width) + "\n\n")

	b.WriteString("  " + stSection.Render("◆ schedule") + "\n\n")
	for _, fl := range f.fields {
		desc := crontab.DescribeField(fl.kind, fl.input.Value())
		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			stFieldLabel.Render(fl.label),
			fl.input.View(),
			stFieldDesc.Render(desc),
		)
		b.WriteString(row + "\n")
	}
	b.WriteString("\n  " + stSection.Render("◆ command") + "\n\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, stFieldLabel.Render("command"), f.command.View()) + "\n\n")

	schedule := f.scheduleString()
	if err := crontab.ValidateSchedule(schedule); err != nil {
		b.WriteString("  " + stError.Render("✗ invalid schedule: "+err.Error()) + "\n\n")
	} else {
		b.WriteString("  " + stSection.Render("◆ next 5 runs") + "\n\n")
		runs, _ := crontab.NextRunsFor(schedule, 5, time.Now())
		for i, r := range runs {
			marker := stMuted.Render("    ")
			if i == 0 {
				marker = lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("  → ")
			}
			b.WriteString(marker + stValue.Render(r.Format("Mon 2006-01-02 15:04")) + "\n")
		}
		b.WriteString("\n")
	}

	if f.saveError != "" {
		b.WriteString("  " + stError.Render("✗ "+f.saveError) + "\n\n")
	}

	b.WriteString(divider(width) + "\n")
	b.WriteString(statusBar(width,
		[2]string{"tab", "next"},
		[2]string{"^s", "save"},
		[2]string{"esc", "cancel"},
		[2]string{"^h", "hourly"},
		[2]string{"^d", "daily"},
		[2]string{"^w", "weekly"},
		[2]string{"^t", "monthly"},
	))
	return b.String()
}
