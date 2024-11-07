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

	"log/slog"

	"github.com/snowplow-product/snowplow-cli/internal/console"
	snplog "github.com/snowplow-product/snowplow-cli/internal/logging"
	"github.com/snowplow-product/snowplow-cli/internal/model"
	"github.com/snowplow-product/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
)

var downloadCommand = &cobra.Command{
	Use:   "download {directory ./data-structures}",
	Short: "Donwload all data products, event specs and source apps from BDP Console",
	Args:  cobra.MaximumNArgs(1),
	Long: `Downloads the latest versions of all data products, event specs and source apps from BDP Console.

If no directory is provided then defaults to 'data-products' in the current directory. Source apps are stored in the nested 'source-apps' directory`,
	Example: `  $ snowplow-cli dp download
  $ snowplow-cli dp download./my-data-products`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")
		format, _ := cmd.Flags().GetString("output-format")

		dataProductsFolder := util.DataProductsFolder

		if len(args) != 0 {
			dataProductsFolder = args[0]
		}

		files := util.Files{DataProductsLocation: dataProductsFolder, SourceAppsLocation: util.SourceAppsFolder, ExtentionPreference: format}
		cnx := context.Background()

		c, err := console.NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			snplog.LogFatal(err)
		}

		res, err := console.GetDataProductsAndRelatedResources(cnx, c)
		if err != nil {
			snplog.LogFatal(err)
		}

		var sas []model.CliResource[model.SourceAppData]
		for _, sa := range res.SourceApplication {
			model := model.CliResource[model.SourceAppData]{
				ResourceType: "source-application",
				ApiVersion:   "v1",
				ResourceName: sa.Id,
				Data:         sa.ToCanonical(),
			}
			sas = append(sas, model)
		}

		fileNameToSa, err := files.CreateSourceApps(sas)
		if err != nil {
			snplog.LogFatal(err)
		}

		slog.Info("wrote source applications", "count", len(sas))

		var saIdToPath = make(map[string]string)
		for path, sa := range fileNameToSa {
			saIdToPath[sa.ResourceName] = path
		}

		var esIdToRes = make(map[string]console.RemoteEventSpec)
		for _, sa := range res.TrackingScenarios {
			esIdToRes[sa.Id] = sa
		}

		var dps []model.CliResource[model.DataProductCanonicalData]
		for _, dp := range res.DataProducts {
			model := model.CliResource[model.DataProductCanonicalData]{
				ResourceType: "data-product",
				ApiVersion:   "v1",
				ResourceName: dp.Id,
				Data:         dp.ToCanonical(saIdToPath, esIdToRes , files.DataProductsLocation),
			}
			dps = append(dps, model)
		}

		_, err = files.CreateDataProducts(dps)
		if err != nil {
			snplog.LogFatal(err)
		}

		slog.Info("wrote data products", "count", len(dps))

	},
}

func init() {
	DataProductsCmd.AddCommand(downloadCommand)

	downloadCommand.PersistentFlags().StringP("output-format", "f", "yaml", "Format of the files to read/write. json or yaml are supported")
}
