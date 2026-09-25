package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/D3nn7/slack-status-cli/internal/config"
	"github.com/D3nn7/slack-status-cli/internal/domain"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	dir := t.TempDir()
	m := New(Options{
		Paths: config.Paths{
			Config:    dir + "/config.json",
			Templates: dir + "/templates.json",
			Schedules: dir + "/schedules.json",
			State:     dir + "/state.json",
		},
		Config: config.Defaults(),
	})
	m.width = 80
	m.height = 20
	return m
}

func TestDashboardFitsHeight(t *testing.T) {
	m := newTestModel(t)
	for i := 0; i < 30; i++ {
		m.templates = append(m.templates, domain.Template{
			ID:    domain.NewID(),
			Label: "Template with a long name",
			Text:  "Status text",
			Emoji: ":house:",
		})
	}
	m.schedules = append(m.schedules, domain.Schedule{
		ID:      domain.NewID(),
		Label:   "Focus",
		Text:    "Focus time",
		Emoji:   ":dart:",
		Enabled: true,
		Recurrence: &domain.Recurrence{
			StartTime: "09:00",
			EndTime:   "17:00",
		},
	})
	out := m.View()
	lines := strings.Split(out, "\n")
	if len(lines) > m.height {
		t.Fatalf("view has %d lines, exceeds height %d", len(lines), m.height)
	}
	if !strings.Contains(out, "Slack Status") {
		t.Error("expected the header to remain visible")
	}
	if !strings.Contains(out, "Templates") {
		t.Error("expected the Templates title to be visible")
	}
	if !strings.Contains(out, "1-") {
		t.Error("expected a scroll range indicator")
	}
	t.Log("\n" + out)
}

func TestDashboardResponsive(t *testing.T) {
	sizes := [][2]int{{30, 15}, {50, 20}, {80, 24}, {120, 40}, {200, 50}}
	for _, size := range sizes {
		w, h := size[0], size[1]
		m := newTestModel(t)
		m.width, m.height = w, h
		for i := 0; i < 30; i++ {
			m.templates = append(m.templates, domain.Template{
				ID:    domain.NewID(),
				Label: "Template with a very long name " + string(rune('A'+i%26)),
				Text:  "A fairly long status text",
				Emoji: ":house_with_garden:",
			})
		}
		out := m.View()
		lines := strings.Split(out, "\n")
		if len(lines) > h {
			t.Errorf("%dx%d: %d lines exceed height", w, h, len(lines))
		}
		for i, line := range lines {
			if lipgloss.Width(line) > w {
				t.Errorf("%dx%d: line %d width %d exceeds terminal width: %q", w, h, i, lipgloss.Width(line), line)
				break
			}
		}
	}
}

func TestScreensResponsive(t *testing.T) {
	sizes := [][2]int{{40, 16}, {60, 20}, {100, 30}}
	for _, size := range sizes {
		w, h := size[0], size[1]
		m := newTestModel(t)
		m.width, m.height = w, h
		m.templates = append(m.templates, domain.Template{ID: "t1", Label: "Office", Text: "At the office", Emoji: ":office:", UntilTime: "16:30"})
		m.schedules = append(m.schedules, domain.Schedule{ID: "s1", Text: "Focus time", Emoji: ":dart:", Enabled: true,
			Recurrence: &domain.Recurrence{Weekdays: []time.Weekday{time.Monday, time.Friday}, StartTime: "09:00", EndTime: "17:00"}})

		check := func(name string) {
			out := m.View()
			if lines := strings.Split(out, "\n"); len(lines) > h {
				t.Errorf("%dx%d %s: %d lines exceed height", w, h, name, len(lines))
			}
			for _, line := range strings.Split(out, "\n") {
				if lipgloss.Width(line) > w {
					t.Errorf("%dx%d %s: line width %d exceeds %d", w, h, name, lipgloss.Width(line), w)
					break
				}
			}
		}

		m.screen = screenHelp
		check("help")
		m.screen = screenCalendar
		check("calendar")
		m.screen = screenCalStatus
		check("calstatus")
		m.screen = screenConfirm
		check("confirm")

		m.composer = newComposer(composerNew, domain.Status{}, time.Now())
		m.composer.setWidth(m.inputWidth())
		m.screen = screenComposer
		check("composer")

		m.templateEditor = newTemplateEditor()
		m.templateEditor.setWidth(m.inputWidth())
		m.screen = screenTemplateEditor
		check("template-editor")

		m.scheduleEditor = newScheduleEditor(time.Now())
		m.scheduleEditor.setWidth(m.inputWidth())
		m.screen = screenScheduleEditor
		check("schedule-editor")

		m.settings = newSettingsEditor(config.Defaults())
		m.settings.setWidth(m.inputWidth())
		m.screen = screenSettings
		check("settings")
	}
}

