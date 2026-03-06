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
	Short: "Sync data products, event specs and source apps to Snowplow Console, then release event specs",
	Long: `Sync data products, event specs and source apps to Snowplow Console, then release event specs.

This command runs 'sync' first, then releases the event specs.
Releasing sets the event spec status to 'published' and pushes them to the pipeline, enabling event spec inference.
Only event specs that exist locally will be released. Each event spec must have an event defined, and all referenced events and entities must be published to the production environment.

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
