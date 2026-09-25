package tui

import (
	"strings"
	"time"

	"github.com/D3nn7/slack-status-cli/internal/domain"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Schedule editor focus indices.
const (
	schFocusLabel = iota
	schFocusText
	schFocusEmoji
	schFocusType
	schFocusWhen
	schFocusStart
	schFocusEnd
	schFieldCount
)

var weekdayLabels = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

type scheduleEditor struct {
	editing     bool
	editID      string
	label       textinput.Model
	text        textinput.Model
	emoji       textinput.Model
	once        bool
	date        textinput.Model
	start       textinput.Model
	end         textinput.Model
	weekdays    [7]bool
	focus       int
	defaultDate string
}

func newScheduleEditor(defaultDate time.Time) scheduleEditor {
	e := scheduleEditor{
		once:        true,
		defaultDate: defaultDate.Format("2006-01-02"),
	}
	e.label = newField("e.g. Focus time", 40)
	e.text = newField("e.g. Focus time", 80)
	e.emoji = newField(":dart:", 40)
	e.date = newField("2026-01-31", 10)
	e.date.SetValue(e.defaultDate)
	e.start = newField("09:00", 5)
	e.end = newField("10:00", 5)
	e.start.SetValue("09:00")
	e.end.SetValue("10:00")
	for i := range e.weekdays {
		e.weekdays[i] = i < 5 // Mon-Fri preselected
	}
	e.label.Focus()
	return e
}

func (e *scheduleEditor) load(s domain.Schedule) {
	e.editing = true
	e.editID = s.ID
	e.label.SetValue(s.Label)
	e.text.SetValue(s.Text)
	e.emoji.SetValue(s.Emoji)
	if s.Recurrence != nil {
		e.once = false
		e.start.SetValue(s.Recurrence.StartTime)
		e.end.SetValue(s.Recurrence.EndTime)
		for i := range e.weekdays {
			e.weekdays[i] = false
		}
		if len(s.Recurrence.Weekdays) == 0 {
			for i := range e.weekdays {
				e.weekdays[i] = true
			}
		} else {
			for _, wd := range s.Recurrence.Weekdays {
				if idx := weekdayIndex(wd); idx >= 0 {
					e.weekdays[idx] = true
				}
			}
		}
	} else {
		e.once = true
		e.date.SetValue(s.Start.Format("2006-01-02"))
		e.start.SetValue(s.Start.Format("15:04"))
		e.end.SetValue(s.End.Format("15:04"))
	}
	e.focus = schFocusLabel
	e.syncFocus()
}

func weekdayIndex(wd time.Weekday) int {
	for i := 0; i < 7; i++ {
		if time.Weekday((i+1)%7) == wd {
			return i
		}
	}
	return -1
}

func (e *scheduleEditor) cycleFocus(delta int) {
	e.focus = (e.focus + delta + schFieldCount) % schFieldCount
	e.syncFocus()
}

func (e *scheduleEditor) syncFocus() {
	fields := []*textinput.Model{&e.label, &e.text, &e.emoji, &e.date, &e.start, &e.end}
	for i, f := range fields {
		if i == e.focus {
			f.Focus()
		} else {
			f.Blur()
		}
	}
}

func (e *scheduleEditor) setWidth(w int) {
	e.label.Width = w
	e.text.Width = w
	e.emoji.Width = w
	e.date.Width = 12
	e.start.Width = 8
	e.end.Width = 8
}

func (e *scheduleEditor) updateInputs(msg tea.Msg) tea.Cmd {
	fields := []*textinput.Model{&e.label, &e.text, &e.emoji, &e.date, &e.start, &e.end}
	var cmds []tea.Cmd
	for _, f := range fields {
		var cmd tea.Cmd
		*f, cmd = f.Update(msg)
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// toggleWeekday toggles a weekday by its display position (0=Mon).
func (e *scheduleEditor) toggleWeekday(idx int) {
	if idx >= 0 && idx < 7 {
		e.weekdays[idx] = !e.weekdays[idx]
	}
}

func (e scheduleEditor) build(now time.Time) (domain.Schedule, error) {
	label := strings.TrimSpace(e.label.Value())
	text := strings.TrimSpace(e.text.Value())
	em := strings.TrimSpace(e.emoji.Value())
	if label == "" || text == "" || em == "" {
		return domain.Schedule{}, err0("name, text and emoji are required")
	}
	if _, ok := domain.ParseClock(strings.TrimSpace(e.start.Value())); !ok {
		return domain.Schedule{}, err0("start time must be in HH:MM format")
	}
	if _, ok := domain.ParseClock(strings.TrimSpace(e.end.Value())); !ok {
		return domain.Schedule{}, err0("end time must be in HH:MM format")
	}

	id := e.editID
	if id == "" {
		id = domain.NewID()
	}
	s := domain.Schedule{
		ID:      id,
		Label:   label,
		Text:    text,
		Emoji:   em,
		Enabled: true,
	}
	if e.once {
		day, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(e.date.Value()), now.Location())
		if err != nil {
			return domain.Schedule{}, err0("date must be in YYYY-MM-DD format")
		}
		startClock, _ := domain.ParseClock(strings.TrimSpace(e.start.Value()))
		endClock, _ := domain.ParseClock(strings.TrimSpace(e.end.Value()))
		s.Start = time.Date(day.Year(), day.Month(), day.Day(), startClock.Hour(), startClock.Minute(), 0, 0, now.Location())
		s.End = time.Date(day.Year(), day.Month(), day.Day(), endClock.Hour(), endClock.Minute(), 0, 0, now.Location())
		if !s.End.After(s.Start) {
			return domain.Schedule{}, err0("end time must be after start time")
		}
	} else {
		var days []time.Weekday
		for i, on := range e.weekdays {
			if on {
				days = append(days, time.Weekday((i+1)%7))
			}
		}
		s.Recurrence = &domain.Recurrence{
			Weekdays:  days,
			StartTime: strings.TrimSpace(e.start.Value()),
			EndTime:   strings.TrimSpace(e.end.Value()),
		}
	}
	return s, nil
}

func (e scheduleEditor) view() string {
	var b strings.Builder
	title := "New schedule"
	if e.editing {
		title = "Edit schedule"
	}
	b.WriteString(styleLabel.Render(title))
	b.WriteString("\n\n")

	b.WriteString(renderInputRow("Name", e.label.View(), e.focus == schFocusLabel))
	b.WriteString("\n")
	b.WriteString(renderInputRow("Status text", e.text.View(), e.focus == schFocusText))
	b.WriteString("\n")

	glyph := emojiDisplay(e.emoji.Value())
	b.WriteString(renderInputRow("Emoji", glyph+"  "+e.emoji.View(), e.focus == schFocusEmoji))
	b.WriteString("\n")

	typ := "One-off"
	if !e.once {
		typ = "Weekly"
	}
	b.WriteString(renderSelectRow("Type", typ+"  (Space)", e.focus == schFocusType))
	b.WriteString("\n")

	if e.once {
		b.WriteString(renderInputRow("Date", e.date.View(), e.focus == schFocusWhen))
	} else {
		b.WriteString(renderWeekdayRow(e.weekdays, e.focus == schFocusWhen))
	}
	b.WriteString("\n")
	b.WriteString(renderInputRow("Start", e.start.View(), e.focus == schFocusStart))
	b.WriteString("\n")
	b.WriteString(renderInputRow("End", e.end.View(), e.focus == schFocusEnd))
	b.WriteString("\n\n")
	b.WriteString(styleSubtle.Render("Enter to save · Tab to switch · Esc to cancel"))
	return b.String()
}

func renderWeekdayRow(selected [7]bool, focused bool) string {
	var parts []string
	for i, label := range weekdayLabels {
		if selected[i] {
			parts = append(parts, styleSelected.Render(label))
		} else {
			parts = append(parts, lipgloss.NewStyle().Foreground(colGrey).Render(label))
		}
	}
	marker := fieldMarker(focused)
	hint := ""
	if focused {
		hint = styleSubtle.Render("  (1-7 to toggle)")
	}
	return marker + lipgloss.NewStyle().Foreground(colGrey).Width(12).Render("Weekdays") +
		lipgloss.JoinHorizontal(lipgloss.Top, parts...) + hint
}
