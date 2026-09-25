package tui

import (
	"strings"

	"github.com/D3nn7/slack-status-cli/internal/emoji"
	"github.com/mattn/go-runewidth"
)

// emojiDisplay renders a shortcode for display. Known shortcodes become their
// unicode glyph, raw glyphs pass through, and unknown custom shortcodes (which
// have no glyph) become a compact placeholder instead of the long name.
func emojiDisplay(shortcode string) string {
	v := strings.TrimSpace(shortcode)
	if v == "" {
		return "[ ]"
	}
	if strings.HasPrefix(v, ":") {
		if c := emoji.Char(v); c != v {
			return c
		}
		return "?"
	}
	return v
}

// truncate shortens s to at most width display columns, appending an ellipsis
// when content is cut. It is display-width aware so CJK/emoji align correctly.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= width {
		return s
	}
	out := ""
	w := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > width-1 {
			break
		}
		out += string(r)
		w += rw
	}
	return out + "…"
}
