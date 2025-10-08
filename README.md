# Snowplow CLI

`snowplow-cli` means to bring Snowplow Console into the command line

<!-- TODO: Add screenshot -->

Snowplow CLI is available for [Snowplow BDP](https://docs.snowplow.io/docs/feature-comparison/) clients

## Documentation

Documentation for snowplow-cli is available over [here](https://docs.snowplow.io/docs/understanding-tracking-design/managing-your-data-structures/cli/)

## Installation

snowplow-cli can be installed with [homebrew](https://brew.sh/)

```
brew install snowplow/taps/snowplow-cli
snowplow-cli --help
```

For systems where homebrew is not available binaries for multiple platforms can be found in [releases](https://github.com/snowplow/snowplow-cli/releases)

Example installation for `linux_x86_64` using `curl`

```bash
curl -L -o snowplow-cli https://github.com/snowplow/snowplow-cli/releases/latest/download/snowplow-cli_linux_x86_64
chmod u+x snowplow-cli
./snowplow-cli --help
```

We also publish to [npm](https://www.npmjs.com/package/@snowplow/snowplow-cli)

```bash
npx @snowplow/snowplow-cli --help
```

## Configuration

Snowplow CLI requires configuration to use most of its functionality. Configuration can be provided through multiple sources with the following precedence order (highest to lowest):

1. **Command-line flags** (e.g., `--api-key`, `--org-id`)
2. **Environment variables** (e.g., `SNOWPLOW_CONSOLE_API_KEY`, `SNOWPLOW_CONSOLE_ORG_ID`)
3. **Environment (.env) files**
4. **YAML configuration files**

### Configuration Methods

#### 1. YAML Configuration File

Create a YAML config file:

- Unix/Darwin: `mkdir -p ~/.config/snowplow && touch $HOME/.config/snowplow/snowplow.yml`
<!-- TODO: Windows -->

#### 2. Environment (.env) File

Create a `.env` file in your current directory or specify a custom path with `--env-file`:

```bash
# .env file
SNOWPLOW_CONSOLE_ORG_ID=********-****-****-****-************
SNOWPLOW_CONSOLE_API_KEY_ID=********-****-****-****-************
SNOWPLOW_CONSOLE_API_KEY=********-****-****-****-************
```

The CLI will automatically search for `.env` files in:
- Current working directory (`.env`)
- Config directories (`~/.config/snowplow/.env`)

Or specify a custom path: `snowplow-cli --env-file /path/to/custom.env`

#### 3. Environment Variables

Set environment variables directly:

```bash
export SNOWPLOW_CONSOLE_ORG_ID=********-****-****-****-************
export SNOWPLOW_CONSOLE_API_KEY_ID=********-****-****-****-************
export SNOWPLOW_CONSOLE_API_KEY=********-****-****-****-************
```

### Minimal Configuration

You will need to provide the console organization id, API key and API secret.
You can find the instructions on how to get the API key and secret in the [documentation](https://docs.snowplow.io/docs/using-the-snowplow-console/managing-console-api-authentication/#credentials-ui-v3)

**YAML format** (`snowplow.yml`):

```yaml
console:
  org-id: ********-****-****-****-************
  api-key-id: ********-****-****-****-************
  api-key: ********-****-****-****-************
```

**Environment file format** (`.env`):

```bash
SNOWPLOW_CONSOLE_ORG_ID=********-****-****-****-************
SNOWPLOW_CONSOLE_API_KEY_ID=********-****-****-****-************
SNOWPLOW_CONSOLE_API_KEY=********-****-****-****-************
```


### Claude Code GitHub Actions Integration

This repository includes a GitHub Actions workflow that connects directly with Claude for automated code assistance. The workflow runs when:

- `@claude` is mentioned in issue comments, PR reviews, or review comments  
- An issue is opened or assigned with `@claude` in the title or body  
- A pull request is opened, triggering an automatic Claude review

This lets team members request Claude’s help on code reviews, bug fixes, and development tasks directly in GitHub. Claude can read CI results, write to PRs, and manage issues as needed for collaboration.