# What this repo is
`snowplow-cli` is the official command-line interface for Snowplow CDI. It is the eventual home for *all* Snowplow CLI tooling. Its core is a git-based workflow for tracking design: data structures and data products (with their event specs and source apps) live as local YAML/JSON files that the CLI downloads, generates, validates, diffs against what is in Console, and publishes or syncs back. It can also send events to a collector for testing purposes.

# Terminology
"Data products" and "tracking plans" are the same thing. The product renamed data products to tracking plans, but the CLI still uses the old name everywhere (`data-products` / `dp` commands, `DataProduct` types, file formats).

# Non-obvious packages
- `internal/setup/` is the credential setup wizard (`setup`); `internal/wizard/` holds the `signals setup` internals.
- `internal/changes/` is change tracking for `ds publish`; `internal/amend/` edits files in place.

# Conventions
**CLI framework:** Cobra + Viper for config. New top-level command families live under their own subpackage in `cmd/`. Each family exposes one exported `*cobra.Command` and is registered in `cmd/root.go`. Leaf commands sit in sibling files in the same package.

**License header:** Every Go source file starts with the Snowplow Limited Use License Agreement v1.0 header. Match the exact wording used in existing files when creating new ones.

**Config precedence (highest → lowest):** CLI flags → env vars (`SNOWPLOW_CONSOLE_*`) → `.env` file → YAML at `~/.config/snowplow/snowplow.yml` (or platform equivalent). Console credentials (`org-id`, `api-key-id`, `api-key`) are required for any command that hits Console — initialize them in a command's `PersistentPreRunE` via `config.InitConsoleConfig(cmd)` and register flags with `config.InitConsoleFlags(cmd)` (see `cmd/dp/data_products.go`).

**Signals config is separate:** `signals setup` / `signals status` use `SNOWPLOW_SIGNALS_*` values in the project's `.env` (`internal/wizard/verify.go`), not `internal/config`. The names deliberately don't overlap with `SNOWPLOW_CONSOLE_*`, so neither picks up the other's key.

**Help text is documentation:** The hidden `generate-docs` command (`cmd/docs.go`) builds the markdown command reference from the Cobra tree, so `Short` / `Long` / `Example` strings and flag descriptions are user-facing copy.

**File formats:** The JSON schemas in `internal/validation/schema/` are embedded and used for local validation of data structures, data products and source apps. When changing a file format or the types in `internal/model`, update the matching schema.

**Logging:** Use `slog` via `internal/logging`. Don't write to stdout/stderr directly from command code. Respect `--debug`, `--quiet`, `--silent`, `--json-output` — these are persistent root flags and `snplog.InitLogging(cmd)` wires them up.

**Exit codes:** Cobra returns non-zero on error. Don't call `os.Exit` from leaf command logic; return an error from `RunE`.

**Comments:** No new code comments unless the user asks for them.

# Build, test, lint

```bash
go build ./...
go test ./...
go vet ./...
golangci-lint run        # config in .golangci.yml (see CI workflow)
```

The release pipeline is GitHub Actions and publishes to GitHub Releases, Homebrew (`snowplow/taps`), and npm (`@snowplow/snowplow-cli` via `npm/`). Release steps and known issues are in `RELEASING.md`.

# Working with this codebase

- **Look at neighbours first.** When adding a leaf command, copy the pattern from a similar one in `cmd/dp/` or `cmd/ds/` — flag setup, `PersistentPreRunE`, error wrapping, and logging all have an established style.
- **Don't invent new config keys.** If you need a new credential or endpoint, add it to `internal/config` and `env.example` so all three input methods stay in sync.
- **Console API calls go through `internal/console`.** Don't reach for `net/http` directly from command code.
- **Keep `cmd/` thin.** It should be argument parsing, flag wiring, and a call into `internal/`. Business logic belongs in `internal/`.
- **Tests live next to the code** (`*_test.go`). `internal/testdata/` holds fixtures.

# MCP server

The MCP server in this repo (`cmd/mcp.go`) was an experiment. It's not expected to get any updates, and will probably be deprecated in the future.
