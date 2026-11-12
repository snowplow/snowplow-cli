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
	Short: "Validate tracking plans and source applications with Snowplow Console",
	Args:  cobra.ArbitraryArgs,
	Long:  `Sends all tracking plans and source applications from <path> for validation by Snowplow Console.`,
	Example: `  $ snowplow-cli dp validate ./data-products ./source-applications
  $ snowplow-cli dp validate ./src`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := snplog.InitLogging(cmd); err != nil {
			return err
		}

		if err := config.InitConsoleConfig(cmd); err != nil {
			slog.Error("config failure", "error", err)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return err
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		searchPaths := []string{}

		if len(args) == 0 {
			searchPaths = append(searchPaths, util.DataProductsFolder)
			slog.Debug("validation", "msg", fmt.Sprintf("no path provided, using default (./%s)", util.DataProductsFolder))
		}

		searchPaths = append(searchPaths, args...)

		basePath, err := os.Getwd()
		if err != nil {
			slog.Error(err.Error())
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return err
		}

		err = validation.ValidateDataProductsFromCmd(ctx, cmd, searchPaths, basePath)
		if err != nil {
			slog.Error(err.Error())
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return err
		}

		return nil
	},
}

var validateOfflineCmd = &cobra.Command{
	Use:   "offline [paths...]",
	Short: "Validate event specification versions against a git ref, without Snowplow Console",
	Args:  cobra.ArbitraryArgs,
	Long:  `Checks the event specification versions declared in data products from <path> against the files at the merge base of HEAD and --base-ref, without connecting to Snowplow Console. Needs git installed, the full git history and the ref fetched (for example actions/checkout with fetch-depth: 0).`,
	Example: `  $ snowplow-cli dp validate offline --base-ref origin/main
  $ snowplow-cli dp validate offline --base-ref origin/main --fix ./data-products`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return snplog.InitLogging(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		searchPaths := []string{}

		if len(args) == 0 {
			searchPaths = append(searchPaths, util.DataProductsFolder)
			slog.Debug("validation", "msg", fmt.Sprintf("no path provided, using default (./%s)", util.DataProductsFolder))
		}

		searchPaths = append(searchPaths, args...)

		basePath, err := os.Getwd()
		if err != nil {
			slog.Error(err.Error())
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return err
		}

		err = validation.ValidateDataProductsAgainstBaseFromCmd(ctx, cmd, searchPaths, basePath)
		if err != nil {
			slog.Error(err.Error())
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return err
		}

		return nil
	},
}

func init() {
	DataProductsCmd.AddCommand(validateCmd)
	validateCmd.AddCommand(validateOfflineCmd)

	validateCmd.Flags().Bool("gh-annotate", false, "Output suitable for github workflow annotation (ignores -s)")
	validateCmd.Flags().Bool("full", false, "Perform compatibility check on all files, not only the ones that were changed")
	validateCmd.Flags().IntP("concurrency", "c", 3, "The number of validation requests to perform at once (maximum 10)")
	validateCmd.Flags().Bool("fix", false, "Write the version Snowplow Console will assign into each event specification, replacing a declared one that differs and adding it where none is declared and it can be known")

	validateOfflineCmd.Flags().Bool("gh-annotate", false, "Output suitable for github workflow annotation (ignores -s)")
	validateOfflineCmd.Flags().String("base-ref", "", "The git ref whose merge base with HEAD holds the versions to check against")
	validateOfflineCmd.Flags().Bool("fix", false, "Write the version each event specification will get into its file, replacing a declared one that differs and adding it where none is declared and it can be known from the base")
	_ = validateOfflineCmd.MarkFlagRequired("base-ref")
}
