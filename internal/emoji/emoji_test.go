package emoji

import "testing"

func TestSearchByName(t *testing.T) {
	results := Search("coffee", 0)
	if len(results) == 0 {
		t.Fatal("expected at least one result for 'coffee'")
	}
	if results[0].Shortcode() != ":coffee:" {
		t.Errorf("unexpected first result: %s", results[0].Shortcode())
	}
}

func TestSearchByKeyword(t *testing.T) {
	results := Search("homeoffice", 0)
	found := false
	for _, e := range results {
		if e.Name == "house_with_garden" || e.Name == "desktop_computer" {
			found = true
		}
	}
	if !found {
		t.Error("expected a home-office related emoji")
	}
}

func TestSearchLimit(t *testing.T) {
	results := Search("", 5)
	if len(results) != 5 {
		t.Fatalf("got %d, want 5", len(results))
	}
}

func TestCharLookup(t *testing.T) {
	if got := Char(":coffee:"); got != "☕" {
		t.Errorf("got %q, want coffee glyph", got)
	}
	if got := Char(":custom_thing:"); got != ":custom_thing:" {
		t.Errorf("unknown shortcodes should pass through, got %q", got)
	}
}

func TestNoDuplicateNames(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range catalog {
		if seen[e.Name] {
			t.Errorf("duplicate emoji name %q", e.Name)
		}
		seen[e.Name] = true
	}
}
