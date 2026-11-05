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
	"github.com/snowplow/snowplow-cli/internal/config"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/spf13/cobra"
)

var SignalsCmd = &cobra.Command{
	Use:   "signals",
	Short: "Work with Snowplow Signals",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := snplog.InitLogging(cmd); err != nil {
			return err
		}
		// Only the host is needed: setup signs in, and status reads the project's .env.
		return config.InitConsoleConfigForSetup(cmd)
	},
}

func init() {
	config.InitConsoleFlags(SignalsCmd)
	SignalsCmd.AddCommand(setupCmd)
	SignalsCmd.AddCommand(statusCmd)
}
