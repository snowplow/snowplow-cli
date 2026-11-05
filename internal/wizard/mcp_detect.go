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
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const snowplowMCPPath = "/api/agent/mcp"

// hasSnowplowMCP treats a missing or unreadable config as no Snowplow MCP.
// At worst, the user is offered an MCP they already have.
func hasSnowplowMCP(a agent, dir string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	switch a.ID {
	case agentClaude:
		return claudeHasSnowplowMCP(home, dir)
	case agentCodex:
		codexHome := filepath.Join(home, ".codex")
		if d := os.Getenv("CODEX_HOME"); d != "" {
			codexHome = d
		}
		return codexFileHasSnowplowMCP(filepath.Join(codexHome, "config.toml")) ||
			codexFileHasSnowplowMCP(filepath.Join(dir, ".codex", "config.toml"))
	case agentCursor:
		return mcpFileHasSnowplowMCP(filepath.Join(home, ".cursor", "mcp.json")) ||
			mcpFileHasSnowplowMCP(filepath.Join(dir, ".cursor", "mcp.json"))
	}
	return false
}

func claudeHasSnowplowMCP(home, dir string) bool {
	if claudePluginHasSnowplowMCP(home, dir) {
		return true
	}
	var doc struct {
		MCPServers map[string]mcpServer `json:"mcpServers"`
		Projects   map[string]struct {
			MCPServers map[string]mcpServer `json:"mcpServers"`
		} `json:"projects"`
	}
	if readJSON(filepath.Join(home, ".claude.json"), &doc) {
		if serversHaveSnowplowMCP(doc.MCPServers) {
			return true
		}
		for _, d := range projectPaths(dir) {
			if serversHaveSnowplowMCP(doc.Projects[d].MCPServers) {
				return true
			}
		}
	}
	return mcpFileHasSnowplowMCP(filepath.Join(dir, ".mcp.json"))
}

// The Snowplow plugin brings the Snowplow skills along with its MCP.
func claudeHasSnowplowPlugin(dir string) bool {
	home, err := os.UserHomeDir()
	return err == nil && claudePluginHasSnowplowMCP(home, dir)
}

// Plugin MCP servers live in each plugin's folder, not in .claude.json.
func claudePluginHasSnowplowMCP(home, dir string) bool {
	var installed struct {
		Plugins map[string][]struct {
			Scope       string `json:"scope"`
			ProjectPath string `json:"projectPath"`
			InstallPath string `json:"installPath"`
		} `json:"plugins"`
	}
	if !readJSON(filepath.Join(home, ".claude", "plugins", "installed_plugins.json"), &installed) {
		return false
	}

	// Later settings files override earlier ones.
	enabled := map[string]bool{}
	for _, path := range []string{
		filepath.Join(home, ".claude", "settings.json"),
		filepath.Join(dir, ".claude", "settings.json"),
		filepath.Join(dir, ".claude", "settings.local.json"),
	} {
		var settings struct {
			EnabledPlugins map[string]bool `json:"enabledPlugins"`
		}
		if readJSON(path, &settings) {
			maps.Copy(enabled, settings.EnabledPlugins)
		}
	}

	paths := projectPaths(dir)
	for name, installs := range installed.Plugins {
		if !enabled[name] {
			continue
		}
		for _, i := range installs {
			if (i.Scope == "user" || slices.Contains(paths, i.ProjectPath)) &&
				mcpFileHasSnowplowMCP(filepath.Join(i.InstallPath, ".mcp.json")) {
				return true
			}
		}
	}
	return false
}

// A project may be recorded under its symlinked or its resolved path.
func projectPaths(dir string) []string {
	if resolved, err := filepath.EvalSymlinks(dir); err == nil && resolved != dir {
		return []string{dir, resolved}
	}
	return []string{dir}
}

func mcpFileHasSnowplowMCP(path string) bool {
	var doc struct {
		MCPServers map[string]mcpServer `json:"mcpServers"`
	}
	return readJSON(path, &doc) && serversHaveSnowplowMCP(doc.MCPServers)
}

// mcp-remote takes the URL as an argument rather than in url.
type mcpServer struct {
	URL  string   `json:"url"`
	Args []string `json:"args"`
}

// Matched by URL, not name: people add the Snowplow MCP under names of their own.
func serversHaveSnowplowMCP(servers map[string]mcpServer) bool {
	for _, s := range servers {
		if strings.Contains(s.URL, snowplowMCPPath) || strings.Contains(strings.Join(s.Args, " "), snowplowMCPPath) {
			return true
		}
	}
	return false
}

func codexFileHasSnowplowMCP(path string) bool {
	b, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(b), snowplowMCPPath)
}

func readJSON(path string, v any) bool {
	b, err := os.ReadFile(path)
	return err == nil && json.Unmarshal(b, v) == nil
}
