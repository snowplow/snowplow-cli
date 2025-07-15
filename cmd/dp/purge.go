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

	"github.com/snowplow/snowplow-cli/internal/console"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/publish"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
)

type purgeApi struct {
	client *console.ApiClient
	cnx    context.Context
}

func (a purgeApi) DeleteSourceApp(sa console.RemoteSourceApplication) error {
	return console.DeleteSourceApp(a.cnx, a.client, sa)
}

func (a purgeApi) DeleteDataProduct(dp console.RemoteDataProduct) error {
	return console.DeleteDataProduct(a.cnx, a.client, dp)
}

func (a purgeApi) FetchDataProduct() (*console.DataProductsAndRelatedResources, error) {
	return console.GetDataProductsAndRelatedResources(a.cnx, a.client)
}

var purgeCommand = &cobra.Command{
	Use:   "purge {directory ./data-products}",
	Short: "Purges (permanently removes) all remote data products and source apps that do not exist locally",
	Long: `Purges (permanently removes) all remote data products and source apps that do not exist locally.

If no directory is provided then defaults to 'data-products' in the current directory. Source apps are stored in the nested 'source-apps' directory`,
	Example: `  $ snowplow-cli dp purge
  $ snowplow-cli dp purge ./my-data-products`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")
		yes, _ := cmd.Flags().GetBool("yes")

		searchPaths := []string{}

		if len(args) == 0 {
			searchPaths = append(searchPaths, util.DataProductsFolder)
			slog.Debug("purge", "msg", fmt.Sprintf("no path provided, using default (./%s)", util.DataProductsFolder))
		}

		searchPaths = append(searchPaths, args...)

		files, err := util.MaybeResourcesfromPaths(searchPaths)
		if err != nil {
			snplog.LogFatal(err)
		}

		cnx := context.Background()

		c, err := console.NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			snplog.LogFatal(err)
		}

		err = publish.Purge(cnx, purgeApi{c, cnx}, files, yes)
		if err != nil {
			snplog.LogFatal(err)
		}

	},
}

func init() {
	DataProductsCmd.AddCommand(purgeCommand)
	purgeCommand.PersistentFlags().BoolP("yes", "y", false, "commit to purge")
}
