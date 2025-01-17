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

	"github.com/snowplow-product/snowplow-cli/internal/console"
	snplog "github.com/snowplow-product/snowplow-cli/internal/logging"
	"github.com/snowplow-product/snowplow-cli/internal/publish"
	"github.com/snowplow-product/snowplow-cli/internal/util"
	"github.com/snowplow-product/snowplow-cli/internal/validation"
	"github.com/spf13/cobra"
)

var publishCommand = &cobra.Command{
	Use:   "publish {directory ./data-products}",
	Short: "Publish all data products, event specs and source apps to BDP Console",
	Long: `Publish the local version versions of all data products, event specs and source apps from BDP Console.

If no directory is provided then defaults to 'data-products' in the current directory. Source apps are stored in the nested 'source-apps' directory`,
	Example: `  $ snowplow-cli dp publish
  $ snowplow-cli dp download ./my-data-products`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		ghOut, _ := cmd.Flags().GetBool("gh-annotate")
		managedFrom, _ := cmd.Flags().GetString("managed-from")

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

		basePath, err := os.Getwd()
		if err != nil {
			snplog.LogFatal(err)
		}

		cnx := context.Background()

		c, err := console.NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			snplog.LogFatal(err)
		}

		changes, err := publish.FindChanges(cnx, c, files)
		if err != nil {
			snplog.LogFatal(err)
		}

		publish.LockChanged(changes, managedFrom)

		validation.Validate(cnx, c, files, searchPaths, basePath, ghOut, false, changes.IdToFileName)

		err = publish.Publish(cnx, c, changes, dryRun)
		if err != nil {
			snplog.LogFatal(err)
		}

	},
}

func init() {
	DataProductsCmd.AddCommand(publishCommand)
	publishCommand.PersistentFlags().Bool("gh-annotate", false, "Output suitable for github workflow annotation (ignores -s)")
	publishCommand.PersistentFlags().BoolP("dry-run", "d", false, "Only print planned changes without performing them")
}
