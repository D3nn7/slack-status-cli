package main

import (
	"errors"
	"fmt"
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

func buildDurationValueInput(unit durationUnit) []textinput.Model {
	fields := []string{fmt.Sprintf("Anzahl %s", durationUnitLabel(unit))}
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

func parsePositiveInt(v string) (int, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, errors.New("value required")
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	if i <= 0 {
		return 0, errors.New("value must be greater than 0")
	}
	return i, nil
}

func durationUnitLabel(unit durationUnit) string {
	switch unit {
	case durationDays:
		return "Tage"
	case durationHours:
		return "Stunden"
	case durationMinutes:
		return "Minuten"
	default:
		return "Minuten"
	}
}
