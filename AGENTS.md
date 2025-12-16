# Repository Guidelines

## Project Structure & Module Organization
- CLI entrypoint lives in `src/index.ts`; all prompts, Slack API calls, and template management are centralized here.
- Runtime configuration is read from `config.json` (copy `config.example.json` as a starting point); status templates are stored in `templates.json` under the top-level array key `templates`.
- No test directory exists yet; add new utilities under `src/` and keep user-facing templates/config in the repository root for easy editing by non-developers.

## Build, Test, and Development Commands
- `npm install` — install dependencies (TypeScript, ts-node, Slack SDK, Inquirer, Chalk, Conf).
- `npm start` — run the CLI via `ts-node` (requires a valid `config.json` with `slackToken`).
- `npm test` — currently a placeholder; replace with real test runner when tests are added.

## Coding Style & Naming Conventions
- Language: TypeScript targeting Node; keep imports ES-style.
- Indentation: 4 spaces, semicolons enabled; prefer explicit return types on exported functions.
- Naming: camelCase for variables/functions, PascalCase for interfaces/types, uppercase snake case for constants (e.g., `CONFIG_FILE`).
- Error handling: fail fast with clear Chalk-colored console output; exit with non-zero status for blocking errors (missing config, missing templates).

## Testing Guidelines
- No automated tests exist yet. When adding tests, place them alongside source files or in `__tests__` folders and wire `npm test` to your chosen runner (e.g., vitest/jest).
- Mirror CLI flows with fixture configs/templates and mock Slack API responses; ensure prompts and status payloads are covered.
- Aim for coverage on Slack profile updates (`users.profile.set`) and template CRUD helpers before merging significant changes.

## Commit & Pull Request Guidelines
- Use concise, imperative commit subjects (e.g., `Add duration handling to status setter`); group related CLI/menu changes together.
- Pull requests should describe user-facing behavior changes, include setup steps (e.g., sample `config.json`), and note any Slack-scoped permissions required.
- If UI/menu text changes, include a short before/after note or screenshot of the prompt flow.

## Security & Configuration Tips
- Do not commit real Slack tokens; keep `config.json` local and consider adding `.env` support if secrets expand.
- Validate user input for status text/emoji before calling Slack; avoid writing malformed templates by keeping them under the `templates` array with `label`, `text`, `emoji`, and optional `durationInMinutes`/`untilTime`.
