package tui

import (
	"strconv"
	"strings"

	"github.com/D3nn7/slack-status-cli/internal/domain"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Template editor focus indices.
const (
	tplFocusLabel = iota
	tplFocusText
	tplFocusEmoji
	tplFocusDuration
	tplFocusUntil
	tplFocusSelector
	tplFieldCount
)

type templateEditor struct {
	editing     bool
	editID      string
	label       textinput.Model
	text        textinput.Model
	emoji       textinput.Model
	duration    textinput.Model
	until       textinput.Model
	useSelector bool
	focus       int
}

func newTemplateEditor() templateEditor {
	e := templateEditor{}
	e.label = newField("e.g. At the office", 40)
	e.text = newField("e.g. At the office", 80)
	e.emoji = newField(":office:", 40)
	e.duration = newField("e.g. 60 (optional)", 10)
	e.until = newField("e.g. 16:30 (optional)", 10)
	e.label.Focus()
	return e
}

func (e *templateEditor) load(t domain.Template) {
	e.editing = true
	e.editID = t.ID
	e.label.SetValue(t.Label)
	e.text.SetValue(t.Text)
	e.emoji.SetValue(t.Emoji)
	if t.DurationInMinutes != nil {
		e.duration.SetValue(strconv.Itoa(*t.DurationInMinutes))
	} else {
		e.duration.SetValue("")
	}
	e.until.SetValue(t.UntilTime)
	e.useSelector = t.UseDurationSelector
	e.focus = tplFocusLabel
	e.syncFocus()
}

func (e *templateEditor) cycleFocus(delta int) {
	e.focus = (e.focus + delta + tplFieldCount) % tplFieldCount
	e.syncFocus()
}

func (e *templateEditor) syncFocus() {
	fields := []*textinput.Model{&e.label, &e.text, &e.emoji, &e.duration, &e.until}
	for i, f := range fields {
		if i == e.focus {
			f.Focus()
		} else {
			f.Blur()
		}
	}
}

func (e *templateEditor) setWidth(w int) {
	e.label.Width = w
	e.text.Width = w
	e.emoji.Width = w
	e.duration.Width = 10
	e.until.Width = 10
}

func (e *templateEditor) updateInputs(msg tea.Msg) tea.Cmd {
	fields := []*textinput.Model{&e.label, &e.text, &e.emoji, &e.duration, &e.until}
	var cmds []tea.Cmd
	for _, f := range fields {
		var cmd tea.Cmd
		*f, cmd = f.Update(msg)
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (e templateEditor) build() (domain.Template, error) {
	label := strings.TrimSpace(e.label.Value())
	text := strings.TrimSpace(e.text.Value())
	em := strings.TrimSpace(e.emoji.Value())
	if label == "" || text == "" || em == "" {
		return domain.Template{}, err0("name, text and emoji are required")
	}
	var duration *int
	if v := strings.TrimSpace(e.duration.Value()); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return domain.Template{}, err0("duration must be a positive number of minutes")
		}
		duration = &n
	}
	until := strings.TrimSpace(e.until.Value())
	if until != "" {
		if _, ok := domain.ParseClock(until); !ok {
			return domain.Template{}, err0("time must be in HH:MM format")
		}
	}
	id := e.editID
	if id == "" {
		id = domain.NewID()
	}
	return domain.Template{
		ID:                  id,
		Label:               label,
		Text:                text,
		Emoji:               em,
		DurationInMinutes:   duration,
		UntilTime:           until,
		UseDurationSelector: e.useSelector,
	}, nil
}

func (e templateEditor) view() string {
	var b strings.Builder
	title := "New template"
	if e.editing {
		title = "Edit template"
	}
	b.WriteString(styleLabel.Render(title))
	b.WriteString("\n\n")

	b.WriteString(renderInputRow("Name", e.label.View(), e.focus == tplFocusLabel))
	b.WriteString("\n")
	b.WriteString(renderInputRow("Status text", e.text.View(), e.focus == tplFocusText))
	b.WriteString("\n")

	glyph := emojiDisplay(e.emoji.Value())
	b.WriteString(renderInputRow("Emoji", glyph+"  "+e.emoji.View(), e.focus == tplFocusEmoji))
	b.WriteString("\n")
	b.WriteString(renderInputRow("Duration (m)", e.duration.View(), e.focus == tplFocusDuration))
	b.WriteString("\n")
	b.WriteString(renderInputRow("Until time", e.until.View(), e.focus == tplFocusUntil))
	b.WriteString("\n")

	sel := yesNo(e.useSelector)
	b.WriteString(renderSelectRow("Ask duration", sel+"  (Space)", e.focus == tplFocusSelector))
	b.WriteString("\n\n")
	b.WriteString(styleSubtle.Render("Enter to save · Tab to switch · Esc to cancel"))
	return b.String()
}

func newField(placeholder string, limit int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = limit
	return ti
}
