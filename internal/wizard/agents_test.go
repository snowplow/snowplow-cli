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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func stubEnv(t *testing.T, onPath ...string) (home, work string) {
	t.Helper()
	home, work, bin := t.TempDir(), t.TempDir(), t.TempDir()
	for _, name := range onPath {
		if err := os.WriteFile(filepath.Join(bin, name), nil, 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", bin)
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CLAUDECODE", "")
	return home, work
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectAgents(t *testing.T) {
	home, _ := stubEnv(t, "claude", "codex")
	if err := os.MkdirAll(filepath.Join(home, ".cursor"), 0755); err != nil {
		t.Fatal(err)
	}

	agents := detectAgents()
	var ids []string
	for _, a := range agents {
		ids = append(ids, a.ID)
	}
	if strings.Join(ids, ",") != "claude-code,codex,cursor" {
		t.Fatalf("unexpected agents %v", ids)
	}
	if agents[2].canLaunch() {
		t.Fatal("Cursor should never be started")
	}
}

func TestCanLaunchInsideClaudeCode(t *testing.T) {
	stubEnv(t)
	claude, codex := agent{ID: agentClaude, Bin: "/bin/claude"}, agent{ID: agentCodex, Bin: "/bin/codex"}
	if !claude.canLaunch() {
		t.Fatal("expected Claude Code to be launchable")
	}
	t.Setenv("CLAUDECODE", "1")
	if claude.canLaunch() || !codex.canLaunch() {
		t.Fatal("only Claude Code itself can't be started from a Claude Code session")
	}
}
