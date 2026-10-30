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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func stubEnv(t *testing.T, onPath map[string]string) (home string, calls *[]string) {
	t.Helper()
	home = t.TempDir()
	var c []string
	origLook, origHome, origRun, origWork := lookPath, homeDir, runCmd, workDir
	work := t.TempDir()
	workDir = func() (string, error) { return work, nil }
	lookPath = func(name string) (string, error) {
		if p, ok := onPath[name]; ok {
			return p, nil
		}
		return "", exec.ErrNotFound
	}
	homeDir = func() (string, error) { return home, nil }
	runCmd = func(name string, args ...string) error {
		c = append(c, name+" "+strings.Join(args, " "))
		return nil
	}
	t.Cleanup(func() { lookPath, homeDir, runCmd, workDir = origLook, origHome, origRun, origWork })
	return home, &c
}

func TestDetectAgents(t *testing.T) {
	home, _ := stubEnv(t, map[string]string{"claude": "/bin/claude", "codex": "/bin/codex"})
	if err := os.MkdirAll(filepath.Join(home, ".cursor"), 0755); err != nil {
		t.Fatal(err)
	}

	agents := DetectAgents()
	var ids []string
	for _, a := range agents {
		ids = append(ids, a.ID)
	}
	if strings.Join(ids, ",") != "claude,codex,cursor" {
		t.Fatalf("unexpected agents %v", ids)
	}
	if agents[2].CanLaunch() {
		t.Fatal("Cursor without its CLI should not be launchable")
	}
	if agents[0].SkillsDir != filepath.Join(home, ".claude", "skills") || agents[1].SkillsDir != filepath.Join(home, ".agents", "skills") {
		t.Fatalf("unexpected skills dirs %q %q", agents[0].SkillsDir, agents[1].SkillsDir)
	}
}

func TestCursorCLIIgnoresOtherAgentBinaries(t *testing.T) {
	stubEnv(t, map[string]string{"agent": "/usr/local/bin/agent"})
	if got := cursorCLI(); got != "" {
		t.Fatalf("expected no Cursor CLI, got %q", got)
	}
}

func TestAddClaudeAndCodexMCP(t *testing.T) {
	_, calls := stubEnv(t, map[string]string{"claude": "/bin/claude", "codex": "/bin/codex"})

	for _, a := range DetectAgents() {
		if err := a.AddMCP("/bin/snowplow-cli", []string{"mcp"}); err != nil {
			t.Fatal(err)
		}
	}

	want := []string{
		"/bin/claude mcp remove --scope user snowplow",
		"/bin/claude mcp add --scope user snowplow -- /bin/snowplow-cli mcp",
		"/bin/codex mcp remove snowplow",
		"/bin/codex mcp add snowplow -- /bin/snowplow-cli mcp",
	}
	if strings.Join(*calls, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got:\n%s", strings.Join(*calls, "\n"))
	}
}

func TestAddClaudeMCPWhenProjectHasItsOwn(t *testing.T) {
	cases := map[string]func(t *testing.T, home, project string){
		"project .mcp.json": func(t *testing.T, home, project string) {
			writeFile(t, filepath.Join(project, ".mcp.json"), `{"mcpServers":{"snowplow":{"type":"http","url":"https://x"}}}`)
		},
		"local scope in ~/.claude.json": func(t *testing.T, home, project string) {
			b, _ := json.Marshal(map[string]any{"projects": map[string]any{
				project: map[string]any{"mcpServers": map[string]any{"snowplow": map[string]any{"command": "old"}}},
			}})
			writeFile(t, filepath.Join(home, ".claude.json"), string(b))
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			home, calls := stubEnv(t, map[string]string{"claude": "/bin/claude"})
			project, _ := workDir()
			setup(t, home, project)

			if err := DetectAgents()[0].AddMCP("/bin/snowplow-cli", []string{"mcp"}); err != nil {
				t.Fatal(err)
			}
			want := []string{
				"/bin/claude mcp remove --scope user snowplow",
				"/bin/claude mcp add --scope user snowplow -- /bin/snowplow-cli mcp",
				"/bin/claude mcp remove --scope local snowplow",
				"/bin/claude mcp add --scope local snowplow -- /bin/snowplow-cli mcp",
			}
			if strings.Join(*calls, "\n") != strings.Join(want, "\n") {
				t.Fatalf("got:\n%s", strings.Join(*calls, "\n"))
			}
		})
	}
}

