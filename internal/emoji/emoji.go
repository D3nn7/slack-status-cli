// Package emoji provides a small, curated emoji catalogue with search for the
// Slack-style picker. Slack itself is addressed by shortcode (":coffee:"); the
// unicode glyph is used purely for on-screen rendering.
package emoji

import "strings"

// Emoji couples a Slack shortcode with its unicode glyph and search keywords.
type Emoji struct {
	Name     string // shortcode without colons, e.g. "coffee"
	Char     string // unicode glyph
	Keywords []string
}

// Shortcode returns the Slack form, e.g. ":coffee:".
func (e Emoji) Shortcode() string { return ":" + e.Name + ":" }

// Label returns the shortcode with its glyph for display.
func (e Emoji) Label() string {
	if e.Char == "" {
		return e.Shortcode()
	}
	return e.Char + "  " + e.Shortcode()
}

// Search returns the catalogue entries matching query. An empty query returns
// the first limit entries. Name and keywords are matched case-insensitively.
func Search(query string, limit int) []Emoji {
	q := strings.ToLower(strings.TrimSpace(query))
	var out []Emoji
	for _, e := range catalog {
		if q == "" || matches(e, q) {
			out = append(out, e)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out
}

func matches(e Emoji, q string) bool {
	if strings.Contains(e.Name, q) {
		return true
	}
	for _, kw := range e.Keywords {
		if strings.Contains(kw, q) {
			return true
		}
	}
	return false
}

// Char resolves a shortcode or a raw glyph to a display glyph. Unknown custom
// shortcodes are returned unchanged so they still show as ":name:".
func Char(shortcode string) string {
	name := strings.Trim(shortcode, ":")
	if name == "" {
		return ""
	}
	for _, e := range catalog {
		if e.Name == name {
			return e.Char
		}
	}
	return shortcode
}
