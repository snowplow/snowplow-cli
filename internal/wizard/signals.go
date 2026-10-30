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
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/snowplow/snowplow-cli/internal/config"
	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/setup"
)

type SignalsOptions struct {
	ClientID, Auth0Domain, ConsoleHost     string
	ProjectDir                             string
	SkillsRef                              string
	NoLaunch                               bool
	SavedOrgID, SavedAPIKeyID, SavedAPIKey string
}

// The agent's first message names the skill rather than using a slash command,
// because each agent invokes skills differently.
const signalsStarterPrompt = "Use the signals-setup skill to set up Snowplow Signals in this project. " +
	"Setup is done when all four steps are complete: Tracking, Attribute groups, Services, and Used by your app."

var (
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
	cyan   = color.New(color.FgCyan)
)

func SetupSignals(ctx context.Context, opts SignalsOptions) error {
	// Checked before signing in, so no key is created that would end up in a commit.
	if GitTracked(opts.ProjectDir, ".env") {
		return errors.New(".env is committed to git, so the Signals key would be committed too. " +
			"Stop tracking it with `git rm --cached .env`, commit that, then run this again")
	}
	if GitDirty(opts.ProjectDir) {
		_, _ = yellow.Println("! This project has uncommitted changes. Your agent will edit files, so consider committing first.")
	}

	session, err := setup.Login(ctx, opts.ClientID, opts.Auth0Domain, opts.ConsoleHost)
	if err != nil {
		return err
	}
	org := session.Org

	env, err := pickSignalsEnvironment(ctx, opts, session)
	if err != nil {
		return err
	}

	if err := ensureCLIKey(ctx, opts, session); err != nil {
		return err
	}

	envPath := filepath.Join(opts.ProjectDir, ".env")
	if err := ensureAppKey(ctx, opts, session, env.APIURL(), envPath); err != nil {
		return err
	}

	agents := DetectAgents()
	setupAgents(ctx, agents, opts.SkillsRef)

	_, _ = green.Printf("\n✓ Snowplow Signals is ready for your agent in %s.\n", org.Name)
	return handOff(agents, opts.NoLaunch)
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
		return console.SignalsEnvironment{}, fmt.Errorf("the Signals environment in %s is still being set up; try again when it's ready in Console", s.Org.Name)
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

func ensureCLIKey(ctx context.Context, opts SignalsOptions, s *setup.Session) error {
	if opts.SavedOrgID == s.Org.ID && opts.SavedAPIKeyID != "" && opts.SavedAPIKey != "" {
		if _, err := console.NewApiClient(ctx, opts.ConsoleHost, opts.SavedAPIKeyID, opts.SavedAPIKey, opts.SavedOrgID); err == nil {
			perms, err := console.GetAPIKeyPermissions(ctx, s.AccessToken, opts.ConsoleHost, s.Org.ID, opts.SavedAPIKeyID)
			switch {
			case err != nil:
				slog.Debug("couldn't check the saved key's permissions, reusing it", "error", err)
				_, _ = green.Println("✓ Using your saved snowplow-cli key for the Snowplow MCP")
				return nil
			case console.Allows(perms, s.Org.ID, console.SignalsSetupCapabilities):
				_, _ = green.Println("✓ Using your saved snowplow-cli key for the Snowplow MCP")
				return nil
			default:
				_, _ = yellow.Println("! Your saved snowplow-cli key can't set up Signals and tracking (it may be read-only), so a new one will replace it")
			}
		}
	}

	if opts.SavedOrgID != "" && opts.SavedOrgID != s.Org.ID {
		if !interactive() || !confirm(fmt.Sprintf("snowplow-cli is set up for another organization. Switch it to %s?", s.Org.Name)) {
			return errors.New("snowplow-cli is set up for another organization; run `snowplow-cli setup` to switch, then run this again")
		}
	}

	key, err := console.CreateAPIKey(ctx, s.AccessToken, opts.ConsoleHost, s.Org.ID, false)
	if err != nil {
		return fmt.Errorf("failed to create the snowplow-cli key: %w", err)
	}
	if err := config.SaveConfig(s.Org.ID, key.ID, key.Secret, opts.ConsoleHost); err != nil {
		return err
	}
	_, _ = green.Println("✓ Created a snowplow-cli key for the Snowplow MCP")
	return nil
}

func ensureAppKey(ctx context.Context, opts SignalsOptions, s *setup.Session, apiURL, envPath string) error {
	if c, err := ReadSignalsCredentials(envPath); err == nil && c.OrgID == s.Org.ID && c.APIURL == apiURL {
		if VerifySignals(ctx, opts.ConsoleHost, envPath).OK {
			_, _ = green.Printf("✓ Signals credentials in %s already work\n", envPath)
			return nil
		}
	}

	project := filepath.Base(opts.ProjectDir)
	key, err := console.CreateScopedAPIKey(ctx, s.AccessToken, opts.ConsoleHost, s.Org.ID,
		fmt.Sprintf("%s Signals app key (%s)", s.Email, project), console.SignalsReadCapabilities)
	if err != nil {
		return fmt.Errorf("failed to create the Signals app key: %w", err)
	}

	if err := UpsertDotenv(envPath, map[string]string{
		EnvSignalsAPIURL:   apiURL,
		EnvSignalsOrgID:    s.Org.ID,
		EnvSignalsAPIKeyID: key.ID,
		EnvSignalsAPIKey:   key.Secret,
	}); err != nil {
		return err
	}
	_, _ = green.Printf("✓ Signals credentials written to %s (key limited to reading Signals)\n", envPath)

	if changed, err := EnsureGitignored(opts.ProjectDir, ".env"); err != nil {
		_, _ = yellow.Printf("! Couldn't add .env to .gitignore: %v. Make sure it isn't committed.\n", err)
	} else if changed {
		_, _ = green.Println("✓ Added .env to .gitignore")
	}
	return nil
}

// setupAgents reports and skips failures: one agent's problem shouldn't stop the rest.
func setupAgents(ctx context.Context, agents []Agent, skillsRef string) {
	if len(agents) == 0 {
		command, args := MCPCommand()
		_, _ = yellow.Println("! No supported agent found (Claude Code, Codex or Cursor).")
		fmt.Printf("  Add this MCP server to your agent: %s %s\n", command, strings.Join(args, " "))
		return
	}

	command, args := MCPCommand()
	for _, a := range agents {
		if err := a.AddMCP(command, args); err != nil {
			_, _ = yellow.Printf("! %s: couldn't add the Snowplow MCP: %v\n", a.Name, err)
			continue
		}
		_, _ = green.Printf("✓ %s: Snowplow MCP added\n", a.Name)
	}
	if ClaudePluginInstalled() {
		_, _ = yellow.Println("! The snowplow plugin is also installed in Claude Code. Its MCP server repeats the same tools and asks for a second login; disable it with /plugin.")
	}

	skills, err := FetchSkills(ctx, skillsRef)
	if err != nil {
		_, _ = yellow.Printf("! Couldn't download the Snowplow skills: %v\n", err)
		return
	}
	done := map[string]bool{}
	for _, a := range agents {
		if done[a.SkillsDir] {
			continue
		}
		done[a.SkillsDir] = true
		skipped, err := InstallSkills(a.SkillsDir, skills)
		if err != nil {
			_, _ = yellow.Printf("! Couldn't install skills into %s: %v\n", a.SkillsDir, err)
			continue
		}
		_, _ = green.Printf("✓ Snowplow skills installed in %s\n", a.SkillsDir)
		for _, name := range skipped {
			_, _ = yellow.Printf("! Skipped %s: %s already has a skill with that name\n", name, a.SkillsDir)
		}
	}
}

func handOff(agents []Agent, noLaunch bool) error {
	var launchable []Agent
	for _, a := range agents {
		if a.CanLaunch() {
			launchable = append(launchable, a)
		}
	}

	if !noLaunch && interactive() && len(launchable) > 0 {
		options := make([]string, 0, len(launchable)+1)
		for _, a := range launchable {
			options = append(options, "Start "+a.Name)
		}
		options = append(options, "Show the prompt, I'll start my agent myself")

		i, err := choose("How do you want to continue?", options)
		if err != nil {
			return err
		}
		if i < len(launchable) {
			return Launch(launchable[i], signalsStarterPrompt)
		}
	}

	_, _ = cyan.Println("\nStart your agent in this project and send:")
	fmt.Printf("\n  %s\n\n", signalsStarterPrompt)
	return nil
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

func confirm(question string) bool {
	fmt.Printf("%s (y/N) ", question)
	var input string
	_, _ = fmt.Scanln(&input)
	return strings.EqualFold(input, "y")
}
