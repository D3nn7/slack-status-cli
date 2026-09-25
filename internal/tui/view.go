package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/D3nn7/slack-status-cli/internal/domain"
	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 {
		return "Lade …"
	}
	var sections []string
	sections = append(sections, m.renderHeader())

	switch m.screen {
	case screenDashboard:
		sections = append(sections, m.renderStatusCard(), m.renderDashboard())
	case screenCalendar:
		sections = append(sections, m.renderStatusCard(), m.panel(m.calendar.view(m)))
	case screenComposer, screenTemplateEditor, screenScheduleEditor, screenSettings:
		sections = append(sections, m.panel(m.formView()))
	case screenEmoji:
		sections = append(sections, m.panel(m.emojiPicker.view()))
	case screenDuration:
		sections = append(sections, m.panel(m.durationPicker.view()))
	case screenCalStatus:
		sections = append(sections, m.panel(m.renderCalStatus()))
	case screenConfirm:
		sections = append(sections, m.panel(m.renderConfirm()))
	case screenHelp:
		sections = append(sections, m.panel(m.renderHelp()))
	}

	sections = append(sections, m.renderFooter())
	return clampLines(lipgloss.JoinVertical(lipgloss.Left, sections...), m.height)
}

func (m Model) formView() string {
	switch m.screen {
	case screenComposer:
		return m.composer.view(m.panelWidth())
	case screenTemplateEditor:
		return m.templateEditor.view()
	case screenScheduleEditor:
		return m.scheduleEditor.view()
	case screenSettings:
		return m.settings.view()
	}
	return ""
}

func (m Model) panel(content string) string {
	return styleCard.Width(m.panelWidth() - 4).Render(content)
}

func (m Model) renderHeader() string {
	title := styleTitle.Render("  Slack Status  ")
	user := ""
	if m.user.Label() != "" {
		user = styleAccent.Render("  @" + m.user.Label())
	}
	line := title + user
	return lipgloss.NewStyle().Padding(0, 1).Render(line)
}

func (m Model) renderStatusCard() string {
	glyph := emojiDisplay(m.status.Emoji)
	inner := m.panelWidth() - 8
	if inner < 10 {
		inner = 10
	}
	text := m.status.Text
	if text == "" {
		text = styleSubtle.Render("No status set")
	} else {
		text = styleValue.Render(truncate(text, inner-4))
	}
	if m.loading {
		text += "  " + styleAccent.Render(m.spinner.View())
	}

	expiry := "never expires"
	if !m.status.Expiration.IsZero() {
		remaining := time.Until(m.status.Expiration).Round(time.Minute)
		expiry = fmt.Sprintf("clears at %s", m.status.Expiration.Local().Format("15:04"))
		if remaining > 0 {
			expiry += " (" + humanizeDuration(remaining) + ")"
		} else {
			expiry += " (expired)"
		}
	}
	sub := "@" + fallback(m.user.Label(), "unbekannt") + " · " + expiry

	if m.compactLayout() {
		// Two-line header without a bordered card to save vertical space.
		return lipgloss.NewStyle().Padding(0, 1).Render(
			lipgloss.NewStyle().Width(3).Render(glyph) + " " + text + "\n" +
				styleSubtle.Render(truncate(sub+" · "+plainCalIndicator(m), inner)))
	}

	first := fmt.Sprintf("%s  %s", lipgloss.NewStyle().Width(3).Render(glyph), text)
	second := styleSubtle.Render(truncate(sub, inner))
	third := m.renderCalIndicator()

	body := first + "\n" + second + "\n" + third
	return styleCard.Width(m.panelWidth() - 4).Render(body)
}

// plainCalIndicator returns an unstyled, truncatable calendar status.
func plainCalIndicator(m Model) string {
	if !m.calSyncEnabled {
		return "Calendar off"
	}
	if m.override.Active {
		return "active: " + m.override.Label
	}
	if m.lastPollErr != nil {
		return "Calendar error"
	}
	return "Calendar active"
}

func (m Model) renderCalIndicator() string {
	if !m.calSyncEnabled {
		return styleFaint.Render("Calendar sync: disabled")
	}
	if m.override.Active {
		source := "Meeting"
		if m.override.Source == "schedule" {
			source = "Schedule"
		}
		return styleSuccess.Render("[active] " + source + ": " + m.override.Label)
	}
	if m.lastPollErr != nil {
		return styleError.Render("[error] Calendar: " + m.lastPollErr.Error())
	}
	info := ""
	if !m.lastPoll.IsZero() {
		info = " (last poll " + m.lastPoll.Local().Format("15:04") + ")"
	}
	if m.polling {
		return styleAccent.Render(m.spinner.View() + " Calendar: syncing ...")
	}
	return styleAccent.Render("Calendar sync: active" + info)
}

