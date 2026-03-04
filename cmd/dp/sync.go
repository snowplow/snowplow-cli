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

var syncCommand = &cobra.Command{
	Use:     "sync {directory ./data-products}",
	Aliases: []string{"publish"},
	Short:   "Sync all data products, event specs and source apps to CDI Console",
	Long: `Sync the local versions of all data products, event specs and source apps to CDI Console.

This command syncs local files with remote data products event specs and source apps, creating or updating them as needed.
Event specs status will not be changed by running this command.
Use 'release' to also release event specs, which changes the status in CDI Console to "published" and enable event spec inference.

If no directory is provided then defaults to 'data-products' in the current directory. Source apps are stored in the nested 'source-apps' directory`,
	Example: `  $ snowplow-cli dp sync
  $ snowplow-cli dp sync ./my-data-products`,
	Run: func(cmd *cobra.Command, args []string) {
		runDpWorkflow(cmd, args, func(cnx context.Context, c *console.ApiClient, changes *release.DataProductChangeSet, dryRun bool) error {
			return release.Sync(cnx, c, changes, dryRun, false)
		})
	},
}

func init() {
	DataProductsCmd.AddCommand(syncCommand)
	addCommonDpFlags(syncCommand)
}
