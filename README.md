# Snowplow CLI

`snowplow-cli` means to bring Snowplow Console into the command line

<!-- TODO: Add screenshot -->

Snowplow CLI is available for [BDP Enterprise](https://docs.snowplow.io/docs/getting-started-on-bdp/private-managed-cloud/) and [BDP Cloud](https://docs.snowplow.io/docs/getting-started-on-bdp/cloud/) clients

<!-- ## Documentation -->
<!-- TODO: Add docs link-->

## Installation
Binaries for most popular platforms and architectures are available in the [releases](https://github.com/snowplow-product/snowplow-cli/releases)

Example installation for `darwin_amd64`.

```bash
curl -L -o snowplow-cli https://github.com/snowplow-product/snowplow-cli/releases/latest/download/snowplow-cli_darwin_amd64
chmod u+x snowplow-cli
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
  api-key-secret: ********-****-****-****-************
```
