package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/D3nn7/slack-status-cli/internal/domain"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return New(Paths{
		Templates: filepath.Join(dir, "templates.json"),
		Schedules: filepath.Join(dir, "schedules.json"),
		State:     filepath.Join(dir, "state.json"),
	})
}

func TestTemplatesRoundTripAndIDs(t *testing.T) {
	st := newTestStore(t)
	templates := []domain.Template{
		{Label: "Office", Text: "At the office", Emoji: ":office:"},
	}
	if err := st.SaveTemplates(templates); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := st.LoadTemplates()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("got %d templates, want 1", len(loaded))
	}
	if loaded[0].ID == "" {
		t.Error("expected an ID to be assigned on load")
	}
	if loaded[0].Label != "Office" {
		t.Errorf("unexpected label: %q", loaded[0].Label)
	}
}

func TestLoadLegacyTemplatesObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "templates.json")
	legacy := `{"templates":[{"label":"Office","text":"At the office","emoji":":office:"}]}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	st := New(Paths{Templates: path, Schedules: filepath.Join(dir, "s.json"), State: filepath.Join(dir, "st.json")})
	loaded, err := st.LoadTemplates()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Label != "Office" {
		t.Fatalf("unexpected templates: %+v", loaded)
	}
}

func TestLoadTemplatesBareArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "templates.json")
	if err := os.WriteFile(path, []byte(`[{"label":"A","text":"a","emoji":":a:"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	st := New(Paths{Templates: path, Schedules: filepath.Join(dir, "s.json"), State: filepath.Join(dir, "st.json")})
	loaded, err := st.LoadTemplates()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Label != "A" {
		t.Fatalf("unexpected templates: %+v", loaded)
	}
}

func TestSchedulesRoundTrip(t *testing.T) {
	st := newTestStore(t)
	schedules := []domain.Schedule{
		{Label: "Focus", Text: "Focus time", Emoji: ":dart:", Enabled: true},
	}
	if err := st.SaveSchedules(schedules); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := st.LoadSchedules()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 1 || loaded[0].ID == "" {
		t.Fatalf("unexpected schedules: %+v", loaded)
	}
}

func TestSnapshotLifecycle(t *testing.T) {
	st := newTestStore(t)
	if _, err := st.LoadSnapshot(); err != nil {
		t.Fatalf("missing snapshot should not error: %v", err)
	}
	snap := Snapshot{HasBase: true, BaseStatus: domain.Status{Text: "alt", Emoji: ":x:"}, RefID: "evt"}
	if err := st.SaveSnapshot(snap); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := st.LoadSnapshot()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !loaded.HasBase || loaded.BaseStatus.Text != "alt" || loaded.RefID != "evt" {
		t.Errorf("unexpected snapshot: %+v", loaded)
	}
	if err := st.ClearSnapshot(); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if _, err := st.LoadSnapshot(); err != nil {
		t.Fatalf("load after clear: %v", err)
	}
}
