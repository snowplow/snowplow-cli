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
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/snowplow-product/snowplow-cli/internal/console"
	snplog "github.com/snowplow-product/snowplow-cli/internal/logging"
	"github.com/snowplow-product/snowplow-cli/internal/publish"
	"github.com/snowplow-product/snowplow-cli/internal/util"
	"github.com/snowplow-product/snowplow-cli/internal/validation"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [paths...]",
	Short: "Validate data structures with BDP Console",
	Args:  cobra.ArbitraryArgs,
	Long:  `Sends all data products and source applications from <path> for validation by BDP Console.`,
	Example: `  $ snowplow-cli dp validate ./data-products ./source-applications
  $ snowplow-cli dp validate ./src`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")
		ghOut, _ := cmd.Flags().GetBool("gh-annotate")
		full, _ := cmd.Flags().GetBool("full")

		searchPaths := []string{}

		if len(args) == 0 {
			searchPaths = append(searchPaths, util.DataProductsFolder)
			slog.Debug("validation", "msg", fmt.Sprintf("no path provided, using default (./%s)", util.DataProductsFolder))
		}

		searchPaths = append(searchPaths, args...)

		files, err := util.MaybeResourcesfromPaths(searchPaths)
		if err != nil {
			snplog.LogFatal(err)
		}

		arg0, err := os.Executable()
		if err != nil {
			snplog.LogFatal(err)
		}
		basePath := filepath.Dir(arg0)

		cnx := context.Background()

		c, err := console.NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			snplog.LogFatal(err)
		}

		changes, err := publish.FindChanges(cnx, c, files)
		if err != nil {
			snplog.LogFatal(err)
		}

		validation.Validate(cnx, c, files, searchPaths, basePath, ghOut, full, changes.IdToFileName)
	},
}

func init() {
	DataProductsCmd.AddCommand(validateCmd)

	validateCmd.PersistentFlags().Bool("gh-annotate", false, "Output suitable for github workflow annotation (ignores -s)")
	validateCmd.PersistentFlags().Bool("full", false, "Perform compatibility check on all files, not only the ones that were changed")
}
