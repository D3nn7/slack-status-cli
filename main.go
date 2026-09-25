// Command slack-status-cli is a terminal UI for managing your Slack status,
// including templates, scheduled and recurring statuses, an emoji picker and
// automatic overrides driven by calendar meetings.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/D3nn7/slack-status-cli/internal/config"
	"github.com/D3nn7/slack-status-cli/internal/slackapi"
	"github.com/D3nn7/slack-status-cli/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("slack-status-cli", version)
		return
	}

	paths := config.ResolvePaths()
	cfg, cfgErr := config.Load(paths.Config)

	var client slackapi.Client
	if cfgErr == nil && cfg.SlackToken != "" {
		client = slackapi.New(cfg.SlackToken)
	}

	model := tui.New(tui.Options{
		Paths:     paths,
		Config:    cfg,
		Client:    client,
		ConfigErr: cfgErr,
	})

	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
