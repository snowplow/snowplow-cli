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
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
)

const (
	// The skills and plugins CLIs' names for the agents.
	agentClaude = "claude-code"
	agentCodex  = "codex"
	agentCursor = "cursor"
)

type agent struct {
	ID   string
	Name string
	// Bin is the command that starts the agent, or "" for one that isn't started.
	Bin string
}

// Claude Code refuses to start inside one of its own sessions.
func (a agent) canLaunch() bool {
	if a.ID == agentClaude && os.Getenv("CLAUDECODE") != "" {
		return false
	}
	return a.Bin != ""
}

func detectAgents() []agent {
	var agents []agent
	if bin, err := exec.LookPath("claude"); err == nil {
		agents = append(agents, agent{ID: agentClaude, Name: "Claude Code", Bin: bin})
	}
	if bin, err := exec.LookPath("codex"); err == nil {
		agents = append(agents, agent{ID: agentCodex, Name: "Codex", Bin: bin})
	}
	// Cursor is set up but never started: it's mostly used as an editor.
	if home, err := os.UserHomeDir(); err == nil {
		if _, err := os.Stat(filepath.Join(home, ".cursor")); err == nil {
			agents = append(agents, agent{ID: agentCursor, Name: "Cursor"})
		}
	}
	return agents
}

func claudeArgs(prompt string) []string {
	return []string{prompt, "--settings", claudeSessionSettings}
}

// Stops Claude Code's file tools, but not shell commands, from reading the app's key in .env.
const claudeSessionSettings = `{"permissions":{"deny":["Read(./.env)","Read(./.env.*)"]}}`

func launch(bin string, args []string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	// Ctrl-C and closing the terminal reach us too: ignore them while the agent has the terminal.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGHUP)
	defer signal.Stop(signals)

	// The agent's own exit status, e.g. from Ctrl-C, isn't a setup failure.
	var exitErr *exec.ExitError
	if err := cmd.Run(); err != nil && !errors.As(err, &exitErr) {
		return err
	}
	return nil
}
