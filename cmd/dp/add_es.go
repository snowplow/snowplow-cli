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

	"github.com/snowplow/snowplow-cli/internal/amend"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/spf13/cobra"
)

var addEsCmd = &cobra.Command{
	Use:     "add-event-spec {path}",
	Short:   "Add new event spec to an existing data product",
	Aliases: []string{"add-es"},
	Args:    cobra.ExactArgs(1),
	Long: `Adds one or more event specifications to an existing data product file.
The command takes the path to a data product file and adds the specified event specifications to it.
The command will attempt to keep the formatting and comments of the original file intact, but it's a best effort approach. Some comments might be deleted, some formatting changes might occur.`,
	Example: `  $ snowplow-cli dp add-event-spec --event-spec user_login --event-spec page_view ./my-data-product.yaml
  $ snowplow-cli dp add-es ./data-products/analytics.yaml -e "checkout_completed" -e "item_purchased"`,
	Run: func(cmd *cobra.Command, args []string) {

		esNames, _ := cmd.Flags().GetStringArray("event-spec")

		dpFilePath := args[0]

		if err := amend.AddEventSpecsToFile(esNames, dpFilePath); err != nil {
			snplog.LogFatal(err)
		}

		slog.Info("Successfully added event specifications", "count", len(esNames), "file", dpFilePath)

	},
}

func init() {
	DataProductsCmd.AddCommand(addEsCmd)
	addEsCmd.Flags().StringArrayP("event-spec", "e", []string{}, "Name of event spec to add")
	addEsCmd.MarkFlagsOneRequired("event-spec")
}
