package utils

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

var (
	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Padding(0, 1)

	cellStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")).
		Padding(0, 1)

	borderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
)

func PrintTable(headers []string, rows [][]string) {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(borderStyle).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	PrintGeneric(t.Render())
}
