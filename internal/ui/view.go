package ui

import (
	"fmt"
	"strings"

	"github.com/JonathanInTheClouds/gstash/internal/git"
	"github.com/charmbracelet/lipgloss"
)

const listWidth = 36

var (
	subtle    = lipgloss.Color("#6B7280")
	highlight = lipgloss.Color("#7C3AED")
	accent    = lipgloss.Color("#A78BFA")
	green     = lipgloss.Color("#34D399")
	red       = lipgloss.Color("#F87171")
	amber     = lipgloss.Color("#F59E0B")
	white     = lipgloss.Color("#F9FAFB")
	dimmed    = lipgloss.Color("#9CA3AF")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Background(highlight).
			Foreground(white).
			Bold(true).
			Width(listWidth).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Foreground(white).
			Width(listWidth).
			Padding(0, 1)

	branchStyle = lipgloss.NewStyle().
			Foreground(green).
			Bold(true)

	dateStyle = lipgloss.NewStyle().
			Foreground(dimmed)

	dividerStyle = lipgloss.NewStyle().
			Foreground(subtle)

	statusStyle = lipgloss.NewStyle().
			Foreground(dimmed).
			Italic(true).
			Padding(0, 1)

	warningStyle = lipgloss.NewStyle().
			Foreground(amber).
			Bold(true).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(red).
			Bold(true).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Padding(0, 1)

	inputStyle = lipgloss.NewStyle().
			Foreground(accent).
			Padding(0, 1)
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Hard error — show full screen message
	if m.err != nil && m.warning == "" {
		return errorStyle.Render("✗ "+m.err.Error()) +
			"\n\n" + helpStyle.Render("Press any key to continue, q to quit.")
	}

	divider := dividerStyle.Render(strings.Repeat("│\n", m.height-2))

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderList(),
		divider,
		m.renderPreview(),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeader(),
		body,
		m.renderStatusBar(),
	)
}

func (m Model) renderHeader() string {
	title := titleStyle.Render("⎇  gstash")
	active := m.activeStashes()
	count := statusStyle.Render(fmt.Sprintf("%d stash(es)", len(active)))
	pad := m.width - lipgloss.Width(title) - lipgloss.Width(count)
	if pad < 0 {
		pad = 0
	}
	return title + strings.Repeat(" ", pad) + count
}

func (m Model) renderList() string {
	active := m.activeStashes()

	if len(active) == 0 {
		msg := "No stashes found."
		if m.mode == modeSearch {
			msg = "No matches."
		}
		return lipgloss.NewStyle().
			Foreground(dimmed).
			Italic(true).
			Width(listWidth).
			Padding(1, 2).
			Render(msg)
	}

	var rows []string
	for i, s := range active {
		label := truncate(s.Message, listWidth-4)
		branch := branchStyle.Render(truncate(s.Branch, 16))
		date := dateStyle.Render(git.RelativeTime(s.Date))
		line2 := branch + "  " + date

		if i == m.cursor {
			rows = append(rows,
				selectedStyle.Render(label),
				selectedStyle.Render(line2),
				selectedStyle.Render(""),
			)
		} else {
			rows = append(rows,
				normalStyle.Render(label),
				normalStyle.Render(line2),
				normalStyle.Render(""),
			)
		}
	}

	return strings.Join(rows, "\n")
}

func (m Model) renderPreview() string {
	previewWidth := m.width - listWidth - 3
	if previewWidth < 10 {
		return ""
	}

	if m.diff == "" {
		return lipgloss.NewStyle().
			Foreground(dimmed).
			Italic(true).
			Padding(1, 2).
			Render("Select a stash to preview its diff.")
	}

	lines := strings.Split(m.diff, "\n")
	var colored []string
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			colored = append(colored, lipgloss.NewStyle().Foreground(green).Render(line))
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			colored = append(colored, lipgloss.NewStyle().Foreground(red).Render(line))
		case strings.HasPrefix(line, "@@"):
			colored = append(colored, lipgloss.NewStyle().Foreground(accent).Render(line))
		case strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "index "):
			colored = append(colored, lipgloss.NewStyle().Foreground(dimmed).Render(line))
		default:
			colored = append(colored, line)
		}
	}

	m.preview.SetContent(strings.Join(colored, "\n"))
	return m.preview.View()
}

func (m Model) renderStatusBar() string {
	// Priority: warning > rename input > search input > help
	if m.warning != "" {
		return warningStyle.Render(m.warning)
	}

	if m.mode == modeRename {
		return inputStyle.Render("Rename: ") + m.renameInput.View()
	}

	if m.mode == modeSearch {
		return inputStyle.Render("/") + m.searchInput.View()
	}

	if m.message != "" {
		return statusStyle.Render(m.message)
	}

	return helpStyle.Render("↑↓ navigate   / search   a apply   p pop   d drop   r rename   pgup/pgdn scroll   q quit")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}