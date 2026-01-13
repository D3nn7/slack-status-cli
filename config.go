package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolvePath(name string) (string, error) {
	paths := []string{name, filepath.Join("..", name)}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("%s not found in current or parent directory", name)
}

func ensureTemplatesFile() (string, error) {
	path, err := resolvePath(templatesName)
	if err == nil {
		return path, nil
	}
	root := ".."
	if _, err := os.Stat(filepath.Join(".", templatesName)); err == nil {
		root = "."
	}
	target := filepath.Join(root, templatesName)
	payload := templatePayload{Templates: []template{}}
	data, _ := json.MarshalIndent(payload, "", "  ")
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return "", err
	}
	return target, nil
}

func loadConfig(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, err
	}
	if cfg.SlackToken == "" {
		return config{}, errors.New("slackToken missing in config.json")
	}
	return cfg, nil
}

func effectiveConfirmDelete(cfg config) bool {
	if cfg.ConfirmDelete == nil {
		return true
	}
	return *cfg.ConfirmDelete
}

func defaultConfigPath() string {
	return filepath.Join("..", configName)
}

func configPathForSave(path string) string {
	if strings.TrimSpace(path) != "" {
		return path
	}
	return defaultConfigPath()
}
