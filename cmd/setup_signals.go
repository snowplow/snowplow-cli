/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package cmd

import (
	"os"

	"github.com/snowplow/snowplow-cli/internal/config"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/wizard"
	"github.com/spf13/cobra"
)

var SetupSignalsCmd = &cobra.Command{
	Use:   "signals",
	Short: "Set up Snowplow Signals in this project with your coding agent",
	Long: `Sign in once, then get this project and your coding agent ready for Snowplow Signals:

  - writes Signals credentials for your app to .env (a key that can only read Signals)
  - adds the Snowplow MCP and skills to Claude Code, Codex and Cursor
  - starts your agent with a prompt that walks through the Signals setup

The Snowplow MCP uses snowplow-cli's own key, so your agent needs no separate login.`,
	Example: `  $ snowplow-cli setup signals
  $ npx @snowplow/snowplow-cli setup signals`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := snplog.InitLogging(cmd); err != nil {
			return err
		}
		if err := config.InitConsoleConfigForSetup(cmd); err != nil {
			return err
		}

		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		flag := func(name string) string {
			v, _ := cmd.Flags().GetString(name)
			return v
		}
		noLaunch, _ := cmd.Flags().GetBool("no-launch")

		return wizard.SetupSignals(cmd.Context(), wizard.SignalsOptions{
			ClientID:      flag("client-id"),
			Auth0Domain:   flag("auth0-domain"),
			ConsoleHost:   flag("host"),
			ProjectDir:    dir,
			SkillsRef:     flag("skills-ref"),
			NoLaunch:      noLaunch,
			SavedOrgID:    flag("org-id"),
			SavedAPIKeyID: flag("api-key-id"),
			SavedAPIKey:   flag("api-key"),
		})
	},
}

func init() {
	SetupSignalsCmd.Flags().String("skills-ref", wizard.DefaultSkillsRef, "snowplow/skills git ref (branch or tag) to install skills from")
	SetupSignalsCmd.Flags().Bool("no-launch", false, "Don't offer to start an agent; print the prompt instead")
	SetupCmd.AddCommand(SetupSignalsCmd)
}
