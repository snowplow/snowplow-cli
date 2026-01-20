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

	"github.com/snowplow/snowplow-cli/internal/console"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/release"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/snowplow/snowplow-cli/internal/validation"
	"github.com/spf13/cobra"
)

var releaseCommand = &cobra.Command{
	Use:   "release {directory ./data-products}",
	Short: "Publish and release all data products, event specs and source apps to CDI Console",
	Long: `Publish and release the local versions of all data products, event specs and source apps to CDI Console.

This command syncs local files with remote data products, then releases any draft event specs. It will filter out the event specs without the event, and only attempt to publish the event specs that are part of the data products in the directory.
Releasing marks event specs as published and enables event spec inference.
Use 'sync' to only sync without releasing, and not change the status of event specs

If no directory is provided then defaults to 'data-products' in the current directory. Source apps are stored in the nested 'source-apps' directory`,
	Example: `  $ snowplow-cli dp release
  $ snowplow-cli dp release ./my-data-products`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		ghOut, _ := cmd.Flags().GetBool("gh-annotate")
		managedFrom, _ := cmd.Flags().GetString("managed-from")
		concurrentReq, _ := cmd.Flags().GetInt("concurrency")

		if concurrentReq > 10 {
			concurrentReq = 10
			slog.Debug("validation", "msg", "concurrency set to > 10, limited to 10")
		}

		if concurrentReq < 1 {
			concurrentReq = 1
			slog.Debug("validation", "msg", "concurrency set to < 1, increased to 1")
		}

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

		changes, err := release.FindChanges(cnx, c, files)
		if err != nil {
			snplog.LogFatal(err)
		}

		release.LockChanged(changes, managedFrom)

		err = validation.Validate(cnx, c, files, searchPaths, basePath, ghOut, false, changes.IdToFileName, concurrentReq)
		if err != nil {
			snplog.LogFatal(err)
		}

		err = release.Release(cnx, c, changes, dryRun)
		if err != nil {
			snplog.LogFatal(err)
		}
	},
}

func init() {
	DataProductsCmd.AddCommand(releaseCommand)
	releaseCommand.PersistentFlags().Bool("gh-annotate", false, "Output suitable for github workflow annotation (ignores -s)")
	releaseCommand.PersistentFlags().BoolP("dry-run", "d", false, "Only print planned changes without performing them")
	releaseCommand.PersistentFlags().IntP("concurrency", "c", 3, "The number of validation requests to perform at once (maximum 10)")
}