func TestTemplatePagination(t *testing.T) {
	m := newTestModel(t)
	m.width, m.height = 80, 50
	for i := 0; i < 30; i++ {
		m.templates = append(m.templates, domain.Template{ID: domain.NewID(), Label: "T", Text: "x", Emoji: ":x:"})
	}
	if got := m.dashboardPageSize(); got != templatePageSize {
		t.Fatalf("page size = %d, want %d", got, templatePageSize)
	}
	out := m.View()
	if !strings.Contains(out, "of 30") || !strings.Contains(out, "Page 1/") {
		t.Errorf("expected a page indicator, got:\n%s", out)
	}
	// Page down moves by a full page.
	next, _ := m.handleDashboard(tea.KeyMsg{Type: tea.KeyPgDown})
	m = next.(Model)
	if m.tplCursor != templatePageSize {
		t.Errorf("cursor after page down = %d, want %d", m.tplCursor, templatePageSize)
	}
}

func TestEmojiDisplayPlaceholder(t *testing.T) {
	if got := emojiDisplay(":coffee:"); got != "☕" {
		t.Errorf("known shortcode = %q, want glyph", got)
	}
	if got := emojiDisplay(":custom_sepp:"); got != "?" {
		t.Errorf("custom shortcode = %q, want placeholder", got)
	}
	if got := emojiDisplay("🏢"); got != "🏢" {
		t.Errorf("raw glyph = %q, want passthrough", got)
	}
	if got := emojiDisplay(""); got != "[ ]" {
		t.Errorf("empty = %q, want [ ]", got)
	}
}

func TestDashboardEmptyState(t *testing.T) {
	m := newTestModel(t)
	out := m.View()
	if !strings.Contains(out, "No templates") {
		t.Error("expected the empty state hint")
	}
}

func TestPickerFilterAndNavigation(t *testing.T) {
	items := []pickerItem{
		{Title: "coffee", Value: ":coffee:"},
		{Title: "calendar", Value: ":calendar:"},
		{Title: "dart", Value: ":dart:"},
	}
	p := newPicker("test", true, items)
	p.query = "cal"
	p.applyFilter()
	if len(p.filtered) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(p.filtered))
	}
	got, ok := p.selected()
	if !ok || got.Title != "calendar" {
		t.Fatalf("unexpected selection: %+v", got)
	}
}

func TestPickerNavigationWraps(t *testing.T) {
	p := newPicker("test", false, []pickerItem{{Title: "a"}, {Title: "b"}})
	p.move(-1)
	if p.cursor != 1 {
		t.Errorf("expected wrap to last item, got %d", p.cursor)
	}
	p.move(1)
	if p.cursor != 0 {
		t.Errorf("expected wrap to first item, got %d", p.cursor)
	}
}

func TestComposerBuildStatus(t *testing.T) {
	now := time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC)
	c := newComposer(composerNew, domain.Status{}, now)
	c.text.SetValue("Focus")
	c.emoji = ":dart:"
	c.clearMode = domain.Clear1Hour

	status, err := c.buildStatus(now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Text != "Focus" || status.Emoji != ":dart:" {
		t.Errorf("unexpected status: %+v", status)
	}
	if want := now.Add(time.Hour); !status.Expiration.Equal(want) {
		t.Errorf("expiration got %v, want %v", status.Expiration, want)
	}
}

func TestComposerValidation(t *testing.T) {
	now := time.Now()
	c := newComposer(composerNew, domain.Status{}, now)
	c.emoji = ":x:"
	if _, err := c.buildStatus(now); err == nil {
		t.Error("expected error for missing text")
	}
	c.text.SetValue("hi")
	c.emoji = ""
	if _, err := c.buildStatus(now); err == nil {
		t.Error("expected error for missing emoji")
	}
}

func TestMinutesUntilNextMonday(t *testing.T) {
	// Friday 2026-01-16 12:00 -> Monday 2026-01-19 00:00 = 60 hours.
	friday := time.Date(2026, time.January, 16, 12, 0, 0, 0, time.UTC)
	got := minutesUntilNextMonday(friday)
	if got != 60*60 {
		t.Errorf("got %d minutes, want %d", got, 60*60)
	}
}
