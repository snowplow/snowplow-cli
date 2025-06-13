/**
 * Copyright (c) 2013-present Snowplow Analytics Ltd.
 * All rights reserved.
 * This software is made available by Snowplow Analytics, Ltd.,
 * under the terms of the Snowplow Limited Use License Agreement, Version 1.0
 * located at https://docs.snowplow.io/limited-use-license-1.0
 * BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
 * OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
 */

package dp

import (
	"fmt"
	"log/slog"
	"os"

	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/snowplow/snowplow-cli/internal/validation"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [paths...]",
	Short: "Validate data products and source applications with BDP Console",
	Args:  cobra.ArbitraryArgs,
	Long:  `Sends all data products and source applications from <path> for validation by BDP Console.`,
	Example: `  $ snowplow-cli dp validate ./data-products ./source-applications
  $ snowplow-cli dp validate ./src`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		searchPaths := []string{}

		if len(args) == 0 {
			searchPaths = append(searchPaths, util.DataProductsFolder)
			slog.Debug("validation", "msg", fmt.Sprintf("no path provided, using default (./%s)", util.DataProductsFolder))
		}

		searchPaths = append(searchPaths, args...)

		basePath, err := os.Getwd()
		if err != nil {
			snplog.LogFatal(err)
		}

		err = validation.ValidateDataProductsFromCmd(ctx, cmd, searchPaths, basePath)
		if err != nil {
			snplog.LogFatal(err)
		}

	},
}

func init() {
	DataProductsCmd.AddCommand(validateCmd)

	validateCmd.PersistentFlags().Bool("gh-annotate", false, "Output suitable for github workflow annotation (ignores -s)")
	validateCmd.PersistentFlags().Bool("full", false, "Perform compatibility check on all files, not only the ones that were changed")
	validateCmd.PersistentFlags().IntP("concurrency", "c", 3, "The number of validation requests to perform at once (maximum 10)")
}
