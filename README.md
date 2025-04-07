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
