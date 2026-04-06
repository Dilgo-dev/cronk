package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Dilgo-dev/cronk/internal/crontab"
)

type detailModel struct {
	job crontab.Job
}

var (
	detailLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Width(16)
	detailValue = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	detailHead  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFAF00"))
)

func (d detailModel) View() string {
	var b strings.Builder
	b.WriteString(formTitle.Render("cronk - job detail") + "\n\n")

	b.WriteString(detailLabel.Render("schedule") + detailValue.Render(d.job.Schedule) + "\n")
	b.WriteString(detailLabel.Render("command") + detailValue.Render(d.job.Command) + "\n")
	status := "enabled"
	if d.job.Disabled {
		status = "disabled"
	}
	b.WriteString(detailLabel.Render("status") + detailValue.Render(status) + "\n\n")

	b.WriteString(detailHead.Render("decoded fields") + "\n")
	if d.job.Reboot {
		b.WriteString("  runs once at system boot\n")
	} else {
		parts := strings.Fields(d.job.Schedule)
		labels := []string{"minute", "hour", "day of month", "month", "day of week"}
		kinds := []crontab.FieldKind{crontab.FieldMinute, crontab.FieldHour, crontab.FieldDom, crontab.FieldMonth, crontab.FieldDow}
		if len(parts) == 5 {
			for i, p := range parts {
				desc := crontab.DescribeField(kinds[i], p)
				b.WriteString("  " + detailLabel.Render(labels[i]) + detailValue.Render(p) + fieldDesc.Render("  "+desc) + "\n")
			}
		} else {
			b.WriteString("  " + d.job.Schedule + "\n")
		}
	}
	b.WriteString("\n")

	b.WriteString(detailHead.Render("next 10 runs") + "\n")
	if d.job.Reboot {
		b.WriteString("  on reboot\n")
	} else {
		runs, err := crontab.NextRunsFor(d.job.Schedule, 10, time.Now())
		if err != nil {
			b.WriteString(fieldErr.Render("  cannot compute: "+err.Error()) + "\n")
		} else {
			for _, r := range runs {
				b.WriteString(previewLine.Render("  "+r.Format("Mon 2006-01-02 15:04")) + "\n")
			}
		}
	}

	b.WriteString("\n" + helpStyle.Render("esc/q back"))
	return b.String()
}
