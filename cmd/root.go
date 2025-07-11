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

	"github.com/snowplow/snowplow-cli/cmd/dp"
	"github.com/snowplow/snowplow-cli/cmd/ds"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "snowplow-cli",
	Short: "Snowplow CLI",
	Long: `Work with Snowplow from the command line

Configuration required: org-id, api-key-id, api-key
Get credentials: https://docs.snowplow.io/docs/using-the-snowplow-console/managing-console-api-authentication/`,
	Example: `  $ snowplow-cli data-structures download
  $ snowplow-cli ds validate
  $ snowplow-cli mcp`,
	Version: util.Version,
}

func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().String("config", "",
		`Config file. Defaults to $HOME/.config/snowplow/snowplow.yml
Then on:
  Unix $XDG_CONFIG_HOME/snowplow/snowplow.yml
  Darwin $HOME/Library/Application Support/snowplow/snowplow.yml
  Windows %AppData%\snowplow\snowplow.yml`,
	)
	RootCmd.PersistentFlags().String("env-file", "",
		`Environment file (.env). Defaults to .env in current directory
Then on:
  Unix $HOME/.config/snowplow/.env
  Darwin $HOME/Library/Application Support/snowplow/.env
  Windows %AppData%\snowplow\.env`,
	)
	RootCmd.PersistentFlags().Bool("debug", false, "Log output level to Debug")
	RootCmd.PersistentFlags().BoolP("quiet", "q", false, "Log output level to Warn")
	RootCmd.PersistentFlags().BoolP("silent", "s", false, "Disable output")
	RootCmd.PersistentFlags().Bool("json-output", false, "Log output as json")
	RootCmd.AddCommand(ds.DataStructuresCmd)
	RootCmd.AddCommand(dp.DataProductsCmd)
	RootCmd.AddCommand(McpCmd)
}
