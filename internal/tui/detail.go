package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Dilgo-dev/cronk/internal/crontab"
)

type detailModel struct {
	job        crontab.Job
	dateFormat string
}

func (d detailModel) fmtDate(t time.Time) string {
	if d.dateFormat == "" {
		return t.Format("Mon 2006-01-02 15:04")
	}
	return t.Format(d.dateFormat)
}

func (d detailModel) View(width int) string {
	var b strings.Builder
	b.WriteString(topBar(width, "job detail", "") + "\n")
	b.WriteString(divider(width) + "\n\n")

	var left strings.Builder
	left.WriteString("  " + stSection.Render("◆ overview") + "\n\n")
	left.WriteString(stFieldLabel.Render("schedule") + stValueAccent.Render(d.job.Schedule) + "\n")
	left.WriteString(stFieldLabel.Render("command") + stValue.Render(d.job.Command) + "\n")
	statusVal := stOk.Render("● enabled")
	if d.job.Disabled {
		statusVal = stMuted.Render("○ disabled")
	}
	left.WriteString(stFieldLabel.Render("status") + statusVal + "\n\n")

	left.WriteString("  " + stSection.Render("◆ decoded") + "\n\n")
	if d.job.Reboot {
		left.WriteString("  " + stMuted.Render("runs once at system boot") + "\n")
	} else {
		parts := strings.Fields(d.job.Schedule)
		labels := []string{"minute", "hour", "day of month", "month", "day of week"}
		kinds := []crontab.FieldKind{crontab.FieldMinute, crontab.FieldHour, crontab.FieldDom, crontab.FieldMonth, crontab.FieldDow}
		if len(parts) == 5 {
			for i, p := range parts {
				desc := crontab.DescribeField(kinds[i], p)
				row := lipgloss.JoinHorizontal(
					lipgloss.Top,
					stFieldLabel.Render(labels[i]),
					lipgloss.NewStyle().Foreground(colorYellow).Render(pad(p, 6)),
					stFieldDesc.Render(desc),
				)
				left.WriteString(row + "\n")
			}
		}
	}

	var right strings.Builder
	right.WriteString(stSection.Render("◆ next 10 runs") + "\n\n")
	if d.job.Reboot {
		right.WriteString(stMuted.Render("on reboot") + "\n")
	} else {
		runs, err := crontab.NextRunsFor(d.job.Schedule, 10, time.Now())
		if err != nil {
			right.WriteString(stError.Render("cannot compute: "+err.Error()) + "\n")
		} else {
			for i, r := range runs {
				marker := stMuted.Render("  ")
				if i == 0 {
					marker = lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("→ ")
				}
				right.WriteString(marker + stValue.Render(d.fmtDate(r)) + "\n")
			}
		}
	}

	leftBlock := lipgloss.NewStyle().Width((clampWidth(width) / 2) - 2).Render(left.String())
	rightBlock := lipgloss.NewStyle().Padding(0, 0, 0, 2).Render(right.String())
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightBlock) + "\n\n")

	b.WriteString(divider(width) + "\n")
	b.WriteString(statusBar(width, [2]string{"esc/q", "back"}))
	return b.String()
}
