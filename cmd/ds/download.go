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
	"fmt"
	"log/slog"

	"github.com/snowplow/snowplow-cli/internal/console"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/model"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download {directory ./data-structures}",
	Short: "Download data structures from BDP Console",
	Args:  cobra.MaximumNArgs(1),
	Long: `Downloads data structures from BDP Console.

By default, downloads the latest versions of all data structures from your development environment.
If no directory is provided then defaults to 'data-structures' in the current directory.

By default, data structures with empty schemaType (legacy format) are skipped.
Use --include-legacy to include them (they will be set to 'entity' schemaType).

You can download specific data structures using --vendor, --name, and --format flags.
You can also download a specific version using --version flag, or all versions using --all-versions flag.
Use --env flag to filter deployments by environment (DEV, PROD).`,
	Example: `  $ snowplow-cli ds download

  Download data structures matching com.example/event_name* or com.example.subdomain*
  $ snowplow-cli ds download --match com.example/event_name --match com.example.subdomain

  Download with custom output format and directory
  $ snowplow-cli ds download --output-format json ./my-data-structures

  Include legacy data structures with empty schemaType
  $ snowplow-cli ds download --include-legacy

  Download a specific data structure
  $ snowplow-cli ds download --vendor com.example --name login_click --format jsonschema

  Download a specific version of a data structure
  $ snowplow-cli ds download --vendor com.example --name login_click --format jsonschema --version 1-0-0

  Download all versions of a data structure
  $ snowplow-cli ds download --vendor com.example --name login_click --format jsonschema --all-versions

  Download only production deployments
  $ snowplow-cli ds download --vendor com.example --name login_click --format jsonschema --all-versions --env PROD`,
	Run: func(cmd *cobra.Command, args []string) {
		dataStructuresFolder := util.DataStructuresFolder
		if len(args) > 0 {
			dataStructuresFolder = args[0]
		}
		format, _ := cmd.Flags().GetString("output-format")
		match, _ := cmd.Flags().GetStringArray("match")
		includeLegacy, _ := cmd.Flags().GetBool("include-legacy")
		plain, _ := cmd.Flags().GetBool("plain")

		// Flags for specific data structure download
		vendor, _ := cmd.Flags().GetString("vendor")
		name, _ := cmd.Flags().GetString("name")
		formatFlag, _ := cmd.Flags().GetString("format")
		version, _ := cmd.Flags().GetString("version")
		allVersions, _ := cmd.Flags().GetBool("all-versions")
		env, _ := cmd.Flags().GetString("env")

		files := util.Files{DataStructuresLocation: dataStructuresFolder, ExtentionPreference: format}

		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")

		cnx := context.Background()

		c, err := console.NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			snplog.LogFatalMsg("client creation fail", err)
		}

		var dss []model.DataStructure

		// Check if we're downloading a specific data structure
		var includeVersions bool
		if vendor != "" && name != "" && formatFlag != "" {
			// Validate mutually exclusive flags
			if version != "" && allVersions {
				snplog.LogFatalMsg("validation error", fmt.Errorf("--version and --all-versions are mutually exclusive"))
			}

			// Generate hash for the specific data structure
			dsHash := console.GenerateDataStructureHash(org, vendor, name, formatFlag)

			if allVersions {
				// Download all versions
				dss, err = console.GetAllDataStructureVersions(cnx, c, dsHash, env)
				if err != nil {
					snplog.LogFatalMsg("failed to fetch all data structure versions", err)
				}
				slog.Info("downloaded all versions", "vendor", vendor, "name", name, "count", len(dss), "env", env)
				includeVersions = true
			} else if version != "" {
				// Download specific version
				ds, err := console.GetSpecificDataStructureVersion(cnx, c, dsHash, version)
				if err != nil {
					snplog.LogFatalMsg("failed to fetch specific data structure version", err)
				}
				dss = []model.DataStructure{*ds}
				slog.Info("downloaded specific version", "vendor", vendor, "name", name, "version", version)
				includeVersions = true
			} else {
				// Download latest version
				ds, err := console.GetSpecificDataStructure(cnx, c, dsHash)
				if err != nil {
					snplog.LogFatalMsg("failed to fetch specific data structure", err)
				}
				dss = []model.DataStructure{*ds}
				slog.Info("downloaded specific data structure", "vendor", vendor, "name", name)
				includeVersions = false // Latest version doesn't need version suffix
			}
		} else {
			// Download all data structures
			dss, err = console.GetAllDataStructures(cnx, c, match, includeLegacy)
			if err != nil {
				snplog.LogFatalMsg("data structure fetch failed", err)
			}
			includeVersions = false // Bulk download gets latest versions without version suffix
		}

		err = files.CreateDataStructuresWithVersions(dss, plain, includeVersions)
		if err != nil {
			snplog.LogFatal(err)
		}

		slog.Info("wrote data structures", "count", len(dss))
	},
}

func init() {
	DataStructuresCmd.AddCommand(downloadCmd)

	downloadCmd.PersistentFlags().StringP("output-format", "f", "yaml", "Format of the files to read/write. json or yaml are supported")
	downloadCmd.PersistentFlags().StringArrayP("match", "", []string{}, "Match for specific data structure to download (eg. --match com.example/event_name or --match com.example)")
	downloadCmd.PersistentFlags().Bool("include-legacy", false, "Include legacy data structures with empty schemaType (will be set to 'entity')")
	downloadCmd.PersistentFlags().Bool("plain", false, "Don't include any comments in yaml files")

	// New flags for specific data structure download
	downloadCmd.PersistentFlags().String("vendor", "", "Vendor of the specific data structure to download (requires --name and --format)")
	downloadCmd.PersistentFlags().String("name", "", "Name of the specific data structure to download (requires --vendor and --format)")
	downloadCmd.PersistentFlags().String("format", "jsonschema", "Format of the specific data structure to download (requires --vendor and --name)")
	downloadCmd.PersistentFlags().String("version", "", "Specific version of the data structure to download (optional, defaults to latest)")
	downloadCmd.PersistentFlags().Bool("all-versions", false, "Download all versions of the data structure (mutually exclusive with --version)")
	downloadCmd.PersistentFlags().String("env", "", "Filter deployments by environment (DEV, PROD) - only applies to --all-versions")
}
