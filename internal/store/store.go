// Package store persists templates, schedules and crash-recovery state as
// human-editable JSON files using atomic writes.
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/D3nn7/slack-status-cli/internal/domain"
)

// Store is a file-backed repository.
type Store struct {
	templatesPath string
	schedulesPath string
	statePath     string
}

// Paths configures the store's file locations.
type Paths struct {
	Templates string
	Schedules string
	State     string
}

// New returns a Store for the given paths.
func New(p Paths) *Store {
	return &Store{
		templatesPath: p.Templates,
		schedulesPath: p.Schedules,
		statePath:     p.State,
	}
}

// Snapshot captures the status that was active before an automatic override,
// so it can be restored when the override ends or after a crash.
type Snapshot struct {
	HasBase    bool          `json:"hasBase"`
	BaseStatus domain.Status `json:"baseStatus,omitzero"`
	Source     string        `json:"source,omitempty"`
	RefID      string        `json:"refId,omitempty"`
}

// TemplatesPath returns the configured templates file path.
func (s *Store) TemplatesPath() string { return s.templatesPath }

// SchedulesPath returns the configured schedules file path.
func (s *Store) SchedulesPath() string { return s.schedulesPath }

// templateFile is the on-disk wrapper for templates. The object form
// ({"templates": [...]}) keeps backwards compatibility with the legacy CLI.
type templateFile struct {
	Templates []domain.Template `json:"templates"`
}

// scheduleFile is the on-disk wrapper for schedules.
type scheduleFile struct {
	Schedules []domain.Schedule `json:"schedules"`
}

// LoadTemplates reads templates, upgrading legacy entries with IDs. Both the
// object form and a bare JSON array are accepted.
func (s *Store) LoadTemplates() ([]domain.Template, error) {
	data, err := readFile(s.templatesPath)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var file templateFile
	if err := json.Unmarshal(data, &file); err == nil && file.Templates != nil {
		return domain.EnsureTemplateIDs(file.Templates), nil
	}
	var arr []domain.Template
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, err
	}
	return domain.EnsureTemplateIDs(arr), nil
}

// SaveTemplates writes templates atomically in the object form.
func (s *Store) SaveTemplates(templates []domain.Template) error {
	return WriteJSON(s.templatesPath, templateFile{
		Templates: domain.EnsureTemplateIDs(templates),
	})
}

// LoadSchedules reads schedules, upgrading legacy entries with IDs. Both the
// object form and a bare JSON array are accepted.
func (s *Store) LoadSchedules() ([]domain.Schedule, error) {
	data, err := readFile(s.schedulesPath)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var file scheduleFile
	if err := json.Unmarshal(data, &file); err == nil && file.Schedules != nil {
		return domain.EnsureScheduleIDs(file.Schedules), nil
	}
	var arr []domain.Schedule
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, err
	}
	return domain.EnsureScheduleIDs(arr), nil
}

// SaveSchedules writes schedules atomically in the object form.
func (s *Store) SaveSchedules(schedules []domain.Schedule) error {
	return WriteJSON(s.schedulesPath, scheduleFile{
		Schedules: domain.EnsureScheduleIDs(schedules),
	})
}

// LoadSnapshot reads the crash-recovery snapshot. A missing file is not an
// error and yields a zero snapshot.
func (s *Store) LoadSnapshot() (Snapshot, error) {
	return readOrEmpty[Snapshot](s.statePath)
}

// SaveSnapshot writes the snapshot atomically.
func (s *Store) SaveSnapshot(snap Snapshot) error {
	return WriteJSON(s.statePath, snap)
}

// ClearSnapshot removes the snapshot file, ignoring a missing file.
func (s *Store) ClearSnapshot() error {
	err := os.Remove(s.statePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// ReadJSON decodes a JSON file into T.
func ReadJSON[T any](path string) (T, error) {
	var out T
	data, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	return out, nil
}

// WriteJSON encodes v as indented JSON and writes it atomically.
func WriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// readFile returns the file contents, treating a missing file as empty.
func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

func readOrEmpty[T any](path string) (T, error) {
	var zero T
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return zero, nil
		}
		return zero, err
	}
	if len(data) == 0 {
		return zero, nil
	}
	if err := json.Unmarshal(data, &zero); err != nil {
		return zero, err
	}
	return zero, nil
}
