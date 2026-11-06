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

  - writes Signals credentials for your app to .env (a key that can read and edit Signals)
  - installs the Snowplow plugin (skills and MCP)
  - starts your agent with a prompt that walks through the Signals setup

The Snowplow MCP signs in with your Snowplow account the first time your agent uses it.

Requires Node.js 18 or later.`,
	Example: `  $ snowplow-cli signals setup
  $ snowplow-cli signals setup --pipeline Production
  $ npx @snowplow/snowplow-cli signals setup`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		host, err := cmd.Flags().GetString("host")
		if err != nil {
			return err
		}
		pipeline, err := cmd.Flags().GetString("pipeline")
		if err != nil {
			return err
		}
		return wizard.SetupSignals(cmd.Context(), wizard.SignalsOptions{ConsoleHost: host, ProjectDir: dir, Pipeline: pipeline})
	},
}

func init() {
	setupCmd.Flags().String("pipeline", "", "Name or ID of the pipeline whose Signals environment to use")
}
