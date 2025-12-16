package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/slack-go/slack"
	tea "github.com/charmbracelet/bubbletea"
)

func fetchStatusCmd(client *slack.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return errMsg{errors.New("no Slack client configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		profile, err := client.GetUserProfileContext(ctx, &slack.GetUserProfileParameters{})
		if err != nil {
			return errMsg{err}
		}

		user := profile.DisplayName
		if user == "" {
			user = profile.RealName
		}

		exp := ""
		if profile.StatusExpiration > 0 {
			t := time.Unix(int64(profile.StatusExpiration), 0).Local()
			exp = t.Format("15:04")
		}

		info := statusInfo{
			User:       user,
			Text:       profile.StatusText,
			Emoji:      profile.StatusEmoji,
			Expiration: exp,
		}
		return statusMsg(info)
	}
}

func setStatusCmd(client *slack.Client, text, emoji string, duration *int, until string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return errMsg{errors.New("no Slack client configured")}
		}
		expiration := int64(0)

		if duration != nil {
			expiration = time.Now().Add(time.Duration(*duration) * time.Minute).Unix()
		}
		if until != "" {
			t, err := time.Parse("15:04", until)
			if err != nil {
				return errMsg{fmt.Errorf("until time must be HH:MM")}
			}
			now := time.Now()
			target := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
			expiration = target.Unix()
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.SetUserCustomStatusContext(ctx, text, emoji, expiration); err != nil {
			return errMsg{err}
		}
		return setStatusMsg("Status updated")
	}
}

func loadTemplatesCmd(path string) tea.Cmd {
	return func() tea.Msg {
		data, err := os.ReadFile(path)
		if err != nil {
			return errMsg{err}
		}
		var payload templatePayload
		if err := json.Unmarshal(data, &payload); err != nil {
			return errMsg{err}
		}
		if payload.Templates == nil {
			payload.Templates = []template{}
		}
		return templatesMsg(payload.Templates)
	}
}

func saveTemplateCmd(path string, existing []template, newTemplate template) tea.Cmd {
	return func() tea.Msg {
		list := append([]template{}, existing...)
		list = append(list, newTemplate)
		if err := writeTemplates(path, list); err != nil {
			return errMsg{err}
		}
		return savedTemplatesMsg(list)
	}
}

func deleteTemplateCmd(path string, existing []template, label string) tea.Cmd {
	return func() tea.Msg {
		filtered := make([]template, 0, len(existing))
		for _, t := range existing {
			if t.Label != label {
				filtered = append(filtered, t)
			}
		}
		if err := writeTemplates(path, filtered); err != nil {
			return errMsg{err}
		}
		return savedTemplatesMsg(filtered)
	}
}

func saveConfigCmd(path, token string, confirmDelete bool) tea.Cmd {
	return func() tea.Msg {
		target := configPathForSave(path)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client := slack.New(token)
		if _, err := client.AuthTestContext(ctx); err != nil {
			return errMsg{fmt.Errorf("token validation failed: %w", err)}
		}

		cfg := config{
			SlackToken:    token,
			ConfirmDelete: &confirmDelete,
		}

		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return errMsg{err}
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return errMsg{err}
		}

		return configUpdatedMsg{
			cfg:    cfg,
			client: client,
			msg:    "Settings saved",
			path:   target,
		}
	}
}

func writeTemplates(path string, templates []template) error {
	payload := templatePayload{Templates: templates}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