func (m Model) renderDashboard() string {
	width := m.panelWidth()
	pageSize := m.dashboardPageSize()
	inner := width - 4 // card padding

	start := m.windowStart(pageSize)
	end := min(start+pageSize, len(m.templates))
	total := len(m.templates)
	pages := 0
	page := 0
	if total > 0 {
		pages = (total + pageSize - 1) / pageSize
		page = start/pageSize + 1
	}

	// Compact mode drops the description column on narrow terminals.
	compact := inner < 54

	var b strings.Builder
	title := "Templates"
	if total > 0 {
		title += fmt.Sprintf("   %d-%d of %d", start+1, end, total)
		if pages > 1 {
			title += fmt.Sprintf("   Page %d/%d", page, pages)
		}
	}
	b.WriteString(styleLabel.Render(truncate(title, inner)))
	b.WriteString("\n")

	if total == 0 {
		b.WriteString(styleSubtle.Render(truncate("  No templates - press c to add one.", inner)))
		b.WriteString("\n")
	} else {
		for i := start; i < end; i++ {
			b.WriteString(m.renderTemplateRow(m.templates[i], i == m.tplCursor, inner, compact))
			b.WriteString("\n")
		}
	}

	if next, ok := m.nextScheduleSummary(); ok {
		b.WriteString(styleFaint.Render(truncate("Next schedule: "+next, inner)))
	}
	return styleCard.Width(width - 4).Render(b.String())
}

func (m Model) renderTemplateRow(t domain.Template, selected bool, inner int, compact bool) string {
	glyph := emojiDisplay(t.Emoji)
	marker := "  "
	if selected {
		marker = "> "
	}
	glyphCell := lipgloss.NewStyle().Width(3).Render(glyph)

	if compact {
		labelWidth := inner - len(marker) - 3 - 1
		line := marker + glyphCell + " " + truncate(t.Label, labelWidth)
		if selected {
			return styleRowSelected.Render(line)
		}
		return styleValue.Render(line)
	}

	labelWidth := inner/3 + 4
	if labelWidth < 12 {
		labelWidth = 12
	}
	if labelWidth > 28 {
		labelWidth = 28
	}
	label := lipgloss.NewStyle().Width(labelWidth).Render(truncate(t.Label, labelWidth))

	desc := t.Text
	if extra := templateExpiry(t); extra != "" {
		desc += "  " + extra
	}
	descWidth := inner - len(marker) - 3 - 2 - labelWidth
	desc = truncate(desc, descWidth)
	line := marker + glyphCell + " " + label + "  " + desc
	if selected {
		return styleRowSelected.Render(line)
	}
	return styleValue.Render(line)
}

// nextScheduleSummary returns a one-line description of the schedule that is
// currently active or starts next, if any.
func (m Model) nextScheduleSummary() (string, bool) {
	now := time.Now()
	var best *domain.Schedule
	var bestStart time.Time
	for i := range m.schedules {
		s := m.schedules[i]
		start, ok := s.NextStart(now)
		if !ok {
			continue
		}
		if s.Active(now) {
			start = now
		}
		if best == nil || start.Before(bestStart) {
			c := s
			best = &c
			bestStart = start
		}
	}
	if best == nil {
		return "", false
	}
	prefix := ""
	if best.Active(now) {
		prefix = "running: "
	}
	return prefix + best.Text + "  " + scheduleSummary(*best), true
}

// clampLines limits the rendered view to the terminal height so the header and
// status card never scroll off screen.
func clampLines(s string, max int) string {
	if max <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= max {
		return s
	}
	return strings.Join(lines[:max], "\n")
}

func templateExpiry(t domain.Template) string {
	switch {
	case t.UseDurationSelector:
		return "(choose duration)"
	case t.DurationInMinutes != nil:
		return fmt.Sprintf("(%d min)", *t.DurationInMinutes)
	case t.UntilTime != "":
		return "(until " + t.UntilTime + ")"
	}
	return ""
}

func scheduleSummary(s domain.Schedule) string {
	if s.Recurrence != nil {
		days := "daily"
		if len(s.Recurrence.Weekdays) > 0 {
			var labels []string
			for _, wd := range s.Recurrence.Weekdays {
				labels = append(labels, weekdayName(wd))
			}
			days = strings.Join(labels, ",")
		}
		return days + " " + s.Recurrence.StartTime + "–" + s.Recurrence.EndTime
	}
	return s.Start.Local().Format("02.01. 15:04") + "–" + s.End.Local().Format("15:04")
}

