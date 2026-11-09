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

	"github.com/snowplow/snowplow-cli/internal/config"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/snowplow/snowplow-cli/internal/validation"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [paths...]",
	Short: "Validate data products and source applications with Snowplow Console",
	Args:  cobra.ArbitraryArgs,
	Long:  `Sends all data products and source applications from <path> for validation by Snowplow Console.`,
	Example: `  $ snowplow-cli dp validate ./data-products ./source-applications
  $ snowplow-cli dp validate ./src
  $ snowplow-cli dp validate --base-ref origin/main`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := snplog.InitLogging(cmd); err != nil {
			return err
		}

		if baseRef, _ := cmd.Flags().GetString("base-ref"); baseRef != "" {
			return nil
		}

		if err := config.InitConsoleConfig(cmd); err != nil {
			slog.Error("config failure", "error", err)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return err
		}

		return nil
	},
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

		if baseRef, _ := cmd.Flags().GetString("base-ref"); baseRef != "" {
			err = validation.ValidateDataProductsAgainstBaseFromCmd(ctx, cmd, searchPaths, basePath)
		} else {
			err = validation.ValidateDataProductsFromCmd(ctx, cmd, searchPaths, basePath)
		}
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
	validateCmd.PersistentFlags().String("base-ref", "", "Check declared event specification versions against the files at the merge base of HEAD and this git ref, without connecting to Snowplow Console. Needs the full git history and the ref fetched (for example actions/checkout with fetch-depth: 0)")
	validateCmd.PersistentFlags().Bool("fix", false, "Write the version Snowplow Console will assign into each event specification that declares a different one")
}
