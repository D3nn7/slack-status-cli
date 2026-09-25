package tui

import (
	"strconv"
	"strings"
	"time"

	"github.com/D3nn7/slack-status-cli/internal/domain"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type composerMode int

const (
	composerNew composerMode = iota
	composerEdit
)

// Field focus indices for the composer.
const (
	focusEmoji = iota
	focusText
	focusClear
	focusCustom
)

type composer struct {
	mode       composerMode
	emoji      string
	text       textinput.Model
	clearMode  domain.ClearMode
	customMins textinput.Model
	focus      int
}

func newComposer(mode composerMode, existing domain.Status, now time.Time) composer {
	ti := textinput.New()
	ti.Placeholder = "What's happening?"
	ti.CharLimit = 100
	ti.SetValue(existing.Text)
	ti.Focus()

	cm := textinput.New()
	cm.Placeholder = "e.g. 45"
	cm.CharLimit = 5

	c := composer{
		mode:       mode,
		emoji:      existing.Emoji,
		text:       ti,
		clearMode:  domain.ClearNever,
		customMins: cm,
		focus:      focusText,
	}
	if mode == composerEdit && !existing.Expiration.IsZero() {
		c.clearMode = domain.ClearCustom
		remaining := existing.Expiration.Sub(now).Round(time.Minute)
		if remaining > 0 {
			c.customMins.SetValue(strconv.Itoa(int(remaining.Minutes())))
		}
	}
	return c
}

func (c composer) fieldCount() int {
	if c.clearMode == domain.ClearCustom {
		return 4
	}
	return 3
}

func (c *composer) cycleFocus(delta int) {
	n := c.fieldCount()
	c.focus = (c.focus + delta + n) % n
	c.syncFocus()
}

func (c *composer) syncFocus() {
	if c.focus == focusText {
		c.text.Focus()
	} else {
		c.text.Blur()
	}
	if c.focus == focusCustom {
		c.customMins.Focus()
	} else {
		c.customMins.Blur()
	}
}

func (c *composer) setWidth(w int) {
	c.text.Width = w
	c.customMins.Width = 8
}

func (c *composer) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	c.text, cmd = c.text.Update(msg)
	cmds = append(cmds, cmd)
	c.customMins, cmd = c.customMins.Update(msg)
	cmds = append(cmds, cmd)
	return tea.Batch(cmds...)
}

// buildStatus validates the fields and materialises the status.
func (c composer) buildStatus(now time.Time) (domain.Status, error) {
	text := strings.TrimSpace(c.text.Value())
	if text == "" {
		return domain.Status{}, errNoText
	}
	if strings.TrimSpace(c.emoji) == "" {
		return domain.Status{}, errNoEmoji
	}
	var custom time.Duration
	if c.clearMode == domain.ClearCustom {
		mins, err := strconv.Atoi(strings.TrimSpace(c.customMins.Value()))
		if err != nil || mins <= 0 {
			return domain.Status{}, err0("duration must be a positive number of minutes")
		}
		custom = time.Duration(mins) * time.Minute
	}
	return domain.Status{
		Text:       text,
		Emoji:      strings.TrimSpace(c.emoji),
		Expiration: c.clearMode.Expiration(now, custom),
	}, nil
}

// view renders the Slack-like composer.
func (c composer) view(width int) string {
	var b strings.Builder

	title := "Update your status"
	if c.mode == composerEdit {
		title = "Edit status"
	}
	b.WriteString(styleLabel.Render(title))
	b.WriteString("\n\n")

	// Emoji field.
	glyph := emojiDisplay(c.emoji)
	emojiRow := lipgloss.JoinHorizontal(lipgloss.Top,
		fieldMarker(c.focus == focusEmoji),
		lipgloss.NewStyle().Width(3).Render(glyph),
		fieldValue(c.focus == focusEmoji, "Choose emoji (Enter)"),
	)
	b.WriteString(emojiRow)
	b.WriteString("\n")

	// Text field.
	b.WriteString(renderInputRow("Status", c.text.View(), c.focus == focusText))
	b.WriteString("\n")

	// Clear-after field.
	clear := domain.ClearModeLabel(c.clearMode)
	b.WriteString(renderSelectRow("Clear after", clear, c.focus == focusClear))
	b.WriteString("\n")

	// Custom duration field.
	if c.clearMode == domain.ClearCustom {
		b.WriteString(renderInputRow("Minutes", c.customMins.View(), c.focus == focusCustom))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleSubtle.Render("Enter to save · Tab to switch · Esc to cancel"))
	return b.String()
}

func renderInputRow(label, value string, focused bool) string {
	marker := fieldMarker(focused)
	labelStyle := styleSubtle
	if focused {
		labelStyle = styleAccent
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top,
		marker,
		labelStyle.Width(12).Render(label),
		value,
	)
	return row
}

func renderSelectRow(label, value string, focused bool) string {
	marker := fieldMarker(focused)
	labelStyle := styleSubtle
	if focused {
		labelStyle = styleAccent
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		marker,
		labelStyle.Width(12).Render(label),
		fieldValue(focused, value+"  ▾"),
	)
}

func fieldMarker(focused bool) string {
	if focused {
		return styleAccent.Render("› ")
	}
	return "  "
}

func fieldValue(focused bool, value string) string {
	if focused {
		return styleSelected.Render(value)
	}
	return styleValue.Render(value)
}
