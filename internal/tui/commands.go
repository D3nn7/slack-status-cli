package tui

import (
	"context"
	"time"

	"github.com/D3nn7/slack-status-cli/internal/calendar"
	"github.com/D3nn7/slack-status-cli/internal/config"
	"github.com/D3nn7/slack-status-cli/internal/domain"
	"github.com/D3nn7/slack-status-cli/internal/slackapi"
	"github.com/D3nn7/slack-status-cli/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

const apiTimeout = 8 * time.Second

type settingsSavedMsg struct {
	cfg    config.Config
	client slackapi.Client
}

func loadTemplatesCmd(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		templates, err := st.LoadTemplates()
		if err != nil {
			return errMsg{err}
		}
		return templatesMsg(templates)
	}
}

func saveTemplatesCmd(st *store.Store, templates []domain.Template) tea.Cmd {
	return func() tea.Msg {
		if err := st.SaveTemplates(templates); err != nil {
			return errMsg{err}
		}
		return savedMsg{note: "Templates saved"}
	}
}

func loadSchedulesCmd(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		schedules, err := st.LoadSchedules()
		if err != nil {
			return errMsg{err}
		}
		return schedulesMsg(schedules)
	}
}

func saveSchedulesCmd(st *store.Store, schedules []domain.Schedule) tea.Cmd {
	return func() tea.Msg {
		if err := st.SaveSchedules(schedules); err != nil {
			return errMsg{err}
		}
		return savedMsg{note: "Schedules saved"}
	}
}

func loadSnapshotCmd(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		snap, err := st.LoadSnapshot()
		if err != nil {
			return errMsg{err}
		}
		return snapshotMsg(snap)
	}
}

func saveSnapshotCmd(st *store.Store, snap store.Snapshot) tea.Cmd {
	return func() tea.Msg {
		if err := st.SaveSnapshot(snap); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func clearSnapshotCmd(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		if err := st.ClearSnapshot(); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func fetchUserCmd(client slackapi.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()
		user, err := client.AuthTest(ctx)
		if err != nil {
			return errMsg{err}
		}
		return userMsg(user)
	}
}

func fetchStatusCmd(client slackapi.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()
		status, err := client.GetStatus(ctx)
		if err != nil {
			return errMsg{err}
		}
		return statusMsg(status)
	}
}

func setStatusCmd(client slackapi.Client, status domain.Status, note string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return errMsg{errNoClient}
		}
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()
		if err := client.SetStatus(ctx, status); err != nil {
			return errMsg{err}
		}
		return statusAppliedMsg{status: status, note: note}
	}
}

func saveSettingsCmd(path string, cfg config.Config, client slackapi.Client) tea.Cmd {
	return func() tea.Msg {
		if err := config.Save(path, cfg); err != nil {
			return errMsg{err}
		}
		return settingsSavedMsg{cfg: cfg, client: client}
	}
}

func validateTokenCmd(path string, cfg config.Config, token string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()
		client := slackapi.New(token)
		if _, err := client.AuthTest(ctx); err != nil {
			return errMsg{errTokenInvalid(err)}
		}
		cfg.SlackToken = token
		if err := config.Save(path, cfg); err != nil {
			return errMsg{err}
		}
		return settingsSavedMsg{cfg: cfg, client: client}
	}
}

func fetchEventsCmd(cfg config.Config) tea.Cmd {
	return func() tea.Msg {
		url := cfg.Calendar.ICSUrl
		if url == "" {
			return eventsMsg{fetchedAt: time.Now()}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		events, err := calendar.NewFetcher().Fetch(ctx, url)
		if err != nil {
			return errMsg{err}
		}
		events = calendar.MarkMeetings(events, cfg.Calendar.Keywords)
		return eventsMsg{events: events, fetchedAt: time.Now()}
	}
}

func scheduleTickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}
