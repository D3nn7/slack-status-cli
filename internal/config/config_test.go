package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingYieldsDefaultsAndError(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected an error for a missing config")
	}
	if !cfg.ConfirmDeleteEnabled() {
		t.Error("confirm delete should default to true")
	}
	if cfg.Calendar.PollingIntervalSeconds != defaultPollSecs {
		t.Errorf("got polling interval %d, want %d", cfg.Calendar.PollingIntervalSeconds, defaultPollSecs)
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := Defaults()
	cfg.SlackToken = "xoxp-test"
	cfg.Calendar.Enabled = true
	cfg.Calendar.ICSUrl = "https://example.com/cal.ics"

	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not written: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.SlackToken != "xoxp-test" || loaded.Calendar.ICSUrl != "https://example.com/cal.ics" {
		t.Errorf("unexpected loaded config: %+v", loaded)
	}
}

func TestApplyDefaultsClampsPolling(t *testing.T) {
	cfg := Config{}
	applyDefaults(&cfg)
	if cfg.Calendar.PollingIntervalSeconds != defaultPollSecs {
		t.Errorf("got %d, want %d", cfg.Calendar.PollingIntervalSeconds, defaultPollSecs)
	}
}
