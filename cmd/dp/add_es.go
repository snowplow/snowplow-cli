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
	"path/filepath"

	"github.com/snowplow/snowplow-cli/internal/amend"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
)

var addEsCmd = &cobra.Command{
	Use:     "add-event-spec {path}",
	Short:   "Add new event spec to an existing data product",
	Aliases: []string{"add-es"},
	Args:    cobra.ExactArgs(1),
	Long: `Adds one or more event specifications to an existing data product file.

The command takes the path to a data product file and adds the specified event specifications to it.
Event specifications must exist in the data products directory before they can be added.
Please note that the path is relative to the configured data-products directory, that defaults to ./data-products`,
	Example: `  $ snowplow-cli dp add-event-spec my-data-product.yaml --event-specs "user_login" --event-specs "page_view"
  $ snowplow-cli dp add-es ./products/analytics.yaml -e "checkout_completed" -e "item_purchased"`,
	Run: func(cmd *cobra.Command, args []string) {

		esNames, _ := cmd.Flags().GetStringArray("event-specs")
		dataproductDirectory, _ := cmd.Flags().GetString("data-products-directory")

		dpFilePath := args[0]
		dpFullPath := filepath.Join(dataproductDirectory, dpFilePath)

		if err := amend.AddEventSpecsToFile(esNames, dataproductDirectory, dpFilePath); err != nil {
			snplog.LogFatal(err)
		}

		slog.Info("Successfully added event specifications", "count", len(esNames), "file", dpFullPath)

	},
}

func init() {
	DataProductsCmd.AddCommand(addEsCmd)
	addEsCmd.Flags().StringArrayP("event-specs", "e", []string{}, "Name of event spec to add")
	addEsCmd.Flags().String("data-products-directory", util.DataProductsFolder, "Directory to write data products to")
}
