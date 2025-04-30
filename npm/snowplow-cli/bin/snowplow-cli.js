#!/usr/bin/env node

const { spawn } = require("child_process");
const os = require("os");

function getPlatformIdentifier() {
  const platform = os.platform();
  const arch = os.arch();

  const platformMap = {
    darwin: {
      x64: "darwin-amd64",
      arm64: "darwin-arm64",
    },
    linux: {
      ia32: "linux-386",
      x64: "linux-amd64",
      arm64: "linux-arm64",
    },
    win32: {
      ia32: "win32-386",
      x64: "win32-amd64",
      arm64: "win32-arm64",
    },
  };

  if (!platformMap[platform] || !platformMap[platform][arch]) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }

  return platformMap[platform][arch];
}

function getExecutablePath() {
  const platformId = getPlatformIdentifier();
  const isWindows = os.platform() === "win32";
  const executableName = isWindows ? "snowplow-cli.exe" : "snowplow-cli";

  try {
    const packageName = `@snowplow/snowplow-cli-${platformId}`;
    const packagePath = require.resolve(`${packageName}/bin/${executableName}`);
    return packagePath;
  } catch (err) {
    throw new Error(
      `Failed to find executable for ${platformId}: ${err.message}`,
    );
  }
}

const executablePath = getExecutablePath();
const childProcess = spawn(executablePath, process.argv.slice(2), {
  stdio: "inherit",
});

childProcess.on("error", (err) => {
  console.error(`Failed to run snowplow-cli: ${err.message}`);
  process.exit(1);
});

childProcess.on("exit", (code) => {
  process.exit(code);
});
