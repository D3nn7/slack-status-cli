package main

import "github.com/slack-go/slack"

type viewState int

const (
	viewDashboard viewState = iota
	viewManual
	viewEditCurrent
	viewCreateTemplate
	viewDeleteConfirm
	viewSettings
)

const (
	appName       = "Slack Status TUI"
	configName    = "config.json"
	templatesName = "templates.json"
)

type template struct {
	Label             string `json:"label"`
	Text              string `json:"text"`
	Emoji             string `json:"emoji"`
	DurationInMinutes *int   `json:"durationInMinutes,omitempty"`
	UntilTime         string `json:"untilTime,omitempty"`
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
