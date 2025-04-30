#!/bin/bash
set -euo pipefail

keys=(
  "darwin_amd64_v1"
  "darwin_arm64_v8.0"
  "linux_386_sse2"
  "linux_amd64_v1"
  "linux_arm64_v8.0"
  "windows_386_sse2"
  "windows_amd64_v1"
  "windows_arm64_v8.0"
)

values=(
  "darwin-amd64"
  "darwin-arm64"
  "linux-386"
  "linux-amd64"
  "linux-arm64"
  "win32-386"
  "win32-amd64"
  "win32-arm64"
)

if [ -z "$version" ]; then
  echo "no version set in env" >&2
  exit 1
fi

cp ../README.md snowplow-cli/
cp ../LICENSE.md snowplow-cli/
jq ".version = \"$version\" | .optionalDependencies |= with_entries(.value = \"$version\")" \
  snowplow-cli/package.json.template >snowplow-cli/package.json

for i in "${!keys[@]}"; do
  p="${values[i]}"

  if [ -z "$p" ]; then
    echo "unknown platform $p" >&2
    exit 1
  fi

  mkdir -p platforms/snowplow-cli-$p/bin
  cp ../README.md platforms/snowplow-cli-$p/
  cp ../LICENSE.md platforms/snowplow-cli-$p/
  cp ../dist/snowplow-cli_${keys[i]}/snowplow-cli* platforms/snowplow-cli-$p/bin

  IFS='-' read -r osPart archPart <<<"${values[i]}"

  cat >"platforms/snowplow-cli-$p/package.json" <<EOT
{
  "name": "@snowplow/snowplow-cli-${values[i]}",
  "version": "$version",
  "license": "see license in LICENSE.md",
  "repository": "https://github.com/snowplow/snowplow-cli",
  "description": "snowplow cli for $osPart ($archPart)",
  "os": ["$osPart"],
  "cpu": ["$archPart"],
  "files": ["bin"]
}
EOT

  npm publish --access public ./platforms/snowplow-cli-$p
done

npm publish --access public ./snowplow-cli
