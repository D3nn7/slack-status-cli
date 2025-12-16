package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.client != nil {
		cmds = append(cmds, fetchStatusCmd(m.client))
	}
	if m.templatesPath != "" {
		cmds = append(cmds, loadTemplatesCmd(m.templatesPath))
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		w := msg.Width/2 - 4
		h := msg.Height - 10
		if w < 30 {
			w = 30
		}
		if h < 10 {
			h = 10
		}
		m.templateList.SetSize(w, h)
	case statusMsg:
		m.status = statusInfo(msg)
		m.message = "Status refreshed"
		m.err = nil
	case templatesMsg:
		m.templates = msg
		items := make([]list.Item, 0, len(msg))
		for _, t := range msg {
			tCopy := t
			items = append(items, templateItem(tCopy))
		}
		m.templateList.SetItems(items)
		if len(items) == 0 {
			m.templateList.Title = "Status Templates (empty - press c to add)"
		} else {
			m.templateList.Title = "Status Templates"
		}
		m.message = "Templates loaded"
		m.err = nil
	case savedTemplatesMsg:
		m.state = viewDashboard
		m.inputs = nil
		m.focusIndex = 0
		return m, tea.Batch(loadTemplatesCmd(m.templatesPath), messageCmd("Templates saved"))
	case setStatusMsg:
		m.message = string(msg)
		if m.client != nil {
			return m, fetchStatusCmd(m.client)
		}
		return m, nil
	case errMsg:
		m.err = msg.err
		m.message = ""
	case configUpdatedMsg:
		m.cfg = msg.cfg
		m.confirmDelete = effectiveConfirmDelete(msg.cfg)
		m.client = msg.client
		m.configPath = msg.path
		m.message = msg.msg
		m.err = nil
		if m.client != nil {
			return m, fetchStatusCmd(m.client)
		}
		return m, nil
	case tea.KeyMsg:
		if m.state == viewDashboard {
			var (
				cmd     tea.Cmd
				handled bool
			)
			m, cmd, handled = m.handleDashboardKey(msg)
			if handled {
				return m, cmd
			}
		} else {
			return m.handleFormKey(msg)
		}
	}

	if m.state == viewDashboard {
		var cmd tea.Cmd
		m.templateList, cmd = m.templateList.Update(msg)
		return m, cmd
	}

	if m.state == viewSettings {
		cmd := m.updateInputs(msg)
		return m, cmd
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m model) handleDashboardKey(msg tea.KeyMsg) (model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit, true
	case "r":
		return m, tea.Batch(fetchStatusCmd(m.client), loadTemplatesCmd(m.templatesPath), messageCmd("Refreshing.")), true
	case "enter":
		item := m.selectedTemplate()
		if item == nil {
			return m, nil, true
		}
		return m, setStatusCmd(m.client, item.Text, item.Emoji, item.DurationInMinutes, item.UntilTime), true
	case "a", "n":
		return m.enterManualForm(), nil, true
	case "e":
		return m.enterEditForm(), nil, true
	case "c":
		return m.enterCreateTemplateForm(), nil, true
	case "x", "delete", "backspace":
		if m.selectedTemplate() == nil {
			return m, nil, true
		}
		if !m.confirmDelete {
			t := m.selectedTemplate()
			if t != nil {
				return m, deleteTemplateCmd(m.templatesPath, m.templates, t.Label), true
			}
			return m, nil, true
		}
		m.state = viewDeleteConfirm
		m.message = "Delete selected template? (y/n)"
		return m, nil, true
	case "s":
		return m.enterSettings(), nil, true
	case "?":
		m.message = "Keys: enter use template \a a manual \a e edit current \a c create template \a x delete \a s settings \a r refresh \a q quit"
		return m, nil, true
	}
	return m, nil, false
}

func (m model) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.backToDashboard(), nil
	}

	if m.state == viewDeleteConfirm {
		if msg.String() == "y" {
			t := m.selectedTemplate()
			if t != nil {
				return m, deleteTemplateCmd(m.templatesPath, m.templates, t.Label)
			}
		}
		return m.backToDashboard(), nil
	}

	switch msg.String() {
	case "tab", "shift+tab", "ctrl+n", "ctrl+p":
		if msg.String() == "shift+tab" || msg.String() == "ctrl+p" {
			m.focusIndex--
		} else {
			m.focusIndex++
		}
		if m.focusIndex >= len(m.inputs) {
			m.focusIndex = 0
		}
		if m.focusIndex < 0 {
			m.focusIndex = len(m.inputs) - 1
		}
		for i := range m.inputs {
			if i == m.focusIndex {
				m.inputs[i].Focus()
			} else {
				m.inputs[i].Blur()
			}
		}
		return m, nil
	case "enter":
		if m.state == viewSettings {
			return m.submitSettingsForm()
		}
		return m.submitForm()
	case "t", " ":
		if m.state == viewSettings {
			m.confirmDelete = !m.confirmDelete
			return m, nil
		}
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	if len(m.inputs) == 0 {
		return nil
	}
	var cmds []tea.Cmd
	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m model) submitForm() (tea.Model, tea.Cmd) {
	switch m.state {
	case viewManual, viewEditCurrent:
		text := strings.TrimSpace(m.inputs[0].Value())
		emoji := strings.TrimSpace(m.inputs[1].Value())
		duration, err := parseOptionalInt(m.inputs[2].Value())
		if err != nil {
			return m.withError(fmt.Errorf("duration: %w", err)), nil
		}
		until := strings.TrimSpace(m.inputs[3].Value())
		if text == "" || emoji == "" {
			return m.withError(errors.New("text and emoji are required")), nil
		}
		m.state = viewDashboard
		return m, setStatusCmd(m.client, text, emoji, duration, until)
	case viewCreateTemplate:
		label := strings.TrimSpace(m.inputs[0].Value())
		text := strings.TrimSpace(m.inputs[1].Value())
		emoji := strings.TrimSpace(m.inputs[2].Value())
		duration, err := parseOptionalInt(m.inputs[3].Value())
		if err != nil {
			return m.withError(fmt.Errorf("duration: %w", err)), nil
		}
		until := strings.TrimSpace(m.inputs[4].Value())
		if label == "" || text == "" || emoji == "" {
			return m.withError(errors.New("label, text, and emoji are required")), nil
		}
		newTemplate := template{
			Label:             label,
			Text:              text,
			Emoji:             emoji,
			DurationInMinutes: duration,
			UntilTime:         until,
		}
		m.state = viewDashboard
		return m, saveTemplateCmd(m.templatesPath, m.templates, newTemplate)
	}
	return m, nil
}

func (m model) submitSettingsForm() (tea.Model, tea.Cmd) {
	if len(m.inputs) == 0 {
		return m.withError(errors.New("no inputs to submit")), nil
	}
	token := strings.TrimSpace(m.inputs[0].Value())
	if token == "" {
		return m.withError(errors.New("slack token is required")), nil
	}
	m.state = viewDashboard
	return m, saveConfigCmd(m.configPath, token, m.confirmDelete)
}
