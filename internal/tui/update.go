package tui

import (
	"time"

	"github.com/D3nn7/slack-status-cli/internal/domain"
	"github.com/D3nn7/slack-status-cli/internal/scheduler"
	"github.com/D3nn7/slack-status-cli/internal/slackapi"
	"github.com/D3nn7/slack-status-cli/internal/store"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type emojiTarget int

const (
	emojiForComposer emojiTarget = iota
	emojiForTemplate
	emojiForSchedule
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.emojiPicker.setSize(m.panelWidth()-6, m.contentHeight()-3)
		m.composer.setWidth(m.inputWidth())
		m.templateEditor.setWidth(m.inputWidth())
		m.scheduleEditor.setWidth(m.inputWidth())
		m.settings.setWidth(m.inputWidth())
		return m, nil

	case tickMsg:
		return m.handleTick()

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case userMsg:
		m.user = slackapi.User(msg)
		return m, nil

	case statusMsg:
		m.status = domain.Status(msg)
		m.err = nil
		m.loading = false
		m.message = "Status updated"
		return m, nil

	case templatesMsg:
		m.templates = msg
		if m.tplCursor >= len(m.templates) {
			m.tplCursor = max(0, len(m.templates)-1)
		}
		m.loading = false
		return m, nil

	case schedulesMsg:
		m.schedules = msg
		m.loading = false
		return m, nil

	case eventsMsg:
		m.events = msg.events
		m.lastPoll = msg.fetchedAt
		m.lastPollErr = nil
		m.polling = false
		return m.evaluate()

	case snapshotMsg:
		m.snapshot = store.Snapshot(msg)
		if m.snapshot.HasBase && m.snapshot.RefID != "" {
			m.override = scheduler.Override{Active: true, RefID: m.snapshot.RefID, Source: scheduler.Source(m.snapshot.Source)}
		}
		return m, nil

	case savedMsg:
		m.message = msg.note
		m.err = nil
		return m, nil

	case statusAppliedMsg:
		m.status = msg.status
		m.message = msg.note
		m.err = nil
		return m, nil

	case settingsSavedMsg:
		m.cfg = msg.cfg
		m.client = msg.client
		m.calSyncEnabled = m.cfg.Calendar.Enabled && m.cfg.Calendar.ICSUrl != ""
		m.message = "Settings saved"
		m.err = nil
		return m, tea.Batch(fetchUserCmd(m.client), fetchStatusCmd(m.client))

	case errMsg:
		m.err = msg.err
		m.loading = false
		m.polling = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m.routeInputs(msg)
}

func (m Model) handleTick() (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{scheduleTickCmd(schedulerTickPeriod)}
	if m.calSyncEnabled && m.client != nil {
		interval := time.Duration(m.cfg.Calendar.PollingIntervalSeconds) * time.Second
		if m.lastPoll.IsZero() || time.Since(m.lastPoll) >= interval {
			m.polling = true
			cmds = append(cmds, fetchEventsCmd(m.cfg))
		}
	}
	evaluated, cmd := m.evaluate()
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return evaluated, tea.Batch(cmds...)
}

// evaluate computes the desired override and applies/restores the Slack status
// as needed. It only issues API calls when the desired state changes.
func (m Model) evaluate() (tea.Model, tea.Cmd) {
	ov := scheduler.Evaluate(scheduler.Inputs{
		Now:       time.Now(),
		Schedules: m.schedules,
		Events:    m.events,
		Config:    m.currentSchedulerConfig(),
	})

	if ov.Active {
		unchanged := m.override.Active && m.override.RefID == ov.RefID && m.override.Source == ov.Source
		if unchanged {
			return m, nil
		}
		var cmds []tea.Cmd
		if !m.snapshot.HasBase {
			m.snapshot = store.Snapshot{
				HasBase:    true,
				BaseStatus: m.status,
				Source:     string(ov.Source),
				RefID:      ov.RefID,
			}
		} else {
			m.snapshot.Source = string(ov.Source)
			m.snapshot.RefID = ov.RefID
		}
		m.override = ov
		cmds = append(cmds, saveSnapshotCmd(m.store, m.snapshot),
			setStatusCmd(m.client, ov.Status, overrideNote(ov)))
		return m, tea.Batch(cmds...)
	}

	if m.override.Active {
		base := m.snapshot.BaseStatus
		m.override = scheduler.Override{}
		m.snapshot = store.Snapshot{}
		return m, tea.Batch(
			clearSnapshotCmd(m.store),
			setStatusCmd(m.client, base, "Automatic status cleared"),
		)
	}
	return m, nil
}

func overrideNote(ov scheduler.Override) string {
	switch ov.Source {
	case scheduler.SourceMeeting:
		return "Meeting active: " + ov.Label
	case scheduler.SourceSchedule:
		return "Schedule active: " + ov.Label
	default:
		return "Status set automatically"
	}
}

// --- key routing ---

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	switch m.screen {
	case screenComposer:
		return m.handleComposer(msg)
	case screenEmoji:
		return m.handleEmoji(msg)
	case screenDuration:
		return m.handleDuration(msg)
	case screenTemplateEditor:
		return m.handleTemplateEditor(msg)
	case screenScheduleEditor:
		return m.handleScheduleEditor(msg)
	case screenCalendar:
		return m.handleCalendar(msg)
	case screenSettings:
		return m.handleSettings(msg)
	case screenCalStatus:
		if msg.String() == "esc" || msg.String() == "q" {
			return m.goBack(), nil
		}
		return m, nil
	case screenHelp:
		return m.goBack(), nil
	case screenConfirm:
		return m.handleConfirm(msg)
	default:
		return m.handleDashboard(msg)
	}
}

