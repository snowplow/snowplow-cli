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

	"github.com/snowplow/snowplow-cli/internal/console"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download {directory ./data-structures}",
	Short: "Download all data structures from BDP Console",
	Args:  cobra.MaximumNArgs(1),
	Long: `Downloads the latest versions of all data structures from BDP Console.

Will retrieve schema contents from your development environment.
If no directory is provided then defaults to 'data-structures' in the current directory.

By default, data structures with empty schemaType (legacy format) are skipped.
Use --include-legacy to include them (they will be set to 'entity' schemaType).`,
	Example: `  $ snowplow-cli ds download

  Download data structures matching com.example/event_name* or com.example.subdomain*
  $ snowplow-cli ds download --match com.example/event_name --match com.example.subdomain

  Download with custom output format and directory
  $ snowplow-cli ds download --output-format json ./my-data-structures

  Include legacy data structures with empty schemaType
  $ snowplow-cli ds download --include-legacy`,
	Run: func(cmd *cobra.Command, args []string) {
		dataStructuresFolder := util.DataStructuresFolder
		if len(args) > 0 {
			dataStructuresFolder = args[0]
		}
		format, _ := cmd.Flags().GetString("output-format")
		match, _ := cmd.Flags().GetStringArray("match")
		includeLegacy, _ := cmd.Flags().GetBool("include-legacy")
		plain, _ := cmd.Flags().GetBool("plain")
		files := util.Files{DataStructuresLocation: dataStructuresFolder, ExtentionPreference: format}

		includeDrafts, err := cmd.Flags().GetBool("include-drafts")
		if err != nil {
			snplog.LogFatalMsg("failed to include drafts", err)
		}

		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")

		cnx := context.Background()

		c, err := console.NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			snplog.LogFatalMsg("client creation fail", err)
		}

		dss, err := console.GetAllDataStructures(cnx, c, match, includeLegacy)
		if err != nil {
			snplog.LogFatalMsg("data structure fetch failed", err)
		}

		allDss := dss

		if includeDrafts {
			dssDrafts, err := console.GetAllDataStructuresDrafts(cnx, c, match)
			if err != nil {
				snplog.LogFatalMsg("data structure drafts fetch failed", err)
			}
			allDss = append(allDss, dssDrafts...)

			slog.Info("wrote data structures drafts", "count", len(dssDrafts))

		}

		err = files.CreateDataStructures(allDss, plain)
		if err != nil {
			snplog.LogFatal(err)
		}

		slog.Info("wrote data structures", "count", len(dss))
	},
}

func init() {
	DataStructuresCmd.AddCommand(downloadCmd)

	downloadCmd.PersistentFlags().Bool("include-drafts", false, "Include drafts data structures")

	downloadCmd.PersistentFlags().StringP("output-format", "f", "yaml", "Format of the files to read/write. json or yaml are supported")
	downloadCmd.PersistentFlags().StringArrayP("match", "", []string{}, "Match for specific data structure to download (eg. --match com.example/event_name or --match com.example)")
	downloadCmd.PersistentFlags().Bool("include-legacy", false, "Include legacy data structures with empty schemaType (will be set to 'entity')")
	downloadCmd.PersistentFlags().Bool("plain", false, "Don't include any comments in yaml files")
}
