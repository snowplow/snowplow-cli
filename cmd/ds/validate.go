/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package ds

import (
	"errors"

	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/validation"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [paths...] default: [./data-structures]",
	Short: "Validate data structures with BDP Console",
	Args:  cobra.ArbitraryArgs,
	Long:  `Sends all data structures from <path> for validation by BDP Console.`,
	Example: `  $ snowplow-cli ds validate
  $ snowplow-cli ds validate ./my-data-structures ./my-other-data-structures`,
	Run: func(cmd *cobra.Command, args []string) {
		err := validation.ValidateDataStructuresFromCmd(cmd.Context(), cmd, args)
		if err != nil {
			snplog.LogFatal(errors.New("validation failed: " + err.Error()))
		}
	},
}

func init() {
	DataStructuresCmd.AddCommand(validateCmd)

	validateCmd.PersistentFlags().Bool("gh-annotate", false, "Output suitable for github workflow annotation (ignores -s)")
}
