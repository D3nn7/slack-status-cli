package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/D3nn7/slack-status-cli/internal/domain"
	"github.com/charmbracelet/lipgloss"
)

type calendarModel struct {
	month  time.Time
	cursor time.Time
}

func newCalendarModel() calendarModel {
	now := time.Now()
	return calendarModel{
		month:  time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
		cursor: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()),
	}
}

func (c *calendarModel) moveDay(delta int) {
	c.cursor = c.cursor.AddDate(0, 0, delta)
	if c.cursor.Month() != c.month.Month() || c.cursor.Year() != c.month.Year() {
		c.month = time.Date(c.cursor.Year(), c.cursor.Month(), 1, 0, 0, 0, 0, c.cursor.Location())
	}
}

func (c *calendarModel) moveMonth(delta int) {
	c.month = c.month.AddDate(0, delta, 0)
	last := c.month.AddDate(0, 1, -1)
	day := c.cursor.Day()
	if day > last.Day() {
		day = last.Day()
	}
	c.cursor = time.Date(c.month.Year(), c.month.Month(), day, 0, 0, 0, 0, c.month.Location())
}

func (c *calendarModel) moveWeek(delta int) { c.moveDay(delta * 7) }

func (c calendarModel) view(m Model) string {
	var b strings.Builder
	header := c.month.Format("January 2006")
	b.WriteString(styleLabel.Render(header))
	b.WriteString("\n\n")

	b.WriteString(styleSubtle.Render("Mon Tue Wed Thu Fri Sat Sun"))
	b.WriteString("\n")

	firstOffset := (int(c.month.Weekday()) - int(time.Monday) + 7) % 7
	for i := 0; i < firstOffset; i++ {
		b.WriteString("    ")
	}
	daysInMonth := c.month.AddDate(0, 1, -1).Day()
	for day := 1; day <= daysInMonth; day++ {
		date := time.Date(c.month.Year(), c.month.Month(), day, 0, 0, 0, 0, c.month.Location())
		b.WriteString(c.renderDay(date, m))
		weekdayIdx := (firstOffset + day - 1) % 7
		if weekdayIdx == 6 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n\n")

	b.WriteString(c.renderAgenda(m))
	return b.String()
}

func (c calendarModel) renderDay(date time.Time, m Model) string {
	label := fmt.Sprintf("%2d", date.Day())
	isCursor := sameDay(date, c.cursor)
	hasEvent := len(eventsOnDay(m.events, date)) > 0
	hasSchedule := len(schedulesOnDay(m.schedules, date)) > 0

	marker := " "
	switch {
	case hasEvent && hasSchedule:
		marker = "*"
	case hasEvent:
		marker = "*"
	case hasSchedule:
		marker = "+"
	}

	text := label + marker
	switch {
	case isCursor:
		return styleSelected.Render(text) + " "
	case hasEvent || hasSchedule:
		return styleAccent.Render(text) + " "
	default:
		return lipgloss.NewStyle().Foreground(colGrey).Render(text) + " "
	}
}

func (c calendarModel) renderAgenda(m Model) string {
	day := c.cursor
	label := day.Format("Monday, 02.01.2006")
	var b strings.Builder
	b.WriteString(styleLabel.Render(label))
	b.WriteString("\n")

	wrote := false
	for _, ev := range eventsOnDay(m.events, day) {
		when := "all day"
		if !ev.AllDay {
			when = ev.Start.Local().Format("15:04") + "–" + ev.End.Local().Format("15:04")
		}
		tag := ""
		if ev.Meeting {
			tag = styleWarning.Render("  [Meeting]")
		}
		b.WriteString(fmt.Sprintf("  %s  %s%s\n", styleSubtle.Render(when), ev.Subject, tag))
		wrote = true
	}
	for _, s := range schedulesOnDay(m.schedules, day) {
		when := ""
		if s.Recurrence != nil {
			when = s.Recurrence.StartTime + "–" + s.Recurrence.EndTime + " (weekly)"
		} else {
			when = s.Start.Local().Format("15:04") + "–" + s.End.Local().Format("15:04")
		}
		b.WriteString(fmt.Sprintf("  %s  %s  %s\n", styleSubtle.Render(when), s.Text, styleSuccess.Render("[schedule]")))
		wrote = true
	}
	if !wrote {
		b.WriteString(styleSubtle.Render("  no entries"))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(styleSubtle.Render("←/→ day · ↑/↓ week · n/p month · Enter schedule · Esc back"))
	return b.String()
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func eventsOnDay(events []domain.Event, day time.Time) []domain.Event {
	var out []domain.Event
	for _, ev := range events {
		if sameDay(ev.Start.Local(), day) {
			out = append(out, ev)
		}
	}
	return out
}

func schedulesOnDay(schedules []domain.Schedule, day time.Time) []domain.Schedule {
	var out []domain.Schedule
	for _, s := range schedules {
		if !s.Enabled {
			continue
		}
		if s.Recurrence != nil {
			if len(s.Recurrence.Weekdays) == 0 || containsWeekday(s.Recurrence.Weekdays, day.Weekday()) {
				out = append(out, s)
			}
			continue
		}
		if sameDay(s.Start.Local(), day) {
			out = append(out, s)
		}
	}
	return out
}

func containsWeekday(days []time.Weekday, wd time.Weekday) bool {
	for _, d := range days {
		if d == wd {
			return true
		}
	}
	return false
}
