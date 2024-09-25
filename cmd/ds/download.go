/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package ds

import (
	"context"
	"log/slog"

	"github.com/snowplow-product/snowplow-cli/internal/console"
	. "github.com/snowplow-product/snowplow-cli/internal/logging"
	"github.com/snowplow-product/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download {directory ./data-structures}",
	Short: "Download all data structures from BDP Console",
	Args:  cobra.MaximumNArgs(1),
	Long: `Downloads the latest versions of all data structures from BDP Console.

Will retrieve schema contents from your development environment.
If no directory is provided then defaults to 'data-structures' in the current directory.`,
	Example: `  $ snowplow-cli ds download
  $ snowplow-cli ds download --output-format json ./my-data-structures`,
	Run: func(cmd *cobra.Command, args []string) {
		dataStructuresFolder := util.DataStructuresFolder
		if len(args) > 0 {
			dataStructuresFolder = args[0]
		}
		format, _ := cmd.Flags().GetString("output-format")
		files := util.Files{DataStructuresLocation: dataStructuresFolder, ExtentionPreference: format}

		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key-secret")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")

		cnx := context.Background()

		c, err := console.NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			LogFatalMsg("client creation fail", err)
		}

		dss, err := console.GetAllDataStructures(cnx, c)
		if err != nil {
			LogFatalMsg("data structure fetch failed", err)
		}

		err = files.CreateDataStructures(dss)
		if err != nil {
			LogFatal(err)
		}

		slog.Info("wrote data structures", "count", len(dss))
	},
}

func init() {
	DataStructuresCmd.AddCommand(downloadCmd)

	downloadCmd.PersistentFlags().StringP("output-format", "f", "yaml", "Format of the files to read/write. json or yaml are supported")
}
