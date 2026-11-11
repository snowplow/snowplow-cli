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
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestHasSnowplowMCP(t *testing.T) {
	claude, codex, cursor := agent{ID: agentClaude}, agent{ID: agentCodex}, agent{ID: agentCursor}
	const remote = `{"mcpServers":{"snowplow-mcp":{"type":"http","url":"https://console.snowplowanalytics.com/api/agent/mcp"}}}`

	t.Run("nothing configured", func(t *testing.T) {
		_, work := stubEnv(t)
		for _, a := range []agent{claude, codex, cursor} {
			if hasSnowplowMCP(a, work) {
				t.Fatalf("%s: expected no Snowplow MCP", a.ID)
			}
		}
	})

	t.Run("claude user scope", func(t *testing.T) {
		home, work := stubEnv(t)
		writeFile(t, filepath.Join(home, ".claude.json"), remote)
		if !hasSnowplowMCP(claude, work) {
			t.Fatal("expected the user-scope server to count")
		}
	})

	t.Run("claude local scope only for its own project", func(t *testing.T) {
		home, work := stubEnv(t)
		doc, _ := json.Marshal(map[string]any{"projects": map[string]any{
			"/some/other/project": map[string]any{"mcpServers": map[string]any{
				"snowplow": map[string]any{"type": "http", "url": "https://console.snowplowanalytics.com/api/agent/mcp"},
			}},
		}})
		writeFile(t, filepath.Join(home, ".claude.json"), string(doc))
		if hasSnowplowMCP(claude, work) {
			t.Fatal("another project's server shouldn't count")
		}
		writeFile(t, filepath.Join(work, ".mcp.json"), remote)
		if !hasSnowplowMCP(claude, work) {
			t.Fatal("expected the project's .mcp.json server to count")
		}
	})

	t.Run("snowplow-cli mcp without console tools", func(t *testing.T) {
		home, work := stubEnv(t)
		writeFile(t, filepath.Join(home, ".claude.json"), `{"mcpServers":{"snowplow":{"command":"snowplow-cli","args":["mcp"]}}}`)
		if hasSnowplowMCP(claude, work) {
			t.Fatal("local-only tools shouldn't count as the Snowplow MCP")
		}
	})

	t.Run("codex and cursor", func(t *testing.T) {
		home, work := stubEnv(t)
		writeFile(t, filepath.Join(home, ".codex", "config.toml"), "[mcp_servers.snowplow-mcp]\ncommand = \"npx\"\nargs = [\"mcp-remote\", \"https://console.snowplowanalytics.com/api/agent/mcp\"]\n")
		writeFile(t, filepath.Join(work, ".cursor", "mcp.json"), remote)
		if !hasSnowplowMCP(codex, work) || !hasSnowplowMCP(cursor, work) {
			t.Fatal("expected Codex and Cursor servers to count")
		}
	})

	const codexServer = "[mcp_servers.snowplow-mcp]\nurl = \"https://console.snowplowanalytics.com/api/agent/mcp\"\n"

	t.Run("codex project config", func(t *testing.T) {
		_, work := stubEnv(t)
		writeFile(t, filepath.Join(work, ".codex", "config.toml"), codexServer)
		if !hasSnowplowMCP(codex, work) {
			t.Fatal("expected the project's Codex server to count")
		}
	})

	t.Run("codex commented out", func(t *testing.T) {
		home, work := stubEnv(t)
		writeFile(t, filepath.Join(home, ".codex", "config.toml"), "# [mcp_servers.snowplow-mcp]\n# url = \"https://console.snowplowanalytics.com/api/agent/mcp\"\n")
		if hasSnowplowMCP(codex, work) {
			t.Fatal("a commented-out server shouldn't count")
		}
	})

	t.Run("CODEX_HOME", func(t *testing.T) {
		_, work := stubEnv(t)
		codexHome := t.TempDir()
		t.Setenv("CODEX_HOME", codexHome)
		writeFile(t, filepath.Join(codexHome, "config.toml"), codexServer)
		if !hasSnowplowMCP(codex, work) {
			t.Fatal("expected config in CODEX_HOME to count")
		}
	})
}

func TestIsSnowplowMCPMatchesWhereItConnects(t *testing.T) {
	var servers map[string]mcpServer
	_ = json.Unmarshal([]byte(`{
		"my own name": {"command": "npx", "args": ["-y", "mcp-remote", "https://console.snowplowanalytics.com/api/agent/mcp", "3334"]},
		"snowplow": {"command": "snowplow-cli", "args": ["mcp"]},
		"notes": {"command": "notes-mcp", "env": {"SEE": "https://console.snowplowanalytics.com/api/agent/mcp"}},
		"other": {"type": "http", "url": "https://mcp.example.com/api/agent/mcp"}
	}`), &servers)
	only := func(name string) map[string]mcpServer { return map[string]mcpServer{name: servers[name]} }
	if !serversHaveSnowplowMCP(only("my own name")) {
		t.Fatal("mcp-remote to the Snowplow MCP should match under any name")
	}
	if serversHaveSnowplowMCP(only("snowplow")) || serversHaveSnowplowMCP(only("notes")) {
		t.Fatal("only a server's url or args should count, not its name or other fields")
	}
	if serversHaveSnowplowMCP(only("other")) {
		t.Fatal("another host's /api/agent/mcp isn't the Snowplow MCP")
	}
}
