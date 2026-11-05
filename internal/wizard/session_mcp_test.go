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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/setup"
)

func TestClaudeSessionArgsPutPromptBeforeMCPConfig(t *testing.T) {
	args := claudeArgs("the prompt", claudeSessionMCPArgs("https://console.example.com")...)
	n := len(args)
	if n < 3 || args[0] != "the prompt" || args[n-2] != "--mcp-config" {
		t.Fatalf("the prompt must come first and nothing may follow the config, got %q", args)
	}
}

func TestClaudeSessionArgsDenyReadingEnv(t *testing.T) {
	args := claudeArgs("the prompt")
	var settings struct {
		Permissions struct {
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	if args[1] != "--settings" || json.Unmarshal([]byte(args[2]), &settings) != nil {
		t.Fatalf("expected --settings with valid JSON, got %q", args)
	}
	if strings.Join(settings.Permissions.Deny, ",") != "Read(./.env),Read(./.env.*)" {
		t.Fatalf("unexpected deny rules %v", settings.Permissions.Deny)
	}
}

// fakeKeyConsole serves key creation and deletion, recording what it's asked.
func fakeKeyConsole(t *testing.T, createStatus int) (url string, created *string, deleted *[]string) {
	t.Helper()
	var body string
	var deletes []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/msc/v1/organizations/org/credentials/v3/api-keys":
			b, _ := io.ReadAll(r.Body)
			body = string(b)
			w.WriteHeader(createStatus)
			_, _ = w.Write([]byte(`{"id":"session-key-id","key":"session-key-secret"}`))
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/msc/v1/organizations/org/credentials/v2/api-keys/"):
			deletes = append(deletes, strings.TrimPrefix(r.URL.Path, "/api/msc/v1/organizations/org/credentials/v2/api-keys/"))
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)
	return ts.URL, &body, &deletes
}

// fakeBin is an executable that records the arguments and environment it's
// started with in <bin>.args and <bin>.env.
func fakeBin(t *testing.T, name string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), name)
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$0.args\"\nenv > \"$0.env\"\n"
	if err := os.WriteFile(bin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func TestLaunchWithSessionMCP(t *testing.T) {
	session := &setup.Session{AccessToken: "token", Email: "dev@example.com", Org: console.Organization{ID: "org"}}

	t.Run("key lives only for the session", func(t *testing.T) {
		host, created, deleted := fakeKeyConsole(t, http.StatusCreated)
		bin := fakeBin(t, "claude")
		claude := agent{ID: agentClaude, Bin: bin}
		if err := launchWithSessionMCP(context.Background(), SignalsOptions{ConsoleHost: host, ProjectDir: "/p/shop"}, session, claude); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(*created, `"resourceType":"ENVIRONMENTS"`) || strings.Contains(*created, `"resourceType":"*"`) {
			t.Fatalf("expected a key with the session permissions only, got %s", *created)
		}
		args, _ := os.ReadFile(bin + ".args")
		env, _ := os.ReadFile(bin + ".env")
		if !strings.HasPrefix(string(args), signalsStarterPrompt+"\n") || strings.Contains(string(args), "session-key-secret") {
			t.Fatalf("expected the prompt and no key in the agent's arguments, got %s", args)
		}
		if !strings.Contains(string(env), envMCPAPIKey+"=session-key-secret\n") {
			t.Fatal("expected the key in the agent's environment")
		}
		if strings.Join(*deleted, ",") != "session-key-id" {
			t.Fatalf("expected the key to be revoked after the session, got %v", *deleted)
		}
	})

	t.Run("key revoked when the agent fails to start", func(t *testing.T) {
		host, _, deleted := fakeKeyConsole(t, http.StatusCreated)
		missing := agent{ID: agentClaude, Bin: filepath.Join(t.TempDir(), "claude")}
		if err := launchWithSessionMCP(context.Background(), SignalsOptions{ConsoleHost: host}, session, missing); err == nil {
			t.Fatal("expected the launch error")
		}
		if len(*deleted) != 1 {
			t.Fatalf("expected the key to be revoked, got %v", *deleted)
		}
	})

	t.Run("no key falls back to the prompt", func(t *testing.T) {
		host, _, deleted := fakeKeyConsole(t, http.StatusForbidden)
		bin := fakeBin(t, "claude")
		if err := launchWithSessionMCP(context.Background(), SignalsOptions{ConsoleHost: host}, session, agent{ID: agentClaude, Bin: bin}); err != nil {
			t.Fatalf("a key the user can't create shouldn't fail setup: %v", err)
		}
		if _, err := os.Stat(bin + ".args"); err == nil || len(*deleted) != 0 {
			t.Fatal("nothing should be launched or revoked without a key")
		}
	})
}

func TestClaudeSessionMCPArgsKeepKeyOutOfArguments(t *testing.T) {
	args := claudeSessionMCPArgs("https://console.example.com/")
	if len(args) != 2 || args[0] != "--mcp-config" {
		t.Fatalf("unexpected args %v", args)
	}
	var config struct {
		MCPServers map[string]struct {
			Type    string            `json:"type"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(args[1]), &config); err != nil {
		t.Fatal(err)
	}
	s := config.MCPServers[mcpServerName]
	if s.Type != "http" || s.URL != "https://console.example.com/api/agent/mcp" {
		t.Fatalf("unexpected server %+v", s)
	}
	if s.Headers["X-API-Key"] != "${"+envMCPAPIKey+"}" || s.Headers["X-API-Key-ID"] != "${"+envMCPAPIKeyID+"}" || s.Headers["X-Org-Id"] != "${"+envMCPOrgID+"}" {
		t.Fatalf("headers should only name environment variables, got %v", s.Headers)
	}
}
