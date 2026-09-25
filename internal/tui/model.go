// Package tui contains the Bubble Tea user interface: a Slack-inspired status
// composer, template library, emoji picker, calendar view and scheduling.
package tui

import (
	"time"

	"github.com/D3nn7/slack-status-cli/internal/config"
	"github.com/D3nn7/slack-status-cli/internal/domain"
	"github.com/D3nn7/slack-status-cli/internal/scheduler"
	"github.com/D3nn7/slack-status-cli/internal/slackapi"
	"github.com/D3nn7/slack-status-cli/internal/store"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// screen enumerates the top-level views.
type screen int

const (
	screenDashboard screen = iota
	screenComposer
	screenEmoji
	screenDuration
	screenTemplateEditor
	screenScheduleEditor
	screenCalendar
	screenSettings
	screenCalStatus
	screenConfirm
	screenHelp
)

const (
	defaultWidth        = 80
	schedulerTickPeriod = 30 * time.Second
	templatePageSize    = 8
)

// Options configures a new Model.
type Options struct {
	Paths      config.Paths
	Config     config.Config
	Client     slackapi.Client
	ConfigErr  error
	TemplatesE error
}

// Model is the root Bubble Tea model.
type Model struct {
	paths  config.Paths
	cfg    config.Config
	client slackapi.Client
	store  *store.Store

	user      slackapi.User
	status    domain.Status
	templates []domain.Template
	schedules []domain.Schedule
	events    []domain.Event
	snapshot  store.Snapshot
	override  scheduler.Override

	screen     screen
	prevScreen screen

	width   int
	height  int
	message string
	err     error
	loading bool

	tplCursor int

	composer       composer
	emojiPicker    picker
	durationPicker picker
	templateEditor templateEditor
	scheduleEditor scheduleEditor
	settings       settingsEditor
	calendar       calendarModel

	emojiTarget     emojiTarget
	pendingTemplate *domain.Template
	pendingSchedule *domain.Schedule

	spinner spinner.Model

	confirmPrompt string
	confirmAction func(*Model) tea.Cmd

	calSyncEnabled bool
	lastPoll       time.Time
	lastPollErr    error
	polling        bool
}

// New builds the initial model.
func New(opts Options) Model {
	st := store.New(store.Paths{
		Templates: opts.Paths.Templates,
		Schedules: opts.Paths.Schedules,
		State:     opts.Paths.State,
	})
	m := Model{
		paths:          opts.Paths,
		cfg:            opts.Config,
		client:         opts.Client,
		store:          st,
		screen:         screenDashboard,
		width:          defaultWidth,
		height:         30,
		calSyncEnabled: opts.Config.Calendar.Enabled && opts.Config.Calendar.ICSUrl != "",
		message:        "Welcome! Press ? for help.",
		err:            opts.ConfigErr,
		loading:        opts.Client != nil,
		spinner:        spinner.New(spinner.WithSpinner(spinner.Line)),
	}
	m.calendar = newCalendarModel()
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		loadTemplatesCmd(m.store),
		loadSchedulesCmd(m.store),
		loadSnapshotCmd(m.store),
		scheduleTickCmd(schedulerTickPeriod),
	}
	if m.loading {
		cmds = append(cmds, m.spinner.Tick)
	}
	if m.client != nil {
		cmds = append(cmds, fetchUserCmd(m.client), fetchStatusCmd(m.client))
	}
	if m.calSyncEnabled && m.client != nil {
		cmds = append(cmds, fetchEventsCmd(m.cfg))
	}
	return tea.Batch(cmds...)
}

// --- messages ---

type userMsg slackapi.User
type statusMsg domain.Status
type templatesMsg []domain.Template
type schedulesMsg []domain.Schedule
type eventsMsg struct {
	events    []domain.Event
	fetchedAt time.Time
}
type snapshotMsg store.Snapshot
type tickMsg time.Time
type savedMsg struct{ note string }
type errMsg struct{ err error }
type statusAppliedMsg struct {
	status domain.Status
	note   string
}

// --- sizing helpers ---

// panelWidth returns the content width for the main panels, adapting to the
// terminal width (with a small margin) and clamped to a sane range.
func (m Model) panelWidth() int {
	w := m.width - 4
	if w < 20 {
		w = m.width - 2
	}
	if w < 10 {
		w = 10
	}
	if w > 120 {
		w = 120
	}
	return w
}

// contentHeight returns the available height for scrollable content.
func (m Model) contentHeight() int {
	h := m.height - 12
	if h < 3 {
		h = 3
	}
	return h
}

// compactLayout is true on small terminals, where chrome is reduced.
func (m Model) compactLayout() bool {
	return m.height < 20
}

// dashboardRowCount is the number of template rows that fit on the dashboard
// without pushing the header or footer off screen.
func (m Model) dashboardRowCount() int {
	reserve := 14
	if m.compactLayout() {
		reserve = 10
	}
	rows := m.height - reserve
	if rows < 1 {
		rows = 1
	}
	if rows > 30 {
		rows = 30
	}
	return rows
}

// inputWidth is the width for text inputs inside forms.
func (m Model) inputWidth() int {
	w := m.panelWidth() - 18
	if w < 6 {
		w = 6
	}
	if w > 80 {
		w = 80
	}
	return w
}

// dashboardPageSize is the number of templates per page: a fixed cap for
// readability, shrunk further when the terminal is short.
func (m Model) dashboardPageSize() int {
	size := templatePageSize
	if rows := m.dashboardRowCount(); rows < size {
		size = rows
	}
	if size < 1 {
		size = 1
	}
	return size
}

// windowStart returns the first visible template index for the page that
// contains the cursor.
func (m Model) windowStart(pageSize int) int {
	if pageSize < 1 {
		pageSize = 1
	}
	return (m.tplCursor / pageSize) * pageSize
}

// currentSchedulerConfig maps the app config into scheduler inputs.
func (m Model) currentSchedulerConfig() scheduler.Config {
	c := m.cfg.Calendar
	return scheduler.Config{
		UseEventTitle: c.UseEventTitle,
		MeetingEmoji:  c.DefaultEmoji,
		MeetingText:   c.DefaultText,
		MeetingOnly:   c.MeetingOnly,
		Keywords:      c.Keywords,
	}
}
