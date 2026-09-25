# Repository Guidelines

## Project Overview
Terminal UI (Go + Bubble Tea) for managing a Slack custom status: templates,
scheduling, recurrence, emoji picker, calendar view and calendar-driven
meeting overrides.

## Structure
- `main.go` — thin entrypoint (flag parsing, wiring, `tea.NewProgram`).
- `internal/domain` — pure core types: `Status`, `Template`, `Schedule`,
  `Recurrence`, `Event`, `ClearMode`. No I/O.
- `internal/config` — config document, platform-aware path resolution, atomic
  JSON writes. Legacy `calendar-sync.json` is merged in.
- `internal/store` — JSON repositories for templates, schedules and the
  crash-recovery snapshot. Always use its atomic `WriteJSON`.
- `internal/slackapi` — `Client` interface plus the `slack-go` implementation.
  Depend on the interface so tests can use fakes.
- `internal/calendar` — ICS fetch/parse, Windows→IANA timezone mapping,
  meeting classification.
- `internal/scheduler` — **pure** decision engine (`Evaluate`); keep it free of
  I/O so it stays unit-testable.
- `internal/emoji` — curated catalogue and search.
- `internal/tui` — Bubble Tea model, update loop and views. Screens are
  dispatched from `update.go`; rendering helpers live in the matching `*.go`.

## Build / Test / Dev
- `go run .` — run the TUI.
- `go build ./...` — compile everything.
- `go test ./...` — unit tests.
- `make build` / `make release` — native / cross-platform binaries.
- `make review` — `gofmt` + `go vet` + tests (run before committing).
- `.\build.ps1` — cross-compile on Windows without `make`.

## Coding Style
- Language: Go, ES-style imports, `gofmt` formatted (tabs).
- 4-space-equivalent indentation via gofmt; keep functions focused.
- Naming: `camelCase` unexported, `PascalCase` exported, `UPPER_SNAKE` only for
  true constants. German UI strings are fine; code and comments in English.
- Error handling: return wrapped errors (`fmt.Errorf("…: %w", err)`), surface
  them in the TUI as red footer messages. Never `panic` for user input.
- UI text, code and comments are English; keep phrasing consistent.

## Testing Guidelines
- Tests live next to the code as `*_test.go` in the same package.
- Prioritise `internal/domain`, `internal/scheduler`, `internal/calendar`,
  `internal/store` and `internal/config`.
- Use `t.TempDir()` for filesystem tests; never touch the user's real config.
- Add a fuzz/table test when parsing external formats (ICS, clock strings).

## Commit & PR Guidelines
- Concise imperative subjects (e.g. `Add recurring status schedules`).
- Group related core/TUI changes together; describe user-facing behaviour.
- Note any Slack scopes or config changes required in the PR description.

## Security & Configuration
- Never commit real Slack tokens; `config.json`, `state.json`,
  `schedules.json` and calendar files are gitignored.
- Validate status text/emoji and clock strings before calling the Slack API.
