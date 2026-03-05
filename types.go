package main

import (
	"time"

	"github.com/slack-go/slack"
)

type viewState int

const (
	viewDashboard viewState = iota
	viewManual
	viewEditCurrent
	viewCreateTemplate
	viewDeleteConfirm
	viewSettings
	viewDurationSelector
	viewDurationValue
	viewCalSyncStatus
)

const (
	appName       = "Slack Status TUI"
	configName    = "config.json"
	templatesName = "templates.json"
)

type template struct {
	Label               string `json:"label"`
	Text                string `json:"text"`
	Emoji               string `json:"emoji"`
	DurationInMinutes   *int   `json:"durationInMinutes,omitempty"`
	UntilTime           string `json:"untilTime,omitempty"`
	UseDurationSelector bool   `json:"useDurationSelector,omitempty"`
}

type templatePayload struct {
	Templates []template `json:"templates"`
}

type config struct {
	SlackToken    string `json:"slackToken"`
	ConfirmDelete *bool  `json:"confirmDelete,omitempty"`
}

type statusInfo struct {
	User       string
	Text       string
	Emoji      string
	Expiration string
}

type durationUnit int

const (
	durationDays durationUnit = iota
	durationHours
	durationMinutes
	durationNextMonday
)

type statusMsg statusInfo
type templatesMsg []template
type setStatusMsg string
type savedTemplatesMsg []template
type errMsg struct{ err error }

type configUpdatedMsg struct {
	cfg    config
	client *slack.Client
	msg    string
	path   string
}

// Calendar sync config (calendar-sync.json)
type calSyncConfig struct {
	Enabled                bool   `json:"enabled"`
	ICSUrl                 string `json:"icsUrl"`
	DefaultEmoji           string `json:"defaultEmoji"`
	DefaultText            string `json:"defaultText"`
	UseEventTitle          bool   `json:"useEventTitle"`
	PollingIntervalSeconds int    `json:"pollingIntervalSeconds"`
	StatePath              string `json:"statePath"`
	Debug                  bool   `json:"debug"`
	DebugLogPath           string `json:"debugLogPath"`
}

type calEvent struct {
	ID        string
	Subject   string
	StartTime time.Time
	EndTime   time.Time
	IsAllDay  bool
}

type savedStatus struct {
	Text              string `json:"text"`
	Emoji             string `json:"emoji"`
	ExpirationUnix    int64  `json:"expirationUnix"`
	SavedAt           int64  `json:"savedAt"`
	ActiveEventID     string `json:"activeEventId,omitempty"`
	ActiveEventEndUTC string `json:"activeEventEndUtc,omitempty"`
}

type calSyncState struct {
	ActiveEventID   string
	ActiveEventEnd  time.Time
	LastPollAt      time.Time
	LastPollErr     error
	StatusSaved     bool
	StatusSavedText string
	pendingEvent    *calEvent
}

// Calendar sync tea.Msg types
type calSyncTickMsg struct{}
type calEventsMsg struct {
	Events    []calEvent
	FetchedAt time.Time
}
type calStatusSetMsg struct {
	EventID     string
	EventEnd    time.Time
	StatusText  string
	StatusEmoji string
}
type calStatusRestoredMsg struct{ PreviousText string }
type calStatusSavedMsg struct{ Snapshot savedStatus }
type calSyncErrMsg struct {
	Err     error
	IsFatal bool
}
