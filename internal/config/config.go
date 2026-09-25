// Package config loads and persists the application configuration and resolves
// platform-appropriate file locations.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	appDirName      = "slack-status-cli"
	configFileName  = "config.json"
	legacyCalFile   = "calendar-sync.json"
	templatesName   = "templates.json"
	schedulesName   = "schedules.json"
	stateName       = "state.json"
	debugLogName    = "calendar-sync-debug.log"
	defaultPollSecs = 60
	minPollSecs     = 30
)

// CalendarConfig configures the ICS calendar sync / meeting override.
type CalendarConfig struct {
	Enabled                bool     `json:"enabled"`
	ICSUrl                 string   `json:"icsUrl,omitempty"`
	DefaultEmoji           string   `json:"defaultEmoji,omitempty"`
	DefaultText            string   `json:"defaultText,omitempty"`
	UseEventTitle          bool     `json:"useEventTitle,omitempty"`
	MeetingOnly            bool     `json:"meetingOnly"`
	Keywords               []string `json:"keywords,omitempty"`
	PollingIntervalSeconds int      `json:"pollingIntervalSeconds,omitempty"`
}

// Config is the root configuration document.
type Config struct {
	SlackToken    string         `json:"slackToken"`
	ConfirmDelete *bool          `json:"confirmDelete,omitempty"`
	Calendar      CalendarConfig `json:"calendar,omitzero"`
}

// Paths bundles every file the application reads or writes.
type Paths struct {
	Dir       string
	Config    string
	Templates string
	Schedules string
	State     string
	DebugLog  string
}

// Defaults returns a config populated with sensible defaults.
func Defaults() Config {
	confirm := true
	return Config{
		ConfirmDelete: &confirm,
		Calendar: CalendarConfig{
			DefaultEmoji:           ":calendar:",
			DefaultText:            "In einem Meeting",
			MeetingOnly:            true,
			PollingIntervalSeconds: defaultPollSecs,
		},
	}
}

// ResolvePaths determines where the application stores its files. Precedence:
//  1. $SLACK_STATUS_HOME
//  2. a legacy config.json or templates.json in the current/parent directory
//  3. the platform user config directory
func ResolvePaths() Paths {
	if home := strings.TrimSpace(os.Getenv("SLACK_STATUS_HOME")); home != "" {
		return pathsIn(home)
	}
	if dir, ok := legacyDir(); ok {
		return pathsIn(dir)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = "."
	}
	return pathsIn(filepath.Join(base, appDirName))
}

func pathsIn(dir string) Paths {
	return Paths{
		Dir:       dir,
		Config:    filepath.Join(dir, configFileName),
		Templates: filepath.Join(dir, templatesName),
		Schedules: filepath.Join(dir, schedulesName),
		State:     filepath.Join(dir, stateName),
		DebugLog:  filepath.Join(dir, debugLogName),
	}
}

func legacyDir() (string, bool) {
	candidates := []string{".", ".."}
	for _, c := range candidates {
		for _, name := range []string{configFileName, templatesName, legacyCalFile} {
			if _, err := os.Stat(filepath.Join(c, name)); err == nil {
				abs, err := filepath.Abs(c)
				if err != nil {
					continue
				}
				return abs, true
			}
		}
	}
	return "", false
}

// Load reads the configuration from path. Missing fields are filled with
// defaults. A missing file yields the defaults together with an error so the
// caller can show a setup hint.
func Load(path string) (Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, fmt.Errorf("no configuration found at %s", path)
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("konfiguration lesen: %w", err)
	}
	applyDefaults(&cfg)

	// Merge legacy calendar-sync.json when the new section is empty.
	if !cfg.Calendar.Enabled && cfg.Calendar.ICSUrl == "" {
		mergeLegacyCalendar(filepath.Dir(path), &cfg)
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.ConfirmDelete == nil {
		confirm := true
		cfg.ConfirmDelete = &confirm
	}
	c := &cfg.Calendar
	if c.PollingIntervalSeconds < minPollSecs {
		c.PollingIntervalSeconds = defaultPollSecs
	}
	if c.DefaultEmoji == "" {
		c.DefaultEmoji = ":calendar:"
	}
	if c.DefaultText == "" {
		c.DefaultText = "In einem Meeting"
	}
}

type legacyCalendar struct {
	Enabled                bool     `json:"enabled"`
	ICSUrl                 string   `json:"icsUrl"`
	DefaultEmoji           string   `json:"defaultEmoji"`
	DefaultText            string   `json:"defaultText"`
	UseEventTitle          bool     `json:"useEventTitle"`
	PollingIntervalSeconds int      `json:"pollingIntervalSeconds"`
	Keywords               []string `json:"keywords"`
}

func mergeLegacyCalendar(dir string, cfg *Config) {
	data, err := os.ReadFile(filepath.Join(dir, legacyCalFile))
	if err != nil {
		return
	}
	var legacy legacyCalendar
	if err := json.Unmarshal(data, &legacy); err != nil {
		return
	}
	cfg.Calendar.Enabled = legacy.Enabled
	cfg.Calendar.ICSUrl = legacy.ICSUrl
	cfg.Calendar.UseEventTitle = legacy.UseEventTitle
	cfg.Calendar.Keywords = legacy.Keywords
	if legacy.DefaultEmoji != "" {
		cfg.Calendar.DefaultEmoji = legacy.DefaultEmoji
	}
	if legacy.DefaultText != "" {
		cfg.Calendar.DefaultText = legacy.DefaultText
	}
	if legacy.PollingIntervalSeconds > 0 {
		cfg.Calendar.PollingIntervalSeconds = legacy.PollingIntervalSeconds
	}
	applyDefaults(cfg)
}

// Save writes the configuration atomically to path.
func Save(path string, cfg Config) error {
	applyDefaults(&cfg)
	return writeJSON(path, cfg)
}

// ConfirmDeleteEnabled resolves the tri-state pointer with a true default.
func (c Config) ConfirmDeleteEnabled() bool {
	return c.ConfirmDelete == nil || *c.ConfirmDelete
}
