package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

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
	if m.calSyncEnabled && m.client != nil {
		cmds = append(cmds, pollCalendarCmd(m.calSyncCfg))
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		w, h := panelSize(msg.Width, msg.Height)
		m.templateList.SetSize(w, h)
		m.durationList.SetSize(w, h)
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

	// ── Calendar sync messages ──────────────────────────────────────────
	case calSyncTickMsg:
		if m.calSyncEnabled {
			return m, pollCalendarCmd(m.calSyncCfg)
		}
		return m, nil

	case calEventsMsg:
		m.calSync.LastPollAt = msg.FetchedAt
		m.calSync.LastPollErr = nil
		return m.handleCalEvents(filterActiveEvents(msg.Events, msg.FetchedAt), msg.FetchedAt)

	case calStatusSavedMsg:
		m.calSync.StatusSaved = true
		m.calSync.StatusSavedText = msg.Snapshot.Text
		if m.calSync.pendingEvent != nil {
			ev := *m.calSync.pendingEvent
			m.calSync.pendingEvent = nil
			return m, setMeetingStatusCmd(m.client, m.calSyncCfg, ev)
		}
		return m, nil

	case calStatusSetMsg:
		m.calSync.ActiveEventID = msg.EventID
		m.calSync.ActiveEventEnd = msg.EventEnd
		interval := time.Duration(m.calSyncCfg.PollingIntervalSeconds) * time.Second
		return m, tea.Batch(
			fetchStatusCmd(m.client),
			startCalSyncTickCmd(interval),
		)

	case calStatusRestoredMsg:
		m.calSync.ActiveEventID = ""
		m.calSync.ActiveEventEnd = time.Time{}
		m.calSync.StatusSaved = false
		m.calSync.StatusSavedText = ""
		m.calSync.pendingEvent = nil
		interval := time.Duration(m.calSyncCfg.PollingIntervalSeconds) * time.Second
		return m, tea.Batch(
			fetchStatusCmd(m.client),
			startCalSyncTickCmd(interval),
		)

	case calSyncErrMsg:
		m.calSync.LastPollErr = msg.Err
		if msg.IsFatal {
			m.calSyncEnabled = false
			return m, nil
		}
		interval := time.Duration(m.calSyncCfg.PollingIntervalSeconds) * time.Second
		return m, startCalSyncTickCmd(interval)

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
		if item.UseDurationSelector {
			return m.enterDurationSelector(*item), nil, true
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
	case "C":
		if m.calSyncEnabled {
			m.state = viewCalSyncStatus
			return m, nil, true
		}
		return m, nil, false
	case "?":
		m.message = "Keys: enter use template \a a manual \a e edit current \a c create template \a x delete \a s settings \a C cal-sync \a r refresh \a q quit"
		return m, nil, true
	}
	return m, nil, false
}

func (m model) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state == viewDurationSelector {
		return m.handleDurationSelectorKey(msg)
	}
	if m.state == viewDurationValue {
		return m.handleDurationValueKey(msg)
	}
	if m.state == viewCalSyncStatus {
		if msg.String() == "esc" {
			return m.backToDashboard(), nil
		}
		return m, nil
	}
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

func (m model) handleDurationSelectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.cancelDurationSelector(), nil
	case "enter":
		item, ok := m.durationList.SelectedItem().(durationOption)
		if !ok {
			return m, nil
		}
		if item.Unit == durationNextMonday {
			minutes := minutesUntilNextMonday(time.Now())
			return m.applyDurationMinutes(minutes)
		}
		m.state = viewDurationValue
		m.durationUnit = item.Unit
		m.message = "Dauer eingeben"
		m.inputs = buildDurationValueInput(item.Unit)
		m.focusIndex = 0
		return m, nil
	}
	var cmd tea.Cmd
	m.durationList, cmd = m.durationList.Update(msg)
	return m, cmd
}

func (m model) handleDurationValueKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.backToDurationSelector(), nil
	case "enter":
		return m.submitDurationValue()
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

func (m model) submitDurationValue() (tea.Model, tea.Cmd) {
	if len(m.inputs) == 0 {
		return m.withError(errors.New("duration value missing")), nil
	}
	value, err := parsePositiveInt(m.inputs[0].Value())
	if err != nil {
		return m.withError(fmt.Errorf("duration: %w", err)), nil
	}
	minutes := value
	switch m.durationUnit {
	case durationDays:
		minutes = value * 24 * 60
	case durationHours:
		minutes = value * 60
	case durationMinutes:
		minutes = value
	}
	return m.applyDurationMinutes(minutes)
}

func (m model) applyDurationMinutes(minutes int) (tea.Model, tea.Cmd) {
	if m.pendingTemplate == nil {
		return m.withError(errors.New("no template selected")), nil
	}
	if minutes <= 0 {
		return m.withError(errors.New("duration must be greater than 0")), nil
	}
	t := *m.pendingTemplate
	duration := minutes
	m.state = viewDashboard
	m.inputs = nil
	m.focusIndex = 0
	m.pendingTemplate = nil
	return m, setStatusCmd(m.client, t.Text, t.Emoji, &duration, "")
}

func minutesUntilNextMonday(now time.Time) int {
	daysUntil := (int(time.Monday) - int(now.Weekday()) + 7) % 7
	if daysUntil == 0 {
		daysUntil = 7
	}
	target := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	target = target.AddDate(0, 0, daysUntil)
	minutes := int(target.Sub(now).Minutes())
	if minutes < 1 {
		return 1
	}
	return minutes
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

// handleCalEvents is the calendar sync state machine. It is called after every poll.
func (m model) handleCalEvents(nowEvents []calEvent, now time.Time) (tea.Model, tea.Cmd) {
	interval := time.Duration(m.calSyncCfg.PollingIntervalSeconds) * time.Second

	if m.calSync.ActiveEventID == "" {
		// CASE A: Kein aktives Meeting verfolgt.
		if len(nowEvents) == 0 {
			logCal("State A: kein laufendes Meeting → nächster Poll in %s", interval)
			return m, startCalSyncTickCmd(interval)
		}
		ev := earliestStartEvent(nowEvents)
		logCal("State A→Meeting: frühestes Event %q (%s) → Status sichern", ev.Subject, ev.StartTime.Local().Format("15:04"))
		m.calSync.pendingEvent = &ev
		return m, saveCurrentStatusCmd(m.client, m.calSyncCfg.StatePath)
	}

	// CASE B: Wir verfolgen gerade ein aktives Meeting.
	for _, ev := range nowEvents {
		if ev.ID == m.calSync.ActiveEventID {
			logCal("State B1: Meeting %q läuft noch → warten", m.calSync.ActiveEventID[:min(8, len(m.calSync.ActiveEventID))])
			return m, startCalSyncTickCmd(interval)
		}
	}

	logCal("State B2: Meeting %q beendet → Status wiederherstellen", m.calSync.ActiveEventID[:min(8, len(m.calSync.ActiveEventID))])
	return m, restorePreviousStatusCmd(m.client, m.calSyncCfg.StatePath)
}
