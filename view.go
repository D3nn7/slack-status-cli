package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.width == 0 {
		return "Loading layout..."
	}

	header := renderHeader()
	statusCard := renderStatusCard(m.status, m.err, m.calSync, m.calSyncEnabled)
	body := m.renderBody()
	footer := renderFooter(m.status.User)

	return lipgloss.JoinVertical(lipgloss.Left, header, statusCard, body, footer)
}

func (m model) renderBody() string {
	if m.state == viewCalSyncStatus {
		return renderCalSyncStatusView(m.calSync, m.calSyncEnabled)
	}

	if m.state == viewDashboard || m.state == viewDeleteConfirm {
		left := lipgloss.JoinVertical(lipgloss.Left, renderPanelTitle("Templates"), m.templateList.View())
		help := renderHelp(m.state == viewDeleteConfirm, m.message)
		right := lipgloss.JoinVertical(lipgloss.Left, renderPanelTitle("Actions"), help)
		return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	if m.state == viewDurationSelector {
		left := lipgloss.JoinVertical(lipgloss.Left, renderPanelTitle("Dauer"), m.durationList.View())
		help := renderDurationHelp(m.message)
		right := lipgloss.JoinVertical(lipgloss.Left, renderPanelTitle("Actions"), help)
		return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	if m.state == viewSettings {
		return lipgloss.JoinVertical(lipgloss.Left, renderSettingsView(m))
	}

	if m.state == viewDurationValue {
		return lipgloss.JoinVertical(lipgloss.Left, renderDurationValueForm(m))
	}

	form := renderForm(m.state, m.inputs)
	return lipgloss.JoinVertical(lipgloss.Left, form)
}

func renderHeader() string {
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#b4f9f8")).Bold(true).Padding(1, 2).Background(lipgloss.Color("#11131f")).Render(appName)
	sub := lipgloss.NewStyle().Foreground(lipgloss.Color("#7dc4e4")).Padding(1, 2).Render("Bubble Tea + Lip Gloss makeover for your Slack status")
	return lipgloss.JoinHorizontal(lipgloss.Top, title, sub)
}

func renderStatusCard(info statusInfo, err error, calSync calSyncState, calEnabled bool) string {
	indicator := renderCalSyncIndicator(calSync, calEnabled)
	base := fmt.Sprintf("User: %s\nStatus: %s %s\nExpires: %s\n%s",
		missing(info.User, "unknown"),
		missing(info.Text, "-"),
		info.Emoji,
		missing(info.Expiration, "none"),
		indicator,
	)
	if err != nil {
		base += "\n\n" + err.Error()
	}
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7dc4e4")).
		Padding(1, 2).
		Width(80).
		Render(base)
	return card
}

func renderPanelTitle(text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#c6a0f6")).Bold(true).Padding(0, 1).Render(text)
}

func renderHelp(confirm bool, message string) string {
	if confirm {
		return lipgloss.NewStyle().Padding(1, 2).Render("Press y to confirm deletion or any other key to cancel.")
	}
	hints := []string{
		"enter use template",
		"a manual status",
		"e edit current",
		"c create template",
		"s settings",
		"C cal-sync",
		"x delete template",
		"r refresh",
		"q quit",
	}
	helpText := lipgloss.NewStyle().Foreground(lipgloss.Color("#8aadf4")).Render(strings.Join(hints, " \a "))
	msg := ""
	if message != "" {
		msg = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#a6da95")).Render(message)
	}
	return lipgloss.NewStyle().Padding(1, 2).Width(40).Render(helpText + msg)
}

func renderDurationHelp(message string) string {
	hints := []string{
		"enter Auswahl",
		"esc abbrechen",
	}
	helpText := lipgloss.NewStyle().Foreground(lipgloss.Color("#8aadf4")).Render(strings.Join(hints, " \a "))
	msg := ""
	if message != "" {
		msg = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#a6da95")).Render(message)
	}
	return lipgloss.NewStyle().Padding(1, 2).Width(40).Render(helpText + msg)
}

func renderForm(state viewState, inputs []textinput.Model) string {
	title := "Manual Status"
	if state == viewEditCurrent {
		title = "Edit Current Status"
	} else if state == viewCreateTemplate {
		title = "Create Template"
	}
	var b strings.Builder
	b.WriteString(renderPanelTitle(title))
	b.WriteString("\n\n")
	for _, input := range inputs {
		b.WriteString(input.View())
		b.WriteString("\n\n")
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#8aadf4")).Render("Enter to submit \a Esc to cancel \a Tab to switch fields"))
	card := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#c6a0f6")).
		Padding(1, 2).
		Width(80).
		Render(b.String())
	return card
}

func renderDurationValueForm(m model) string {
	title := fmt.Sprintf("Dauer (%s)", durationUnitLabel(m.durationUnit))
	var b strings.Builder
	b.WriteString(renderPanelTitle(title))
	b.WriteString("\n\n")
	for _, input := range m.inputs {
		b.WriteString(input.View())
		b.WriteString("\n\n")
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#8aadf4")).Render("Enter to submit \a Esc to cancel"))
	card := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#c6a0f6")).
		Padding(1, 2).
		Width(80).
		Render(b.String())
	return card
}

func renderFooter(user string) string {
	footer := fmt.Sprintf("%s \a signed in as %s", appName, missing(user, "unknown"))
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cad3f5")).
		Background(lipgloss.Color("#24273a")).
		Padding(0, 2).
		Width(lipgloss.Width(footer) + 4).
		Render(footer)
}

func renderSettingsView(m model) string {
	tokenView := ""
	if len(m.inputs) > 0 {
		tokenView = m.inputs[0].View()
	}
	confirm := "no"
	if m.confirmDelete {
		confirm = "yes"
	}
	body := fmt.Sprintf(
		"Logged in as: %s\nConfirm deletions: %s (toggle with t)\n\nConfig path: %s\n\n%s\n\nEnter to save \a Esc to cancel",
		missing(m.status.User, "unknown"),
		confirm,
		m.configPath,
		tokenView,
	)
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#c6a0f6")).
		Padding(1, 2).
		Width(80).
		Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, renderPanelTitle("Settings"), card)
}

func missing(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

type templateItem template

func (t templateItem) Title() string {
	return t.Label
}

func (t templateItem) Description() string {
	var parts []string
	if t.Text != "" {
		parts = append(parts, t.Text)
	}
	if t.DurationInMinutes != nil {
		parts = append(parts, fmt.Sprintf("%dm", *t.DurationInMinutes))
	}
	if t.UntilTime != "" {
		parts = append(parts, fmt.Sprintf("until %s", t.UntilTime))
	}
	return strings.Join(parts, " \a ")
}

func (t templateItem) FilterValue() string {
	return t.Label + t.Text + t.Emoji
}

func newTemplateDelegate() list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(lipgloss.Color("#c6a0f6"))
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("#a6da95"))
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(lipgloss.Color("#8aadf4"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("#eed49f"))
	return delegate
}

type durationOption struct {
	Label string
	Unit  durationUnit
}

func (d durationOption) Title() string {
	return d.Label
}

func (d durationOption) Description() string {
	return ""
}

func (d durationOption) FilterValue() string {
	return d.Label
}

// renderCalSyncIndicator renders a one-line status indicator for the cal-sync feature.
func renderCalSyncIndicator(s calSyncState, enabled bool) string {
	if !enabled {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6e738d")).Render("Cal-Sync: deaktiviert")
	}
	if s.ActiveEventID != "" {
		until := ""
		if !s.ActiveEventEnd.IsZero() {
			until = " until " + s.ActiveEventEnd.Local().Format("15:04")
		}
		text := s.StatusSavedText
		if text == "" {
			text = "meeting"
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#a6da95")).Render("Cal-Sync: in meeting" + until + " (previous: " + text + ")")
	}
	if s.LastPollErr != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ed8796")).Render("Cal-Sync: error — " + s.LastPollErr.Error())
	}
	pollInfo := ""
	if !s.LastPollAt.IsZero() {
		pollInfo = " (last poll " + s.LastPollAt.Local().Format("15:04:05") + ")"
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#7dc4e4")).Render("Cal-Sync: idle" + pollInfo)
}

// renderCalSyncStatusView renders the calendar sync status detail panel.
func renderCalSyncStatusView(s calSyncState, enabled bool) string {
	var b strings.Builder
	b.WriteString(renderPanelTitle("Calendar Sync Status"))
	b.WriteString("\n\n")

	if !enabled {
		b.WriteString("Cal-Sync ist deaktiviert.\ncalendar-sync.json erstellen um es zu aktivieren.\n")
	} else {
		b.WriteString("Status: Active\n")
		if !s.LastPollAt.IsZero() {
			b.WriteString(fmt.Sprintf("Last poll: %s\n", s.LastPollAt.Local().Format("15:04:05")))
		}
		if s.LastPollErr != nil {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#ed8796")).Render("Last error: "+s.LastPollErr.Error()) + "\n")
		}
		if s.ActiveEventID != "" {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#a6da95")).Render("Active meeting: yes") + "\n")
			if !s.ActiveEventEnd.IsZero() {
				remaining := time.Until(s.ActiveEventEnd).Round(time.Minute)
				b.WriteString(fmt.Sprintf("  Ends at: %s (%s remaining)\n",
					s.ActiveEventEnd.Local().Format("15:04"),
					remaining.String()))
			}
		} else {
			b.WriteString("Active meeting: none\n")
		}
		if s.StatusSaved {
			b.WriteString(fmt.Sprintf("Saved status: %q\n", s.StatusSavedText))
		}
	}

	b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#8aadf4")).Render("Esc to go back"))
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7dc4e4")).
		Padding(1, 2).
		Width(80).
		Render(b.String())
	return card
}

func newDurationList(width, height int) list.Model {
	items := []list.Item{
		durationOption{Label: "Tage", Unit: durationDays},
		durationOption{Label: "Stunden", Unit: durationHours},
		durationOption{Label: "Minuten", Unit: durationMinutes},
		durationOption{Label: "Bis naechste Woche Montag", Unit: durationNextMonday},
	}
	delegate := newTemplateDelegate()
	ls := list.New(items, delegate, width, height)
	ls.Title = "Dauer"
	ls.SetShowStatusBar(false)
	ls.SetShowHelp(false)
	ls.DisableQuitKeybindings()
	ls.SetFilteringEnabled(false)
	return ls
}
