package tui

import (
	"strings"
	"time"

	"github.com/D3nn7/slack-status-cli/internal/domain"
	"github.com/D3nn7/slack-status-cli/internal/emoji"
)

// newEmojiPicker builds the searchable emoji picker.
func newEmojiPicker(height int) picker {
	entries := emoji.Search("", 0)
	items := make([]pickerItem, 0, len(entries))
	for _, e := range entries {
		items = append(items, pickerItem{
			Title:    e.Label(),
			Desc:     "",
			Keywords: strings.Join(e.Keywords, " "),
			Value:    e.Shortcode(),
		})
	}
	p := newPicker("Emoji", true, items)
	p.height = height
	return p
}

// newDurationPicker builds the "clear after" selector.
func newDurationPicker() picker {
	var items []pickerItem
	for _, mode := range domain.AllClearModes() {
		items = append(items, pickerItem{
			Title: domain.ClearModeLabel(mode),
			Value: mode,
		})
	}
	p := newPicker("Clear after", false, items)
	return p
}

// newTemplateDurationPicker asks for an ad-hoc duration when a template opts
// into the duration selector.
func newTemplateDurationPicker() picker {
	items := []pickerItem{
		{Title: "30 minutes", Value: 30},
		{Title: "1 hour", Value: 60},
		{Title: "2 hours", Value: 120},
		{Title: "4 hours", Value: 240},
		{Title: "8 hours", Value: 480},
		{Title: "Until next Monday", Value: -1},
	}
	return newPicker("Duration", false, items)
}

func templateDurationMinutes(item pickerItem) int {
	v, _ := item.Value.(int)
	if v == -1 {
		return minutesUntilNextMonday(time.Now())
	}
	if v <= 0 {
		return 30
	}
	return v
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
