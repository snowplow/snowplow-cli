/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package signals

import (
	"os"

	"github.com/snowplow/snowplow-cli/internal/wizard"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up Snowplow Signals in this project with your coding agent",
	Long: `Sign in once, then get this project and your coding agent ready for Snowplow Signals:

  - writes Signals credentials for your app to .env (a key that can only read Signals)
  - adds the Snowplow skills to Claude Code, Codex and Cursor
  - starts your agent with a prompt that walks through the Signals setup

When it starts Claude Code, the Snowplow MCP is connected for that session only,
with a key that is removed when the session ends. Your agent's own MCP setup is
never changed; to keep the MCP for later sessions, install it from the docs.`,
	Example: `  $ snowplow-cli signals setup
  $ npx @snowplow/snowplow-cli signals setup`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		host, _ := cmd.Flags().GetString("host")
		return wizard.SetupSignals(cmd.Context(), wizard.SignalsOptions{ConsoleHost: host, ProjectDir: dir})
	},
}