func (m Model) handleDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.prevScreen = screenDashboard
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "?":
		m.prevScreen = screenDashboard
		m.screen = screenHelp
		return m, nil
	case "up", "k":
		if m.tplCursor > 0 {
			m.tplCursor--
		}
		return m, nil
	case "down", "j":
		if m.tplCursor < len(m.templates)-1 {
			m.tplCursor++
		}
		return m, nil
	case "pgdown", "right":
		size := m.dashboardPageSize()
		m.tplCursor = min(m.tplCursor+size, max(0, len(m.templates)-1))
		return m, nil
	case "pgup", "left":
		size := m.dashboardPageSize()
		m.tplCursor = max(m.tplCursor-size, 0)
		return m, nil
	case "home":
		m.tplCursor = 0
		return m, nil
	case "end":
		m.tplCursor = max(0, len(m.templates)-1)
		return m, nil
	case "r":
		m.loading = true
		m.message = ""
		cmds := []tea.Cmd{m.spinner.Tick}
		if m.client != nil {
			cmds = append(cmds, fetchStatusCmd(m.client), fetchUserCmd(m.client))
		}
		cmds = append(cmds, loadTemplatesCmd(m.store), loadSchedulesCmd(m.store))
		if m.calSyncEnabled {
			cmds = append(cmds, fetchEventsCmd(m.cfg))
		}
		return m, tea.Batch(cmds...)
	case "enter":
		t, ok := m.selectedTemplate()
		if !ok {
			return m, nil
		}
		if t.UseDurationSelector {
			m.pendingTemplate = &t
			m.durationPicker = newTemplateDurationPicker()
			m.prevScreen = m.screen
			m.screen = screenDuration
			return m, nil
		}
		status := t.Status(time.Now(), nil)
		return m, setStatusCmd(m.client, status, "Status applied: "+t.Label)
	case "n":
		m.composer = newComposer(composerNew, domain.Status{}, time.Now())
		m.composer.setWidth(m.inputWidth())
		m.composer.text.Focus()
		m.screen = screenComposer
		return m, nil
	case "e":
		m.composer = newComposer(composerEdit, m.status, time.Now())
		m.composer.setWidth(m.inputWidth())
		m.composer.text.Focus()
		m.screen = screenComposer
		return m, nil
	case "c":
		m.templateEditor = newTemplateEditor()
		m.templateEditor.setWidth(m.inputWidth())
		m.screen = screenTemplateEditor
		return m, nil
	case "x", "delete":
		t, ok := m.selectedTemplate()
		if !ok {
			return m, nil
		}
		if m.cfg.ConfirmDeleteEnabled() {
			return m.confirm("Delete template \""+t.Label+"\"?", func(mm *Model) tea.Cmd {
				return mm.deleteTemplate(t.ID)
			}), nil
		}
		return m, m.deleteTemplate(t.ID)
	case "p":
		ed := newScheduleEditor(time.Now())
		ed.setWidth(m.inputWidth())
		m.scheduleEditor = ed
		m.screen = screenScheduleEditor
		return m, nil
	case "C":
		m.screen = screenCalendar
		return m, nil
	case "s":
		m.settings = newSettingsEditor(m.cfg)
		m.settings.setWidth(m.inputWidth())
		m.screen = screenSettings
		return m, nil
	case "v":
		m.screen = screenCalStatus
		return m, nil
	}
	return m, nil
}

