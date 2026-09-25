# slack-status-cli

A terminal UI to manage your Slack status, built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Slack-style status composer** — emoji, status text and a "Clear after"
  selector (30 min, 1 h, 4 h, today, this week or a custom duration).
- **Emoji picker** with live search over a curated catalogue, plus support for
  custom workspace shortcodes (e.g. `:sepp:`).
- **Template library** — create, apply, edit and delete reusable status
  presets, with optional fixed duration, "until HH:MM" or an ad-hoc duration
  prompt.
- **Scheduling** — pre-plan a status for a specific date/time window.
- **Recurring statuses** — weekly rules (e.g. "Home Office, Mon–Fri 08:00–17:00").
- **Calendar view** — month grid with an agenda of meetings and planned statuses.
- **Calendar sync / meeting override** — polls an ICS feed and temporarily
  overrides your status while a meeting or huddle is running, then restores the
  previous status afterwards. The previous status is persisted, so recovery
  works even across restarts.
- **Cross-platform** — Windows, macOS and Linux (amd64 + arm64).

## Requirements

- Go 1.25+ (to build from source)
- A Slack **user token** (`xoxp-…`) with the `users.profile:read` and
  `users.profile:write` scopes

## Setup

Run the CLI once and configure the token via the settings screen (`s`), or copy
`config.example.json` to the configuration location:

| OS      | Path                                                        |
|---------|-------------------------------------------------------------|
| Windows | `%AppData%\slack-status-cli\config.json`                    |
| macOS   | `~/Library/Application Support/slack-status-cli/config.json`|
| Linux   | `~/.config/slack-status-cli/config.json`                    |

For development or portable setups the app also accepts a `config.json` in the
current/parent directory, and honours the `SLACK_STATUS_HOME` environment
variable as an override for the data directory.

```json
{
  "slackToken": "xoxp-dein-slack-token",
  "confirmDelete": true,
  "calendar": {
    "enabled": true,
    "icsUrl": "https://outlook.office365.com/owa/calendar/TOKEN/calendar.ics",
    "defaultEmoji": ":calendar:",
    "defaultText": "In a meeting",
    "useEventTitle": true,
    "meetingOnly": true,
    "keywords": ["kickoff", "retro"],
    "pollingIntervalSeconds": 60
  }
}
```

Legacy `calendar-sync.json` files are still read automatically when the new
`calendar` section is empty.

## Build & run

```sh
go run .                       # run from source
make build                     # native binary in ./bin
make release                   # cross-compiled binaries in ./dist
```

On Windows without `make`:

```powershell
.\build.ps1
go run .
```

## Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Apply selected template |
| `n` | Set / update a status (composer) |
| `e` | Edit the current status |
| `c` | Create a template |
| `p` | Pre-plan a status |
| `x` / `Del` | Delete the selected template |
| `k` / `C` | Calendar view |
| `v` | Calendar sync status |
| `s` | Settings |
| `r` | Refresh status, templates and calendar |
| `?` | Toggle key hints |
| `q` / `Ctrl+C` | Quit |

Inside the composer: `Tab` cycles fields, `Enter` on the emoji field opens the
picker, `Enter` on "Clear after" opens the duration selector, `Esc` cancels.

In the calendar: `←/→` day, `↑/↓` week, `n/p` month, `Enter` pre-plan on the
selected day.

## Status templates

Templates are stored in `templates.json` next to the configuration. Each entry
supports:

| Field | Description |
|-------|-------------|
| `id` | Stable identifier (auto-assigned) |
| `label` | Display name in the list |
| `text` | Status text |
| `emoji` | Slack emoji shortcode, e.g. `:coffee:` |
| `durationInMinutes` | Optional fixed auto-expiry |
| `untilTime` | Optional expiry as `HH:MM` |
| `useDurationSelector` | Prompt for a duration when applied |

## Scheduling & recurrence

Planned statuses live in `schedules.json`. A schedule is applied automatically
while active and takes precedence over the manual status but **not** over a
running meeting. Once it ends, the previous status is restored.

- One-off: a start and end date/time.
- Recurring: selected weekdays plus a start/end time (overnight spans such as
  `22:00–06:00` are supported).

## Calendar sync / meeting override

When enabled, the app polls the configured ICS URL (Outlook, Google Calendar,
Apple Calendar, Nextcloud, …) and:

1. Detects timed, in-progress events. With `meetingOnly: true` only subjects
   that look like meetings are considered (built-in keywords plus `keywords`).
2. Saves your current status, then sets the meeting status with an expiry at
   the event end. `useEventTitle` uses the event subject as the status text.
3. Restores the saved status when the override ends.

Meetings take priority over schedules; among concurrent meetings the
earliest-started one wins. All-day events are ignored. The previous status is
persisted to `state.json` for crash recovery.

## Project layout

```
internal/domain     core types (status, template, schedule, recurrence, event)
internal/config     configuration + platform-aware paths, atomic writes
internal/store      JSON repositories (templates, schedules, state)
internal/slackapi   testable Slack API wrapper
internal/calendar   ICS fetch/parse, timezone mapping, meeting detection
internal/scheduler  pure decision engine for overrides
internal/emoji      emoji catalogue + search
internal/tui        Bubble Tea models, views and widgets
```

## Development

```sh
make review   # gofmt + go vet + go test
```
