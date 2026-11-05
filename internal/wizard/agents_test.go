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

// fakeBin is an executable that records the arguments of each call, followed
// by "--", in <bin>.args.
func fakeBin(t *testing.T, name string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), name)
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" -- >> \"$0.args\"\n"
	if err := os.WriteFile(bin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func TestLaunch(t *testing.T) {
	bin := fakeBin(t, "claude")
	if err := launch(bin, []string{"the prompt"}); err != nil {
		t.Fatal(err)
	}
	if args, _ := os.ReadFile(bin + ".args"); string(args) != "the prompt\n--\n" {
		t.Fatalf("unexpected args %q", args)
	}
	if err := launch(filepath.Join(t.TempDir(), "missing"), nil); err == nil {
		t.Fatal("expected an error when the agent can't start")
	}
}

func TestClaudeArgsDenyReadingEnv(t *testing.T) {
	args := claudeArgs("the prompt")
	var settings struct {
		Permissions struct {
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	if args[0] != "the prompt" || args[1] != "--settings" || json.Unmarshal([]byte(args[2]), &settings) != nil {
		t.Fatalf("expected the prompt and --settings with valid JSON, got %q", args)
	}
	if strings.Join(settings.Permissions.Deny, ",") != "Read(./.env),Read(./.env.*)" {
		t.Fatalf("unexpected deny rules %v", settings.Permissions.Deny)
	}
}
