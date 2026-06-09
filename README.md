# Snowplow CLI

`snowplow-cli` means to bring Snowplow Console into the command line

<!-- TODO: Add screenshot -->

Snowplow CLI is available for [Snowplow BDP](https://docs.snowplow.io/docs/feature-comparison/) clients

## Documentation 
Documentation for snowplow-cli is available over [here](https://docs.snowplow.io/docs/understanding-tracking-design/managing-your-data-structures/cli/)

## Installation
snowplow-cli can be installed with [homebrew](https://brew.sh/)
```
brew install snowplow-product/taps/snowplow-cli
snowplow-cli --help
```

For systems where homebrew is not available binaries for multiple platforms can be found in [releases](https://github.com/snowplow-product/snowplow-cli/releases)

Example installation for `linux_x86_64` using `curl`

```bash
curl -L -o snowplow-cli https://github.com/snowplow-product/snowplow-cli/releases/latest/download/snowplow-cli_linux_x86_64
chmod u+x snowplow-cli
./snowplow-cli --help
```

## Configuration
Snowplow CLI requires a configuration, to use most of its functionality

### Create a config file
- Unix/Darwin: `mkdir -p ~/.config/snowplow && touch $HOME/.config/snowplow/snowplow.yml`
<!-- TODO: Windows -->

### Minimal configuration
You will need to provide the console organization id, API key and API secret.
You can find the instructions on how to get the API key and secret in the [documentation](https://docs.snowplow.io/docs/using-the-snowplow-console/managing-console-api-authentication/#credentials-ui-v3)

Your `snowplow.yml` content should look like following
```yaml
console:
  org-id: ********-****-****-****-************
  api-key-id: ********-****-****-****-************
  api-key: ********-****-****-****-************
```

## Sending events (`events send`)

Send a single self-describing event to a Snowplow collector:

```bash
snowplow-cli events send \
  --collector collector.example.com \
  --schema iglu:com.snowplowanalytics.snowplow/custom_event/jsonschema/1-0-0 \
  --json '{"category":"test","action":"click"}'
```

Or pass a full self-describing JSON:

```bash
snowplow-cli events send \
  --collector collector.example.com \
  --sdjson '{"schema":"iglu:com.snowplowanalytics.snowplow/custom_event/jsonschema/1-0-0","data":{"category":"test","action":"click"}}'
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--collector` | `-c` | — | Collector domain (required) |
| `--app-id` | `-a` | `snowplowcli` | Application ID |
| `--method` | `-m` | `POST` | HTTP method (`POST`/`GET`) |
| `--protocol` | `-p` | `https` | Protocol (`http`/`https`) |
| `--sdjson` | `-J` | — | Self-describing JSON `{"schema":...,"data":...}` |
| `--schema` | `-d` | — | Schema (data structure) URI |
| `--json` | `-j` | — | Non-self-describing JSON data |
| `--ip-address` | `-i` | — | Custom IP address |
| `--entities` | `-e` | `[]` | JSON array of entities to attach |

Exit codes: `0` (2xx/3xx), `4` (4xx), `5` (5xx), `1` (validation or other error).

### Migrating from `snowplow-tracking-cli`

`events send` aims to replace the standalone `snowplow-tracking-cli`. The behavior — building a
self-describing event and sending it once — is unchanged, including the `0/4/5/1` exit
codes and the validation rules. Only the command prefix and some flag names change.

```bash
# before
snowplow-tracking-cli --collector collector.example.com --schema iglu:... --json '{...}'

# after
snowplow-cli events send --collector collector.example.com --schema iglu:... --json '{...}'
```

Flag mapping:

| Old | New | Note |
|-----|-----|------|
| `--appid` / `-id` | `--app-id` / `-a` | renamed (kebab-case); old name still works but is deprecated; default now `snowplowcli` |
| `--sdjson` / `-sdj` | `--sdjson` / `-J` | new shorthand |
| `--schema` / `-s` | `--schema` / `-d` | new shorthand  |
| `--ipaddress` / `-ip` | `--ip-address` / `-i` | renamed (kebab-case); old name still works but is deprecated |
| `--contexts` / `-ctx` | `--entities` / `-e` | renamed to current Snowplow terminology |
