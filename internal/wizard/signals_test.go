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
	"net/http/httptest"
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

func Test_ensureAppKey_ReplacesKeySignalsRejects(t *testing.T) {
	consoleURL, signalsURL := fakeConsoleAndSignals(t, http.StatusForbidden, ``)
	envPath := writeEnv(t, signalsURL, "secret")

	// The fake Console has no key creation endpoint, so reaching it fails.
	s := &setup.Session{AccessToken: "token", Email: "dev@example.com", Org: console.Organization{ID: "org"}}
	opts := SignalsOptions{ConsoleHost: consoleURL, ProjectDir: filepath.Dir(envPath)}
	err := ensureAppKey(context.Background(), opts, s, signalsURL, envPath)
	if err == nil || !strings.Contains(err.Error(), "failed to create the Signals app key") {
		t.Fatalf("expected a new key to be created, got %v", err)
	}
}

func Test_ensureAppKey_KeepsKeyWhenSignalsFails(t *testing.T) {
	consoleURL, signalsURL := fakeConsoleAndSignals(t, http.StatusServiceUnavailable, ``)
	envPath := writeEnv(t, signalsURL, "secret")
	before, _ := os.ReadFile(envPath)

	s := &setup.Session{AccessToken: "token", Email: "dev@example.com", Org: console.Organization{ID: "org"}}
	opts := SignalsOptions{ConsoleHost: consoleURL, ProjectDir: filepath.Dir(envPath)}
	err := ensureAppKey(context.Background(), opts, s, signalsURL, envPath)
	if err == nil || !strings.Contains(err.Error(), "the Signals API returned 503") {
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

func Test_pickSignalsEnvironment(t *testing.T) {
	const (
		ready    = `{"status":"ready","pipeline":{"id":"p1","name":"Prod"},"config":{"personalizationApiHost":"prod.signals.example"}}`
		creating = `{"status":"creating","pipeline":{"id":"p3","name":"New"}}`
	)
	// Several ready environments prompt, so that case isn't covered here.
	tests := []struct{ name, body, wantURL, wantErr string }{
		{name: "none", body: `[]`, wantErr: "has no Signals environment"},
		{name: "none ready", body: `[` + creating + `]`, wantErr: "no Signals environment in Acme is ready yet"},
		{name: "one ready", body: `[` + creating + `,` + ready + `]`, wantURL: "https://prod.signals.example"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			s := &setup.Session{AccessToken: "token", Org: console.Organization{ID: "org", Name: "Acme"}}
			env, err := pickSignalsEnvironment(context.Background(), SignalsOptions{ConsoleHost: srv.URL}, s)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected an error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if env.APIURL() != tt.wantURL {
				t.Fatalf("expected %s, got %s", tt.wantURL, env.APIURL())
			}
		})
	}
}

func TestInstallPlugin(t *testing.T) {
	home, work := stubEnv(t)
	// Claude Code already has the plugin, and Codex has the MCP from the docs.
	plugin := filepath.Join(home, ".claude", "plugins", "cache", "snowplow", "snowplow", "0.1.0")
	writeFile(t, filepath.Join(plugin, ".mcp.json"), `{"mcpServers":{"snowplow":{"command":"npx","args":["mcp-remote","https://console.snowplowanalytics.com/api/agent/mcp"]}}}`)
	writeFile(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"),
		`{"version":2,"plugins":{"snowplow@snowplow":[{"scope":"user","installPath":"`+plugin+`"}]}}`)
	writeFile(t, filepath.Join(home, ".claude", "settings.json"), `{"enabledPlugins":{"snowplow@snowplow":true}}`)
	writeFile(t, filepath.Join(home, ".codex", "config.toml"), "[mcp_servers.snowplow-mcp]\nargs = [\"mcp-remote\", \"https://console.snowplowanalytics.com/api/agent/mcp\"]\n")
	npx := fakeBin(t, "npx")
	t.Setenv("PATH", filepath.Dir(npx))

	installPlugin(context.Background(), []agent{{ID: agentClaude}, {ID: agentCodex}, {ID: agentCursor}}, work)

	args, _ := os.ReadFile(npx + ".args")
	want := "-y\nplugins@1\nadd\nsnowplow/skills\n-y\n--target\nclaude-code\n--\n" +
		"-y\nplugins@1\nadd\nsnowplow/skills\n-y\n--target\ncursor\n--\n" +
		"-y\nskills@1\nadd\nsnowplow/skills\n-g\n-y\n-a\ncodex\n--\n"
	if string(args) != want {
		t.Fatalf("expected the plugin for Claude Code (an update) and Cursor, and only skills for Codex, got:\n%s", args)
	}
}
