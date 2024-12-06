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
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	snplog "github.com/snowplow-product/snowplow-cli/internal/logging"
	"github.com/snowplow-product/snowplow-cli/internal/model"
	"github.com/snowplow-product/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:     "generate [paths...]",
	Short:   "Generate new data products and source applications locally",
	Aliases: []string{"gen"},
	Args:    cobra.NoArgs,
	Long: `Will write new data products and/or source application to file based on the arguments provided.

Example:
  $ snowplow-cli dp gen --source-app "Mobile app"
  Will result in a new source application getting written to './data-products/source-applications/mobile-app.yaml'

  $ snowplow-cli dp gen --data-product "Ad tracking" --output-format json --data-products-directory dir1
  Will result in a new data product getting written to './dir1/ad-tracking.json'
`,
	Example: `  $ snowplow-cli dp generate --source-app "Mobile app" --source-app "Web app" --data-product "Signup flow"`,
	Run: func(cmd *cobra.Command, args []string) {
		outFmt, _ := cmd.Flags().GetString("output-format")

		sourceAppDirectory, _ := cmd.Flags().GetString("source-apps-directory")
		sourceApps, _ := cmd.Flags().GetStringArray("source-app")

		dataproductDirectory, _ := cmd.Flags().GetString("data-products-directory")
		dataProducts, _ := cmd.Flags().GetStringArray("data-product")

		err := os.MkdirAll(sourceAppDirectory, os.ModePerm)
		if err != nil && !os.IsExist(err) {
			snplog.LogFatal(err)
		}
		err = os.MkdirAll(dataproductDirectory, os.ModePerm)
		if err != nil && !os.IsExist(err) {
			snplog.LogFatal(err)
		}

		for _, app := range sourceApps {
			appn := util.ResourceNameToFileName(app)
			fpath, err := util.WriteSerializableToFile(buildSaTpl(app), sourceAppDirectory, appn, outFmt)
			if err != nil {
				snplog.LogFatal(err)
			}
			slog.Info("generate", "msg", "wrote", "kind", "source app", "file", fpath)
		}

		for _, app := range dataProducts {
			appn := util.ResourceNameToFileName(app)
			if err != nil {
				snplog.LogFatal(err)
			}
			fpath, err := util.WriteSerializableToFile(buildDpTpl(app), dataproductDirectory, appn, outFmt)
			if err != nil {
				snplog.LogFatal(err)
			}
			slog.Info("generate", "msg", "wrote", "kind", "data product", "file", fpath)
		}

	},
}

func buildDpTpl(name string) model.CliResource[model.DataProductCanonicalData] {
	return model.CliResource[model.DataProductCanonicalData]{
		ApiVersion:   "v1",
		ResourceType: "data-product",
		ResourceName: uuid.NewString(),
		Data: model.DataProductCanonicalData{
			Name:                name,
			SourceApplications:  []model.Ref{},
			EventSpecifications: []model.EventSpecCanonical{},
		},
	}
}

func buildSaTpl(name string) model.CliResource[model.SourceAppData] {
	return model.CliResource[model.SourceAppData]{
		ApiVersion:   "v1",
		ResourceType: "source-application",
		ResourceName: uuid.NewString(),
		Data: model.SourceAppData{
			Name:     name,
			AppIds:   []string{},
			Entities: &model.EntitiesDef{},
		},
	}
}

func init() {
	DataProductsCmd.AddCommand(generateCmd)

	generateCmd.Flags().String("output-format", "yaml", "File format (yaml|json)")
	generateCmd.Flags().String("data-products-directory", util.DataProductsFolder, "Directory to write data products to")
	generateCmd.Flags().String("source-apps-directory", filepath.Join(util.DataProductsFolder, util.SourceAppsFolder), "Directory to write source apps to")

	generateCmd.Flags().StringArray("source-app", []string{}, "Name of source app to generate")
	generateCmd.Flags().StringArray("data-product", []string{}, "Name of data product to generate")

	generateCmd.MarkFlagsOneRequired("source-app", "data-product")
}