func weekdayName(wd time.Weekday) string {
	names := map[time.Weekday]string{
		time.Monday: "Mon", time.Tuesday: "Tue", time.Wednesday: "Wed",
		time.Thursday: "Thu", time.Friday: "Fri", time.Saturday: "Sat", time.Sunday: "Sun",
	}
	return names[wd]
}

func (m Model) renderCalStatus() string {
	var b strings.Builder
	b.WriteString(styleLabel.Render("Calendar Sync"))
	b.WriteString("\n\n")
	if !m.calSyncEnabled {
		b.WriteString(styleWarning.Render("Disabled. Enable the sync in Settings (s)."))
		b.WriteString("\n")
	} else {
		b.WriteString("Status: " + styleSuccess.Render("active") + "\n")
		if !m.lastPoll.IsZero() {
			b.WriteString("Last poll: " + m.lastPoll.Local().Format("15:04:05") + "\n")
		}
		if m.lastPollErr != nil {
			b.WriteString(styleError.Render("Last error: "+m.lastPollErr.Error()) + "\n")
		}
		if m.override.Active {
			b.WriteString(styleSuccess.Render("Active override: "+m.override.Label) + "\n")
			if !m.override.Expires.IsZero() {
				b.WriteString("  Ends: " + m.override.Expires.Local().Format("15:04") + "\n")
			}
		} else {
			b.WriteString("No active override\n")
		}
		if m.snapshot.HasBase {
			b.WriteString(fmt.Sprintf("Saved status: %q\n", m.snapshot.BaseStatus.Text))
		}
	}
	b.WriteString("\n")
	b.WriteString(styleSubtle.Render("Esc back"))
	return b.String()
}

func (m Model) renderConfirm() string {
	return styleWarning.Render(m.confirmPrompt) + "\n\n" +
		styleSubtle.Render("y = confirm · any other key = cancel")
}

func (m Model) renderHelp() string {
	rows := [][2]string{
		{"Enter", "Apply template"},
		{"n", "Set / update status (composer)"},
		{"e", "Edit current status"},
		{"c", "Create a new template"},
		{"x / Del", "Delete selected template"},
		{"p", "Schedule a status (one-off or weekly)"},
		{"↑/↓ · PgUp/PgDn", "Move within / page through templates"},
		{"Home / End", "First / last template"},
		{"C", "Calendar view"},
		{"v", "Calendar sync status"},
		{"s", "Settings"},
		{"r", "Refresh everything"},
		{"? / Esc", "Close help"},
		{"q / Ctrl+C", "Quit"},
	}
	var b strings.Builder
	b.WriteString(styleLabel.Render("Help"))
	b.WriteString("\n\n")
	for _, row := range rows {
		b.WriteString("  " + styleSelected.Render(row[0]))
		b.WriteString("  " + styleValue.Render(row[1]))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(styleSubtle.Render("In the composer: Tab cycles fields · Enter chooses emoji/duration · Esc cancels"))
	return b.String()
}

func (m Model) renderFooter() string {
	var lines []string
	if m.err != nil {
		lines = append(lines, styleError.Render("Error: "+m.err.Error()))
	} else if m.message != "" {
		lines = append(lines, styleSuccess.Render(m.message))
	}
	lines = append(lines, styleSubtle.Render(m.keyHints()))
	body := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Padding(0, 1).Width(m.panelWidth()).Render(body)
}

func (m Model) keyHints() string {
	switch m.screen {
	case screenComposer:
		return "Enter to save · Tab field · Esc to cancel"
	case screenTemplateEditor, screenScheduleEditor, screenSettings:
		return "Enter to save · Tab field · Space toggles · Esc to cancel"
	case screenEmoji:
		return "type to filter · Enter to select · Esc to cancel"
	case screenDuration:
		return "Enter to select · Esc to cancel"
	case screenCalendar:
		return "←/→ day · ↑/↓ week · n/p month · Enter schedule · Esc back"
	case screenCalStatus:
		return "Esc back"
	case screenHelp:
		return "Esc or ? to close"
	case screenConfirm:
		return "y to confirm · any other key to cancel"
	default:
		return "Enter apply · n Status · c Template · p Schedule · C Calendar · ? Help"
	}
}

func fallback(v, fb string) string {
	if strings.TrimSpace(v) == "" {
		return fb
	}
	return v
}

func humanizeDuration(d time.Duration) string {
	mins := int(d.Minutes())
	if mins < 60 {
		return fmt.Sprintf("%d min", mins)
	}
	hours := mins / 60
	rest := mins % 60
	if rest == 0 {
		return fmt.Sprintf("%d h", hours)
	}
	return fmt.Sprintf("%d h %d min", hours, rest)
}
