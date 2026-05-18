# CLAUDE.md

Guidance for Claude working in this repository. Keep this file accurate as the codebase evolves.

## What this repo is

`snowplow-cli` is the official command-line interface for Snowplow CDI. It is the eventual home for *all* Snowplow CLI tooling. Today it primarily wraps the Snowplow Console API (data structures, data products, event specs, source apps) and ships an MCP server. Other Snowplow CLI tools (notably `snowplow-tracking-cli`) are being merged in over time — see `docs/specs/` for in-flight work.

Module path: `github.com/snowplow/snowplow-cli`. Go version: 1.25.5. License: Snowplow Limited Use License Agreement v1.0 (header at the top of every source file).

## Layout

```
main.go                    Thin entrypoint → cmd.Execute()
cmd/                       Cobra command tree
  root.go                  Root command, persistent flags, subcommand wiring
  dp/                      `data-products` (alias `dp`) — download, validate, generate, sync, release, purge, add-es
  ds/                      `data-structures` (alias `ds`) — download, validate, generate, publish dev/prod
  setup.go                 Interactive credential setup wizard
  status.go                Print current resolved config
  mcp.go                   MCP server (model context protocol)
  docs.go                  Generate command docs
internal/
  config/                  Viper-based config: flag/env/.env/yaml precedence, Console credentials
  console/                 Console API client (OAuth2, requests for DP/DS/keys/org/migrations)
  download/                Shared download helpers
  logging/                 slog wrapper with debug/quiet/silent/json-output modes
  model/                   Domain types: DataProduct, DataStructure, EventSpec, SourceApp
  validation/              Local + remote validation
  release/                 Sync/release/purge for data products
  changes/                 Change tracking for DS publish
  amend/                   File amendment utilities
  setup/                   Setup wizard internals
  util/                    Constants, file I/O, version string
npm/                       npm distribution wrapper
.github/workflows/         CI (ci.yaml), release (cd.yaml), lint, Claude actions
env.example                Template for SNOWPLOW_CONSOLE_* credentials
```

## Conventions

**CLI framework:** Cobra (`github.com/spf13/cobra` v1.10) + Viper for config. New top-level command families live under their own subpackage in `cmd/` (see `cmd/dp/`, `cmd/ds/` as the templates to follow). Each family exposes one exported `*cobra.Command` and is registered in `cmd/root.go`. Leaf commands sit in sibling files in the same package.

**License header:** Every Go source file starts with the Snowplow Limited Use License Agreement v1.0 header. Match the exact wording used in existing files when creating new ones.

**Config precedence (highest → lowest):** CLI flags → env vars (`SNOWPLOW_CONSOLE_*`) → `.env` file → YAML at `~/.config/snowplow/snowplow.yml` (or platform equivalent). Console credentials (`org-id`, `api-key-id`, `api-key`) are required for any command that hits Console — initialize them in a command's `PersistentPreRunE` via `config.InitConsoleConfig(cmd)` and register flags with `config.InitConsoleFlags(cmd)` (see `cmd/dp/data_products.go`).

**Logging:** Use `slog` via `internal/logging`. Don't write to stdout/stderr directly from command code. Respect `--debug`, `--quiet`, `--silent`, `--json-output` — these are persistent root flags and `snplog.InitLogging(cmd)` wires them up.

**Exit codes:** Cobra returns non-zero on error. Don't call `os.Exit` from leaf command logic; return an error from `RunE`.

**Comments:** Match the existing style — terse, only where the *why* isn't obvious from the code.

## Build, test, lint

```bash
go build ./...
go test ./...
go vet ./...
golangci-lint run        # config in .golangci.yml (see CI workflow)
```

The release pipeline is GitHub Actions (`.github/workflows/cd.yaml`) and publishes to GitHub Releases, Homebrew (`snowplow/taps`), and npm (`@snowplow/snowplow-cli` via `npm/`).

## Working with this codebase

- **Look at neighbours first.** When adding a leaf command, copy the pattern from a similar one in `cmd/dp/` or `cmd/ds/` — flag setup, `PersistentPreRunE`, error wrapping, and logging all have an established style.
- **Don't invent new config keys.** If you need a new credential or endpoint, add it to `internal/config` and `env.example` so all three input methods stay in sync.
- **Console API calls go through `internal/console`.** Don't reach for `net/http` directly from command code.
- **Keep `cmd/` thin.** It should be argument parsing, flag wiring, and a call into `internal/`. Business logic belongs in `internal/`.
- **Tests live next to the code** (`*_test.go`). `internal/testdata/` holds fixtures.

## Active work

Open specs live in `docs/specs/`. Read the relevant one before touching code that overlaps with in-flight changes.
