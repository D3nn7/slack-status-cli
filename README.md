# slack-status-cli

A terminal UI for managing your Slack status, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- View your current Slack status at a glance
- Apply status templates with a single keypress
- Create, edit, and delete reusable templates
- Set custom status with optional duration or expiry time
- **Outlook Calendar Sync** — automatically sets your Slack status when a meeting starts and restores your previous status when it ends

## Requirements

- Go 1.24+
- A Slack user token (`xoxp-…`) with `users.profile:write` and `users.profile:read` scopes

## Setup

1. Copy `config.example.json` to `config.json` and fill in your Slack token:
   ```json
   { "slackToken": "xoxp-your-token-here" }
   ```

2. Run the TUI:
   ```
   go run .
   ```

## Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Apply selected template |
| `a` / `n` | Set a manual status |
| `e` | Edit current status |
| `c` | Create a new template |
| `x` / `Del` | Delete selected template |
| `s` | Settings |
| `C` | Calendar sync status panel |
| `r` | Refresh status & templates |
| `?` | Show key hints |
| `q` / `Ctrl+C` | Quit |

## Status Templates

Templates are stored in `templates.json`. Each template supports:

| Field | Description |
|-------|-------------|
| `label` | Display name in the list |
| `text` | Status text |
| `emoji` | Slack emoji (e.g. `:coffee:`) |
| `durationInMinutes` | Optional auto-expiry in minutes |
| `untilTime` | Optional expiry as `HH:MM` |
| `useDurationSelector` | If `true`, prompts for duration on apply |

## Calendar Sync

When enabled, the app polls a calendar ICS URL and automatically:

- Sets your Slack status to the meeting title (or a default text) when a meeting starts
- Sets the status expiry to the meeting's end time
- Restores your previous status when the meeting ends

Overlapping meetings use **first-started-wins** priority. Works with any calendar source that provides an ICS URL — Outlook, Google Calendar, Apple Calendar, Nextcloud, etc.

### Get your ICS URL

| Source | Where to find it |
|--------|-----------------|
| **Outlook Web (OWA / Microsoft 365)** | Settings → Calendar → Shared calendars → Publish → copy ICS link |
| **Google Calendar** | Settings → calendar → "Secret address in iCal format" |
| **Apple iCloud** | Calendar.app → share icon → copy link |
| **Nextcloud** | Calendar app → share → copy private link |

### Create `calendar-sync.json`

Copy `calendar-sync.example.json` to `calendar-sync.json`:

```json
{
  "enabled": true,
  "icsUrl": "https://outlook.office365.com/owa/calendar/TOKEN/calendar.ics",
  "defaultEmoji": ":calendar:",
  "defaultText": "In einem Meeting",
  "useEventTitle": true,
  "pollingIntervalSeconds": 60,
  "statePath": "calendar-sync-state.json"
}
```

| Field | Description |
|-------|-------------|
| `enabled` | Master switch |
| `icsUrl` | ICS URL of your calendar |
| `defaultEmoji` | Emoji when `useEventTitle` is false |
| `defaultText` | Status text when `useEventTitle` is false |
| `useEventTitle` | Use the meeting subject as status text |
| `pollingIntervalSeconds` | Poll interval in seconds (minimum 30) |
| `statePath` | Path for the previous-status snapshot file |

### How it works

```
Init()
 └─> pollCalendarCmd  (first poll immediately)
       └─> calEventsMsg → handleCalEvents → startCalSyncTickCmd(60s)
             └─> calSyncTickMsg → pollCalendarCmd → …
```

The previous status (text, emoji, expiry) is saved to `calendar-sync-state.json` before a meeting status is set. If the app is restarted mid-meeting, it recovers the saved state from that file and will restore it when the meeting ends.

## Configuration

`config.json` fields:

| Field | Description |
|-------|-------------|
| `slackToken` | Slack user token |
| `confirmDelete` | Show confirmation before deleting a template (default: `true`) |
