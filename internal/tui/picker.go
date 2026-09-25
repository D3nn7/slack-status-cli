package tui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// pickerItem is a single row of a picker.
type pickerItem struct {
	Title    string
	Desc     string
	Keywords string // searchable but not rendered
	Value    any
}

// pickerKeyResult reports how a key was consumed by a picker.
type pickerKeyResult int

const (
	pickerIgnored pickerKeyResult = iota
	pickerMoved
	pickerConfirmed
	pickerCancelled
)

// picker is a reusable, optionally filterable list widget.
type picker struct {
	title    string
	items    []pickerItem
	filtered []int
	cursor   int
	query    string
	filter   bool
	height   int
	width    int
}

func newPicker(title string, filter bool, items []pickerItem) picker {
	p := picker{title: title, filter: filter, height: 10, width: 60}
	p.setItems(items)
	return p
}

func (p *picker) setItems(items []pickerItem) {
	p.items = items
	p.cursor = 0
	p.applyFilter()
}

func (p *picker) setSize(width, height int) {
	p.width = width
	p.height = height
}

func (p *picker) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(p.query))
	p.filtered = p.filtered[:0]
	for i, it := range p.items {
		haystack := strings.ToLower(it.Title + " " + it.Desc + " " + it.Keywords)
		if q == "" || strings.Contains(haystack, q) {
			p.filtered = append(p.filtered, i)
		}
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = max(0, len(p.filtered)-1)
	}
}

func (p *picker) selected() (pickerItem, bool) {
	if p.cursor < 0 || p.cursor >= len(p.filtered) {
		return pickerItem{}, false
	}
	return p.items[p.filtered[p.cursor]], true
}

func (p *picker) move(delta int) {
	if len(p.filtered) == 0 {
		return
	}
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = len(p.filtered) - 1
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = 0
	}
}

// handleKey processes a key event and reports the outcome.
func (p *picker) handleKey(msg tea.KeyMsg) pickerKeyResult {
	switch msg.String() {
	case "esc":
		return pickerCancelled
	case "enter":
		if len(p.filtered) == 0 {
			return pickerIgnored
		}
		return pickerConfirmed
	case "up", "k", "ctrl+p":
		p.move(-1)
		return pickerMoved
	case "down", "j", "ctrl+n":
		p.move(1)
		return pickerMoved
	case "backspace":
		if p.filter && p.query != "" {
			p.query = p.query[:len(p.query)-1]
			p.applyFilter()
			return pickerMoved
		}
		return pickerIgnored
	case "ctrl+u":
		if p.filter {
			p.query = ""
			p.applyFilter()
			return pickerMoved
		}
	}
	if p.filter && msg.Type == tea.KeyRunes {
		p.query += string(msg.Runes)
		p.applyFilter()
		return pickerMoved
	}
	return pickerIgnored
}

func (p picker) view() string {
	var b strings.Builder
	if p.title != "" {
		b.WriteString(styleLabel.Render(p.title))
		b.WriteString("\n")
	}
	if p.filter {
		query := p.query
		if query == "" {
			query = styleSubtle.Render("type to filter ...")
		}
		b.WriteString(styleCard.Padding(0, 1).Width(p.width - 6).Render(query))
		b.WriteString("\n")
	}

	if len(p.filtered) == 0 {
		b.WriteString(styleSubtle.Render("  no matches"))
		return b.String()
	}

	visible := p.height
	if visible < 1 {
		visible = 1
	}
	start := 0
	if p.cursor >= visible {
		start = p.cursor - visible + 1
	}
	end := min(start+visible, len(p.filtered))

	for i := start; i < end; i++ {
		item := p.items[p.filtered[i]]
		marker := "  "
		if i == p.cursor {
			marker = "> "
		}
		line := marker + item.Title
		if item.Desc != "" {
			line += "  " + item.Desc
		}
		if i == p.cursor {
			b.WriteString(styleRowSelected.Render(line))
		} else {
			b.WriteString(styleCursor.Render(line))
		}
		b.WriteString("\n")
	}
	if len(p.filtered) > visible {
		b.WriteString(styleSubtle.Render(lipgloss.NewStyle().Render(
			"  " + strconv.Itoa(p.cursor+1) + " of " + strconv.Itoa(len(p.filtered)))))
		b.WriteString("\n")
	}
	return b.String()
}