func TestClaudeProjectHasMCPIgnoresOtherProjects(t *testing.T) {
	home, _ := stubEnv(t, nil)
	project, _ := workDir()
	writeFile(t, filepath.Join(project, ".mcp.json"), `{"mcpServers":{"github":{"url":"https://x"}}}`)
	writeFile(t, filepath.Join(home, ".claude.json"), `{"projects":{"/somewhere/else":{"mcpServers":{"snowplow":{}}}}}`)
	if claudeProjectHasMCP() {
		t.Fatal("expected no project entry")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestAddCursorMCPKeepsOtherServers(t *testing.T) {
	home, _ := stubEnv(t, nil)
	path := filepath.Join(home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"github":{"url":"https://x"}},"other":1}`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := addCursorMCP(Agent{}, "/bin/snowplow-cli", []string{"mcp"}); err != nil {
		t.Fatal(err)
	}

	b, _ := os.ReadFile(path)
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	servers := doc["mcpServers"].(map[string]any)
	if _, ok := servers["github"]; !ok || doc["other"] == nil {
		t.Fatalf("existing config lost: %s", b)
	}
	sp := servers["snowplow"].(map[string]any)
	if sp["command"] != "/bin/snowplow-cli" {
		t.Fatalf("unexpected entry %v", sp)
	}
}

func TestAddCursorMCPRefusesInvalidJSON(t *testing.T) {
	home, _ := stubEnv(t, nil)
	path := filepath.Join(home, ".cursor", "mcp.json")
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	_ = os.WriteFile(path, []byte(`{not json`), 0644)

	if err := addCursorMCP(Agent{}, "x", nil); err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
	b, _ := os.ReadFile(path)
	if string(b) != `{not json` {
		t.Fatal("invalid file should be left alone")
	}
}

// stubBinaries lays out fake install locations: "tmp" stands in for the
// system temp folder, everything else counts as a real install.
func stubBinaries(t *testing.T, running string) string {
	t.Helper()
	root := t.TempDir()
	for _, p := range []string{"tmp/go-build1/exe/snowplow-cli", "opt/new/snowplow-cli", "opt/old/snowplow-cli", "npm/_npx/abc/node_modules/x/snowplow-cli"} {
		full := filepath.Join(root, p)
		_ = os.MkdirAll(filepath.Dir(full), 0755)
		_ = os.WriteFile(full, []byte(p), 0755)
	}
	_ = os.Symlink(filepath.Join(root, "opt/new/snowplow-cli"), filepath.Join(root, "opt/link"))

	origExe, origTmp := executable, tempDir
	executable = func() (string, error) { return filepath.Join(root, running), nil }
	tempDir = func() string { return filepath.Join(root, "tmp") }
	t.Cleanup(func() { executable, tempDir = origExe, origTmp })
	return root
}

func TestMCPCommandUsesRunningBinaryOverOlderOnPath(t *testing.T) {
	stubEnv(t, nil)
	root := stubBinaries(t, "opt/new/snowplow-cli")
	stubEnvKeep(t, map[string]string{"snowplow-cli": filepath.Join(root, "opt/old/snowplow-cli")})

	cmd, args := MCPCommand()
	if cmd != filepath.Join(root, "opt/new/snowplow-cli") || strings.Join(args, " ") != "mcp" {
		t.Fatalf("unexpected command %s %v", cmd, args)
	}
}

func TestMCPCommandPrefersStablePathToSameBinary(t *testing.T) {
	stubEnv(t, nil)
	root := stubBinaries(t, "opt/new/snowplow-cli")
	stubEnvKeep(t, map[string]string{"snowplow-cli": filepath.Join(root, "opt/link")})

	if cmd, _ := MCPCommand(); cmd != filepath.Join(root, "opt/link") {
		t.Fatalf("expected the PATH symlink, got %s", cmd)
	}
}

func TestMCPCommandFromNpx(t *testing.T) {
	stubEnv(t, nil)
	stubBinaries(t, "npm/_npx/abc/node_modules/x/snowplow-cli")
	stubEnvKeep(t, map[string]string{"npx": "/bin/npx"})

	cmd, args := MCPCommand()
	if cmd != "/bin/npx" || strings.Join(args, " ") != "-y @snowplow/snowplow-cli mcp" {
		t.Fatalf("unexpected command %s %v", cmd, args)
	}
}

func TestMCPCommandFromGoRunFallsBackToInstalled(t *testing.T) {
	stubEnv(t, nil)
	stubBinaries(t, "tmp/go-build1/exe/snowplow-cli")
	stubEnvKeep(t, map[string]string{"snowplow-cli": "/opt/homebrew/bin/snowplow-cli"})

	if cmd, _ := MCPCommand(); cmd != "/opt/homebrew/bin/snowplow-cli" {
		t.Fatalf("a temporary build shouldn't be the MCP command, got %s", cmd)
	}
}

// stubEnvKeep changes only what's on PATH, keeping the other stubs.
func stubEnvKeep(t *testing.T, onPath map[string]string) {
	t.Helper()
	orig := lookPath
	lookPath = func(name string) (string, error) {
		if p, ok := onPath[name]; ok {
			return p, nil
		}
		return "", exec.ErrNotFound
	}
	t.Cleanup(func() { lookPath = orig })
}

func TestClaudePluginInstalled(t *testing.T) {
	home, _ := stubEnv(t, nil)
	if ClaudePluginInstalled() {
		t.Fatal("no plugins file should mean not installed")
	}
	path := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	_ = os.WriteFile(path, []byte(`{"version":2,"plugins":{"snowplow@snowplow":[{}]}}`), 0644)
	if !ClaudePluginInstalled() {
		t.Fatal("expected the plugin to be detected")
	}
}
