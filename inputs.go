package main

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
)

func buildStatusInputs(text, emoji string) []textinput.Model {
	fields := []string{"Status text", "Emoji (:coffee:)", "Duration (minutes, optional)", "Until time (HH:MM, optional)"}
	values := []string{text, emoji, "", ""}
	inputs := make([]textinput.Model, len(fields))
	for i := range inputs {
		ti := textinput.New()
		ti.Placeholder = fields[i]
		ti.CharLimit = 128
		ti.SetValue(values[i])
		if i == 0 {
			ti.Focus()
		}
		inputs[i] = ti
	}
	return inputs
}

func buildTemplateInputs() []textinput.Model {
	fields := []string{"Template name", "Status text", "Emoji (:house:)", "Duration (minutes, optional)", "Until time (HH:MM, optional)"}
	inputs := make([]textinput.Model, len(fields))
	for i := range inputs {
		ti := textinput.New()
		ti.Placeholder = fields[i]
		ti.CharLimit = 128
		if i == 0 {
			ti.Focus()
		}
		inputs[i] = ti
	}
	return inputs
}

func buildSettingsInputs(token string) []textinput.Model {
	inputs := make([]textinput.Model, 1)
	ti := textinput.New()
	ti.Placeholder = "Slack token"
	ti.CharLimit = 128
	ti.SetValue(token)
	ti.Focus()
	inputs[0] = ti
	return inputs
}

func parseOptionalInt(v string) (*int, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return nil, err
	}
	return &i, nil
}
