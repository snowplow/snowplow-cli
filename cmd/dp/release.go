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

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/release"
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
		runDpWorkflow(cmd, args, func(cnx context.Context, c *console.ApiClient, changes *release.DataProductChangeSet, dryRun bool) error {
			return release.Release(cnx, c, changes, dryRun)
		})
	},
}

func init() {
	DataProductsCmd.AddCommand(releaseCommand)
	addCommonDpFlags(releaseCommand)
}
