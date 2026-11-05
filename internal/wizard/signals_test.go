/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package wizard

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/setup"
)

func Test_ensureAppKey_ReusesWorkingKey(t *testing.T) {
	consoleURL, signalsURL := fakeConsoleAndSignals(t, http.StatusOK, `[{"name":"domain_userid"}]`)
	envPath := writeEnv(t, signalsURL, "secret")
	before, _ := os.ReadFile(envPath)

	s := &setup.Session{AccessToken: "token", Email: "dev@example.com", Org: console.Organization{ID: "org"}}
	opts := SignalsOptions{ConsoleHost: consoleURL, ProjectDir: filepath.Dir(envPath)}
	if err := ensureAppKey(context.Background(), opts, s, signalsURL, envPath); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(envPath); string(after) != string(before) {
		t.Fatalf(".env changed:\n%s", after)
	}
}

func Test_ensureAppKey_ReplacesRejectedKey(t *testing.T) {
	consoleURL, signalsURL := fakeConsoleAndSignals(t, http.StatusOK, `[{"name":"domain_userid"}]`)
	envPath := writeEnv(t, signalsURL, "revoked")

	// The fake Console has no key creation endpoint, so reaching it fails.
	s := &setup.Session{AccessToken: "token", Email: "dev@example.com", Org: console.Organization{ID: "org"}}
	opts := SignalsOptions{ConsoleHost: consoleURL, ProjectDir: filepath.Dir(envPath)}
	err := ensureAppKey(context.Background(), opts, s, signalsURL, envPath)
	if err == nil || !strings.Contains(err.Error(), "failed to create the Signals app key") {
		t.Fatalf("expected a new key to be created, got %v", err)
	}
}

func Test_ensureAppKey_KeepsKeyWhenSignalsFails(t *testing.T) {
	consoleURL, signalsURL := fakeConsoleAndSignals(t, http.StatusUnauthorized, ``)
	envPath := writeEnv(t, signalsURL, "secret")
	before, _ := os.ReadFile(envPath)

	s := &setup.Session{AccessToken: "token", Email: "dev@example.com", Org: console.Organization{ID: "org"}}
	opts := SignalsOptions{ConsoleHost: consoleURL, ProjectDir: filepath.Dir(envPath)}
	err := ensureAppKey(context.Background(), opts, s, signalsURL, envPath)
	if err == nil || !strings.Contains(err.Error(), "the Signals API returned 401") {
		t.Fatalf("expected the check's error, got %v", err)
	}
	if after, _ := os.ReadFile(envPath); string(after) != string(before) {
		t.Fatalf(".env changed:\n%s", after)
	}
}

func Test_ensureAppKey_KeepsKeyForAnotherEnvironment(t *testing.T) {
	consoleURL, signalsURL := fakeConsoleAndSignals(t, http.StatusOK, ``)
	envPath := writeEnv(t, "https://old.signals.example", "secret")

	s := &setup.Session{AccessToken: "token", Email: "dev@example.com", Org: console.Organization{ID: "org"}}
	opts := SignalsOptions{ConsoleHost: consoleURL, ProjectDir: filepath.Dir(envPath)}
	if err := ensureAppKey(context.Background(), opts, s, signalsURL, envPath); err != nil {
		t.Fatal(err)
	}
	if c, _ := readSignalsCredentials(envPath); c.APIURL != signalsURL || c.APIKeyID != "kid" {
		t.Fatalf("expected the same key with the new URL, got %+v", c)
	}
}

func TestInstallSkillsLeavesClaudeToThePlugin(t *testing.T) {
	home, work := stubEnv(t)
	plugin := filepath.Join(home, ".claude", "plugins", "cache", "snowplow", "snowplow", "0.1.0")
	writeFile(t, filepath.Join(plugin, ".mcp.json"), `{"mcpServers":{"snowplow":{"command":"npx","args":["mcp-remote","https://console.snowplowanalytics.com/api/agent/mcp"]}}}`)
	writeFile(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"),
		`{"version":2,"plugins":{"snowplow@snowplow":[{"scope":"user","installPath":"`+plugin+`"}]}}`)
	writeFile(t, filepath.Join(home, ".claude", "settings.json"), `{"enabledPlugins":{"snowplow@snowplow":true}}`)
	npx := fakeBin(t, "npx")
	t.Setenv("PATH", filepath.Dir(npx))

	installSkills(context.Background(), []agent{{ID: agentClaude}, {ID: agentCodex}}, work)

	args, _ := os.ReadFile(npx + ".args")
	if want := "-y\nskills@1\nadd\nsnowplow/skills\n-g\n-y\n-a\ncodex\n"; string(args) != want {
		t.Fatalf("expected only Codex to get the skills, got %q", args)
	}
}
