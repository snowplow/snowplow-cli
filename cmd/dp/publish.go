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

	"github.com/snowplow-product/snowplow-cli/internal/console"
	snplog "github.com/snowplow-product/snowplow-cli/internal/logging"
	"github.com/spf13/cobra"
)

var publishCommand = &cobra.Command{
	Use:   "publish {directory ./data-products}",
	Short: "Publish all data products, event specs and source apps to BDP Console",
	Args:  cobra.MaximumNArgs(1),
	Long: `Publish the local version versions of all data products, event specs and source apps from BDP Console.

If no directory is provided then defaults to 'data-products' in the current directory. Source apps are stored in the nested 'source-apps' directory`,
	Example: `  $ snowplow-cli dp publish
  $ snowplow-cli dp download ./my-data-products`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")

		cnx := context.Background()
		// stub for future logic

		c, err := console.NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			snplog.LogFatal(err)
		}

		_, err = console.GetDataProductsAndRelatedResources(cnx, c)

		if err != nil {
			snplog.LogFatal(err)
		}

	},
}

func init() {
	DataProductsCmd.AddCommand(publishCommand)
}
