package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorBg     = lipgloss.Color("#1A1A1A")
	colorBgAlt  = lipgloss.Color("#252525")
	colorText   = lipgloss.Color("#F7F1FF")
	colorMuted  = lipgloss.Color("#7C7C7C")
	colorDim    = lipgloss.Color("#4A4A4A")
	colorPink   = lipgloss.Color("#FC618D")
	colorCyan   = lipgloss.Color("#5AD4E6")
	colorMint   = lipgloss.Color("#7BD88F")
	colorYellow = lipgloss.Color("#FCE566")
	colorPurple = lipgloss.Color("#948AE3")
)

var (
	stBrand = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorBg).
		Background(colorPink).
		Padding(0, 1)

	stBrandSub = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	stCount = lipgloss.NewStyle().
		Foreground(colorCyan).
		Bold(true).
		Padding(0, 1)

	stDivider = lipgloss.NewStyle().
			Foreground(colorDim)

	stColHeader = lipgloss.NewStyle().
			Foreground(colorPurple).
			Bold(true)

	stRow = lipgloss.NewStyle().
		Foreground(colorText)

	stRowSelected = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorCyan).
			Bold(true)

	stRowDisabled = lipgloss.NewStyle().
			Foreground(colorDim).
			Italic(true)

	stBadgeOn = lipgloss.NewStyle().
			Foreground(colorMint).
			Bold(true)

	stBadgeOff = lipgloss.NewStyle().
			Foreground(colorMuted)

	stError = lipgloss.NewStyle().
		Foreground(colorPink).
		Bold(true)

	stStatusBg = lipgloss.NewStyle().
			Background(colorBgAlt)

	stKey = lipgloss.NewStyle().
		Foreground(colorYellow).
		Background(colorBgAlt).
		Bold(true)

	stKeyDesc = lipgloss.NewStyle().
			Foreground(colorMuted).
			Background(colorBgAlt)

	stKeySep = lipgloss.NewStyle().
			Foreground(colorDim).
			Background(colorBgAlt)

	stSection = lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true)

	stMuted = lipgloss.NewStyle().
		Foreground(colorMuted)

	stValue = lipgloss.NewStyle().
		Foreground(colorText)

	stValueAccent = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	stFieldLabel = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(14).
			Align(lipgloss.Right).
			PaddingRight(2)

	stFieldDesc = lipgloss.NewStyle().
			Foreground(colorMint).
			Italic(true).
			PaddingLeft(2)

	stOk = lipgloss.NewStyle().
		Foreground(colorMint).
		Bold(true)
)

const maxContentWidth = 100

func clampWidth(w int) int {
	if w < 60 {
		return 80
	}
	if w > maxContentWidth {
		return maxContentWidth
	}
	return w
}

func topBar(width int, sub string, right string) string {
	width = clampWidth(width)
	left := stBrand.Render("cronk") + stBrandSub.Render(sub)
	rightR := stCount.Render(right)
	gap := width - lipgloss.Width(left) - lipgloss.Width(rightR)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + rightR
}

func divider(width int) string {
	width = clampWidth(width)
	return stDivider.Render(strings.Repeat("─", width))
}

func statusBar(width int, items ...[2]string) string {
	width = clampWidth(width)
	var parts []string
	for i, it := range items {
		if i > 0 {
			parts = append(parts, stKeySep.Render(" • "))
		}
		parts = append(parts, stKey.Render(" "+it[0]+" ")+stKeyDesc.Render(" "+it[1]))
	}
	content := lipgloss.JoinHorizontal(lipgloss.Left, parts...)
	pad := width - lipgloss.Width(content)
	if pad < 0 {
		pad = 0
	}
	return content + stStatusBg.Render(strings.Repeat(" ", pad))
}
