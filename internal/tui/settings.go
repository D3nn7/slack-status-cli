package tui

import (
	"strings"

	"github.com/D3nn7/slack-status-cli/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Settings focus indices.
const (
	setFocusToken = iota
	setFocusConfirm
	setFocusCalEnabled
	setFocusCalURL
	setFocusCalText
	setFocusCalEmoji
	setFocusUseTitle
	setFocusMeetingOnly
	setFieldCount
)

type settingsEditor struct {
	token        textinput.Model
	url          textinput.Model
	meetingText  textinput.Model
	meetingEmoji textinput.Model
	confirm      bool
	calEnabled   bool
	useTitle     bool
	meetingOnly  bool
	focus        int
}

func newSettingsEditor(cfg config.Config) settingsEditor {
	e := settingsEditor{
		confirm:     cfg.ConfirmDeleteEnabled(),
		calEnabled:  cfg.Calendar.Enabled,
		useTitle:    cfg.Calendar.UseEventTitle,
		meetingOnly: cfg.Calendar.MeetingOnly,
	}
	e.token = newField("xoxp-...", 200)
	e.token.SetValue(cfg.SlackToken)
	e.url = newField("https://outlook.office365.com/.../calendar.ics", 400)
	e.url.SetValue(cfg.Calendar.ICSUrl)
	e.meetingText = newField("In a meeting", 100)
	e.meetingText.SetValue(cfg.Calendar.DefaultText)
	e.meetingEmoji = newField(":calendar:", 40)
	e.meetingEmoji.SetValue(cfg.Calendar.DefaultEmoji)
	e.token.Focus()
	return e
}

func (e *settingsEditor) cycleFocus(delta int) {
	e.focus = (e.focus + delta + setFieldCount) % setFieldCount
	e.syncFocus()
}

func (e *settingsEditor) syncFocus() {
	fields := []*textinput.Model{&e.token, &e.url, &e.meetingText, &e.meetingEmoji}
	// Determine which textinput corresponds to the focused index.
	active := map[int]*textinput.Model{
		setFocusToken:    &e.token,
		setFocusCalURL:   &e.url,
		setFocusCalText:  &e.meetingText,
		setFocusCalEmoji: &e.meetingEmoji,
	}[e.focus]
	for _, f := range fields {
		if f == active {
			f.Focus()
		} else {
			f.Blur()
		}
	}
}

func (e *settingsEditor) toggle() {
	switch e.focus {
	case setFocusConfirm:
		e.confirm = !e.confirm
	case setFocusCalEnabled:
		e.calEnabled = !e.calEnabled
	case setFocusUseTitle:
		e.useTitle = !e.useTitle
	case setFocusMeetingOnly:
		e.meetingOnly = !e.meetingOnly
	}
}

func (e *settingsEditor) setWidth(w int) {
	e.token.Width = w
	e.url.Width = w
	e.meetingText.Width = w
	e.meetingEmoji.Width = 16
}

func (e *settingsEditor) updateInputs(msg tea.Msg) tea.Cmd {
	fields := []*textinput.Model{&e.token, &e.url, &e.meetingText, &e.meetingEmoji}
	var cmds []tea.Cmd
	for _, f := range fields {
		var cmd tea.Cmd
		*f, cmd = f.Update(msg)
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// apply returns a copy of cfg with the editor's values merged in.
func (e settingsEditor) apply(cfg config.Config) config.Config {
	cfg.SlackToken = strings.TrimSpace(e.token.Value())
	confirm := e.confirm
	cfg.ConfirmDelete = &confirm
	cfg.Calendar.Enabled = e.calEnabled
	cfg.Calendar.ICSUrl = strings.TrimSpace(e.url.Value())
	cfg.Calendar.DefaultText = strings.TrimSpace(e.meetingText.Value())
	cfg.Calendar.DefaultEmoji = strings.TrimSpace(e.meetingEmoji.Value())
	cfg.Calendar.UseEventTitle = e.useTitle
	cfg.Calendar.MeetingOnly = e.meetingOnly
	return cfg
}

func (e settingsEditor) view() string {
	var b strings.Builder
	b.WriteString(styleLabel.Render("Settings"))
	b.WriteString("\n\n")
	b.WriteString(renderInputRow("Slack token", e.token.View(), e.focus == setFocusToken))
	b.WriteString("\n\n")

	b.WriteString(renderSelectRow("Confirm deletes", yesNo(e.confirm), e.focus == setFocusConfirm))
	b.WriteString("\n")
	b.WriteString(renderSelectRow("Calendar sync", yesNo(e.calEnabled), e.focus == setFocusCalEnabled))
	b.WriteString("\n")
	b.WriteString(renderInputRow("ICS URL", e.url.View(), e.focus == setFocusCalURL))
	b.WriteString("\n")
	b.WriteString(renderInputRow("Meeting text", e.meetingText.View(), e.focus == setFocusCalText))
	b.WriteString("\n")
	b.WriteString(renderInputRow("Meeting emoji", e.meetingEmoji.View(), e.focus == setFocusCalEmoji))
	b.WriteString("\n")
	b.WriteString(renderSelectRow("Use event title", yesNo(e.useTitle), e.focus == setFocusUseTitle))
	b.WriteString("\n")
	b.WriteString(renderSelectRow("Meetings only", yesNo(e.meetingOnly), e.focus == setFocusMeetingOnly))
	b.WriteString("\n\n")
	b.WriteString(styleSubtle.Render("Enter to save · Tab to switch · Space to toggle · Esc to cancel"))
	return b.String()
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
