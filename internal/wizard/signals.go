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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/setup"
)

type SignalsOptions struct {
	ConsoleHost, ProjectDir string
}

// The agent's first message names the skill rather than using a slash command,
// because each agent invokes skills differently.
const signalsStarterPrompt = "Use the signals-setup skill to set up Snowplow Signals in this project. " +
	"Setup is done when all four steps are complete: Tracking, Attribute groups, Services, and Used by your app. " +
	"If the signals-setup skill isn't available, stop and tell the user to install or update the Snowplow skills."

var (
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
	cyan   = color.New(color.FgCyan)
)

func SetupSignals(ctx context.Context, opts SignalsOptions) error {
	if gitTracked(opts.ProjectDir, ".env") {
		return errors.New(".env is committed to git, so the Signals key would be committed too. " +
			"Stop tracking it with `git rm --cached .env`, commit that, then run this again")
	}
	if gitDirty(opts.ProjectDir) {
		_, _ = yellow.Println("! This project has uncommitted changes. Your agent will edit files, so consider committing first.")
	}

	session, err := setup.Login(ctx, setup.DefaultClientID, setup.DefaultAuth0Domain, opts.ConsoleHost)
	if err != nil {
		return err
	}
	org := session.Org
	if org.ID != session.HomeOrgID {
		_, _ = yellow.Printf("! The Snowplow MCP works in your home organization, not %s, so your agent's Snowplow tools won't see this project's setup.\n", org.Name)
	}

	env, err := pickSignalsEnvironment(ctx, opts, session)
	if err != nil {
		return err
	}

	envPath := filepath.Join(opts.ProjectDir, ".env")
	if err := ensureAppKey(ctx, opts, session, env.APIURL(), envPath); err != nil {
		return err
	}

	agents := detectAgents()
	installPlugin(ctx, agents, opts.ProjectDir)

	_, _ = green.Printf("\n✓ Snowplow Signals is ready for your agent in %s.\n", org.Name)
	return startAgent(agents)
}

func pickSignalsEnvironment(ctx context.Context, opts SignalsOptions, s *setup.Session) (console.SignalsEnvironment, error) {
	all, err := console.GetSignalsEnvironments(ctx, s.AccessToken, opts.ConsoleHost, s.Org.ID)
	if err != nil {
		return console.SignalsEnvironment{}, err
	}
	var ready []console.SignalsEnvironment
	for _, e := range all {
		if e.Status == "ready" && e.APIURL() != "" {
			ready = append(ready, e)
		}
	}

	switch {
	case len(ready) == 0 && len(all) > 0:
		return console.SignalsEnvironment{}, fmt.Errorf("no Signals environment in %s is ready yet; try again when one is ready in Console", s.Org.Name)
	case len(ready) == 0:
		return console.SignalsEnvironment{}, fmt.Errorf("%s has no Signals environment; set one up in Console first: %s", s.Org.Name, opts.ConsoleHost)
	case len(ready) == 1:
		_, _ = green.Printf("✓ Signals environment: %s\n", envLabel(ready[0]))
		return ready[0], nil
	}

	if !interactive() {
		return console.SignalsEnvironment{}, fmt.Errorf("%s has %d Signals environments; run this in a terminal to pick one", s.Org.Name, len(ready))
	}
	labels := make([]string, len(ready))
	for i, e := range ready {
		labels[i] = envLabel(e)
	}
	i, err := choose("Which Signals environment should this project use?", labels)
	if err != nil {
		return console.SignalsEnvironment{}, err
	}
	_, _ = green.Printf("✓ Signals environment: %s\n", labels[i])
	return ready[i], nil
}

func envLabel(e console.SignalsEnvironment) string {
	name := e.Pipeline.Name
	if name == "" {
		name = e.Pipeline.ID
	}
	return fmt.Sprintf("%s (%s)", name, e.APIURL())
}