func (m Model) handleComposer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.goBack(), nil
	case "tab":
		m.composer.cycleFocus(1)
		return m, nil
	case "shift+tab":
		m.composer.cycleFocus(-1)
		return m, nil
	case "enter":
		switch m.composer.focus {
		case focusEmoji:
			m.openEmojiPicker(emojiForComposer)
			return m, nil
		case focusClear:
			m.durationPicker = newDurationPicker()
			m.prevScreen = m.screen
			m.screen = screenDuration
			return m, nil
		default:
			status, err := m.composer.buildStatus(time.Now())
			if err != nil {
				m.err = err
				return m, nil
			}
			m.screen = screenDashboard
			return m, setStatusCmd(m.client, status, "Status applied")
		}
	}
	return m, m.composer.updateInputs(msg)
}

func (m Model) handleEmoji(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result := m.emojiPicker.handleKey(msg)
	switch result {
	case pickerCancelled:
		return m.goBack(), nil
	case pickerConfirmed:
		item, _ := m.emojiPicker.selected()
		shortcode, _ := item.Value.(string)
		return m.applyEmoji(shortcode), nil
	}
	if msg.String() == "enter" && len(m.emojiPicker.filtered) == 0 && m.emojiPicker.query != "" {
		return m.applyEmoji(":" + m.emojiPicker.query + ":"), nil
	}
	return m, nil
}

func (m Model) applyEmoji(shortcode string) Model {
	switch m.emojiTarget {
	case emojiForTemplate:
		m.templateEditor.emoji.SetValue(shortcode)
	case emojiForSchedule:
		m.scheduleEditor.emoji.SetValue(shortcode)
	default:
		m.composer.emoji = shortcode
	}
	return m.goBack()
}

func (m Model) handleDuration(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result := m.durationPicker.handleKey(msg)
	switch result {
	case pickerCancelled:
		m.pendingTemplate = nil
		return m.goBack(), nil
	case pickerConfirmed:
		item, _ := m.durationPicker.selected()
		if m.pendingTemplate != nil {
			minutes := templateDurationMinutes(item)
			t := *m.pendingTemplate
			m.pendingTemplate = nil
			m.screen = screenDashboard
			d := time.Duration(minutes) * time.Minute
			return m, setStatusCmd(m.client, t.Status(time.Now(), &d), "Status applied: "+t.Label)
		}
		mode, _ := item.Value.(domain.ClearMode)
		m.composer.clearMode = mode
		if mode == domain.ClearCustom {
			m.composer.focus = focusCustom
		}
		m.composer.syncFocus()
		return m.goBack(), nil
	}
	return m, nil
}

func (m Model) handleTemplateEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.goBack(), nil
	case "tab":
		m.templateEditor.cycleFocus(1)
		return m, nil
	case "shift+tab":
		m.templateEditor.cycleFocus(-1)
		return m, nil
	case " ":
		if m.templateEditor.focus == tplFocusSelector {
			m.templateEditor.useSelector = !m.templateEditor.useSelector
			return m, nil
		}
	case "enter":
		if m.templateEditor.focus == tplFocusEmoji {
			m.openEmojiPicker(emojiForTemplate)
			return m, nil
		}
		t, err := m.templateEditor.build()
		if err != nil {
			m.err = err
			return m, nil
		}
		return m.saveTemplate(t)
	}
	return m, m.templateEditor.updateInputs(msg)
}

func (m Model) handleScheduleEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.goBack(), nil
	case "tab":
		m.scheduleEditor.cycleFocus(1)
		return m, nil
	case "shift+tab":
		m.scheduleEditor.cycleFocus(-1)
		return m, nil
	case " ":
		if m.scheduleEditor.focus == schFocusType {
			m.scheduleEditor.once = !m.scheduleEditor.once
			return m, nil
		}
	case "1", "2", "3", "4", "5", "6", "7":
		if m.scheduleEditor.focus == schFocusWhen && !m.scheduleEditor.once {
			idx := int(msg.String()[0] - '1')
			m.scheduleEditor.toggleWeekday(idx)
			return m, nil
		}
	case "enter":
		if m.scheduleEditor.focus == schFocusEmoji {
			m.openEmojiPicker(emojiForSchedule)
			return m, nil
		}
		s, err := m.scheduleEditor.build(time.Now())
		if err != nil {
			m.err = err
			return m, nil
		}
		return m.saveSchedule(s)
	}
	return m, m.scheduleEditor.updateInputs(msg)
}

