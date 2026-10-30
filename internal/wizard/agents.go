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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const MCPServerName = "snowplow"

type Agent struct {
	ID   string
	Name string
	// Bin is the terminal command, or "" when only the editor is installed.
	Bin       string
	SkillsDir string
	addMCP    func(a Agent, command string, args []string) error
}

func (a Agent) CanLaunch() bool { return a.Bin != "" }

func (a Agent) AddMCP(command string, args []string) error { return a.addMCP(a, command, args) }

// Overridable in tests.
var (
	lookPath   = exec.LookPath
	homeDir    = os.UserHomeDir
	executable = os.Executable
	tempDir    = os.TempDir
	workDir    = os.Getwd
	runCmd     = func(name string, args ...string) error {
		out, err := exec.Command(name, args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
		return nil
	}
)

func DetectAgents() []Agent {
	home, err := homeDir()
	if err != nil {
		return nil
	}

	var agents []Agent
	if bin, err := lookPath("claude"); err == nil {
		agents = append(agents, Agent{ID: "claude", Name: "Claude Code", Bin: bin,
			SkillsDir: filepath.Join(home, ".claude", "skills"), addMCP: addClaudeMCP})
	}
	if bin, err := lookPath("codex"); err == nil {
		agents = append(agents, Agent{ID: "codex", Name: "Codex", Bin: bin,
			SkillsDir: filepath.Join(home, ".agents", "skills"), addMCP: addCodexMCP})
	}
	cursorBin := cursorCLI()
	if _, err := os.Stat(filepath.Join(home, ".cursor")); err == nil || cursorBin != "" {
		agents = append(agents, Agent{ID: "cursor", Name: "Cursor", Bin: cursorBin,
			SkillsDir: filepath.Join(home, ".agents", "skills"), addMCP: addCursorMCP})
	}
	return agents
}

// cursorCLI finds Cursor's terminal agent. It is installed as `agent`, a name
// other tools also use, so only accept one that resolves into a Cursor folder.
func cursorCLI() string {
	if bin, err := lookPath("cursor-agent"); err == nil {
		return bin
	}
	bin, err := lookPath("agent")
	if err != nil {
		return ""
	}
	resolved, err := filepath.EvalSymlinks(bin)
	if err != nil || !strings.Contains(strings.ToLower(resolved), "cursor") {
		return ""
	}
	return bin
}

func addClaudeMCP(a Agent, command string, args []string) error {
	_ = runCmd(a.Bin, "mcp", "remove", "--scope", "user", MCPServerName)
	if err := runCmd(a.Bin, append([]string{"mcp", "add", "--scope", "user", MCPServerName, "--", command}, args...)...); err != nil {
		return err
	}
	// Claude Code prefers a project's own entry to the user one, so if this
	// project has one, add ours for this project only: local scope wins over both.
	if !claudeProjectHasMCP() {
		return nil
	}
	_ = runCmd(a.Bin, "mcp", "remove", "--scope", "local", MCPServerName)
	return runCmd(a.Bin, append([]string{"mcp", "add", "--scope", "local", MCPServerName, "--", command}, args...)...)
}

func claudeProjectHasMCP() bool {
	dir, err := workDir()
	if err != nil {
		return false
	}
	var project struct {
		MCPServers map[string]any `json:"mcpServers"`
	}
	if b, err := os.ReadFile(filepath.Join(dir, ".mcp.json")); err == nil && json.Unmarshal(b, &project) == nil {
		if _, ok := project.MCPServers[MCPServerName]; ok {
			return true
		}
	}

	home, err := homeDir()
	if err != nil {
		return false
	}
	b, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		return false
	}
	var doc struct {
		Projects map[string]struct {
			MCPServers map[string]any `json:"mcpServers"`
		} `json:"projects"`
	}
	if json.Unmarshal(b, &doc) != nil {
		return false
	}
	// The folder may have been reached through a symlink, so check the resolved path too.
	dirs := []string{dir}
	if resolved, err := filepath.EvalSymlinks(dir); err == nil && resolved != dir {
		dirs = append(dirs, resolved)
	}
	for _, d := range dirs {
		if _, ok := doc.Projects[d].MCPServers[MCPServerName]; ok {
			return true
		}
	}
	return false
}

func addCodexMCP(a Agent, command string, args []string) error {
	_ = runCmd(a.Bin, "mcp", "remove", MCPServerName)
	return runCmd(a.Bin, append([]string{"mcp", "add", MCPServerName, "--", command}, args...)...)
}

func addCursorMCP(_ Agent, command string, args []string) error {
	home, err := homeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".cursor", "mcp.json")

	doc := map[string]any{}
	b, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(b, &doc); err != nil {
			return fmt.Errorf("%s is not valid JSON, not changing it: %w", path, err)
		}
	case errors.Is(err, os.ErrNotExist):
	default:
		return err
	}

	servers, _ := doc["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers[MCPServerName] = map[string]any{"command": command, "args": args}
	doc["mcpServers"] = servers

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0600)
}

// MCPCommand is the command agents run to start the snowplow-cli MCP. It is
// the binary running now, since an older snowplow-cli elsewhere on PATH may
// not serve the Console tools. Paths are absolute because editors started
// from the dock don't get the shell's PATH.
func MCPCommand() (string, []string) {
	self, err := executable()
	if err == nil {
		self = filepath.Clean(self)
		switch {
		case strings.Contains(filepath.ToSlash(self), "/_npx/"):
			// npx's download folder is a cache: run it through npx again.
			if npx, err := lookPath("npx"); err == nil {
				return npx, []string{"-y", "@snowplow/snowplow-cli", "mcp"}
			}
		case !isTemporaryBuild(self):
			// Prefer the PATH entry for the same file, e.g. Homebrew's stable
			// symlink rather than a versioned folder an upgrade removes.
			if onPath, err := lookPath("snowplow-cli"); err == nil && sameFile(onPath, self) {
				return onPath, []string{"mcp"}
			}
			return self, []string{"mcp"}
		}
	}

	// A `go run` build or an unknown location: fall back to an installed copy.
	if bin, err := lookPath("snowplow-cli"); err == nil {
		return bin, []string{"mcp"}
	}
	if npx, err := lookPath("npx"); err == nil {
		return npx, []string{"-y", "@snowplow/snowplow-cli", "mcp"}
	}
	return "snowplow-cli", []string{"mcp"}
}

func isTemporaryBuild(path string) bool {
	tmp, err := filepath.EvalSymlinks(tempDir())
	if err != nil {
		tmp = tempDir()
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	return strings.HasPrefix(path, tmp) || strings.Contains(filepath.ToSlash(path), "/go-build")
}

func sameFile(a, b string) bool {
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	return err == nil && os.SameFile(ai, bi)
}

func ClaudePluginInstalled() bool {
	home, err := homeDir()
	if err != nil {
		return false
	}
	b, err := os.ReadFile(filepath.Join(home, ".claude", "plugins", "installed_plugins.json"))
	if err != nil {
		return false
	}
	var doc struct {
		Plugins map[string]any `json:"plugins"`
	}
	if json.Unmarshal(b, &doc) != nil {
		return false
	}
	for name := range doc.Plugins {
		if strings.HasPrefix(name, "snowplow@") {
			return true
		}
	}
	return false
}

func Launch(a Agent, prompt string) error {
	cmd := exec.Command(a.Bin, prompt)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