func ensureAppKey(ctx context.Context, opts SignalsOptions, s *setup.Session, apiURL, envPath string) error {
	existing, err := readSignalsCredentials(envPath)
	hasOrgKey := err == nil && existing.OrgID == s.Org.ID

	if hasOrgKey {
		// Keys aren't tied to a Signals environment, so only the URL may need updating.
		if err := upsertDotenv(envPath, map[string]string{envSignalsAPIURL: apiURL}); err != nil {
			return err
		}
		res := VerifySignals(ctx, opts.ConsoleHost, envPath)
		switch {
		case res.OK:
			_, _ = green.Printf("✓ Signals credentials in %s already work\n", envPath)
			return nil
		case res.Reason != ReasonAuth:
			return fmt.Errorf("the Signals credentials in %s don't work: %s", envPath, res.Error)
		}
		// Console rejected the existing key. Create a new one
	}

	project := filepath.Base(opts.ProjectDir)
	key, err := console.CreateScopedAPIKey(ctx, s.AccessToken, opts.ConsoleHost, s.Org.ID,
		fmt.Sprintf("%s Signals app key (%s)", s.Email, project), console.SignalsAppCapabilities)
	if err != nil {
		return fmt.Errorf("failed to create the Signals app key: %w", err)
	}

	if err := upsertDotenv(envPath, map[string]string{
		envSignalsAPIURL:   apiURL,
		envSignalsOrgID:    s.Org.ID,
		envSignalsAPIKeyID: key.ID,
		envSignalsAPIKey:   key.Secret,
	}); err != nil {
		return err
	}
	_, _ = green.Printf("✓ Signals credentials written to %s (key that can read and edit Signals)\n", envPath)
	return nil
}

// An agent that already has the Snowplow MCP some other way only gets the
// skills, so it doesn't end up with two. Failures are reported, not fatal.
func installPlugin(ctx context.Context, agents []agent, dir string) {
	if len(agents) == 0 {
		_, _ = yellow.Println("! No supported agent found (Claude Code, Codex or Cursor). Add the Snowplow plugin to yours with: npx plugins add snowplow/skills")
		return
	}

	var skillsOnly []string
	for _, a := range agents {
		if hasSnowplowMCP(a, dir) {
			skillsOnly = append(skillsOnly, a.ID)
			continue
		}
		npx(ctx, "the Snowplow plugin for "+a.Name, "-y", "plugins@1", "add", "snowplow/skills", "-y", "--target", a.ID)
	}
	if len(skillsOnly) > 0 {
		npx(ctx, "the Snowplow skills", append([]string{"-y", "skills@1", "add", "snowplow/skills", "-g", "-y", "-a"}, skillsOnly...)...)
	}
}

func npx(ctx context.Context, what string, args ...string) {
	cmd := exec.CommandContext(ctx, "npx", args...)
	fmt.Printf("Installing %s: %s\n", what, strings.Join(cmd.Args, " "))
	if out, err := cmd.CombinedOutput(); err != nil {
		_, _ = yellow.Printf("! Couldn't install %s (%v). Run the command above yourself.\n%s", what, err, out)
		return
	}
	_, _ = green.Printf("✓ Installed %s\n", what)
}

func startAgent(agents []agent) error {
	var launchable []agent
	var options []string
	for _, a := range agents {
		if a.canLaunch() {
			launchable = append(launchable, a)
			options = append(options, "Start "+a.Name)
		}
	}

	if interactive() && len(launchable) > 0 {
		// Setup is already done, so an invalid choice falls back to showing the prompt.
		i, err := choose("How do you want to continue?", append(options, "Show the prompt, I'll start my agent myself"))
		if err == nil && i < len(launchable) {
			a := launchable[i]
			args := []string{signalsStarterPrompt}
			if a.ID == agentClaude {
				args = claudeArgs(signalsStarterPrompt)
			}
			_, _ = cyan.Println("\nIf your browser opens to sign in to the Snowplow MCP, accept it to continue.")
			return launch(a.Bin, args)
		}
	}

	printPrompt()
	return nil
}

func printPrompt() {
	_, _ = cyan.Println("\nStart your agent in this project and send:")
	fmt.Printf("\n  %s\n\n", signalsStarterPrompt)
}

func interactive() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func choose(question string, options []string) (int, error) {
	_, _ = cyan.Printf("\n%s\n", question)
	for i, o := range options {
		fmt.Printf("  %d. %s\n", i+1, o)
	}
	fmt.Printf("Select [1-%d] (default: 1): ", len(options))
	var input string
	_, _ = fmt.Scanln(&input)
	return setup.SelectIndex(input, len(options), 0)
}