func (m Model) handleCalendar(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return m.goBack(), nil
	case "left", "h":
		m.calendar.moveDay(-1)
	case "right", "l":
		m.calendar.moveDay(1)
	case "up", "k":
		m.calendar.moveWeek(-1)
	case "down", "j":
		m.calendar.moveWeek(1)
	case "n":
		m.calendar.moveMonth(1)
	case "p":
		m.calendar.moveMonth(-1)
	case "enter":
		ed := newScheduleEditor(m.calendar.cursor)
		ed.setWidth(m.inputWidth())
		m.scheduleEditor = ed
		m.prevScreen = screenCalendar
		m.screen = screenScheduleEditor
	}
	return m, nil
}

func (m Model) handleSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.goBack(), nil
	case "tab":
		m.settings.cycleFocus(1)
		return m, nil
	case "shift+tab":
		m.settings.cycleFocus(-1)
		return m, nil
	case " ":
		m.settings.toggle()
		return m, nil
	case "enter":
		newCfg := m.settings.apply(m.cfg)
		m.screen = screenDashboard
		if newCfg.SlackToken == "" {
			m.err = err0("Slack token must not be empty")
			return m, nil
		}
		if newCfg.SlackToken != m.cfg.SlackToken || m.client == nil {
			return m, validateTokenCmd(m.paths.Config, newCfg, newCfg.SlackToken)
		}
		return m, saveSettingsCmd(m.paths.Config, newCfg, m.client)
	}
	return m, m.settings.updateInputs(msg)
}

func (m Model) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "y" || msg.String() == "enter" {
		action := m.confirmAction
		m.confirmAction = nil
		m.confirmPrompt = ""
		m.screen = screenDashboard
		var cmd tea.Cmd
		if action != nil {
			cmd = action(&m)
		}
		return m, cmd
	}
	m.confirmAction = nil
	m.confirmPrompt = ""
	return m.goBack(), nil
}

// --- helpers ---

func (m Model) routeInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenComposer:
		return m, m.composer.updateInputs(msg)
	case screenTemplateEditor:
		return m, m.templateEditor.updateInputs(msg)
	case screenScheduleEditor:
		return m, m.scheduleEditor.updateInputs(msg)
	case screenSettings:
		return m, m.settings.updateInputs(msg)
	}
	return m, nil
}

func (m Model) selectedTemplate() (domain.Template, bool) {
	if m.tplCursor < 0 || m.tplCursor >= len(m.templates) {
		return domain.Template{}, false
	}
	return m.templates[m.tplCursor], true
}

func (m Model) saveTemplate(t domain.Template) (tea.Model, tea.Cmd) {
	updated := make([]domain.Template, 0, len(m.templates)+1)
	replaced := false
	for _, existing := range m.templates {
		if existing.ID == t.ID {
			updated = append(updated, t)
			replaced = true
		} else {
			updated = append(updated, existing)
		}
	}
	if !replaced {
		updated = append(updated, t)
	}
	m.screen = screenDashboard
	return m, saveTemplatesCmd(m.store, updated)
}

func (m Model) deleteTemplate(id string) tea.Cmd {
	filtered := make([]domain.Template, 0, len(m.templates))
	for _, t := range m.templates {
		if t.ID != id {
			filtered = append(filtered, t)
		}
	}
	return saveTemplatesCmd(m.store, filtered)
}

func (m Model) saveSchedule(s domain.Schedule) (tea.Model, tea.Cmd) {
	updated := make([]domain.Schedule, 0, len(m.schedules)+1)
	replaced := false
	for _, existing := range m.schedules {
		if existing.ID == s.ID {
			updated = append(updated, s)
			replaced = true
		} else {
			updated = append(updated, existing)
		}
	}
	if !replaced {
		updated = append(updated, s)
	}
	m.screen = screenDashboard
	return m, saveSchedulesCmd(m.store, updated)
}

func (m Model) confirm(prompt string, action func(*Model) tea.Cmd) Model {
	m.prevScreen = m.screen
	m.screen = screenConfirm
	m.confirmPrompt = prompt
	m.confirmAction = action
	return m
}

func (m Model) goBack() Model {
	switch m.prevScreen {
	case screenConfirm, screenEmoji, screenDuration:
		m.screen = screenDashboard
	default:
		m.screen = m.prevScreen
	}
	m.prevScreen = screenDashboard
	m.confirmPrompt = ""
	m.confirmAction = nil
	return m
}

// openEmojiPicker switches to the emoji picker and remembers where to return.
func (m *Model) openEmojiPicker(target emojiTarget) {
	m.emojiTarget = target
	m.prevScreen = m.screen
	m.emojiPicker = newEmojiPicker(m.contentHeight() - 3)
	m.emojiPicker.setSize(m.panelWidth()-6, m.contentHeight()-3)
	m.screen = screenEmoji
}
