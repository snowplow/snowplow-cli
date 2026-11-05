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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/setup"
)

const mcpServerName = "snowplow"

// Not SNOWPLOW_CONSOLE_*, so snowplow-cli commands the agent runs don't pick up the short-lived key.
const (
	envMCPOrgID    = "SNOWPLOW_MCP_ORG_ID"
	envMCPAPIKeyID = "SNOWPLOW_MCP_API_KEY_ID"
	envMCPAPIKey   = "SNOWPLOW_MCP_API_KEY"
)

// Only Claude Code can be given an MCP server for one session (--mcp-config).
func (a agent) takesSessionMCP() bool { return a.ID == agentClaude }

// The key only exists while the agent runs.
func launchWithSessionMCP(ctx context.Context, opts SignalsOptions, s *setup.Session, a agent) error {
	description := fmt.Sprintf("%s Signals setup session (%s)", s.Email, filepath.Base(opts.ProjectDir))
	key, err := console.CreateScopedAPIKey(ctx, s.AccessToken, opts.ConsoleHost, s.Org.ID, description, console.SignalsSetupCapabilities)
	if err != nil {
		// A key can't exceed its creator's permissions, so some roles can't make this one.
		_, _ = yellow.Printf("! Couldn't create a key for the Snowplow MCP: %v\n", err)
		printPrompt(true)
		return nil
	}

	_, _ = green.Println("✓ Snowplow MCP connected for this session only")
	launchErr := launch(a.Bin, claudeArgs(signalsStarterPrompt, claudeSessionMCPArgs(opts.ConsoleHost)...), []string{
		envMCPOrgID + "=" + s.Org.ID,
		envMCPAPIKeyID + "=" + key.ID,
		envMCPAPIKey + "=" + key.Secret,
	})

	if err := console.DeleteAPIKey(ctx, s.AccessToken, opts.ConsoleHost, s.Org.ID, key.ID); err != nil {
		_, _ = yellow.Printf("! Couldn't remove the session's key: %v\n  Delete %q in Console: %s\n", err, description, opts.ConsoleHost)
	}
	if launchErr != nil {
		return launchErr
	}
	_, _ = cyan.Println("\nThe Snowplow MCP was only connected for that session.")
	fmt.Printf("To use it in your other sessions, install it: %s\n", mcpDocsURL)
	return nil
}

// The prompt goes first and extra flags last: --mcp-config takes several
// values, so anything after it would be read as another config.
func claudeArgs(prompt string, extra ...string) []string {
	return append([]string{prompt, "--settings", claudeSessionSettings}, extra...)
}

// Keeps the agent, and any subagent it starts, from reading the app's key in .env.
const claudeSessionSettings = `{"permissions":{"deny":["Read(./.env)","Read(./.env.*)"]}}`

// Claude Code expands the ${VAR} headers, so the key stays out of the arguments.
func claudeSessionMCPArgs(consoleHost string) []string {
	config, _ := json.Marshal(map[string]any{"mcpServers": map[string]any{
		mcpServerName: map[string]any{
			"type": "http",
			"url":  strings.TrimSuffix(consoleHost, "/") + snowplowMCPPath,
			"headers": map[string]string{
				"X-Org-Id":     "${" + envMCPOrgID + "}",
				"X-API-Key-ID": "${" + envMCPAPIKeyID + "}",
				"X-API-Key":    "${" + envMCPAPIKey + "}",
			},
		},
	}})
	return []string{"--mcp-config", string(config)}
}
